package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/luoxunhao/pi-web-go/internal/events"
	"github.com/luoxunhao/pi-web-go/internal/pigo"
	"github.com/luoxunhao/pi-web-go/internal/session"
)

func TestModelsProxy(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/config":
			_, _ = fmt.Fprint(w, `{"model":"openrouter/free","models":[]}`)
		case "/api/v1/config/providers":
			_, _ = fmt.Fprint(w, `{"defaultModel":"openrouter/free","providers":[{"id":"openrouter","name":"OpenRouter","models":[{"provider":"openrouter","modelId":"free","name":"Free","baseUrl":"https://openrouter.ai/api/v1","protocol":"openai","apiKeyConfigured":true}]}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer fake.Close()

	deps := Dependencies{
		PigoClient: pigo.NewClient(fake.URL, ""),
		Converter:  events.NewConverter(),
		Cursor:     events.NewCursorStore(),
		SessionMgr: session.NewManager(0),
	}
	router := NewRouter(deps)
	req := httptest.NewRequest(http.MethodGet, "/api/models", nil)
	req.Host = "127.0.0.1"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		ModelList []map[string]interface{} `json:"modelList"`
		Default   map[string]interface{}   `json:"defaultModel"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.ModelList) != 1 {
		t.Fatalf("modelList = %#v", body.ModelList)
	}
	if body.ModelList[0]["id"] != "free" {
		t.Fatalf("modelList[0].id = %v, want free", body.ModelList[0]["id"])
	}
	if body.Default["provider"] != "openrouter" {
		t.Fatalf("default = %#v", body.Default)
	}
}
