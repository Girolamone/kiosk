// Package orders covers the path from a cart to a paid order.
//
// The rule that shapes everything here: money is never taken from the client.
// Quantities come from the browser, prices and names are read from the
// database at the moment they are needed. A cart that carried its own prices
// would let anyone buy anything for a penny.
package orders

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
)

var (
	ErrNotFound       = errors.New("cart not found")
	ErrEmptyCart      = errors.New("the cart is empty")
	ErrNotPurchasable = errors.New("that product is not for sale")
)

// MaxQuantity caps a single line. Without a ceiling, a quantity of two
// billion overflows the integer the total is computed in.
const MaxQuantity = 99

type Cart struct {
	ID      string
	StoreID string
	Token   string
	Items   []CartItem
}

// CartItem carries the name and price read from the product, not from
// whoever built the request.
type CartItem struct {
	ProductID      string
	Name           string
	UnitPriceCents int
	Quantity       int
}

func (i CartItem) LineTotalCents() int { return i.UnitPriceCents * i.Quantity }

func (c Cart) TotalCents() int {
	total := 0
	for _, item := range c.Items {
		total += item.LineTotalCents()
	}
	return total
}

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusPaid      OrderStatus = "paid"
	StatusFulfilled OrderStatus = "fulfilled"
	StatusCancelled OrderStatus = "cancelled"
)

type Order struct {
	ID         string
	StoreID    string
	Email      string
	Status     OrderStatus
	TotalCents int
	Currency   string
	Items      []OrderItem
}

type OrderItem struct {
	ProductID      string
	Name           string
	UnitPriceCents int
	Quantity       int
}

// NewCartToken returns an unguessable identifier for an anonymous cart.
//
// It is the only thing standing between a shopper and their basket, so it has
// to be long enough that guessing is pointless: anyone holding the token can
// read and change that cart.
func NewCartToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
