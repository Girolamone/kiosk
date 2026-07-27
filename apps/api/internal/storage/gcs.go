package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"

	gcs "cloud.google.com/go/storage"
)

// GCS stores files in a Cloud Storage bucket. Used in production, where the
// container filesystem is thrown away on every deploy and a restart would
// otherwise take every product photo with it.
type GCS struct {
	client *gcs.Client
	bucket *gcs.BucketHandle
	name   string
}

func NewGCS(ctx context.Context, bucketName string) (*GCS, error) {
	// Credentials come from the environment: the service account on Cloud
	// Run, or the developer's application-default login locally. No key file
	// ever needs to exist, let alone be committed.
	client, err := gcs.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect to cloud storage: %w", err)
	}
	return &GCS{client: client, bucket: client.Bucket(bucketName), name: bucketName}, nil
}

func (g *GCS) Close() error { return g.client.Close() }

func (g *GCS) Put(ctx context.Context, contentType string, r io.Reader) (Object, error) {
	key, err := NewKey(contentType)
	if err != nil {
		return Object{}, err
	}

	writer := g.bucket.Object(key).NewWriter(ctx)
	writer.ContentType = contentType
	// Keys are random and an object is never rewritten, so it can be cached
	// for as long as anyone likes.
	writer.CacheControl = "public, max-age=31536000, immutable"

	size, err := io.Copy(writer, r)
	if err != nil {
		// Close the writer even on failure, or the resumable upload is left
		// open on Google's side.
		_ = writer.Close()
		return Object{}, fmt.Errorf("write to bucket: %w", err)
	}
	// The upload is only durable once Close returns without error; io.Copy
	// succeeding says nothing on its own.
	if err := writer.Close(); err != nil {
		return Object{}, fmt.Errorf("finish upload: %w", err)
	}

	return Object{
		Key:         key,
		URL:         g.URLFor(key),
		ContentType: contentType,
		Size:        size,
	}, nil
}

func (g *GCS) Open(ctx context.Context, key string) (io.ReadCloser, string, error) {
	object := g.bucket.Object(sanitizeKey(key))

	attrs, err := object.Attrs(ctx)
	if errors.Is(err, gcs.ErrObjectNotExist) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", fmt.Errorf("read object attributes: %w", err)
	}

	reader, err := object.NewReader(ctx)
	if errors.Is(err, gcs.ErrObjectNotExist) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", fmt.Errorf("open object: %w", err)
	}
	return reader, attrs.ContentType, nil
}

// URLFor points straight at the bucket. Objects are publicly readable, so the
// browser fetches them without going through this server at all: the API is
// not in the path of every product photo.
func (g *GCS) URLFor(key string) string {
	return "https://storage.googleapis.com/" + g.name + "/" + url.PathEscape(sanitizeKey(key))
}
