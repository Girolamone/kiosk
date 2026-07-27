package catalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// uniqueViolation is the Postgres SQLSTATE for a unique constraint breach.
const uniqueViolation = "23505"

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// uuid columns are cast to text on the way out and from text on the way in, so
// ids stay plain strings across the whole codebase.
const storeColumns = `id::text, owner_id::text, name, slug, description, currency, created_at`

func (r *Repository) StoreBySlug(ctx context.Context, slug string) (Store, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+storeColumns+` FROM stores WHERE slug = $1`, slug)
	return scanStore(row)
}

func (r *Repository) StoreByID(ctx context.Context, id string) (Store, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+storeColumns+` FROM stores WHERE id = $1::uuid`, id)
	return scanStore(row)
}

func (r *Repository) StoresByOwner(ctx context.Context, ownerID string) ([]Store, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+storeColumns+` FROM stores WHERE owner_id = $1::uuid ORDER BY created_at`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("query stores: %w", err)
	}
	defer rows.Close()

	var stores []Store
	for rows.Next() {
		s, err := scanStore(rows)
		if err != nil {
			return nil, err
		}
		stores = append(stores, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read stores: %w", err)
	}
	return stores, nil
}

func (r *Repository) CreateStore(ctx context.Context, s Store) (Store, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO stores (owner_id, name, slug, description, currency)
		VALUES ($1::uuid, $2, $3, $4, $5)
		RETURNING `+storeColumns,
		s.OwnerID, s.Name, s.Slug, s.Description, s.Currency)

	created, err := scanStore(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return Store{}, ErrSlugTaken
		}
		return Store{}, err
	}
	return created, nil
}

const productColumns = `id::text, store_id::text, name, description, price_cents, status, created_at, updated_at`

func (r *Repository) ProductByID(ctx context.Context, id string) (Product, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+productColumns+` FROM products WHERE id = $1::uuid`, id)
	return scanProduct(row)
}

// ProductsByStore lists a store's products. A nil status means only published
// ones: this feeds a public storefront, so the safe default must not leak
// drafts.
func (r *Repository) ProductsByStore(ctx context.Context, storeID string, status *ProductStatus) ([]Product, error) {
	effective := StatusActive
	if status != nil {
		effective = *status
	}

	rows, err := r.pool.Query(ctx, `
		SELECT `+productColumns+`
		  FROM products
		 WHERE store_id = $1::uuid AND status = $2
		 ORDER BY created_at DESC`,
		storeID, string(effective))
	if err != nil {
		return nil, fmt.Errorf("query products: %w", err)
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read products: %w", err)
	}
	return products, nil
}

func (r *Repository) CreateProduct(ctx context.Context, p Product) (Product, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO products (store_id, name, description, price_cents, status)
		VALUES ($1::uuid, $2, $3, $4, $5)
		RETURNING `+productColumns,
		p.StoreID, p.Name, p.Description, p.PriceCents, string(p.Status))
	return scanProduct(row)
}

// ProductPatch carries a partial update. A nil field means "leave alone".
type ProductPatch struct {
	Name        *string
	Description *string
	PriceCents  *int
	Status      *ProductStatus
}

func (r *Repository) UpdateProduct(ctx context.Context, id string, patch ProductPatch) (Product, error) {
	var status *string
	if patch.Status != nil {
		s := string(*patch.Status)
		status = &s
	}

	// COALESCE lets one statement handle any combination of set fields,
	// instead of building SQL by string concatenation.
	row := r.pool.QueryRow(ctx, `
		UPDATE products
		   SET name        = COALESCE($2, name),
		       description = COALESCE($3, description),
		       price_cents = COALESCE($4, price_cents),
		       status      = COALESCE($5::product_status, status)
		 WHERE id = $1::uuid
		RETURNING `+productColumns,
		id, patch.Name, patch.Description, patch.PriceCents, status)
	return scanProduct(row)
}

func (r *Repository) ImagesByProduct(ctx context.Context, productID string) ([]ProductImage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, product_id::text, url, alt_text, position
		  FROM product_images
		 WHERE product_id = $1::uuid
		 ORDER BY position, created_at`, productID)
	if err != nil {
		return nil, fmt.Errorf("query product images: %w", err)
	}
	defer rows.Close()

	var images []ProductImage
	for rows.Next() {
		var img ProductImage
		if err := rows.Scan(&img.ID, &img.ProductID, &img.URL, &img.AltText, &img.Position); err != nil {
			return nil, fmt.Errorf("scan product image: %w", err)
		}
		images = append(images, img)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read product images: %w", err)
	}
	return images, nil
}

// scanner is satisfied by both pgx.Row and pgx.Rows, so single-row and
// multi-row reads share the same scan code.
type scanner interface {
	Scan(dest ...any) error
}

func scanStore(row scanner) (Store, error) {
	var s Store
	err := row.Scan(&s.ID, &s.OwnerID, &s.Name, &s.Slug, &s.Description, &s.Currency, &s.CreatedAt)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Store{}, ErrNotFound
	case err != nil:
		return Store{}, fmt.Errorf("scan store: %w", err)
	}
	return s, nil
}

func scanProduct(row scanner) (Product, error) {
	var p Product
	var status string
	err := row.Scan(&p.ID, &p.StoreID, &p.Name, &p.Description, &p.PriceCents, &status, &p.CreatedAt, &p.UpdatedAt)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Product{}, ErrNotFound
	case err != nil:
		return Product{}, fmt.Errorf("scan product: %w", err)
	}
	p.Status = ProductStatus(status)
	return p, nil
}
