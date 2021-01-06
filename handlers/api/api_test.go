package handlers

import (
	"context"
	"fmt"
	"io"
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

	c, err := newClient(ts.URL, key)
	assert.NoError(t, err)

	res, err := c.Do("GET", "/api/schedules", nil)
	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)
}

type testClient struct {
	root          string
	client        *http.Client
	authorization string
}

func newClient(root, key string) (*testClient, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"owner": "bob",
		"nbf":   time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
	})
	blob, err := token.SignedString([]byte(key))
	if err != nil {
		return nil, err
	}
	return &testClient{
		root:          root,
		client:        &http.Client{},
		authorization: fmt.Sprintf("Bearer %s", blob),
	}, nil
}

func (t *testClient) Do(method, url string, body io.Reader) (*http.Response, error) {
	r, err := http.NewRequest(method, t.root+url, body)
	if err != nil {
		return nil, err
	}
	r.Header.Set("Authorization", t.authorization)
	return t.client.Do(r)
}
