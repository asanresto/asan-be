package resolver

import (
	"asan/graph/db"
	authMiddleware "asan/graph/middleware/auth"
	"asan/graph/model"
	"asan/graph/storage"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"os"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/minio/minio-go/v7"
	"golang.org/x/crypto/bcrypt"
)

func Me(ctx context.Context) (*model.User, error) {
	userId := authMiddleware.ForContext(ctx)
	if userId == "" {
		return nil, errors.New("unauthorized")
	}
	preloads := getNestedPreloads(
		graphql.GetOperationContext(ctx),
		graphql.CollectFieldsCtx(ctx, nil),
		"",
		[]string{""},
		0,
	)
	rows, _ := db.GetPostgresPool().Query(
		context.Background(),
		fmt.Sprintf(`SELECT %s FROM "users" WHERE id=$1;`, strings.Join(preloads[:], ",")),
		userId)
	p, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[model.User])
	if err != nil {
		return nil, nil
	}
	return &p, nil
}

func HandleUsers(ctx context.Context) ([]*model.User, error) {
	userId := authMiddleware.ForContext(ctx)
	if userId == "" {
		return nil, errors.New("unauthorized")
	}
	preloads := GetPreloads(ctx, []string{""})
	rows, _ := db.GetPostgresPool().Query(context.Background(),
		fmt.Sprintf(`SELECT %s FROM users WHERE id!=$1;`, strings.Join(preloads[:], ",")),
		userId)
	users, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (*model.User, error) {
		return pgx.RowToAddrOfStructByNameLax[model.User](row)
	})
	return users, err
}

func UpdateAccount(ctx context.Context, avatar *graphql.Upload, name *string) (bool, error) {
	userId := authMiddleware.ForContext(ctx)
	if userId == "" {
		return false, errors.New("unauthorized")
	}
	var newAvatarUrl *string
	var currentAvatarUrl *string
	if avatar != nil {
		// Check if the user already has an avatar
		err := db.GetPostgresPool().QueryRow(ctx, `SELECT "avatar_url" FROM "users" WHERE "id"=$1;`, userId).Scan(&currentAvatarUrl)
		if err != nil {
			return false, err
		}
		var avatarObjectName string
		if currentAvatarUrl != nil {
			segments := strings.Split(*currentAvatarUrl, "/")
			avatarObjectName = segments[len(segments)-1]
		} else {
			avatarObjectName = uuid.New().String()
		}
		_, err = storage.Upload(ctx, avatarObjectName, avatar.File, avatar.Size, minio.PutObjectOptions{ContentType: avatar.ContentType})
		if err != nil {
			return false, err
		}
		// User didn't have an avatar
		if currentAvatarUrl == nil {
			url, err := url.JoinPath(os.Getenv("MINIO_URL"), os.Getenv("MINIO_BUCKET_NAME"), avatarObjectName)
			if err != nil {
				return false, err
			}
			newAvatarUrl = &url
		}
	}
	// Base SQL update statement
	sql := `UPDATE users SET`
	params := []interface{}{}
	paramCount := 1
	// Append the fields to be updated
	if name != nil {
		sql += fmt.Sprintf(` "name"=$%d,`, paramCount)
		params = append(params, *name)
		paramCount++
	}
	if newAvatarUrl != nil {
		sql += fmt.Sprintf(` "avatar_url"=$%d,`, paramCount)
		params = append(params, newAvatarUrl)
		paramCount++
	}
	// If no fields to update, return
	if len(params) == 0 {
		return true, nil
	}
	// Remove the trailing comma
	sql = sql[:len(sql)-1]
	// Add the WHERE clause
	sql += fmt.Sprintf(` WHERE "id"=$%d`, paramCount)
	params = append(params, userId)
	_, err := db.GetPostgresPool().Exec(ctx, sql, params...)
	if err != nil {
		return false, err
	}
	return true, nil
}

func ChangePassword(ctx context.Context, currentPassword string, newPassword string, confirmPassword string) (bool, error) {
	userId := authMiddleware.ForContext(ctx)
	if userId == "" {
		return false, errors.New("unauthorrized")
	}
	if newPassword != confirmPassword {
		return false, errors.New("passwords don't match")
	}
	pg := db.GetPostgresPool()
	rows, _ := pg.Query(
		ctx,
		`SELECT hashed_password FROM users WHERE id=$1 LIMIT 1;`,
		userId)
	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[model.User])
	if err == pgx.ErrNoRows {
		return false, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(currentPassword))
	if err != nil {
		return false, errors.New("wrong password")
	}
	_, err = db.GetPostgresPool().Exec(ctx, `ALTER TABLE users SET password = $1 WHERE password = $2 `, newPassword, currentPassword)
	if err != nil {
		return false, err
	}
	return true, nil
}

func resizeImage(img image.Image, width, height int) image.Image {
	bounds := img.Bounds()
	imgWidth := bounds.Max.X
	imgHeight := bounds.Max.Y

	// Calculate the scaling factors for width and height.
	scaleX := float64(width) / float64(imgWidth)
	scaleY := float64(height) / float64(imgHeight)

	// Create a new image with the desired dimensions.
	resizedImg := image.NewRGBA(image.Rect(0, 0, width, height))

	// Iterate over each pixel of the resized image and set its color based on the original image.
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Calculate the corresponding position in the original image.
			ox := int(float64(x) / scaleX)
			oy := int(float64(y) / scaleY)

			// Set the color of the pixel in the resized image based on the corresponding pixel in the original image.
			resizedImg.Set(x, y, img.At(ox, oy))
		}
	}

	return resizedImg
}
