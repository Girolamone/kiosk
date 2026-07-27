package graph

import (
	"context"
	"errors"
	"io"

	"github.com/Girolamone/kiosk/apps/api/graph/model"
	"github.com/Girolamone/kiosk/apps/api/internal/ai"
	"github.com/Girolamone/kiosk/apps/api/internal/auth"
	"github.com/Girolamone/kiosk/apps/api/internal/storage"
)

// GenerateProductCopy is the resolver for the generateProductCopy field.
func (r *mutationResolver) GenerateProductCopy(ctx context.Context, imageKey string) (*model.ProductCopy, error) {
	// Generation costs quota, so it is not open to anonymous callers.
	if _, err := auth.RequireUser(ctx); err != nil {
		return nil, err
	}

	file, contentType, err := r.Files.Open(ctx, imageKey)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, errors.New("no such image: upload it first")
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Bounded read. Open takes a key from the client, and a caller should not
	// be able to make the server hold an unbounded amount of a file in memory.
	data, err := io.ReadAll(io.LimitReader(file, storage.MaxUploadBytes))
	if err != nil {
		return nil, err
	}

	generated, err := r.Copywriter.GenerateProductCopy(ctx, ai.Image{
		ContentType: contentType,
		Data:        data,
	})
	if err != nil {
		// ai.ErrUnavailable arrives here when the provider is unset, down or
		// confused. It fails this field only, so a client asking for copy
		// alongside other work still gets the rest of its query.
		return nil, err
	}

	return &model.ProductCopy{
		Title:       generated.Title,
		Description: generated.Description,
		AltText:     generated.AltText,
	}, nil
}
