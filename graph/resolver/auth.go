package resolver

import (
	"asan/graph/db"
	"asan/graph/middleware/auth"
	"asan/graph/middleware/header"
	"asan/graph/model"
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

func LogIn(ctx context.Context, email string, password string) (*model.AuthResponse, error) {
	pg := db.GetPostgresPool()
	rows, _ := pg.Query(
		context.Background(),
		`SELECT id, hashed_password FROM users WHERE email=$1 LIMIT 1;`,
		email)
	p, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[model.User])
	if err == pgx.ErrNoRows {
		return nil, errors.New("invalid credentials")
	}
	err1 := bcrypt.CompareHashAndPassword([]byte(p.HashedPassword), []byte(password))
	if err1 != nil {
		return nil, errors.New("invalid credentials")
	}
	refreshToken := generateRandomString()
	refreshTokenTtl, err2 := strconv.Atoi(os.Getenv("REFRESH_TOKEN_TTL"))
	if err2 != nil {
		panic(errors.New("refresh token time to live not spcecified"))
	}
	accessTokenTtl, err3 := strconv.Atoi(os.Getenv("ACCESS_TOKEN_TTL"))
	if err3 != nil {
		panic(errors.New("refresh token time to live not spcecified"))
	}
	pg.Exec(
		context.Background(),
		`INSERT INTO refresh_tokens (token, user_id, expiry_time) VALUES ($1, $2, $3)`,
		refreshToken,
		p.ID,
		time.Now().Add(time.Second*time.Duration(refreshTokenTtl)))
	accessToken := createJwt(p.ID)
	headers := header.ForContext(ctx).ResHeader
	headers.Add("Set-Cookie", (&http.Cookie{
		Name:     os.Getenv("REFRESH_TOKEN_COOKIE"),
		Value:    refreshToken,
		Path:     "/",
		MaxAge:   refreshTokenTtl,
		HttpOnly: true,
	}).String())
	headers.Add("Set-Cookie", (&http.Cookie{
		Name:     os.Getenv("ACCESS_TOKEN_COOKIE"),
		Value:    accessToken,
		Path:     "/",
		MaxAge:   accessTokenTtl,
		HttpOnly: true,
	}).String())
	return &model.AuthResponse{
		AccessToken:     accessToken,
		AccessTokenTTL:  accessTokenTtl,
		RefreshToken:    refreshToken,
		RefreshTokenTTL: refreshTokenTtl,
	}, nil
}

func SignUp(input model.SignUpPayload) (bool, error) {
	pg := db.GetPostgresPool()
	if input.Password != input.ConfirmPassword {
		return false, errors.New("passwords don't match")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), 4)
	if err != nil {
		panic(err)
	}
	var id string
	if err := pg.QueryRow(
		context.Background(),
		"insert into users (email, hashed_password) values ($1, $2) returning id;",
		input.Email,
		string(hashed[:])).Scan(&id); err != nil {
		if err.(*pgconn.PgError).Code == pgerrcode.UniqueViolation {
			return false, errors.New("email is used")
		}
		return false, err
	}
	go sendVerificationEmail(input.Email)
	return true, nil
}

