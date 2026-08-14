package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/luoxunhao/pi-web-go/internal/files"
	"github.com/luoxunhao/pi-web-go/internal/pigo"
	"github.com/luoxunhao/pi-web-go/internal/session"
)

type agentHandler struct {
	client     *pigo.Client
	sessions   *session.Manager
	fileAccess *files.Access
}

func (h *agentHandler) newSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Cwd           string   `json:"cwd"`
		Type          string   `json:"type"`
		Message       string   `json:"message"`
		Provider      string   `json:"provider"`
		ModelID       string   `json:"modelId"`
		ThinkingLevel string   `json:"thinkingLevel"`
		ToolNames     []string `json:"toolNames"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		agentJSON(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
		return
	}
	if body.Cwd == "" {
		agentJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "cwd is required"})
		return
	}
	stat, err := os.Stat(body.Cwd)
	if err != nil || !stat.IsDir() {
		agentJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "Directory does not exist: " + body.Cwd})
		return
	}

	model := ""
	if body.Provider != "" || body.ModelID != "" {
		model = modelID(body.Provider, body.ModelID)
	} else if cfg, err := h.client.GetConfig(r.Context()); err == nil {
		model = cfg.Model
	}
	created, err := h.client.CreateSession(r.Context(), pigo.NewSessionRequest{
		Directory: body.Cwd,
		Model:     strPtrOrNil(model),
	})
	if err != nil {
		agentJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error(), "code": "prompt_rejected", "accepted": false})
		return
	}
	h.sessions.SetDirectory(created.SessionID, body.Cwd)
	if h.fileAccess != nil {
		_ = h.fileAccess.Add(body.Cwd)
	}
	if body.ThinkingLevel != "" {
		_, _ = h.client.UpdateSessionConfig(r.Context(), created.SessionID, pigo.UpdateSessionRequest{
			Directory: body.Cwd, ThinkingLevel: &body.ThinkingLevel,
		})
	}
	if body.Message != "" {
		_, err = h.client.PromptAsync(r.Context(), created.SessionID, pigo.PromptRequest{
			Directory: body.Cwd,
			Prompt:    []map[string]interface{}{{"type": "text", "text": body.Message}},
		})
		if err != nil {
			agentJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error(), "code": "prompt_rejected", "accepted": false})
			return
		}
	}
	provider, modelID := splitModelID(model)
	agentJSON(w, http.StatusOK, map[string]interface{}{
		"success": true, "sessionId": created.SessionID, "data": nil,
		"model":         map[string]interface{}{"provider": provider, "modelId": modelID},
		"thinkingLevel": body.ThinkingLevel,
	})
}

func (h *agentHandler) get(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	running := h.sessions != nil && containsString(h.sessions.RunningIDs(), sessionID)
	state := map[string]interface{}{
		"sessionId": sessionID, "isStreaming": running, "isPromptRunning": running,
		"isBashRunning": false, "isCompacting": false, "queuedMessages": map[string]interface{}{
			"steering": []interface{}{}, "followUp": []interface{}{},
		},
		"contextUsage": nil, "systemPrompt": nil,
	}
	if !running {
		agentJSON(w, http.StatusOK, map[string]interface{}{"running": false})
		return
	}
	agentJSON(w, http.StatusOK, map[string]interface{}{"running": true, "state": state})
}

func (h *agentHandler) command(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		agentJSON(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
		return
	}
	typ, _ := body["type"].(string)
	directory := h.resolveDirectory(r.Context(), sessionID)
	if directory == "" {
		agentJSON(w, http.StatusNotFound, map[string]interface{}{
			"error": "Session not found", "code": "prompt_rejected", "accepted": false,
		})
		return
	}

	switch typ {
	case "prompt", "steer", "follow_up":
		if images, _ := body["images"].([]interface{}); len(images) > 0 {
			agentJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "Image attachments are not supported by pigo serve yet"})
			return
		}
		message, _ := body["message"].(string)
		if _, err := h.client.PromptAsync(r.Context(), sessionID, pigo.PromptRequest{
			Directory: directory,
			Prompt:    []map[string]interface{}{{"type": "text", "text": message}},
		}); err != nil {
			agentJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error(), "code": "prompt_rejected", "accepted": false})
			return
		}
		agentJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": nil})
	case "abort":
		if err := h.client.CancelPrompt(r.Context(), sessionID); err != nil {
			agentJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
			return
		}
		agentJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": nil})
	case "get_commands":
		commands, err := h.client.ListCommands(r.Context())
		if err != nil {
			agentJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
			return
		}
		out := make([]map[string]interface{}, 0, len(commands.Commands))
		for _, c := range commands.Commands {
			out = append(out, map[string]interface{}{
				"name": c.Name, "description": c.Description, "source": "prompt",
			})
		}
		agentJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": map[string]interface{}{"commands": out}})
	case "get_state":
		status, _ := h.client.GetSessionStatus(r.Context(), sessionID, directory)
		agentJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": map[string]interface{}{
			"sessionId": sessionID, "isStreaming": h.sessions != nil && containsString(h.sessions.RunningIDs(), sessionID),
			"isPromptRunning": false, "isBashRunning": false, "isCompacting": false,
			"queuedMessages": map[string]interface{}{"steering": []interface{}{}, "followUp": []interface{}{}},
			"contextUsage":   nil, "systemPrompt": nil,
			"model":         map[string]interface{}{"provider": providerOf(status.Model), "modelId": modelOf(status.Model)},
			"thinkingLevel": strPtrValue(status.ThinkingLevel),
		}})
	case "set_model":
		provider, _ := body["provider"].(string)
		modelIDValue, _ := body["modelId"].(string)
		model := modelID(provider, modelIDValue)
		if _, err := h.client.UpdateSessionConfig(r.Context(), sessionID, pigo.UpdateSessionRequest{
			Directory: directory, Model: &model,
		}); err != nil {
			agentJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
			return
		}
		agentJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": map[string]interface{}{"id": modelIDValue, "provider": provider}})
	case "set_thinking_level":
		level, _ := body["level"].(string)
		if _, err := h.client.UpdateSessionConfig(r.Context(), sessionID, pigo.UpdateSessionRequest{
			Directory: directory, ThinkingLevel: &level,
		}); err != nil {
			agentJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
			return
		}
		agentJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": nil})
	case "set_session_name":
		name, _ := body["name"].(string)
		if _, err := h.client.ExecuteCommand(r.Context(), sessionID, pigo.CommandRequest{
			Directory: directory, Command: "name", Arguments: &name,
		}); err != nil {
			agentJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
			return
		}
		agentJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": nil})
	case "compact":
		args := ""
		if v, ok := body["customInstructions"].(string); ok {
			args = v
		}
		result, err := h.client.ExecuteCommand(r.Context(), sessionID, pigo.CommandRequest{
			Directory: directory, Command: "compact", Arguments: &args,
		})
		if err != nil {
			agentJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
			return
		}
		agentJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": map[string]interface{}{
			"reason": "manual", "text": strPtrValue(result.Text),
		}})
	case "fork":
		entryID, _ := body["entryId"].(string)
		newID, err := h.forkAt(r.Context(), sessionID, directory, entryID)
		if err != nil {
			agentJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
			return
		}
		agentJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": map[string]interface{}{"cancelled": false, "newSessionId": newID}})
	case "clone":
		result, err := h.client.ExecuteCommand(r.Context(), sessionID, pigo.CommandRequest{Directory: directory, Command: "clone"})
		if err != nil {
			agentJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
			return
		}
		agentJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": map[string]interface{}{"cancelled": false, "newSessionId": parseSessionIDFromText(strPtrValue(result.Text))}})
	case "navigate_tree":
		targetID, _ := body["targetId"].(string)
		if _, err := h.client.LoadSession(r.Context(), sessionID, pigo.LoadSessionRequest{Directory: directory, LeafID: &targetID}); err != nil {
			agentJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
			return
		}
		agentJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": map[string]interface{}{"cancelled": false}})
	case "get_session_stats":
		messages, _ := h.client.GetMessages(r.Context(), sessionID, directory, "", 200)
		agentJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": map[string]interface{}{
			"messageCount": len(messages.Messages), "sessionName": "",
		}})
	case "get_last_assistant_text":
		messages, _ := h.client.GetMessages(r.Context(), sessionID, directory, "", 200)
		text := lastAssistantText(messages.Messages)
		agentJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": map[string]interface{}{"text": text}})
	case "get_tools":
		agentJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": []interface{}{}})
	case "reload", "set_tools", "set_auto_compaction", "clear_queue", "bash", "abort_bash",
		"extension_ui_response", "extension_ui_input", "abort_compaction":
		agentJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "Command not supported by pigo backend: " + typ, "code": "unsupported"})
	default:
		agentJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "Unsupported command: " + typ})
	}
}

func (h *agentHandler) resolveDirectory(ctx context.Context, sessionID string) string {
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
			return s.Directory
		}
	}
	return ""
}

func (h *agentHandler) forkAt(ctx context.Context, sessionID, directory, entryID string) (string, error) {
	messages, err := h.client.GetMessages(ctx, sessionID, directory, "", 200)
	if err != nil {
		return "", err
	}
	index := 0
	for _, m := range messages.Messages {
		if m.Role != "user" {
			continue
		}
		index++
		if m.ID == entryID || (m.EntryID != nil && *m.EntryID == entryID) {
			n := strconv.Itoa(index)
			result, err := h.client.ExecuteCommand(ctx, sessionID, pigo.CommandRequest{
				Directory: directory, Command: "fork", Arguments: &n,
			})
			if err != nil {
				return "", err
			}
			return parseSessionIDFromText(strPtrValue(result.Text)), nil
		}
	}
	return "", fmt.Errorf("invalid entry ID for forking: %s", entryID)
}

var sessionIDPattern = regexp.MustCompile(`[0-9a-zA-Z-]{8,64}`)

func parseSessionIDFromText(text string) string {
	if m := sessionIDPattern.FindString(text); m != "" {
		return m
	}
	return ""
}

func lastAssistantText(messages []pigo.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]
		if m.Role != "assistant" {
			continue
		}
		var b strings.Builder
		for _, block := range m.Content {
			if block["type"] == "text" {
				if text, ok := block["text"].(string); ok {
					b.WriteString(text)
				}
			}
		}
		if b.Len() > 0 {
			return b.String()
		}
	}
	return ""
}

func providerOf(model *string) string {
	if model == nil {
		return ""
	}
	p, _ := splitModelID(*model)
	return p
}

func modelOf(model *string) string {
	if model == nil {
		return ""
	}
	_, m := splitModelID(*model)
	return m
}

func strPtrValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func agentJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
