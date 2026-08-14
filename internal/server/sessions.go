package server

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/luoxunhao/pi-web-go/internal/export"
	"github.com/luoxunhao/pi-web-go/internal/files"
	"github.com/luoxunhao/pi-web-go/internal/pigo"
	"github.com/luoxunhao/pi-web-go/internal/session"
)

type sessionsHandler struct {
	client     *pigo.Client
	sessions   *session.Manager
	fileAccess *files.Access
}

func (h *sessionsHandler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.client.ListSessions(r.Context(), "", 200)
	if err != nil {
		modelJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
		return
	}
	out := make([]map[string]interface{}, 0, len(list.Sessions))
	for _, s := range list.Sessions {
		h.allowDirectory(s.Directory)
		item := map[string]interface{}{
			"id": s.SessionID, "cwd": s.Directory, "name": strPtrValue(s.Title),
			"title": strPtrValue(s.Title), "created": strPtrValue(s.UpdatedAt),
			"modified": strPtrValue(s.UpdatedAt), "updatedAt": strPtrValue(s.UpdatedAt),
			"parentSessionId": strPtrValue(s.ParentSessionID), "messageCount": 0,
			"firstMessage": "", "transient": false,
			"projectRoot": sessionProjectRoot(s.Directory),
		}
		out = append(out, item)
	}
	running := []string{}
	if h.sessions != nil {
		running = h.sessions.RunningIDs()
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{"sessions": out, "runningSessionIds": running})
}

func (h *sessionsHandler) get(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	directory := h.resolveDirectory(r.Context(), sessionID)
	if directory == "" {
		modelJSON(w, http.StatusNotFound, map[string]interface{}{"error": "Session not found"})
		return
	}
	leafID := r.URL.Query().Get("leafId")
	load, err := h.client.LoadSession(r.Context(), sessionID, pigo.LoadSessionRequest{
		Directory: directory, LeafID: strPtrOrNil(leafID),
	})
	if err != nil {
		modelJSON(w, http.StatusNotFound, map[string]interface{}{"error": err.Error()})
		return
	}
	status, _ := h.client.GetSessionStatus(r.Context(), sessionID, directory)
	context := contextFromMessages(load.Messages)
	if status.Model != nil {
		context["model"] = sessionModelContext(r.Context(), h.client, *status.Model)
	} else {
		context["model"] = nil
	}
	if status.ThinkingLevel != nil {
		context["thinkingLevel"] = *status.ThinkingLevel
	}
	info := map[string]interface{}{
		"id": sessionID, "cwd": directory, "name": "", "created": "", "modified": strPtrValue(load.NextCursor),
		"messageCount": len(load.Messages), "firstMessage": firstUserMessage(load.Messages), "transient": false,
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{
		"sessionId": sessionID, "filePath": "", "info": info, "leafId": strPtrValue(load.CurrentLeafID),
		"tree": []interface{}{}, "context": context, "totalActiveMs": 0,
	})
}

