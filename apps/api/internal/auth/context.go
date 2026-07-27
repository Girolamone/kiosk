// Package auth carries the authenticated caller through a request.
package auth

import (
	"context"
	"errors"
)

// ErrUnauthenticated is returned by anything that needs a signed-in caller and
// did not get one.
var ErrUnauthenticated = errors.New("authentication required")

// User is the caller behind the current request.
type User struct {
	ID    string
	Email string
}

// An unexported key type makes it impossible for another package to overwrite
// this value in the context, deliberately or by accident.
type ctxKey struct{}

// NewContext returns ctx carrying u as the authenticated caller.
func NewContext(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, ctxKey{}, u)
}

// FromContext returns the authenticated caller, or false if the request was
// anonymous.
func FromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(ctxKey{}).(User)
	return u, ok
}

// RequireUser returns the authenticated caller, or ErrUnauthenticated.
func RequireUser(ctx context.Context) (User, error) {
	u, ok := FromContext(ctx)
	if !ok {
		return User{}, ErrUnauthenticated
	}
	return u, nil
}
