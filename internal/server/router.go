package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/luoxunhao/pi-web-go/internal/events"
)

// NewRouter builds the pi-web compatible HTTP surface.
func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()
	r.Use(Security(deps.WebPassword, deps.AllowedHosts))

	r.Get("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"healthy": true})
	})

	stream := &events.StreamHandler{
		Client:    deps.PigoClient,
		Cursor:    deps.Cursor,
		Converter: deps.Converter,
	}
	r.Get("/api/agent/{id}/events", stream.ServeHTTP)
	return r
}
