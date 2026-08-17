package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
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
		eh := &engineeringHandler{access: deps.FileAccess, pigoClient: deps.PigoClient}
		r.Get("/api/home", eh.home)
		r.Get("/api/cwd/browse", eh.cwdBrowse)
		r.Post("/api/cwd/validate", eh.cwdValidate)
		r.Post("/api/default-cwd", eh.defaultCwd)
		r.Get("/api/git/status", eh.gitStatus)
		r.Get("/api/git/diff", eh.gitDiff)
		r.Get("/api/worktrees", eh.worktrees)
		r.Post("/api/worktrees", eh.worktreeAdd)
		r.Delete("/api/worktrees", eh.worktreeRemove)
		r.Get("/api/file-index", eh.fileIndex)
		r.Get("/api/app-update", eh.appUpdate)
		r.Get("/api/project-trust", eh.projectTrustGet)
		r.Post("/api/project-trust", eh.projectTrustPost)
	}
	if deps.PigoClient != nil && deps.SessionMgr != nil {
		sh := &sessionsHandler{client: deps.PigoClient, sessions: deps.SessionMgr, fileAccess: deps.FileAccess}
		r.Get("/api/sessions", sh.list)
		r.Get("/api/sessions/{id}", sh.get)
		r.Patch("/api/sessions/{id}", sh.patch)
		r.Delete("/api/sessions/{id}", sh.delete)
		r.Get("/api/sessions/{id}/context", sh.context)
		r.Get("/api/sessions/{id}/state", sh.state)
		r.Post("/api/sessions/{id}/auto-name", sh.autoName)
		r.Get("/api/sessions/{id}/export", sh.exportHTML)
	}
	if deps.PigoClient != nil && deps.SessionMgr != nil && deps.FileAccess != nil {
		r.Get("/api/agent/{id}/bash-output", (&engineeringHandler{access: deps.FileAccess}).bashOutput)
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
	if deps.Static != nil {
		fileServer := http.FileServer(deps.Static)
		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				writeNotFound(w)
				return
			}
			// Serve the exact file when it exists; everything else is the SPA
			// entry point (client-side routing), mirroring the disk-only
			// behavior but working for both http.Dir and embedded FS.
			if serveStaticFile(deps.Static, fileServer, w, r) {
				return
			}
			serveStaticIndex(deps.Static, w, r)
		})
	}
	return r
}

// serveStaticFile serves r.URL.Path from fsys when it names an existing file.
// It returns false when the path is a directory or missing, leaving the caller
// to fall back to the SPA entry point.
func serveStaticFile(fsys http.FileSystem, fileServer http.Handler, w http.ResponseWriter, r *http.Request) bool {
	name := path.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if name == "" || name == "." || strings.Contains(name, "..") {
		// Root, directories, and traversal attempts all go to the SPA entry
		// (embed.FS rejects ".." outright; keep the guard for http.Dir).
		return false
	}
	f, err := fsys.Open(name)
	if err != nil {
		return false
	}
	f.Close()
	fileServer.ServeHTTP(w, r)
	return true
}

// serveStaticIndex serves the SPA entry point (index.html) with a correct
// Content-Type, working for both embedded and disk filesystems.
func serveStaticIndex(fsys http.FileSystem, w http.ResponseWriter, r *http.Request) {
	f, err := fsys.Open("index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	if rs, ok := f.(io.ReadSeeker); ok {
		http.ServeContent(w, r, "index.html", time.Time{}, rs)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.Copy(w, f)
}

func writeNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_, _ = fmt.Fprint(w, `{"error":"Not found"}`)
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
