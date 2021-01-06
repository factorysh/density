package handlers

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/factorysh/batch-scheduler/runner"
	"github.com/factorysh/batch-scheduler/scheduler"
	"github.com/factorysh/batch-scheduler/store"
	"github.com/stretchr/testify/assert"
)

func TestAPI(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "scheduler-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)
	s := scheduler.New(scheduler.NewResources(4, 16*1024), runner.New(dir), store.NewMemoryStore())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.Start(ctx)
	key := "plop"
	mux := MuxAPI(s, key)
	ts := httptest.NewServer(mux)
	defer ts.Close()
	client := &http.Client{}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"owner": "bob",
		"nbf":   time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
	})
	blob, err := token.SignedString([]byte(key))
	assert.NoError(t, err)
	r, err := http.NewRequest("GET", ts.URL+"/api/schedules", nil)
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", blob))
	res, err := client.Do(r)
	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)
}
