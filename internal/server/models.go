package server

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/luoxunhao/pi-web-go/internal/pigo"
)

type modelsHandler struct {
	client *pigo.Client
}

func (h *modelsHandler) listModels(w http.ResponseWriter, r *http.Request) {
	cwd := r.URL.Query().Get("cwd")
	if cwd != "" {
		stat, err := os.Stat(cwd)
		if err != nil || !stat.IsDir() {
			modelJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "Directory does not exist: " + cwd})
			return
		}
	}
	cfg, err := h.client.GetConfig(r.Context())
	if err != nil {
		modelJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
		return
	}
	providers, err := h.client.ListProviders(r.Context())
	if err != nil {
		modelJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
		return
	}

	modelList := make([]map[string]interface{}, 0)
	models := map[string]interface{}{}
	thinkingLevels := map[string]interface{}{}
	for _, p := range providers.Providers {
		for _, m := range p.Models {
			id := modelID(p.ID, m.ModelID)
			name := m.Name
			if name == "" {
				name = m.ModelID
			}
			modelList = append(modelList, map[string]interface{}{
				"id": id, "name": name, "provider": p.ID,
			})
			models[p.ID+":"+m.ModelID] = name
			if m.ThinkingLevels != nil {
				thinkingLevels[p.ID+":"+m.ModelID] = *m.ThinkingLevels
			}
		}
	}
	defaultModel := map[string]interface{}{}
	if provider, model := splitModelID(cfg.Model); provider != "" {
		defaultModel = map[string]interface{}{"provider": provider, "modelId": model}
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{
		"models":            models,
		"modelList":         modelList,
		"defaultModel":      defaultModel,
		"thinkingLevels":    thinkingLevels,
		"thinkingLevelMaps": map[string]interface{}{},
		"thinkingLevelPins": map[string]interface{}{},
	})
}

func (h *modelsHandler) getModelsConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.client.GetConfig(r.Context())
	if err != nil {
		modelJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
		return
	}
	providers, err := h.client.ListProviders(r.Context())
	if err != nil {
		modelJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
		return
	}
	providerMap := map[string]interface{}{}
	for _, p := range providers.Providers {
		entry := map[string]interface{}{
			"id":     p.ID,
			"name":   p.Name,
			"models": p.Models,
		}
		if len(p.Models) > 0 {
			entry["baseUrl"] = p.Models[0].BaseURL
			entry["protocol"] = p.Models[0].Protocol
		}
		providerMap[p.ID] = entry
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{
		"providers":    providerMap,
		"defaultModel": cfg.Model,
	})
}

func (h *modelsHandler) putModelsConfig(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Providers    map[string]providerConfig `json:"providers"`
		DefaultModel string                    `json:"defaultModel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		modelJSON(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
		return
	}
	for id, p := range body.Providers {
		models := make([]pigo.ModelEntry, 0, len(p.Models))
		for _, m := range p.Models {
			models = append(models, m)
		}
		input := pigo.ProviderInput{
			Name:     &p.Name,
			BaseURL:  &p.BaseURL,
			Protocol: &p.Protocol,
			Models:   &models,
		}
		if p.APIKey != "" {
			key := p.APIKey
			input.APIKey = &key
		}
		if err := h.client.UpsertProvider(r.Context(), id, input); err != nil {
			modelJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
			return
		}
	}
	if body.DefaultModel != "" {
		if _, err := h.client.UpdateConfig(r.Context(), pigo.UpdateConfigRequest{Model: body.DefaultModel}); err != nil {
			modelJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
			return
		}
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (h *modelsHandler) discoverModels(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProviderName string `json:"providerName"`
		Provider     struct {
			BaseURL string `json:"baseUrl"`
			API     string `json:"api"`
			APIKey  string `json:"apiKey"`
		} `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		modelJSON(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
		return
	}
	if body.ProviderName == "" || body.Provider.BaseURL == "" {
		modelJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "providerName and baseUrl are required"})
		return
	}
	protocol := protocolFromAPI(body.Provider.API)
	result, err := h.client.DiscoverModels(r.Context(), pigo.DiscoverModelsRequest{
		BaseURL:  body.Provider.BaseURL,
		Protocol: &protocol,
		APIKey:   strPtrOrNil(body.Provider.APIKey),
		Name:     &body.ProviderName,
	})
	if err != nil {
		modelJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
		return
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{
		"models":   result.Models,
		"endpoint": body.Provider.BaseURL,
	})
}

