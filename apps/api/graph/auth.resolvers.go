package graph

import (
	"context"
	"errors"
	"time"

	"github.com/Girolamone/kiosk/apps/api/graph/model"
	"github.com/Girolamone/kiosk/apps/api/internal/account"
	"github.com/Girolamone/kiosk/apps/api/internal/auth"
)

// CreateAccessToken is the resolver for the createAccessToken field.
func (r *mutationResolver) CreateAccessToken(ctx context.Context, email string, password string) (*model.AccessToken, error) {
	user, err := r.Accounts.LogIn(ctx, email, password)
	if err != nil {
		return nil, err
	}

	token, err := r.Tokens.Issue(auth.User{ID: user.ID, Email: user.Email})
	if err != nil {
		return nil, err
	}

	// No cookie is set here. A caller asking for the token is telling us it
	// keeps its own credential, and handing it both would leave a session
	// behind that nothing ever ends.
	return &model.AccessToken{
		Token:     token,
		ExpiresAt: time.Now().Add(r.Tokens.TTL()),
		User:      toUser(user),
	}, nil
}

// SignUp is the resolver for the signUp field.
func (r *mutationResolver) SignUp(ctx context.Context, email string, password string) (*model.User, error) {
	user, err := r.Accounts.SignUp(ctx, email, password)
	if err != nil {
		return nil, err
	}
	if err := r.startSession(ctx, user); err != nil {
		return nil, err
	}
	return toUser(user), nil
}

// LogIn is the resolver for the logIn field.
func (r *mutationResolver) LogIn(ctx context.Context, email string, password string) (*model.User, error) {
	// account.ErrInvalidCredentials is already deliberately vague, so it goes
	// back to the client unchanged.
	user, err := r.Accounts.LogIn(ctx, email, password)
	if err != nil {
		return nil, err
	}
	if err := r.startSession(ctx, user); err != nil {
		return nil, err
	}
	return toUser(user), nil
}

// LogOut is the resolver for the logOut field.
func (r *mutationResolver) LogOut(ctx context.Context) (bool, error) {
	if session, ok := auth.SessionWriterFromContext(ctx); ok {
		session.End()
	}
	// Signing out when already signed out is a success, not an error.
	return true, nil
}

// Me is the resolver for the me field.
func (r *queryResolver) Me(ctx context.Context) (*model.User, error) {
	caller, ok := auth.FromContext(ctx)
	if !ok {
		return nil, nil
	}

	// The token carries an id and an email, but read the row anyway: it is the
	// only way to notice that the account was deleted while a valid token is
	// still sitting in someone's browser.
	user, err := r.Accounts.ByID(ctx, caller.ID)
	if errors.Is(err, account.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toUser(user), nil
}
