package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/luoxunhao/pi-web-go/internal/events"
	"github.com/luoxunhao/pi-web-go/internal/files"
	"github.com/luoxunhao/pi-web-go/internal/pigo"
	"github.com/luoxunhao/pi-web-go/internal/session"
)

func TestAgentEventsSSEConversion(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/events" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		frames := []string{
			"id: 1\nevent: session.status\ndata: {\"id\":1,\"type\":\"session.status\",\"data\":{\"sessionId\":\"s1\",\"messageId\":\"m1\",\"status\":\"running\"}}\n\n",
			"id: 2\nevent: message.part.delta\ndata: {\"id\":2,\"type\":\"message.part.delta\",\"data\":{\"sessionId\":\"s1\",\"messageId\":\"m1\",\"partId\":\"text\",\"delta\":\"hi\"}}\n\n",
			"id: 3\nevent: session.status\ndata: {\"id\":3,\"type\":\"session.status\",\"data\":{\"sessionId\":\"s1\",\"messageId\":\"m1\",\"status\":\"idle\"}}\n\n",
		}
		for _, f := range frames {
			_, _ = fmt.Fprint(w, f)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}))
	defer fake.Close()

	client := pigo.NewClient(fake.URL, "")
	deps := Dependencies{
		PigoClient: client,
		Converter:  events.NewConverter(),
		Cursor:     events.NewCursorStore(),
		SessionMgr: session.NewManager(time.Minute),
	}
	router := NewRouter(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/agent/s1/events", nil)
	req.Host = "127.0.0.1"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	body := rec.Body.String()
	types := collectDataTypes(body)
	want := []string{"agent_start", "message_start", "message_update", "agent_end", "agent_settled", "prompt_done"}
	for _, w := range want {
		found := false
		for _, got := range types {
			if got == w {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing %q in events: %v", w, types)
		}
	}
}

func TestSecurityRequiresPassword(t *testing.T) {
	client := pigo.NewClient("http://127.0.0.1:1", "")
	deps := Dependencies{
		PigoClient:  client,
		Converter:   events.NewConverter(),
		Cursor:      events.NewCursorStore(),
		SessionMgr:  session.NewManager(time.Minute),
		WebPassword: "secret",
	}
	router := NewRouter(deps)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Host = "127.0.0.1"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}

	req.SetBasicAuth("pi", "secret")
	req.Host = "127.0.0.1"
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestSSERequiresPassword(t *testing.T) {
	deps := Dependencies{
		PigoClient:  pigo.NewClient("http://127.0.0.1:1", ""),
		Converter:   events.NewConverter(),
		Cursor:      events.NewCursorStore(),
		SessionMgr:  session.NewManager(time.Minute),
		WebPassword: "secret",
	}
	router := NewRouter(deps)
	req := httptest.NewRequest(http.MethodGet, "/api/agent/s1/events", nil)
	req.Host = "127.0.0.1"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestCORSPreflight(t *testing.T) {
	deps := Dependencies{
		PigoClient: pigo.NewClient("http://127.0.0.1:1", ""),
		Converter:  events.NewConverter(),
		Cursor:     events.NewCursorStore(),
		SessionMgr: session.NewManager(time.Minute),
	}
	router := NewRouter(deps)
	req := httptest.NewRequest(http.MethodOptions, "/api/health", nil)
	req.Host = "127.0.0.1"
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("CORS header = %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestRunningSnapshot(t *testing.T) {
	mgr := session.NewManager(time.Minute)
	mgr.MarkRunning("s1", "/work", "m1")
	deps := Dependencies{
		PigoClient: pigo.NewClient("http://127.0.0.1:1", ""),
		Converter:  events.NewConverter(),
		Cursor:     events.NewCursorStore(),
		SessionMgr: mgr,
	}
	router := NewRouter(deps)
	req := httptest.NewRequest(http.MethodGet, "/api/agent/running", nil)
	req.Host = "127.0.0.1"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"s1"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func collectDataTypes(body string) []string {
	var types []string
	scanner := bufio.NewScanner(strings.NewReader(body))
	var data string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data = strings.TrimPrefix(line, "data: ")
		} else if line == "" && data != "" {
			var ev map[string]interface{}
			if err := json.Unmarshal([]byte(data), &ev); err == nil {
				if typ, ok := ev["type"].(string); ok {
					types = append(types, typ)
				}
			}
			data = ""
		}
	}
	return types
}

var _ io.Reader

func TestProjectTrustGet(t *testing.T) {
	client := pigo.NewClient("http://127.0.0.1:1", "")
	access := files.NewAccess([]string{"/tmp"})
	deps := Dependencies{
		PigoClient:   client,
		Converter:    events.NewConverter(),
		Cursor:       events.NewCursorStore(),
		SessionMgr:   session.NewManager(time.Minute),
		FileAccess:   access,
		AllowedHosts: []string{"localhost", "127.0.0.1", "localhost:5173", "127.0.0.1:5173"},
	}
	router := NewRouter(deps)

	// Missing cwd -> 400
	req := httptest.NewRequest(http.MethodGet, "/api/project-trust", nil)
	req.Host = "localhost:5173"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing cwd: status = %d", rec.Code)
	}

	// Vite proxy origin on port 5173 should now be allowed
	req = httptest.NewRequest(http.MethodGet, "/api/project-trust?cwd=/tmp", nil)
	req.Host = "localhost:5173"
	req.Header.Set("Origin", "http://localhost:5173")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("vite proxy host: status = %d, want 200", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("CORS header = %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}

	// Valid cwd, no pigo client -> returns trusted=false, requiresTrust=false
	req = httptest.NewRequest(http.MethodGet, "/api/project-trust?cwd=/tmp", nil)
	req.Host = "localhost:5173"
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["trusted"] != false {
		t.Fatalf("trusted = %v, want false", resp["trusted"])
	}
}

func TestSessionListAllowsSessionDirectory(t *testing.T) {
	dir := t.TempDir()
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/session" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"sessions": []map[string]interface{}{
				{"sessionId": "s1", "directory": dir},
			},
		})
	}))
	defer fake.Close()

	access := files.NewAccess(nil)
	deps := Dependencies{
		PigoClient: pigo.NewClient(fake.URL, ""),
		Converter:  events.NewConverter(),
		Cursor:     events.NewCursorStore(),
		SessionMgr: session.NewManager(time.Minute),
		FileAccess: access,
	}
	router := NewRouter(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	req.Host = "127.0.0.1"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("session list status = %d", rec.Code)
	}
	if !access.IsAllowed(dir) {
		t.Fatal("session directory was not added to file access allow-list")
	}
	var listBody struct {
		Sessions []map[string]interface{} `json:"sessions"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&listBody); err != nil {
		t.Fatal(err)
	}
	if len(listBody.Sessions) != 1 {
		t.Fatalf("sessions = %#v", listBody.Sessions)
	}
	wantRoot := strings.ReplaceAll(dir, "\\", "/")
	if got := listBody.Sessions[0]["projectRoot"]; got != wantRoot {
		t.Fatalf("projectRoot = %v, want %v", got, wantRoot)
	}

	encodedDir := strings.ReplaceAll(strings.ReplaceAll(dir, "\\", "/"), ":", "%3A")
	req = httptest.NewRequest(http.MethodGet, "/api/files/"+encodedDir+"?type=list", nil)
	req.Host = "127.0.0.1"
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("file list status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestContextFromMessagesNilUsesEmptyArray(t *testing.T) {
	ctx := contextFromMessages(nil)
	messages, ok := ctx["messages"].([]pigo.Message)
	if !ok || messages == nil {
		t.Fatalf("messages = %#v, want non-nil []pigo.Message", ctx["messages"])
	}
}

func TestSessionGetIncludesModelContext(t *testing.T) {
	dir := t.TempDir()
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/load"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"sessionId": "s1", "directory": dir, "messages": []interface{}{}, "hasMore": false,
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/status"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"sessionId": "s1", "directory": dir, "model": "agnes-2.5-flash", "thinkingLevel": "medium",
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/config/providers"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"defaultModel": "openai/agnes-2.5-flash",
				"providers": []map[string]interface{}{
					{"id": "openai", "name": "OpenAI", "models": []map[string]interface{}{
						{"provider": "openai", "modelId": "agnes-2.5-flash"},
					}},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer fake.Close()

	mgr := session.NewManager(time.Minute)
	mgr.SetDirectory("s1", dir)
	deps := Dependencies{
		PigoClient: pigo.NewClient(fake.URL, ""),
		Converter:  events.NewConverter(),
		Cursor:     events.NewCursorStore(),
		SessionMgr: mgr,
	}
	router := NewRouter(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/s1", nil)
	req.Host = "127.0.0.1"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Context struct {
			Model         map[string]interface{} `json:"model"`
			ThinkingLevel string                 `json:"thinkingLevel"`
		} `json:"context"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Context.Model["provider"] != "openai" || body.Context.Model["modelId"] != "agnes-2.5-flash" {
		t.Fatalf("model = %#v", body.Context.Model)
	}
	if body.Context.ThinkingLevel != "medium" {
		t.Fatalf("thinkingLevel = %q", body.Context.ThinkingLevel)
	}
}

