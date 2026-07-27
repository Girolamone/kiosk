package graph

import (
	"context"
	"errors"
	"fmt"

	"github.com/Girolamone/kiosk/apps/api/graph/model"
	"github.com/Girolamone/kiosk/apps/api/internal/catalog"
	"github.com/Girolamone/kiosk/apps/api/internal/orders"
	"github.com/Girolamone/kiosk/apps/api/internal/payments"
)

// SetCartItem is the resolver for the setCartItem field.
func (r *mutationResolver) SetCartItem(ctx context.Context, input model.SetCartItemInput) (*model.Cart, error) {
	store, err := r.Catalog.StoreBySlug(ctx, input.StoreSlug)
	if errors.Is(err, catalog.ErrNotFound) {
		return nil, errNoSuchStore
	}
	if err != nil {
		return nil, err
	}

	if input.Quantity > orders.MaxQuantity {
		return nil, fmt.Errorf("at most %d of one item per order", orders.MaxQuantity)
	}

	// Only published products of this shop can go in this cart. Without the
	// check, anyone could add a draft, or something from another seller, by
	// sending its id.
	if input.Quantity > 0 {
		purchasable, err := r.Orders.ProductIsPurchasable(ctx, store.ID, input.ProductID)
		if err != nil {
			return nil, err
		}
		if !purchasable {
			return nil, orders.ErrNotPurchasable
		}
	}

	cart, err := r.Orders.EnsureCart(ctx, store.ID, deref(input.Token))
	if err != nil {
		return nil, err
	}
	if err := r.Orders.SetItemQuantity(ctx, cart.ID, input.ProductID, input.Quantity); err != nil {
		return nil, err
	}

	// Re-read so the returned cart reflects the write, prices included.
	updated, err := r.Orders.CartByToken(ctx, cart.Token)
	if err != nil {
		return nil, err
	}
	return toCart(updated, store.Currency), nil
}

// CreateCheckout is the resolver for the createCheckout field.
func (r *mutationResolver) CreateCheckout(ctx context.Context, token string, email string) (*model.Checkout, error) {
	cart, err := r.Orders.CartByToken(ctx, token)
	if errors.Is(err, orders.ErrNotFound) {
		return nil, errors.New("that basket no longer exists")
	}
	if err != nil {
		return nil, err
	}
	if len(cart.Items) == 0 {
		return nil, orders.ErrEmptyCart
	}

	store, err := r.Catalog.StoreByID(ctx, cart.StoreID)
	if err != nil {
		return nil, err
	}

	lines := make([]payments.Line, 0, len(cart.Items))
	for _, item := range cart.Items {
		lines = append(lines, payments.Line{
			Name:           item.Name,
			UnitPriceCents: item.UnitPriceCents,
			Quantity:       item.Quantity,
		})
	}

	// The Stripe session is created first because the order needs its id. A
	// failure after this leaves an unpaid session nobody visits, which costs
	// nothing; doing it the other way round would leave an order that can
	// never be paid.
	session, err := r.Payments.CreateSession(ctx, payments.SessionRequest{
		Email:      email,
		Currency:   store.Currency,
		Lines:      lines,
		SuccessURL: fmt.Sprintf("%s/s/%s?paid=1", r.PublicURL, store.Slug),
		CancelURL:  fmt.Sprintf("%s/s/%s/cart", r.PublicURL, store.Slug),
		Reference:  cart.Token,
	})
	if err != nil {
		return nil, err
	}

	if _, err := r.Orders.CreateOrder(ctx, cart, email, store.Currency, session.ID); err != nil {
		return nil, err
	}

	return &model.Checkout{URL: session.URL}, nil
}

// Cart is the resolver for the cart field.
func (r *queryResolver) Cart(ctx context.Context, token string) (*model.Cart, error) {
	cart, err := r.Orders.CartByToken(ctx, token)
	if errors.Is(err, orders.ErrNotFound) {
		// A stale token in someone's browser is an empty basket, not an error.
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	store, err := r.Catalog.StoreByID(ctx, cart.StoreID)
	if err != nil {
		return nil, err
	}
	return toCart(cart, store.Currency), nil
}
