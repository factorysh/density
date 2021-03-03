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
			h := r.Header.Get("Authorization")
			if h == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			bToken := strings.Split(h, " ")
			if len(bToken) != 2 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			t, err := jwt.Parse(bToken[1], func(token *jwt.Token) (interface{}, error) {
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
