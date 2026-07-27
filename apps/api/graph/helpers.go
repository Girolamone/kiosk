package graph

import (
	"context"
	"errors"

	"github.com/Girolamone/kiosk/apps/api/internal/account"
	"github.com/Girolamone/kiosk/apps/api/internal/auth"
	"github.com/Girolamone/kiosk/apps/api/internal/catalog"
)

// errNoSuchProduct is returned both when a product does not exist and when it
// belongs to somebody else. Distinguishing the two would let a caller probe
// for ids that exist but are not theirs.
var errNoSuchProduct = errors.New("no such product")

// errNoSuchStore hides the same distinction for stores.
var errNoSuchStore = errors.New("no such store")

// requireStoreOwner checks that the caller is signed in and owns the store.
func (r *Resolver) requireStoreOwner(ctx context.Context, storeID string) error {
	user, err := auth.RequireUser(ctx)
	if err != nil {
		return err
	}

	store, err := r.Catalog.StoreByID(ctx, storeID)
	if errors.Is(err, catalog.ErrNotFound) {
		return errNoSuchStore
	}
	if err != nil {
		return err
	}
	if store.OwnerID != user.ID {
		return errNoSuchStore
	}
	return nil
}

// startSession issues a token for the user and hands it to whatever the
// transport uses to remember them — an HttpOnly cookie, over HTTP.
func (r *Resolver) startSession(ctx context.Context, u account.User) error {
	session, ok := auth.SessionWriterFromContext(ctx)
	if !ok {
		return errors.New("this transport cannot open a session")
	}

	token, err := r.Tokens.Issue(auth.User{ID: u.ID, Email: u.Email})
	if err != nil {
		return err
	}
	session.Start(token)
	return nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
