package httpapi

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// SPA serves the built React app.
//
// A single-page app owns its own routes, so a request for /dashboard/my-shop
// has no matching file on disk. Serving index.html for anything that is not a
// real file is what makes a page survive being reloaded or opened from a
// bookmark, instead of answering 404 for every route but the first.
func SPA(dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))
	index := filepath.Join(dir, "index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))

		// A path that names a file and is not there is missing, not a route.
		// Falling back to index.html for it answers a request for
		// /favicon.ico with a page of HTML and a 200, which is a lie the
		// browser then has to work out for itself.
		if ext := path.Ext(r.URL.Path); ext != "" && !exists(requested) {
			http.NotFound(w, r)
			return
		}

		if info, err := os.Stat(requested); err == nil && !info.IsDir() {
			// Hashed asset filenames change whenever their content changes,
			// so they can be cached hard. index.html cannot: it is what
			// points at the current hashes.
			if strings.HasPrefix(r.URL.Path, "/assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			files.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, index)
	})
}

func exists(name string) bool {
	info, err := os.Stat(name)
	return err == nil && !info.IsDir()
}
