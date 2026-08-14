package events

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/luoxunhao/pi-web-go/internal/pigo"
)

func TestStreamHandlerConnectedAndReplay(t *testing.T) {
	var gotAfter string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAfter = r.URL.Query().Get("after")
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "id: 9\nevent: session.status\ndata: {\"id\":9,\"type\":\"session.status\",\"data\":{\"sessionId\":\"s1\",\"status\":\"idle\"}}\n\n")
	}))
	defer upstream.Close()

	h := &StreamHandler{
		Client:    pigo.NewClient(upstream.URL, ""),
		Cursor:    NewCursorStore(),
		Converter: NewConverter(),
		Heartbeat: time.Hour,
	}
	router := chi.NewRouter()
	router.Get("/api/agent/{id}/events", h.ServeHTTP)
	server := httptest.NewServer(router)
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/agent/s1/events?directory=C%3A%5Cwork", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Last-Event-ID", "5")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if gotAfter != "5" {
		t.Fatalf("after = %q, want 5", gotAfter)
	}
	if c := h.Cursor.Get("s1"); c != 9 {
		t.Fatalf("cursor = %d, want 9", c)
	}
	body := string(bodyBytes)
	for _, want := range []string{
		`"type":"connected"`,
		`"isStreaming":false`,
		"id: 9\n",
		`"type":"agent_end"`,
		`"type":"agent_settled"`,
		`"type":"prompt_done"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
}

func TestStreamHandlerQueryAfterWins(t *testing.T) {
	var gotAfter string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAfter = r.URL.Query().Get("after")
		w.Header().Set("Content-Type", "text/event-stream")
	}))
	defer upstream.Close()

	h := &StreamHandler{
		Client:    pigo.NewClient(upstream.URL, ""),
		Cursor:    NewCursorStore(),
		Converter: NewConverter(),
		Heartbeat: time.Hour,
	}
	router := chi.NewRouter()
	router.Get("/api/agent/{id}/events", h.ServeHTTP)
	server := httptest.NewServer(router)
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/agent/s1/events?after=3&directory=C%3A%5Cwork", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Last-Event-ID", "5")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if gotAfter != "3" {
		t.Fatalf("after = %q, want 3", gotAfter)
	}
	if !strings.Contains(string(bodyBytes), `"type":"connected"`) {
		t.Fatalf("missing connected event:\n%s", string(bodyBytes))
	}
}
