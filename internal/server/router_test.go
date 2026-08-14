package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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
	want := []string{"agent_start", "message_start", "message_update", "agent_end", "prompt_done"}
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
