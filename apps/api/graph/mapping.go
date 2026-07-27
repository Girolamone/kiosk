package graph

import (
	"github.com/Girolamone/kiosk/apps/api/graph/model"
	"github.com/Girolamone/kiosk/apps/api/internal/catalog"
)

// The domain and the API deliberately keep separate types. The domain follows
// the database (lowercase enum values, storage-shaped fields); the API follows
// GraphQL convention. These functions are the only place the two meet, so
// renaming a column never leaks into the public schema.

func toStore(s catalog.Store) *model.Store {
	return &model.Store{
		ID:          s.ID,
		Name:        s.Name,
		Slug:        s.Slug,
		Description: s.Description,
		Currency:    s.Currency,
		CreatedAt:   s.CreatedAt,
	}
}

func toProduct(p catalog.Product) *model.Product {
	return &model.Product{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		PriceCents:  p.PriceCents,
		Status:      toProductStatus(p.Status),
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func toProducts(ps []catalog.Product) []*model.Product {
	out := make([]*model.Product, 0, len(ps))
	for _, p := range ps {
		out = append(out, toProduct(p))
	}
	return out
}

func toProductImages(imgs []catalog.ProductImage) []*model.ProductImage {
	out := make([]*model.ProductImage, 0, len(imgs))
	for _, img := range imgs {
		out = append(out, &model.ProductImage{
			ID:       img.ID,
			URL:      img.URL,
			AltText:  img.AltText,
			Position: img.Position,
		})
	}
	return out
}

func toProductStatus(s catalog.ProductStatus) model.ProductStatus {
	switch s {
	case catalog.StatusActive:
		return model.ProductStatusActive
	case catalog.StatusArchived:
		return model.ProductStatusArchived
	default:
		return model.ProductStatusDraft
	}
}

func fromProductStatus(s model.ProductStatus) catalog.ProductStatus {
	switch s {
	case model.ProductStatusActive:
		return catalog.StatusActive
	case model.ProductStatusArchived:
		return catalog.StatusArchived
	default:
		return catalog.StatusDraft
	}
}
