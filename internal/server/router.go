package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/luoxunhao/pi-web-go/internal/events"
	"github.com/luoxunhao/pi-web-go/internal/files"
	"github.com/luoxunhao/pi-web-go/internal/session"
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
	if deps.SessionMgr != nil {
		stream.OnEvent = deps.SessionMgr.ObserveEvent
	}
	r.Get("/api/agent/{id}/events", stream.ServeHTTP)

	r.Get("/api/agent/running", func(w http.ResponseWriter, _ *http.Request) {
		ids := []string{}
		if deps.SessionMgr != nil {
			ids = deps.SessionMgr.RunningIDs()
		}
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"runningSessionIds": ids})
	})
	r.Get("/api/agent/running/events", runningEventsHandler(deps.SessionMgr))

	if deps.PigoClient != nil && deps.SessionMgr != nil {
		ah := &agentHandler{client: deps.PigoClient, sessions: deps.SessionMgr, fileAccess: deps.FileAccess}
		r.Post("/api/agent/new", ah.newSession)
		r.Get("/api/agent/{id}", ah.get)
		r.Post("/api/agent/{id}", ah.command)
	}

	if deps.FileAccess != nil {
		r.Handle("/api/files/*", &files.Handler{Access: deps.FileAccess})
	}
	if deps.PigoClient != nil {
		mh := &modelsHandler{client: deps.PigoClient}
		r.Get("/api/models", mh.listModels)
		r.Get("/api/models-config", mh.getModelsConfig)
		r.Put("/api/models-config", mh.putModelsConfig)
		r.Post("/api/models-config/discover", mh.discoverModels)
		r.Post("/api/models-config/test", mh.testModel)
		r.Get("/api/auth/api-key/{provider}", mh.apiKeyGet)
		r.Post("/api/auth/api-key/{provider}", mh.apiKeyPost)
		r.Delete("/api/auth/api-key/{provider}", mh.apiKeyDelete)
		r.Get("/api/auth/providers", mh.providers)
		r.Get("/api/auth/all-providers", mh.allProviders)
	}
	return r
}

func runningEventsHandler(mgr *session.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		if mgr == nil {
			_, _ = fmt.Fprint(w, "data: {\"type\":\"running\",\"runningSessionIds\":[]}\n\n")
			flusher.Flush()
			<-r.Context().Done()
			return
		}

		ch, unsubscribe := mgr.Subscribe()
		defer unsubscribe()
		encode := func(ids []string) {
			_, _ = fmt.Fprintf(w, "data: {\"type\":\"running\",\"runningSessionIds\":%s}\n\n", jsonString(ids))
			flusher.Flush()
		}
		encode(mgr.RunningIDs())

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case ids := <-ch:
				encode(ids)
			case <-ticker.C:
				_, _ = fmt.Fprint(w, ":\n\n")
				flusher.Flush()
			}
		}
	}
}

func jsonString(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
