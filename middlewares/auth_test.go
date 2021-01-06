package middlewares

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
)

func TestAuth(t *testing.T) {
	key := "plop"
	ts := httptest.NewServer(Auth(key, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	})))
	defer ts.Close()

	res, err := http.Get(ts.URL)
	assert.NoError(t, err)
	assert.Equal(t, 401, res.StatusCode)

	client := &http.Client{}

	type fixture struct {
		claim  jwt.MapClaims
		key    []byte
		status int
	}
	for _, a := range []fixture{
		{ // it's ok
			claim: jwt.MapClaims{
				"owner": "bob",
				"nbf":   time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
			},
			key:    []byte(key),
			status: 200,
		},
		{ // owner is missing
			claim: jwt.MapClaims{
				"nbf": time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
			},
			key:    []byte(key),
			status: 400,
		},
		{ // wrong key
			claim: jwt.MapClaims{
				"owner": "bob",
				"nbf":   time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
			},
			key:    []byte("wrong key"),
			status: 401,
		},
	} {
		r, err := http.NewRequest("GET", ts.URL, nil)
		assert.NoError(t, err)
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, a.claim)
		blob, err := token.SignedString(a.key)
		assert.NoError(t, err)
		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", blob))
		res, err = client.Do(r)
		assert.NoError(t, err)
		assert.Equal(t, a.status, res.StatusCode)
	}
}
