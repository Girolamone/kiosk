package graph

import (
	"context"
	"errors"
	"fmt"

	"github.com/Girolamone/kiosk/apps/api/graph/model"
	"github.com/Girolamone/kiosk/apps/api/internal/auth"
	"github.com/Girolamone/kiosk/apps/api/internal/catalog"
)

// CreateStore is the resolver for the createStore field.
func (r *mutationResolver) CreateStore(ctx context.Context, input model.CreateStoreInput) (*model.Store, error) {
	user, err := auth.RequireUser(ctx)
	if err != nil {
		return nil, err
	}

	store, err := catalog.NewStore(user.ID, input.Name, input.Slug, deref(input.Description))
	if err != nil {
		return nil, err
	}

	created, err := r.Catalog.CreateStore(ctx, store)
	if errors.Is(err, catalog.ErrSlugTaken) {
		return nil, fmt.Errorf("the slug %q is already taken", input.Slug)
	}
	if err != nil {
		return nil, err
	}
	return toStore(created), nil
}

// CreateProduct is the resolver for the createProduct field.
func (r *mutationResolver) CreateProduct(ctx context.Context, input model.CreateProductInput) (*model.Product, error) {
	if err := r.requireStoreOwner(ctx, input.StoreID); err != nil {
		return nil, err
	}

	product, err := catalog.NewProduct(input.StoreID, input.Name, deref(input.Description), input.PriceCents)
	if err != nil {
		return nil, err
	}

	created, err := r.Catalog.CreateProduct(ctx, product)
	if err != nil {
		return nil, err
	}
	return toProduct(created), nil
}

// UpdateProduct is the resolver for the updateProduct field.
func (r *mutationResolver) UpdateProduct(ctx context.Context, input model.UpdateProductInput) (*model.Product, error) {
	existing, err := r.Catalog.ProductByID(ctx, input.ID)
	if errors.Is(err, catalog.ErrNotFound) {
		return nil, errNoSuchProduct
	}
	if err != nil {
		return nil, err
	}
	if err := r.requireStoreOwner(ctx, existing.StoreID); err != nil {
		return nil, errNoSuchProduct
	}

	patch := catalog.ProductPatch{
		Name:        input.Name,
		Description: input.Description,
		PriceCents:  input.PriceCents,
	}
	if input.Status != nil {
		status := fromProductStatus(*input.Status)
		patch.Status = &status
	}

	updated, err := r.Catalog.UpdateProduct(ctx, input.ID, patch)
	if err != nil {
		return nil, err
	}
	return toProduct(updated), nil
}

// Images is the resolver for the images field.
func (r *productResolver) Images(ctx context.Context, obj *model.Product) ([]*model.ProductImage, error) {
	images, err := r.Catalog.ImagesByProduct(ctx, obj.ID)
	if err != nil {
		return nil, err
	}
	return toProductImages(images), nil
}

// Store is the resolver for the store field.
func (r *queryResolver) Store(ctx context.Context, slug string) (*model.Store, error) {
	store, err := r.Catalog.StoreBySlug(ctx, slug)
	if errors.Is(err, catalog.ErrNotFound) {
		// A missing store is an absent result, not a failure: the field is
		// nullable and the client gets null rather than an error.
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toStore(store), nil
}

// Product is the resolver for the product field.
func (r *queryResolver) Product(ctx context.Context, id string) (*model.Product, error) {
	product, err := r.Catalog.ProductByID(ctx, id)
	if errors.Is(err, catalog.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toProduct(product), nil
}

// Products is the resolver for the products field.
func (r *storeResolver) Products(ctx context.Context, obj *model.Store, status *model.ProductStatus) ([]*model.Product, error) {
	filter := catalog.StatusActive
	if status != nil {
		filter = fromProductStatus(*status)
	}

	// ACTIVE is the published catalogue and public. Everything else is the
	// seller's private working state, so asking for it means proving you own
	// the store. Defaulting to ACTIVE is not enough on its own: without this
	// check any anonymous caller could just ask for DRAFT by name.
	if filter != catalog.StatusActive {
		if err := r.requireStoreOwner(ctx, obj.ID); err != nil {
			return nil, err
		}
	}

	products, err := r.Catalog.ProductsByStore(ctx, obj.ID, &filter)
	if err != nil {
		return nil, err
	}
	return toProducts(products), nil
}

// errNoSuchProduct is returned both when a product does not exist and when it
// belongs to somebody else. Distinguishing the two would let a caller probe
// for ids that exist but are not theirs.
var errNoSuchProduct = errors.New("no such product")

// requireStoreOwner checks that the caller is signed in and owns the store.
func (r *Resolver) requireStoreOwner(ctx context.Context, storeID string) error {
	user, err := auth.RequireUser(ctx)
	if err != nil {
		return err
	}

	store, err := r.Catalog.StoreByID(ctx, storeID)
	if errors.Is(err, catalog.ErrNotFound) {
		return errors.New("no such store")
	}
	if err != nil {
		return err
	}
	if store.OwnerID != user.ID {
		return errors.New("no such store")
	}
	return nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Mutation returns MutationResolver implementation.
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

// Product returns ProductResolver implementation.
func (r *Resolver) Product() ProductResolver { return &productResolver{r} }

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

// Store returns StoreResolver implementation.
func (r *Resolver) Store() StoreResolver { return &storeResolver{r} }

type (
	mutationResolver struct{ *Resolver }
	productResolver  struct{ *Resolver }
	queryResolver    struct{ *Resolver }
	storeResolver    struct{ *Resolver }
)
