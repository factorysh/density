package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/cristalhq/jwt/v3"
	"github.com/docker/docker/client"
	"github.com/factorysh/density/claims"
	"github.com/factorysh/density/compose"
	"github.com/factorysh/density/runner"
	"github.com/factorysh/density/scheduler"
	"github.com/factorysh/density/store"
	"github.com/factorysh/density/task"
	_ "github.com/factorysh/density/task/compose" // Needed for registering validator
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestAPI(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "scheduler-")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)
	docker, err := client.NewEnvClient()
	assert.NoError(t, err)
	recompose := &task.Recomposator{
		Recomposators: map[string]map[string]interface{}{
			"compose": {
				"VolumeInVolumes": "./volumes",
			},
		},
	}
	err = recompose.Register(docker, "bob")
	assert.NoError(t, err)
	s := scheduler.New(scheduler.NewResources(4, 16*1024), runner.New(dir, recompose), store.NewMemoryStore())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.Start(ctx)
	key := "plop"
	router := mux.NewRouter()
	v := &task.Validator{
		Validators: map[string]map[string]interface{}{
			"compose": compose.StandardConfig,
		},
	}
	err = v.Register()
	assert.NoError(t, err)
	RegisterAPI(router.PathPrefix("/api").Subrouter(), s, v, key)
	ts := httptest.NewServer(router)
	defer ts.Close()

	c, err := newClient(ts.URL, key)
	assert.NoError(t, err)

	var r []interface{}
	res, err := c.Do("GET", "/api/tasks", nil, nil, &r)
	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)
	assert.Len(t, r, 0)

	h := make(http.Header)
	h.Set("content-type", "application/json")
	b := bytes.NewReader([]byte(`{
		"cpu": 2,
		"ram": 128,
		"max_execution_time": "120s",
		"action": {
			"compose": {
				"version": "3",
				"services": {
					"hello": {
						"image":"busybox:latest",
						"command": "echo World"
					}
				}
			}
		}
	}`))
	var ta task.Task
	res, err = c.Do("POST", "/api/tasks", h, b, &ta)
	assert.NoError(t, err)
	assert.Equal(t, 201, res.StatusCode)
	assert.Len(t, r, 0)

	// FIXME test schedule creation with a file upload
}

type testClient struct {
	root          string
	client        *http.Client
	authorization string
}

func newClient(root, key string) (*testClient, error) {
	signer, err := jwt.NewSignerHS(jwt.HS256, []byte(key))
	if err != nil {
		return nil, err
	}

	// create claims (you can create your own, see: Example_BuildUserClaims)
	claims := &claims.Claims{
		Owner: "bob",
	}

	// create a Builder
	builder := jwt.NewBuilder(signer)

	// and build a Token
	token, err := builder.Build(claims)
	if err != nil {
		return nil, err
	}

	return &testClient{
		root:          root,
		client:        &http.Client{},
		authorization: fmt.Sprintf("Bearer %s", token.String()),
	}, nil
}

// Do a request.
// value is a pointer for unmarshaled JSON response
func (t *testClient) Do(method, url string, header http.Header, body io.Reader, value interface{}) (*http.Response, error) {
	r, err := http.NewRequest(method, t.root+url, body)
	if err != nil {
		return nil, err
	}
	if header != nil {
		r.Header = header
	}
	r.Header.Set("Authorization", t.authorization)
	res, err := t.client.Do(r)
	if err != nil {
		return res, err
	}
	defer res.Body.Close()
	ct := res.Header.Get("content-type")
	if ct != "application/json" {
		return res, fmt.Errorf("Wrong content-type : %s", ct)
	}
	raw, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return res, err
	}
	fmt.Println("raw", string(raw))
	err = json.Unmarshal(raw, value)
	return res, err
}
