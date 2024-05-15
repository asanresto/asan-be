package auth

import (
	"asan/graph/middleware"
	"asan/graph/middleware/header"
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// A private key for context that only this package can access. This is important
// to prevent collisions between different context uses
var userCtxKey = &middleware.ContextKey{Name: "user_id"}

// Middleware decodes the share session cookie and packs the session into context
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			accessToken := ""
			accessToken = strings.Replace(request.Header.Get("Authorization"), "Bearer ", "", 1)
			if accessToken == "" {
				accessToken = header.GetCookieFromRequest(request, "ACCESS_TOKEN_COOKIE")
			}
			if accessToken == "" {
				next.ServeHTTP(writer, request)
				return
			}
			claims, err1 := Parse(accessToken)
			if err1 != nil || claims == nil {
				next.ServeHTTP(writer, request)
				return
			}
			userId, err := claims.GetSubject()
			if err != nil {
				next.ServeHTTP(writer, request)
				return
			}
			ctx := context.WithValue(request.Context(), userCtxKey, userId)
			request = request.WithContext(ctx)
			next.ServeHTTP(writer, request)
		})
	}
}

// ForContext finds the userId from the context. REQUIRES Middleware to have run.
func ForContext(ctx context.Context) string {
	raw, _ := ctx.Value(userCtxKey).(string)
	return raw
}

func Parse(accessToken string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	if _, err := jwt.ParseWithClaims(
		accessToken,
		claims,
		func(t *jwt.Token) (interface{}, error) { return []byte(os.Getenv("JWT_KEY")), nil }); err != nil {
		return nil, err
	}
	return claims, nil
}

func GetRefreshTokenCookie(ctx context.Context) string {
	return header.GetCookieFromHeader(header.ForContext(ctx).ReqHeader, os.Getenv("REFRESH_TOKEN_COOKIE"))
}
