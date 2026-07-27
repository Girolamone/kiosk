// Package loaders batches the per-object database reads a GraphQL query fans
// out into.
//
// Resolving `store { products { images } }` calls the images resolver once per
// product. Done naively that is one query per product on top of the query that
// listed them — the N+1 problem. A loader collects the ids requested within a
// short window, fetches them in a single statement, and hands each caller back
// its own slice.
package loaders

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/vikstrous/dataloadgen"

	"github.com/Girolamone/kiosk/apps/api/internal/catalog"
)

// batchWindow is how long a loader waits for more ids before firing. It only
// has to cover the gap between sibling resolvers starting, which is
// microseconds; this is generous.
const batchWindow = time.Millisecond

type Loaders struct {
	ProductImages *dataloadgen.Loader[string, []catalog.ProductImage]
}

type ctxKey struct{}

// Middleware gives every request its own set of loaders.
//
// Per request, not per process, and this is the part that matters: a loader
// caches what it has already fetched. A process-wide loader would serve one
// user stale data from another user's request and never notice a write. The
// cache is only safe because it dies with the request.
func Middleware(repo *catalog.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), ctxKey{}, newLoaders(repo))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// For returns the loaders attached to this request.
func For(ctx context.Context) (*Loaders, error) {
	l, ok := ctx.Value(ctxKey{}).(*Loaders)
	if !ok {
		return nil, fmt.Errorf("no loaders on this request: is the middleware wired up?")
	}
	return l, nil
}

func newLoaders(repo *catalog.Repository) *Loaders {
	return &Loaders{
		ProductImages: dataloadgen.NewLoader(
			fetchProductImages(repo),
			dataloadgen.WithWait(batchWindow),
		),
	}
}

// fetchProductImages must return one slice per requested id, in the same
// order, so each caller gets the images it asked for.
func fetchProductImages(repo *catalog.Repository) func(context.Context, []string) ([][]catalog.ProductImage, []error) {
	return func(ctx context.Context, productIDs []string) ([][]catalog.ProductImage, []error) {
		byProduct, err := repo.ImagesByProducts(ctx, productIDs)
		if err != nil {
			// One failure fails every caller in the batch.
			errs := make([]error, len(productIDs))
			for i := range errs {
				errs[i] = err
			}
			return nil, errs
		}

		out := make([][]catalog.ProductImage, len(productIDs))
		for i, id := range productIDs {
			// A product with no images yields nil, which marshals to an empty
			// list. That is the right answer, not an error.
			out[i] = byProduct[id]
		}
		return out, nil
	}
}
