package owner

import (
	"context"
	"errors"
)

type contextKey string

var (
	userKey = contextKey("owner")
)

// Owner represents an authenticated user info
type Owner struct {
	Name  string
	Admin bool
}

// ToCtx creates a context containing a user key
func (u *Owner) ToCtx(in context.Context) context.Context {
	return context.WithValue(in, userKey, *u)
}

// FromJWT creates a user if one is found in JWT claims
func FromJWT(claims map[string]interface{}) (*Owner, error) {
	var isAdmin bool

	val, ok := claims["owner"]
	if !ok {
		return nil, errors.New("Missing owner in JWT claims")
	}

	name, ok := val.(string)
	if !ok {
		return nil, errors.New("JWT owner claim is not a string")
	}

	val, ok = claims["admin"]
	if ok {
		isAdmin, ok = val.(bool)
		if !ok {
			return nil, errors.New("JWT admin value not valid")
		}
	}

	return &Owner{
		Name:  name,
		Admin: isAdmin,
	}, nil

}

// FromCtx extract a user from a context
func FromCtx(ctx context.Context) (*Owner, error) {
	u, ok := ctx.Value(userKey).(Owner)
	if !ok {
		return nil, errors.New("No user in this context")
	}

	return &u, nil
}
