package middlewares

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/factorysh/density/owner"
	"github.com/factorysh/density/path"
	"github.com/getsentry/sentry-go"
)

// Auth will ensure JWT token is valid
func Auth(key string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := getToken(r)
			if err != nil {
				fmt.Println(err)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			t, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}

				return []byte(key), nil
			})
			if err != nil || !t.Valid {
				fmt.Println(err)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			claims, ok := t.Claims.(jwt.MapClaims)
			if !ok {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if hub := sentry.GetHubFromContext(r.Context()); hub != nil {
				hub.WithScope(func(scope *sentry.Scope) {
					scope.SetExtra("jwt", claims)
				})
			}

			u, err := owner.FromJWT(claims)
			if err != nil {
				fmt.Println(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			ctx := u.ToCtx(r.Context())

			p, err := path.FromJWT(claims)
			if err != nil {
				fmt.Println(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			ctx = p.ToCtx(ctx)

			next.ServeHTTP(w, r.WithContext(ctx))

		})
	}
}

// getToken from Header or Cookie or Param
func getToken(r *http.Request) (string, error) {
	getters := []func(*http.Request) (string, error){getTokenFromHeader, getTokenFromParam}

	for _, fun := range getters {
		token, err := fun(r)
		if err == nil && token != "" {
			return token, err
		}
	}

	return "", fmt.Errorf("All authentication mechanisms failed")
}

func getTokenFromHeader(r *http.Request) (string, error) {
	h := r.Header.Get("Authorization")
	bToken := strings.Split(h, " ")
	if len(bToken) != 2 {
		return "", fmt.Errorf("Invalid authorization header %v", h)
	}

	if bToken[0] != "Bearer" {
		return "", fmt.Errorf("Authorization header is not a bearer token %v", h)
	}

	return bToken[1], nil
}

func getTokenFromParam(r *http.Request) (string, error) {
	return r.URL.Query().Get("token"), nil
}

func getTokenFromCookies(r http.Request) (string, error) {

	cookie, err := r.Cookie("token")
	if err != nil {
		return "", err
	}

	return cookie.Value, nil
}