func (h *modelsHandler) testModel(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProviderName string `json:"providerName"`
		Model        struct {
			ID string `json:"id"`
		} `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		modelJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": err.Error()})
		return
	}
	if body.ProviderName == "" || body.Model.ID == "" {
		modelJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "providerName and model.id are required"})
		return
	}
	result, err := h.client.TestModel(r.Context(), pigo.TestModelRequest{
		ModelID: modelID(body.ProviderName, body.Model.ID),
	})
	if err != nil {
		modelJSON(w, http.StatusBadGateway, map[string]interface{}{"ok": false, "error": err.Error()})
		return
	}
	response := map[string]interface{}{"ok": result.Success}
	if result.ResponseTimeMs != nil {
		response["latencyMs"] = *result.ResponseTimeMs
	}
	if result.ModelResponse != nil {
		response["responseText"] = *result.ModelResponse
	}
	modelJSON(w, http.StatusOK, response)
}

func (h *modelsHandler) apiKeyGet(w http.ResponseWriter, r *http.Request) {
	providerID := chi.URLParam(r, "provider")
	provider, ok := h.findProvider(r, providerID)
	if !ok {
		modelJSON(w, http.StatusNotFound, map[string]interface{}{"error": "provider not found"})
		return
	}
	configured := false
	for _, m := range provider.Models {
		if m.APIKeyConfigured != nil && *m.APIKeyConfigured {
			configured = true
			break
		}
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{
		"provider": providerID, "displayName": provider.Name,
		"configured": configured, "source": "config", "models": len(provider.Models),
	})
}

func (h *modelsHandler) apiKeyPost(w http.ResponseWriter, r *http.Request) {
	providerID := chi.URLParam(r, "provider")
	provider, ok := h.findProvider(r, providerID)
	if !ok {
		modelJSON(w, http.StatusNotFound, map[string]interface{}{"error": "provider not found"})
		return
	}
	var body struct {
		APIKey string `json:"apiKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.APIKey == "" {
		modelJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "apiKey is required"})
		return
	}
	models := provider.Models
	input := pigo.ProviderInput{
		Name:   &provider.Name,
		Models: &models,
		APIKey: &body.APIKey,
	}
	if err := h.client.UpsertProvider(r.Context(), providerID, input); err != nil {
		modelJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
		return
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (h *modelsHandler) apiKeyDelete(w http.ResponseWriter, r *http.Request) {
	providerID := chi.URLParam(r, "provider")
	provider, ok := h.findProvider(r, providerID)
	if !ok {
		modelJSON(w, http.StatusNotFound, map[string]interface{}{"error": "provider not found"})
		return
	}
	models := provider.Models
	empty := ""
	input := pigo.ProviderInput{Name: &provider.Name, Models: &models, APIKey: &empty}
	if err := h.client.UpsertProvider(r.Context(), providerID, input); err != nil {
		modelJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
		return
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (h *modelsHandler) providers(w http.ResponseWriter, r *http.Request) {
	// OAuth is deferred to M3 (issue 13); the endpoint remains for parity and
	// returns an empty list until pigo exposes OAuth credentials.
	modelJSON(w, http.StatusOK, map[string]interface{}{"providers": []interface{}{}})
}

func (h *modelsHandler) allProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := h.client.ListProviders(r.Context())
	if err != nil {
		modelJSON(w, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
		return
	}
	out := make([]map[string]interface{}, 0, len(providers.Providers))
	for _, p := range providers.Providers {
		out = append(out, map[string]interface{}{
			"id": p.ID, "name": p.Name, "apiKeyLogin": true, "oauth": false, "models": len(p.Models),
		})
	}
	modelJSON(w, http.StatusOK, map[string]interface{}{"providers": out})
}

func (h *modelsHandler) findProvider(r *http.Request, providerID string) (pigo.Provider, bool) {
	providers, err := h.client.ListProviders(r.Context())
	if err != nil {
		return pigo.Provider{}, false
	}
	for _, p := range providers.Providers {
		if p.ID == providerID {
			return p, true
		}
	}
	return pigo.Provider{}, false
}

type providerConfig struct {
	Name     string            `json:"name"`
	BaseURL  string            `json:"baseUrl"`
	Protocol string            `json:"protocol"`
	APIKey   string            `json:"apiKey"`
	Models   []pigo.ModelEntry `json:"models"`
}

func protocolFromAPI(api string) string {
	switch api {
	case "anthropic-messages":
		return "anthropic"
	case "google-generative-ai":
		return "google"
	default:
		return "openai"
	}
}

func modelID(provider, model string) string {
	return provider + "/" + model
}

func splitModelID(id string) (string, string) {
	provider, model, ok := strings.Cut(id, "/")
	if !ok {
		return "", ""
	}
	return provider, model
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func modelJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