func (h *sessionsHandler) patch(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	directory := h.resolveDirectory(r.Context(), sessionID)
	if directory == "" {
		modelJSON(w, http.StatusNotFound, map[string]interface{}{"error": "Session not found"})
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		modelJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "name is required"})
		return
	}
	if _, err := h.client.ExecuteCommand(r.Context(), sessionID, pigo.CommandRequest{
		Directory: directory, Command: "name", Arguments: &body.Name,
	}); err != nil {
		modelJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
		return
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func (h *sessionsHandler) delete(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	directory := h.resolveDirectory(r.Context(), sessionID)
	if directory == "" {
		modelJSON(w, http.StatusNotFound, map[string]interface{}{"error": "Session not found"})
		return
	}
	if err := h.client.DeleteSession(r.Context(), sessionID, directory); err != nil {
		modelJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
		return
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func (h *sessionsHandler) context(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	directory := h.resolveDirectory(r.Context(), sessionID)
	if directory == "" {
		modelJSON(w, http.StatusNotFound, map[string]interface{}{"error": "Session not found"})
		return
	}
	leafID := r.URL.Query().Get("leafId")
	load, err := h.client.LoadSession(r.Context(), sessionID, pigo.LoadSessionRequest{
		Directory: directory, LeafID: strPtrOrNil(leafID),
	})
	if err != nil {
		modelJSON(w, http.StatusNotFound, map[string]interface{}{"error": err.Error()})
		return
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{"context": contextFromMessages(load.Messages)})
}

func (h *sessionsHandler) state(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	running := h.sessions != nil && containsString(h.sessions.RunningIDs(), sessionID)
	if !running {
		directory := h.resolveDirectory(r.Context(), sessionID)
		if directory == "" {
			modelJSON(w, http.StatusNotFound, map[string]interface{}{"error": "Session not found"})
			return
		}
		modelJSON(w, http.StatusOK, map[string]interface{}{"running": false})
		return
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{"running": true, "state": map[string]interface{}{
		"sessionId": sessionID, "isStreaming": true,
	}})
}

func (h *sessionsHandler) autoName(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	directory := h.resolveDirectory(r.Context(), sessionID)
	if directory == "" {
		modelJSON(w, http.StatusNotFound, map[string]interface{}{"error": "Session not found"})
		return
	}
	messages, err := h.client.GetMessages(r.Context(), sessionID, directory, "", 200)
	if err != nil {
		modelJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
		return
	}
	title := firstUserMessage(messages.Messages)
	if title == "" {
		title = "Session"
	}
	if _, err := h.client.ExecuteCommand(r.Context(), sessionID, pigo.CommandRequest{
		Directory: directory, Command: "name", Arguments: &title,
	}); err != nil {
		modelJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
		return
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{"title": title, "usage": nil})
}

func (h *sessionsHandler) exportHTML(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	directory := h.resolveDirectory(r.Context(), sessionID)
	if directory == "" {
		modelJSON(w, http.StatusNotFound, map[string]interface{}{"error": "Session not found"})
		return
	}
	messages, err := h.client.GetMessages(r.Context(), sessionID, directory, "", 200)
	if err != nil {
		modelJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
		return
	}
	body := export.SessionHTML(sessionID, messages.Messages)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.URL.Query().Get("inline") != "1" {
		w.Header().Set("Content-Disposition", `attachment; filename="`+sessionID+`.html"`)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

func (h *sessionsHandler) resolveDirectory(ctx context.Context, sessionID string) string {
	if h.sessions != nil {
		if dir := h.sessions.Directory(sessionID); dir != "" {
			return dir
		}
	}
	list, err := h.client.ListSessions(ctx, "", 200)
	if err != nil {
		return ""
	}
	for _, s := range list.Sessions {
		if s.SessionID == sessionID {
			if h.sessions != nil {
				h.sessions.SetDirectory(sessionID, s.Directory)
			}
			h.allowDirectory(s.Directory)
			return s.Directory
		}
	}
	return ""
}

func (h *sessionsHandler) allowDirectory(dir string) {
	if dir != "" && h.fileAccess != nil {
		_ = h.fileAccess.Add(dir)
	}
}

func sessionProjectRoot(cwd string) string {
	if cwd == "" {
		return ""
	}
	root, err := gitOutput(cwd, "rev-parse", "--show-toplevel")
	if err == nil && strings.TrimSpace(root) != "" {
		return strings.TrimSpace(root)
	}
	return filepath.ToSlash(filepath.Clean(cwd))
}

func sessionModelContext(ctx context.Context, client *pigo.Client, model string) map[string]interface{} {
	provider, modelID := splitModelID(model)
	if provider == "" {
		if providers, err := client.ListProviders(ctx); err == nil {
			for _, p := range providers.Providers {
				for _, m := range p.Models {
					if m.ModelID == modelID {
						return map[string]interface{}{"provider": p.ID, "modelId": m.ModelID}
					}
				}
			}
		}
	}
	return map[string]interface{}{"provider": provider, "modelId": modelID}
}

func firstUserMessage(messages []pigo.Message) string {
	for _, m := range messages {
		if m.Role != "user" {
			continue
		}
		for _, block := range m.Content {
			if block["type"] == "text" {
				if text, ok := block["text"].(string); ok && strings.TrimSpace(text) != "" {
					return truncate(text, 60)
				}
			}
		}
	}
	return ""
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func contextFromMessages(messages []pigo.Message) map[string]interface{} {
	if messages == nil {
		messages = []pigo.Message{}
	}
	entryIDs := make([]interface{}, 0, len(messages))
	for _, m := range messages {
		if m.EntryID != nil {
			entryIDs = append(entryIDs, *m.EntryID)
		} else {
			entryIDs = append(entryIDs, m.ID)
		}
	}
	return map[string]interface{}{"messages": messages, "entryIds": entryIDs}
}
