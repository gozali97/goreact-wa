package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// dist holds the built React frontend. The directory is populated by the
// frontend build (vite build → internal/webui/dist) before `go build`.
//
//go:embed all:dist
var dist embed.FS

// placeholder is served when no frontend build is embedded yet (dev backend).
const placeholder = `<!doctype html><html><head><meta charset="utf-8">
<title>WA Proxy</title></head><body style="font-family:sans-serif;padding:2rem">
<h1>WA Proxy backend is running</h1>
<p>The React dashboard has not been built into this binary yet.</p>
<p>Run <code>make build-frontend</code> (or <code>npm run build</code> in
apps/frontend) then rebuild the Go binary.</p>
<p>The REST API is available under <code>/api</code>.</p>
</body></html>`

// FS returns the embedded dist filesystem rooted at the dist directory.
func subFS() (fs.FS, bool) {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil, false
	}
	// Confirm index.html exists; otherwise treat as no build.
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil, false
	}
	return sub, true
}

// Handler returns an http.Handler that serves the embedded SPA with history
// fallback (unknown non-asset routes return index.html).
func Handler() http.Handler {
	sub, ok := subFS()
	if !ok {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(placeholder))
		})
	}

	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqPath := strings.TrimPrefix(r.URL.Path, "/")
		if reqPath == "" {
			reqPath = "index.html"
		}
		// If the requested file exists, serve it; otherwise SPA fallback.
		if _, err := fs.Stat(sub, reqPath); err != nil {
			if path.Ext(reqPath) != "" {
				// Missing asset with an extension → 404.
				http.NotFound(w, r)
				return
			}
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}
