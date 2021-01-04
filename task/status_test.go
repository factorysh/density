package task

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusJSON(t *testing.T) {
	s := Done
	j, err := json.Marshal(s)
	assert.NoError(t, err)
	assert.Equal(t, `"Done"`, string(j))
	var toto int
	err = json.Unmarshal([]byte("42"), &toto)
	assert.NoError(t, err)
	assert.Equal(t, 42, toto)
	var s2 Status
	err = json.Unmarshal([]byte(`"plop"`), &s2)
	assert.Error(t, err)
	err = json.Unmarshal(j, &s2)
	assert.NoError(t, err)
	fmt.Println(string(j), s2.String())
	assert.Equal(t, Done, s2)
}
