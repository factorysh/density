package path

import (
	"context"
	"errors"
	"fmt"
)

type contextKey string

var (
	pathKey = contextKey("path")
)

// PATH identifier in map
const PATH = "path"

// Path type is just a custom string used to add methods on it
type Path string

// ToCtx ctrates a context containing a path key
func (p Path) ToCtx(in context.Context) context.Context {
	return context.WithValue(in, pathKey, p)
}

// FromJWT search for a PATH value inside JWT token claims
func FromJWT(claims map[string]interface{}) (Path, error) {
	val, ok := claims[PATH]
	if !ok {
		return "", nil
	}

	name, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("JWT path claim is not a string")
	}

	return Path(name), nil
}

// FromCtx extract a user from a context
func FromCtx(ctx context.Context) (Path, error) {
	p, ok := ctx.Value(pathKey).(Path)
	if !ok {
		return "", errors.New("No path in this context")
	}

	return p, nil
}
