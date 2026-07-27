package orders

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CartByToken reads a cart and joins each line against the product it points
// at, so names and prices are whatever the catalogue says right now.
func (r *Repository) CartByToken(ctx context.Context, token string) (Cart, error) {
	var cart Cart
	err := r.pool.QueryRow(ctx,
		`SELECT id::text, store_id::text, token FROM carts WHERE token = $1`, token,
	).Scan(&cart.ID, &cart.StoreID, &cart.Token)
	if errors.Is(err, pgx.ErrNoRows) {
		return Cart{}, ErrNotFound
	}
	if err != nil {
		return Cart{}, fmt.Errorf("read cart: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT p.id::text, p.name, p.price_cents, i.quantity
		  FROM cart_items i
		  JOIN products p ON p.id = i.product_id
		 WHERE i.cart_id = $1::uuid
		   -- A product pulled from sale after it was added drops out of the
		   -- cart rather than being sold anyway.
		   AND p.status = 'active'
		 ORDER BY p.name`, cart.ID)
	if err != nil {
		return Cart{}, fmt.Errorf("read cart items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item CartItem
		if err := rows.Scan(&item.ProductID, &item.Name, &item.UnitPriceCents, &item.Quantity); err != nil {
			return Cart{}, fmt.Errorf("scan cart item: %w", err)
		}
		cart.Items = append(cart.Items, item)
	}
	if err := rows.Err(); err != nil {
		return Cart{}, fmt.Errorf("read cart items: %w", err)
	}
	return cart, nil
}

// EnsureCart returns the cart for this token, creating one against the given
// store when the token is unknown or absent.
func (r *Repository) EnsureCart(ctx context.Context, storeID, token string) (Cart, error) {
	if token != "" {
		cart, err := r.CartByToken(ctx, token)
		if err == nil {
			return cart, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return Cart{}, err
		}
	}

	fresh, err := NewCartToken()
	if err != nil {
		return Cart{}, err
	}

	var cart Cart
	err = r.pool.QueryRow(ctx, `
		INSERT INTO carts (store_id, token) VALUES ($1::uuid, $2)
		RETURNING id::text, store_id::text, token`, storeID, fresh,
	).Scan(&cart.ID, &cart.StoreID, &cart.Token)
	if err != nil {
		return Cart{}, fmt.Errorf("create cart: %w", err)
	}
	return cart, nil
}

// SetItemQuantity adds, changes or removes one line. A quantity of zero
// removes it.
func (r *Repository) SetItemQuantity(ctx context.Context, cartID, productID string, quantity int) error {
	if quantity <= 0 {
		_, err := r.pool.Exec(ctx,
			`DELETE FROM cart_items WHERE cart_id = $1::uuid AND product_id = $2::uuid`,
			cartID, productID)
		if err != nil {
			return fmt.Errorf("remove cart item: %w", err)
		}
		return nil
	}

	// UNIQUE (cart_id, product_id) makes this an upsert rather than a second
	// row for the same product.
	_, err := r.pool.Exec(ctx, `
		INSERT INTO cart_items (cart_id, product_id, quantity)
		VALUES ($1::uuid, $2::uuid, $3)
		ON CONFLICT (cart_id, product_id) DO UPDATE SET quantity = EXCLUDED.quantity`,
		cartID, productID, quantity)
	if err != nil {
		return fmt.Errorf("set cart item quantity: %w", err)
	}
	return nil
}

// ProductIsPurchasable reports whether the product is published and belongs to
// the given store, so a cart cannot be filled with another shop's goods or
// with something still in draft.
func (r *Repository) ProductIsPurchasable(ctx context.Context, storeID, productID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM products
			 WHERE id = $1::uuid AND store_id = $2::uuid AND status = 'active')`,
		productID, storeID).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("check product: %w", err)
	}
	return ok, nil
}

// CreateOrder writes a pending order and its lines from the cart, in one
// transaction. Every line copies the name and price as they are now: editing
// the product afterwards must not rewrite what the customer bought.
func (r *Repository) CreateOrder(ctx context.Context, cart Cart, email, currency, sessionID string) (Order, error) {
	if len(cart.Items) == 0 {
		return Order{}, ErrEmptyCart
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Order{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	order := Order{
		StoreID:    cart.StoreID,
		Email:      email,
		Status:     StatusPending,
		TotalCents: cart.TotalCents(),
		Currency:   currency,
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO orders (store_id, email, status, total_cents, currency, stripe_session_id)
		VALUES ($1::uuid, $2, 'pending', $3, $4, $5)
		RETURNING id::text`,
		cart.StoreID, email, order.TotalCents, currency, sessionID,
	).Scan(&order.ID)
	if err != nil {
		return Order{}, fmt.Errorf("create order: %w", err)
	}

	for _, item := range cart.Items {
		if _, err := tx.Exec(ctx, `
			INSERT INTO order_items (order_id, product_id, product_name, unit_price_cents, quantity)
			VALUES ($1::uuid, $2::uuid, $3, $4, $5)`,
			order.ID, item.ProductID, item.Name, item.UnitPriceCents, item.Quantity); err != nil {
			return Order{}, fmt.Errorf("create order item: %w", err)
		}
		order.Items = append(order.Items, OrderItem(item))
	}

	if err := tx.Commit(ctx); err != nil {
		return Order{}, fmt.Errorf("commit order: %w", err)
	}
	return order, nil
}

// MarkPaid settles the order for a checkout session.
//
// Stripe delivers a webhook at least once and will retry, so this has to be
// safe to run repeatedly. The status check in the WHERE clause is what makes
// it so: the second delivery matches no rows and changes nothing.
func (r *Repository) MarkPaid(ctx context.Context, sessionID string) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE orders
		   SET status = 'paid', paid_at = now()
		 WHERE stripe_session_id = $1 AND status = 'pending'`, sessionID)
	if err != nil {
		return false, fmt.Errorf("mark order paid: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// ClearCart empties a cart once its contents have been bought.
func (r *Repository) ClearCart(ctx context.Context, cartID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM cart_items WHERE cart_id = $1::uuid`, cartID)
	if err != nil {
		return fmt.Errorf("clear cart: %w", err)
	}
	return nil
}
