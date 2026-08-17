package webui

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeIndex writes an index.html marker into dir.
func writeIndex(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>disk</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// embedded returns whether a real frontend build currently lives in dist/
// (i.e. frontend-embed has run and committed assets were copied in).
func embedded() bool { return hasIndex(dist, "dist") }

// TestStaticFSServesIndex is state-independent: whatever the source (embedded
// build or disk dir), StaticFS must return a working filesystem whose root
// serves the SPA entry.
func TestStaticFSServesIndex(t *testing.T) {
	dir := t.TempDir()
	writeIndex(t, dir)
	fsys := StaticFS(dir)
	if fsys == nil {
		t.Fatal("StaticFS = nil, want a serving filesystem")
	}
	rec := httptest.NewRecorder()
	http.FileServer(fsys).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "<html") {
		t.Fatalf("GET / body = %q, want an HTML page", body)
	}
}

// TestStaticFSDiskFallback exercises the development path (embedded tree
// empty, disk frontend_dir present → http.Dir). Skipped once a real frontend
// build has been embedded, where the embedded source takes priority.
func TestStaticFSDiskFallback(t *testing.T) {
	if embedded() {
		t.Skip("embedded frontend present; disk fallback not exercised")
	}
	dir := t.TempDir()
	writeIndex(t, dir)
	fsys := StaticFS(dir)
	if fsys == nil {
		t.Fatal("StaticFS = nil, want disk http.Dir")
	}
	// GET / resolves through the directory index to index.html. (Note: a
	// direct /index.html request 301-redirects to ./ — http.FileServer's
	// canonical-path behavior — so exercise the root path instead.)
	rec := httptest.NewRecorder()
	http.FileServer(fsys).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "disk") {
		t.Fatalf("GET / body = %q, want disk marker", body)
	}
}

// TestStaticFSEmbeddedPriority verifies the production path: when the
// embedded tree has a real build, it wins over the disk directory (even one
// that does not exist). Skipped on a fresh clone before frontend-embed ran.
func TestStaticFSEmbeddedPriority(t *testing.T) {
	if !embedded() {
		t.Skip("no embedded frontend yet; embedded priority not exercised")
	}
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	fsys := StaticFS(missing)
	if fsys == nil {
		t.Fatal("StaticFS = nil, want embedded http.FS")
	}
	rec := httptest.NewRecorder()
	http.FileServer(fsys).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "<html") {
		t.Fatalf("GET / body = %q, want an HTML page", body)
	}
}

// TestStaticFSNil verifies that with neither an embedded build nor an existing
// disk directory, StaticFS returns nil so callers disable static hosting.
func TestStaticFSNil(t *testing.T) {
	if embedded() {
		t.Skip("embedded frontend present; nil case not exercisable")
	}
	if got := StaticFS(filepath.Join(t.TempDir(), "missing")); got != nil {
		t.Fatalf("StaticFS = %#v, want nil", got)
	}
}

// TestHasIndexDisk exercises the hasIndex helper against a temp dir.
func TestHasIndexDisk(t *testing.T) {
	dir := t.TempDir()
	if hasIndex(os.DirFS(dir), ".") {
		t.Fatal("empty dir reported as having index.html")
	}
	if err := os.WriteFile(filepath.Join(dir, "index.html"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if !hasIndex(os.DirFS(dir), ".") {
		t.Fatal("dir with index.html not detected")
	}
}
