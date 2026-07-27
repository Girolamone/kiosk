// Package catalog holds the store and product domain: the types, the rules
// they have to satisfy, and the storage they live in. It knows nothing about
// GraphQL or HTTP.
package catalog

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// ErrNotFound is returned when a lookup by id or slug matches no row. Callers
// turn it into whatever their transport considers "absent".
var ErrNotFound = errors.New("not found")

// ErrSlugTaken is returned when a store slug collides with an existing one.
var ErrSlugTaken = errors.New("slug already taken")

// ProductStatus mirrors the product_status enum in Postgres, so the values are
// lowercase here. The GraphQL layer maps them to its own uppercase enum.
type ProductStatus string

const (
	StatusDraft    ProductStatus = "draft"
	StatusActive   ProductStatus = "active"
	StatusArchived ProductStatus = "archived"
)

type Store struct {
	ID          string
	OwnerID     string
	Name        string
	Slug        string
	Description string
	Currency    string
	CreatedAt   time.Time
}

type Product struct {
	ID          string
	StoreID     string
	Name        string
	Description string
	PriceCents  int
	Status      ProductStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ProductImage struct {
	ID        string
	ProductID string
	URL       string
	AltText   string
	Position  int
}

// NewStore validates the inputs for a new store.
func NewStore(ownerID, name, slug, description string) (Store, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Store{}, errors.New("store name is required")
	}
	slug = strings.ToLower(strings.TrimSpace(slug))
	if !slugPattern.MatchString(slug) {
		return Store{}, errors.New("slug must be 3-40 characters of lowercase letters, digits or hyphens")
	}

	return Store{
		OwnerID:     ownerID,
		Name:        name,
		Slug:        slug,
		Description: strings.TrimSpace(description),
		Currency:    "USD",
	}, nil
}

// A slug ends up in a public URL, so keep it to characters that survive one.
var slugPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{1,38})[a-z0-9]$`)

// NewProduct validates the inputs for a new product. Products start as drafts:
// publishing is a deliberate second step.
func NewProduct(storeID, name, description string, priceCents int) (Product, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Product{}, errors.New("product name is required")
	}
	if priceCents < 0 {
		return Product{}, errors.New("price cannot be negative")
	}

	return Product{
		StoreID:     storeID,
		Name:        name,
		Description: strings.TrimSpace(description),
		PriceCents:  priceCents,
		Status:      StatusDraft,
	}, nil
}