// TestRouterStaticHosting verifies the static frontend hosting and SPA
// fallback over an http.FileSystem source (disk dir here; the embedded
// frontend goes through the same http.FS adapter).
func TestRouterStaticHosting(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>spa</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log(1)"), 0o644); err != nil {
		t.Fatal(err)
	}

	deps := Dependencies{
		Converter:  events.NewConverter(),
		Cursor:     events.NewCursorStore(),
		SessionMgr: session.NewManager(time.Minute),
		Static:     http.Dir(dir),
	}
	router := NewRouter(deps)

	cases := []struct {
		name string
		path string
		want int
		body string
	}{
		{"root serves index", "/", http.StatusOK, "<html>spa</html>"},
		{"asset served", "/assets/app.js", http.StatusOK, "console.log(1)"},
		{"client route falls back to index", "/some/client/route", http.StatusOK, "<html>spa</html>"},
		{"api 404 stays json", "/api/not-found", http.StatusNotFound, `{"error":"Not found"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Host = "127.0.0.1"
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("GET %s = %d, want %d (body %q)", tc.path, rec.Code, tc.want, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tc.body) {
				t.Fatalf("GET %s body = %q, want contains %q", tc.path, rec.Body.String(), tc.body)
			}
		})
	}
}

// TestRouterNoStatic confirms the router stays functional (API only) when no
// static source is configured.
func TestRouterNoStatic(t *testing.T) {
	deps := Dependencies{
		Converter:  events.NewConverter(),
		Cursor:     events.NewCursorStore(),
		SessionMgr: session.NewManager(time.Minute),
	}
	router := NewRouter(deps)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "127.0.0.1"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET / without static = %d, want 404", rec.Code)
	}
}
