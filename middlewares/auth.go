package middlewares

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/factorysh/batch-scheduler/owner"
)

// Auth will ensure JWT token is valid
func Auth(key string, next http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

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

		u, err := owner.FromJWT(claims)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ctx := u.ToCtx(r.Context())

		next.ServeHTTP(w, r.WithContext(ctx))

	}
}
