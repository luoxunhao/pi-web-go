// Package webui embeds the built frontend SPA so production deployments are a
// single binary with no external static directory (decision E1: embed in
// production, read from disk in development).
//
// The embedded tree lives in dist/ and is populated by `make build`
// (frontend-embed copies frontend/dist here). The placeholder .gitkeep keeps
// the package compiling on a fresh clone before the frontend has been built.
// When the embedded tree has no index.html, StaticFS falls back to the
// configured disk directory (frontend_dir), which is the development mode.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path"
)

//go:embed all:dist
var dist embed.FS

// StaticFS returns the http.FileSystem serving the frontend:
//   - the embedded build when it contains an index.html (production build), or
//   - the disk directory frontendDir when it exists (development mode), or
//   - nil when neither is available (static hosting is then disabled).
//
// nil is a valid value: callers skip static routing when it is returned.
func StaticFS(frontendDir string) http.FileSystem {
	if hasIndex(dist, "dist") {
		sub, err := fs.Sub(dist, "dist")
		if err == nil {
			return http.FS(sub)
		}
	}
	if frontendDir != "" && dirHasIndex(frontendDir) {
		return http.Dir(frontendDir)
	}
	return nil
}

// hasIndex reports whether dir inside fsys contains an index.html entry.
func hasIndex(fsys fs.FS, dir string) bool {
	_, err := fs.Stat(fsys, path.Join(dir, "index.html"))
	return err == nil
}

// dirHasIndex reports whether dir on disk contains an index.html file.
func dirHasIndex(dir string) bool {
	_, err := os.Stat(path.Join(dir, "index.html"))
	return err == nil
}
