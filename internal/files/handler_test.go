package files

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func testRouter(t *testing.T) (*Access, http.Handler) {
	t.Helper()
	dir := t.TempDir()
	access := NewAccess([]string{dir})
	r := chi.NewRouter()
	r.Handle("/api/files/*", &Handler{Access: access})
	return access, r
}

func TestListAndRead(t *testing.T) {
	_, router := testRouter(t)
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "sub"), 0o700)
	_ = os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o600)

	req := httptest.NewRequest(http.MethodGet, "/api/files/"+strings.ReplaceAll(dir, "\\", "/")+"?type=list", nil)
	req.Host = "127.0.0.1"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("list outside allowed root = %d", rec.Code)
	}

	allowed := t.TempDir()
	_ = os.WriteFile(filepath.Join(allowed, "a.txt"), []byte("hello"), 0o600)
	access := NewAccess([]string{allowed})
	r := chi.NewRouter()
	r.Handle("/api/files/*", &Handler{Access: access})
	req = httptest.NewRequest(http.MethodGet, "/api/files/"+strings.ReplaceAll(filepath.Join(allowed, "a.txt"), "\\", "/")+"?type=read", nil)
	req.Host = "127.0.0.1"
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("read status = %d", rec.Code)
	}
	var body struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Content != "hello" {
		t.Fatalf("content = %q", body.Content)
	}
}

func TestAccessDeniedOutsideRoot(t *testing.T) {
	access := NewAccess([]string{t.TempDir()})
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if access.IsAllowed(outside) {
		t.Fatal("outside root should be denied")
	}
}