func RefreshToken(ctx context.Context, refreshToken *string) (*model.AuthResponse, error) {
	// Refresh token can be passed in the request or in the cookie
	foundRefreshToken := ""
	if refreshToken == nil {
		foundRefreshToken = auth.GetRefreshTokenCookie(ctx)
	} else {
		foundRefreshToken = *refreshToken
	}

	newRefreshToken := generateRandomString()
	refreshTokenTtl, err := strconv.Atoi(os.Getenv("REFRESH_TOKEN_TTL"))
	if err != nil {
		panic(errors.New("refresh token time to live not spcecified"))
	}
	var id string
	if err := db.GetPostgresPool().QueryRow(
		context.Background(),
		`UPDATE refresh_tokens SET token = $1, expiry_time = $2 WHERE token = $3 AND expiry_time >= NOW() RETURNING user_id;`,
		newRefreshToken,
		time.Now().Add(time.Second*time.Duration(refreshTokenTtl)),
		foundRefreshToken).Scan(&id); err != nil {
		return nil, errors.New("invalid refresh token")
	}
	accessToken := createJwt(id)
	accessTokenTtl, _ := strconv.Atoi(os.Getenv("ACCESS_TOKEN_TTL"))

	headers := header.ForContext(ctx).ResHeader
	headers.Add("Set-Cookie", (&http.Cookie{
		Name:     os.Getenv("REFRESH_TOKEN_COOKIE"),
		Value:    newRefreshToken,
		Path:     "/",
		MaxAge:   refreshTokenTtl,
		HttpOnly: true,
	}).String())
	headers.Add("Set-Cookie", (&http.Cookie{
		Name:     os.Getenv("ACCESS_TOKEN_COOKIE"),
		Value:    accessToken,
		Path:     "/",
		MaxAge:   accessTokenTtl,
		HttpOnly: true,
	}).String())
	return &model.AuthResponse{
		AccessToken:     accessToken,
		AccessTokenTTL:  accessTokenTtl,
		RefreshToken:    newRefreshToken,
		RefreshTokenTTL: refreshTokenTtl,
	}, nil
}

func HandleForgotPassword(ctx context.Context, email string) (bool, error) {
	go sendForgotPasswordEmail(email)
	return true, nil
}

func LogOut(ctx context.Context, refreshToken *string) (bool, error) {
	// Refresh token can be passed in the request or in the cookie
	foundRefreshToken := ""
	if refreshToken == nil {
		foundRefreshToken = auth.GetRefreshTokenCookie(ctx)
	} else {
		foundRefreshToken = *refreshToken
	}

	if foundRefreshToken != "" {
		db.GetPostgresPool().Exec(context.Background(),
			`DELETE FROM refresh_tokens WHERE token = $1`, foundRefreshToken)
	}

	resHeader := header.ForContext(ctx).ResHeader

	resHeader.Add("Set-Cookie", (&http.Cookie{
		Name:     os.Getenv("REFRESH_TOKEN_COOKIE"),
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}).String())

	resHeader.Add("Set-Cookie", (&http.Cookie{
		Name:     os.Getenv("ACCESS_TOKEN_COOKIE"),
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}).String())

	return true, nil
}

func createJwt(userId string) string {
	key := []byte(os.Getenv("JWT_KEY"))
	accessTokenTtl, err := strconv.Atoi(os.Getenv("ACCESS_TOKEN_TTL"))
	if err != nil {
		panic(errors.New("access token time to live not spcecified"))
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			Subject:   userId,
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Second * time.Duration(accessTokenTtl))},
		})
	s, err := t.SignedString(key)
	if err != nil {
		panic(err)
	}
	return s
}

func sendVerificationEmail(email string) error {
	return sendEmail(
		[]string{email},
		"Email Verification",
		struct{ Name string }{Name: "Testing123"},
		[]string{"graph/email_templates/verification.html"})
}

func sendForgotPasswordEmail(email string) error {
	return sendEmail(
		[]string{email},
		"Forgot Password",
		struct{ Name string }{Name: "Testing123"},
		[]string{"graph/email_templates/verification.html"})
}

func sendEmail(to []string, subject string, data any, templates []string) error {
	from := os.Getenv("NOREPLY_EMAIL_ADDRESS")
	auth := smtp.PlainAuth("", from, os.Getenv("NOREPLY_EMAIL_PASSWORD"), "smtp.gmail.com")
	t, err1 := template.ParseFiles(strings.Join(templates[:], ","))
	if err1 != nil {
		panic(err1)
	}
	var body bytes.Buffer
	mimeHeaders := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body.Write([]byte(fmt.Sprintf("Subject: %s \n%s\n\n", subject, mimeHeaders)))
	t.Execute(&body, data)
	err := smtp.SendMail(
		"smtp.gmail.com:587",
		auth,
		from,
		to,
		body.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func generateRandomString() string {
	return strings.Replace(uuid.New().String(), "-", "", -1)
}
