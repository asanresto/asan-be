package header

import (
	"asan/graph/middleware"
	"context"
	"net/http"
	"os"
	"strings"
)

var headerKey = &middleware.ContextKey{Name: "header"}

type Headers struct {
	ReqHeader http.Header
	ResHeader http.Header
}

// TODO: We may not need the whole request and response headers, just access token and refresh token cookies
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := context.WithValue(request.Context(), headerKey, Headers{
				ReqHeader: request.Header,
				ResHeader: writer.Header(),
			})
			request = request.WithContext(ctx)
			next.ServeHTTP(writer, request)
		})
	}
}

func ForContext(ctx context.Context) Headers {
	raw, _ := ctx.Value(headerKey).(Headers)
	return raw
}

func GetCookieFromRequest(request *http.Request, cookieName string) string {
	if cookie, err := request.Cookie(os.Getenv(cookieName)); err == nil {
		return cookie.Value
	}
	return ""
}

// Function to get a cookie value by its name from the http.Header
func GetCookieFromHeader(header http.Header, cookieName string) string {
	cookies := header["Cookie"]
	if len(cookies) == 0 {
		return ""
	}

	// Iterate over the cookies in the header
	for _, cookie := range cookies {
		parts := strings.Split(cookie, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, cookieName+"=") {
				return strings.TrimPrefix(part, cookieName+"=")
			}
		}
	}

	return ""
}
