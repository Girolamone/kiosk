// Package httpapi holds the HTTP endpoints that sit alongside GraphQL.
//
// File upload is one of them on purpose. Pushing binary through GraphQL means
// base64 in a JSON document: a third larger on the wire, and the whole thing
// buffered in memory before anything can look at it. A plain multipart POST
// streams.
package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Girolamone/kiosk/apps/api/internal/auth"
	"github.com/Girolamone/kiosk/apps/api/internal/storage"
)

// Upload accepts one image and returns where it was stored. The caller then
// attaches the key to a product through GraphQL.
func Upload(store storage.Store, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := auth.RequireUser(r.Context()); err != nil {
			writeError(w, http.StatusUnauthorized, "sign in to upload")
			return
		}

		// MaxBytesReader caps the request itself, so an oversized upload is
		// refused as it arrives rather than after it has been read.
		r.Body = http.MaxBytesReader(w, r.Body, storage.MaxUploadBytes)

		file, _, err := r.FormFile("file")
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				writeError(w, http.StatusRequestEntityTooLarge, "that image is larger than 8 MB")
				return
			}
			writeError(w, http.StatusBadRequest, "expected a multipart form with a 'file' field")
			return
		}
		defer file.Close()

		contentType, body, err := storage.Sniff(file)
		if errors.Is(err, storage.ErrUnsupportedType) {
			writeError(w, http.StatusUnsupportedMediaType, storage.ErrUnsupportedType.Error())
			return
		}
		if err != nil {
			logger.Error("sniff upload", "err", err)
			writeError(w, http.StatusBadRequest, "could not read that file")
			return
		}

		object, err := store.Put(r.Context(), contentType, body)
		if err != nil {
			logger.Error("store upload", "err", err)
			writeError(w, http.StatusInternalServerError, "could not store that file")
			return
		}

		writeJSON(w, http.StatusCreated, object)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
