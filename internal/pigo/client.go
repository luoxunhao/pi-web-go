package pigo

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to pigo serve's HTTP API. It only carries transport details;
// model/provider configuration is read and written through pigo.
type Client struct {
	BaseURL  string
	Password string
	HTTP     *http.Client
}

type Option func(*Client)

func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.HTTP = hc }
}

func NewClient(baseURL, password string, opts ...Option) *Client {
	c := &Client{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		Password: password,
		HTTP:     &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("pigo api error %d (%s): %s", e.Status, e.Code, e.Message)
	}
	return fmt.Sprintf("pigo api error %d: %s", e.Status, e.Message)
}

func (c *Client) request(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	u := c.BaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.Password != "" {
		req.SetBasicAuth("pigo", c.Password)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeError(resp)
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func decodeError(resp *http.Response) error {
	var body ErrorBody
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err := json.Unmarshal(data, &body); err == nil && body.Error.Message != "" {
		return &APIError{Status: resp.StatusCode, Code: body.Error.Code, Message: body.Error.Message}
	}
	return &APIError{Status: resp.StatusCode, Message: strings.TrimSpace(string(data))}
}

func strPtr(s string) *string { return &s }

// CreateSession creates a pigo session.
func (c *Client) CreateSession(ctx context.Context, req NewSessionRequest) (Session, error) {
	var out Session
	err := c.request(ctx, http.MethodPost, "/api/v1/session", nil, req, &out)
	return out, err
}

func (c *Client) ListSessions(ctx context.Context, directory string, limit int) (SessionListResult, error) {
	var out SessionListResult
	q := url.Values{}
	if directory != "" {
		q.Set("directory", directory)
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	err := c.request(ctx, http.MethodGet, "/api/v1/session", q, nil, &out)
	return out, err
}

func (c *Client) DeleteSession(ctx context.Context, sessionID, directory string) error {
	q := url.Values{"directory": {directory}}
	return c.request(ctx, http.MethodDelete, "/api/v1/session/"+url.PathEscape(sessionID), q, nil, nil)
}

func (c *Client) CloseSession(ctx context.Context, sessionID, directory string) error {
	q := url.Values{"directory": {directory}}
	return c.request(ctx, http.MethodPost, "/api/v1/session/"+url.PathEscape(sessionID)+"/close", q, nil, nil)
}

func (c *Client) UpdateSessionConfig(ctx context.Context, sessionID string, req UpdateSessionRequest) (ConfigOptionsResult, error) {
	var out ConfigOptionsResult
	err := c.request(ctx, http.MethodPatch, "/api/v1/session/"+url.PathEscape(sessionID), nil, req, &out)
	return out, err
}

func (c *Client) SetMode(ctx context.Context, sessionID string, req SetModeRequest) (ModeResult, error) {
	var out ModeResult
	err := c.request(ctx, http.MethodPost, "/api/v1/session/"+url.PathEscape(sessionID)+"/mode", nil, req, &out)
	return out, err
}

func (c *Client) LoadSession(ctx context.Context, sessionID string, req LoadSessionRequest) (SessionLoadResult, error) {
	var out SessionLoadResult
	err := c.request(ctx, http.MethodPost, "/api/v1/session/"+url.PathEscape(sessionID)+"/load", nil, req, &out)
	return out, err
}

func (c *Client) GetSessionStatus(ctx context.Context, sessionID, directory string) (SessionStatusResult, error) {
	var out SessionStatusResult
	q := url.Values{"directory": {directory}}
	err := c.request(ctx, http.MethodGet, "/api/v1/session/"+url.PathEscape(sessionID)+"/status", q, nil, &out)
	return out, err
}

func (c *Client) GetMessages(ctx context.Context, sessionID, directory, before string, limit int) (MessageListResult, error) {
	var out MessageListResult
	q := url.Values{"directory": {directory}}
	if before != "" {
		q.Set("before", before)
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	err := c.request(ctx, http.MethodGet, "/api/v1/session/"+url.PathEscape(sessionID)+"/messages", q, nil, &out)
	return out, err
}

func (c *Client) PromptSync(ctx context.Context, sessionID string, req PromptRequest) (PromptResponse, error) {
	var out PromptResponse
	err := c.request(ctx, http.MethodPost, "/api/v1/session/"+url.PathEscape(sessionID)+"/prompt", nil, req, &out)
	return out, err
}

func (c *Client) PromptAsync(ctx context.Context, sessionID string, req PromptRequest) (PromptAsyncResponse, error) {
	var out PromptAsyncResponse
	err := c.request(ctx, http.MethodPost, "/api/v1/session/"+url.PathEscape(sessionID)+"/prompt_async", nil, req, &out)
	return out, err
}

func (c *Client) CancelPrompt(ctx context.Context, sessionID string) error {
	return c.request(ctx, http.MethodPost, "/api/v1/session/"+url.PathEscape(sessionID)+"/cancel", nil, nil, nil)
}

func (c *Client) ListCommands(ctx context.Context) (CommandListResult, error) {
	var out CommandListResult
	err := c.request(ctx, http.MethodGet, "/api/v1/commands", nil, nil, &out)
	return out, err
}

func (c *Client) ExecuteCommand(ctx context.Context, sessionID string, req CommandRequest) (PromptResponse, error) {
	var out PromptResponse
	err := c.request(ctx, http.MethodPost, "/api/v1/session/"+url.PathEscape(sessionID)+"/command", nil, req, &out)
	return out, err
}

func (c *Client) GetConfig(ctx context.Context) (ConfigResult, error) {
	var out ConfigResult
	err := c.request(ctx, http.MethodGet, "/api/v1/config", nil, nil, &out)
	return out, err
}

func (c *Client) UpdateConfig(ctx context.Context, req UpdateConfigRequest) (ConfigResult, error) {
	var out ConfigResult
	err := c.request(ctx, http.MethodPatch, "/api/v1/config", nil, req, &out)
	return out, err
}

func (c *Client) ListProviders(ctx context.Context) (ProvidersResult, error) {
	var out ProvidersResult
	err := c.request(ctx, http.MethodGet, "/api/v1/config/providers", nil, nil, &out)
	return out, err
}

func (c *Client) DiscoverModels(ctx context.Context, req DiscoverModelsRequest) (DiscoverModelsResult, error) {
	var out DiscoverModelsResult
	err := c.request(ctx, http.MethodPost, "/api/v1/config/providers/discover", nil, req, &out)
	return out, err
}

func (c *Client) TestModel(ctx context.Context, req TestModelRequest) (TestModelResult, error) {
	var out TestModelResult
	err := c.request(ctx, http.MethodPost, "/api/v1/config/providers/test", nil, req, &out)
	return out, err
}

func (c *Client) UpsertProvider(ctx context.Context, providerID string, req ProviderInput) error {
	return c.request(ctx, http.MethodPut, "/api/v1/config/providers/"+url.PathEscape(providerID), nil, req, nil)
}

func (c *Client) DeleteProvider(ctx context.Context, providerID string) error {
	return c.request(ctx, http.MethodDelete, "/api/v1/config/providers/"+url.PathEscape(providerID), nil, nil, nil)
}

func (c *Client) ListModes(ctx context.Context, directory string) (ModesResult, error) {
	var out ModesResult
	q := url.Values{}
	if directory != "" {
		q.Set("directory", directory)
	}
	err := c.request(ctx, http.MethodGet, "/api/v1/modes", q, nil, &out)
	return out, err
}

func (c *Client) ListTrust(ctx context.Context) (TrustListResult, error) {
	var out TrustListResult
	err := c.request(ctx, http.MethodGet, "/api/v1/permission/trust", nil, nil, &out)
	return out, err
}

func (c *Client) SetTrust(ctx context.Context, req SetTrustRequest) error {
	return c.request(ctx, http.MethodPost, "/api/v1/permission/trust", nil, req, nil)
}

func (c *Client) DeleteTrust(ctx context.Context, path string) error {
	q := url.Values{"path": {path}}
	return c.request(ctx, http.MethodDelete, "/api/v1/permission/trust", q, nil, nil)
}

func (c *Client) ReplyPermission(ctx context.Context, sessionID, permissionID, optionID string) error {
	req := PermissionReplyRequest{OptionID: optionID}
	return c.request(ctx, http.MethodPost, "/api/v1/session/"+url.PathEscape(sessionID)+"/permissions/"+url.PathEscape(permissionID)+"/reply", nil, req, nil)
}

// StreamEvents follows pigo's SSE stream and calls fn for each parsed event.
// The returned function cancels the subscription.
func (c *Client) StreamEvents(ctx context.Context, sessionID, directory string, after int64, types string, fn func(DomainEvent) error) error {
	q := url.Values{}
	if sessionID != "" {
		q.Set("sessionId", sessionID)
	}
	if directory != "" {
		q.Set("directory", directory)
	}
	if after > 0 {
		q.Set("after", fmt.Sprintf("%d", after))
	}
	if types != "" {
		q.Set("types", types)
	}
	u := c.BaseURL + "/api/v1/events?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	if c.Password != "" {
		req.SetBasicAuth("pigo", c.Password)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return decodeError(resp)
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	var id string
	var eventName string
	var data strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "id:"):
			id = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
		case strings.HasPrefix(line, "event:"):
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data.WriteString(strings.TrimPrefix(line, "data:"))
		case line == "":
			if data.Len() > 0 {
				var ev DomainEvent
				if err := json.Unmarshal([]byte(data.String()), &ev); err != nil {
					data.Reset()
					continue
				}
				if ev.Type == "" {
					ev.Type = eventName
				}
				if ev.ID == 0 && id != "" {
					fmt.Sscanf(id, "%d", &ev.ID)
				}
				if err := fn(ev); err != nil {
					return err
				}
			}
			id, eventName = "", ""
			data.Reset()
		default:
			// heartbeat comment lines are ignored.
		}
	}
	return scanner.Err()
}
