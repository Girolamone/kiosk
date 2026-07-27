// Package ai turns a product photo into listing copy.
//
// The provider sits behind an interface for two reasons. One is the usual
// one: it can be swapped. The other is that it can be switched off. Copy
// generation is an accelerator, not a requirement — if no key is configured,
// or the provider is down, the seller writes the description themselves and
// nothing else about the product breaks.
package ai

import (
	"context"
	"errors"
)

// ErrUnavailable means copy generation could not run. Callers surface it to
// the user as "write it yourself this time", never as a failure of the
// operation they were actually doing.
var ErrUnavailable = errors.New("copy generation is unavailable")

// Image is a product photo to write about.
type Image struct {
	ContentType string
	Data        []byte
}

// ProductCopy is what a listing needs written.
type ProductCopy struct {
	Title       string
	Description string
	// AltText describes the photo for screen readers and for the case where
	// the image fails to load. It is not the description.
	AltText string
}

type CopyGenerator interface {
	GenerateProductCopy(ctx context.Context, image Image) (ProductCopy, error)
}

// Disabled stands in when no provider is configured, so the rest of the
// application never has to check whether the feature exists.
type Disabled struct{}

func (Disabled) GenerateProductCopy(context.Context, Image) (ProductCopy, error) {
	return ProductCopy{}, ErrUnavailable
}
