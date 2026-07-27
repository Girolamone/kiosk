// Command seed fills a fresh database with a demo shop.
//
// It exists because an empty storefront is a bad first impression: someone
// opening the deployed app should land on something that looks like a real
// shop, not an empty state with a "create your first product" prompt.
//
// The shop sells photographic prints, which means the demo photographs are
// genuinely the product rather than stand-ins, and the copy the model writes
// about them is real output rather than lorem ipsum.
//
//	go run ./cmd/seed          # only fills an empty shop
//	go run ./cmd/seed -reset   # replaces whatever is there
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/Girolamone/kiosk/apps/api/internal/ai"
	"github.com/Girolamone/kiosk/apps/api/internal/config"
	"github.com/Girolamone/kiosk/apps/api/internal/db"
	"github.com/Girolamone/kiosk/apps/api/internal/storage"
)

const (
	demoEmail    = "demo@kiosk.dev"
	demoPassword = "kioskdemo2026"
	storeSlug    = "northlight-press"
	storeName    = "Northlight Press"
	storeAbout   = "Photographic prints, made to order on cotton rag paper."
)

// Fixed photo ids so a reseed produces the same shop rather than a different
// one every time.
var photos = []struct {
	id    string
	cents int
}{
	{"1015", 4500}, {"1016", 5200}, {"1018", 3800}, {"1019", 6000},
	{"1022", 4200}, {"1024", 5500}, {"1039", 4800}, {"1043", 3600},
}

func main() {
	reset := flag.Bool("reset", false, "replace any products already in the demo shop")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if err := run(context.Background(), logger, *reset); err != nil {
		logger.Error("seed failed", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger, reset bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	pool, err := db.Connect(ctx, cfg.DatabaseURL, nil)
	if err != nil {
		return err
	}
	defer pool.Close()

	files, err := newStorage(ctx, cfg)
	if err != nil {
		return err
	}

	var writer ai.CopyGenerator = ai.Disabled{}
	if cfg.GeminiAPIKey != "" {
		writer = ai.NewGemini(cfg.GeminiAPIKey, cfg.GeminiModel)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	var ownerID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash) VALUES ($1, $2)
		ON CONFLICT (email) DO UPDATE SET password_hash = EXCLUDED.password_hash
		RETURNING id::text`, demoEmail, string(hash)).Scan(&ownerID); err != nil {
		return fmt.Errorf("create demo user: %w", err)
	}

	var storeID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO stores (owner_id, name, slug, description)
		VALUES ($1::uuid, $2, $3, $4)
		ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description
		RETURNING id::text`, ownerID, storeName, storeSlug, storeAbout).Scan(&storeID); err != nil {
		return fmt.Errorf("create demo store: %w", err)
	}

	var existing int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM products WHERE store_id = $1::uuid`, storeID).Scan(&existing); err != nil {
		return err
	}
	if existing > 0 && !reset {
		logger.Info("the demo shop already has products, leaving it alone", "products", existing)
		return nil
	}
	if existing > 0 {
		if _, err := pool.Exec(ctx, `DELETE FROM products WHERE store_id = $1::uuid`, storeID); err != nil {
			return fmt.Errorf("clear demo products: %w", err)
		}
	}

	client := &http.Client{Timeout: 60 * time.Second}
	for i, photo := range photos {
		if err := addPrint(ctx, logger, pool, files, writer, client, storeID, photo.id, photo.cents, i); err != nil {
			return err
		}
	}

	logger.Info("done", "shop", "/s/"+storeSlug, "sign in", demoEmail, "password", demoPassword)
	return nil
}

// addPrint downloads one photograph, stores it, asks the model to describe
// it, and publishes it as a product.
func addPrint(
	ctx context.Context,
	logger *slog.Logger,
	pool *pgxpool.Pool,
	files storage.Store,
	writer ai.CopyGenerator,
	client *http.Client,
	storeID, photoID string,
	cents, index int,
) error {
	image, contentType, err := downloadPhoto(ctx, client, photoID)
	if err != nil {
		return err
	}

	object, err := files.Put(ctx, contentType, bytes.NewReader(image))
	if err != nil {
		return fmt.Errorf("store photo %s: %w", photoID, err)
	}

	// The listing is written by the same code path the dashboard uses, so the
	// demo shop shows real output. If the model is unavailable the shop still
	// gets built, just with plainer words.
	name := fmt.Sprintf("Print No. %d", index+1)
	description := "Archival pigment print on cotton rag paper."
	altText := "A photographic print."

	// A batch job can afford to wait where a person filling in a form cannot,
	// so unlike the dashboard this retries. The free Gemini tier rate-limits
	// a run of eight in a row, and a demo shop where two listings say
	// "Print No. 7" undersells the feature it is there to show.
	if copy, err := generateWithRetry(ctx, logger, writer, contentType, image, photoID); err == nil {
		name, description, altText = copy.Title, copy.Description, copy.AltText
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var productID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO products (store_id, name, description, price_cents, status)
		VALUES ($1::uuid, $2, $3, $4, 'active')
		RETURNING id::text`, storeID, name, description, cents).Scan(&productID); err != nil {
		return fmt.Errorf("create demo product: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO product_images (product_id, url, alt_text, position)
		VALUES ($1::uuid, $2, $3, 0)`, productID, object.URL, altText); err != nil {
		return fmt.Errorf("attach demo photo: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	logger.Info("added", "name", name)
	return nil
}

func generateWithRetry(
	ctx context.Context,
	logger *slog.Logger,
	writer ai.CopyGenerator,
	contentType string,
	image []byte,
	photoID string,
) (ai.ProductCopy, error) {
	backoff := 8 * time.Second

	var err error
	for attempt := 1; attempt <= 4; attempt++ {
		var copy ai.ProductCopy
		copy, err = writer.GenerateProductCopy(ctx, ai.Image{ContentType: contentType, Data: image})
		if err == nil {
			return copy, nil
		}

		if attempt < 4 {
			logger.Info("waiting before another try", "photo", photoID, "in", backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ai.ProductCopy{}, ctx.Err()
			}
			backoff *= 2
		}
	}

	logger.Warn("no generated copy for this one", "photo", photoID, "err", err)
	return ai.ProductCopy{}, err
}

func newStorage(ctx context.Context, cfg config.Config) (storage.Store, error) {
	switch cfg.StorageDriver {
	case "local":
		return storage.NewLocal(cfg.LocalStorageDir, "/uploads")
	case "gcs":
		return storage.NewGCS(ctx, cfg.GCSBucket)
	default:
		return nil, fmt.Errorf("unknown STORAGE_DRIVER %q", cfg.StorageDriver)
	}
}

// downloadPhoto fetches one demo photograph.
func downloadPhoto(ctx context.Context, client *http.Client, photoID string) ([]byte, string, error) {
	url := fmt.Sprintf("https://picsum.photos/id/%s/1000/1000", photoID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("download photo %s: %w", photoID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download photo %s: %s", photoID, resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, storage.MaxUploadBytes))
	if err != nil {
		return nil, "", err
	}
	return body, resp.Header.Get("Content-Type"), nil
}
