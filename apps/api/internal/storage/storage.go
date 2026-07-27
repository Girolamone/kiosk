// Package storage keeps uploaded files behind one interface, so nothing else
// in the application has to know whether a product photo lives on local disk
// or in a bucket. Local disk is what makes the project runnable with no cloud
// account; the bucket is what production uses.
package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// MaxUploadBytes caps one upload. Product photos do not need more, and an
// unbounded reader is a way to fill a disk.
const MaxUploadBytes = 8 << 20 // 8 MiB

var (
	ErrUnsupportedType = errors.New("only JPEG, PNG, WebP and GIF images are accepted")
	ErrNotFound        = errors.New("no such object")
)

// Object is a stored file. The tags matter: this struct is serialised
// straight to the upload response, and Go's exported-field names are not the
// shape a JavaScript client expects.
type Object struct {
	Key         string `json:"key"` // opaque storage key, e.g. "a1b2....jpg"
	URL         string `json:"url"` // where a browser can fetch it
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
}

// Store is the whole storage contract. Local and GCS both satisfy it.
type Store interface {
	Put(ctx context.Context, contentType string, r io.Reader) (Object, error)
	Open(ctx context.Context, key string) (io.ReadCloser, string, error)
}

// imageTypes is the allowlist, mapping detected type to file extension.
var imageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

// Sniff identifies content by its actual bytes and returns a reader with the
// bytes it consumed put back.
//
// The browser-supplied Content-Type is deliberately not consulted: it is
// attacker controlled, and a caller can label an HTML file as image/png.
// Deciding from the bytes is what stops that file being stored and later
// served back as something a browser will execute.
func Sniff(r io.Reader) (contentType string, body io.Reader, err error) {
	header := make([]byte, 512)
	n, err := io.ReadFull(r, header)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", nil, fmt.Errorf("read upload: %w", err)
	}
	header = header[:n]

	detected := http.DetectContentType(header)
	if _, ok := imageTypes[detected]; !ok {
		return "", nil, ErrUnsupportedType
	}

	return detected, io.MultiReader(bytes.NewReader(header), r), nil
}

// NewKey returns an unguessable storage key carrying the right extension.
//
// The original filename is discarded on purpose. It is attacker controlled,
// can contain path separators, and leaks whatever the uploader called the
// file.
func NewKey(contentType string) (string, error) {
	ext, ok := imageTypes[contentType]
	if !ok {
		return "", ErrUnsupportedType
	}

	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}
	return hex.EncodeToString(buf) + ext, nil
}
