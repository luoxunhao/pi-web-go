package pigo

// Types below mirror the pigo HTTP API (GET /api/v1/openapi.json). They are
// intentionally kept local to pi-web-go; agent configuration remains owned by
// pigo.

type Session struct {
	SessionID         string             `json:"sessionId"`
	Directory         string             `json:"directory"`
	ConfigOptions     []ConfigOption     `json:"configOptions"`
	AvailableModes    []Mode             `json:"availableModes"`
	AvailableCommands []AvailableCommand `json:"availableCommands"`
}

type ConfigOption struct {
	ID           string                    `json:"id"`
	Name         string                    `json:"name"`
	Type         string                    `json:"type"`
	CurrentValue *string                   `json:"currentValue,omitempty"`
	Options      *[]map[string]interface{} `json:"options,omitempty"`
}

type Mode struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Model        *string   `json:"model,omitempty"`
	SystemPrompt *string   `json:"systemPrompt,omitempty"`
	Tools        *[]string `json:"tools,omitempty"`
}

type AvailableCommand struct {
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	StructuredKinds *[]string `json:"structuredKinds,omitempty"`
}

type SessionSummary struct {
	SessionID        string  `json:"sessionId"`
	Directory        string  `json:"directory"`
	Title            *string `json:"title,omitempty"`
	UpdatedAt        *string `json:"updatedAt,omitempty"`
	ParentSessionID  *string `json:"parentSessionId,omitempty"`
	ParentToolCallID *string `json:"parentToolCallId,omitempty"`
	SubagentType     *string `json:"subagentType,omitempty"`
	SessionKind      *string `json:"sessionKind,omitempty"`
}

type SessionListResult struct {
	Sessions   []SessionSummary `json:"sessions"`
	NextCursor *string          `json:"nextCursor,omitempty"`
}

type LaneState struct {
	Lane   string  `json:"lane"`
	LeafID *string `json:"leafId,omitempty"`
}

type SessionStatusResult struct {
	SessionID     string       `json:"sessionId"`
	Status        string       `json:"status"`
	Model         *string      `json:"model,omitempty"`
	Mode          *string      `json:"mode,omitempty"`
	ThinkingLevel *string      `json:"thinkingLevel,omitempty"`
	QueuedCount   *int         `json:"queuedCount,omitempty"`
	CurrentLeafID *string      `json:"currentLeafId,omitempty"`
	CurrentLane   *string      `json:"currentLane,omitempty"`
	Lanes         *[]LaneState `json:"lanes,omitempty"`
}

type SessionLoadResult struct {
	SessionID     string         `json:"sessionId"`
	Directory     string         `json:"directory"`
	ConfigOptions []ConfigOption `json:"configOptions"`
	Messages      []Message      `json:"messages"`
	HasMore       bool           `json:"hasMore"`
	NextCursor    *string        `json:"nextCursor,omitempty"`
	CurrentLeafID *string        `json:"currentLeafId,omitempty"`
	CurrentLane   *string        `json:"currentLane,omitempty"`
	Lanes         *[]LaneState   `json:"lanes,omitempty"`
}

type Message struct {
	ID        string                   `json:"id"`
	Role      string                   `json:"role"`
	Timestamp string                   `json:"timestamp"`
	Content   []map[string]interface{} `json:"content"`
	EntryID   *string                  `json:"entryId,omitempty"`
	EntryType *string                  `json:"entryType,omitempty"`
	ParentID  *string                  `json:"parentId,omitempty"`
	Model     *string                  `json:"model,omitempty"`
	RawInput  *map[string]interface{}  `json:"rawInput,omitempty"`
	RawOutput *string                  `json:"rawOutput,omitempty"`
	Usage     *map[string]interface{}  `json:"usage,omitempty"`
}

type MessageListResult struct {
	Messages   []Message `json:"messages"`
	HasMore    bool      `json:"hasMore"`
	NextCursor *string   `json:"nextCursor,omitempty"`
}

type NewSessionRequest struct {
	Directory             string                    `json:"directory"`
	Model                 *string                   `json:"model,omitempty"`
	Mode                  *string                   `json:"mode,omitempty"`
	Title                 *string                   `json:"title,omitempty"`
	AdditionalDirectories *[]string                 `json:"additionalDirectories,omitempty"`
	McpServers            *[]map[string]interface{} `json:"mcpServers,omitempty"`
}

type UpdateSessionRequest struct {
	Directory     string  `json:"directory"`
	Model         *string `json:"model,omitempty"`
	ThinkingLevel *string `json:"thinkingLevel,omitempty"`
	Mode          *string `json:"mode,omitempty"`
}

type ConfigOptionsResult struct {
	ConfigOptions []ConfigOption `json:"configOptions"`
}

type SetModeRequest struct {
	Directory string `json:"directory"`
	ModeID    string `json:"modeId"`
}

type ModeResult struct {
	CurrentModeID  string `json:"currentModeId"`
	AvailableModes []Mode `json:"availableModes"`
}

type LoadSessionRequest struct {
	Directory             string    `json:"directory"`
	LeafID                *string   `json:"leafId,omitempty"`
	Before                *string   `json:"before,omitempty"`
	Limit                 *int      `json:"limit,omitempty"`
	AdditionalDirectories *[]string `json:"additionalDirectories,omitempty"`
}

type PromptRequest struct {
	Directory     string                   `json:"directory"`
	Prompt        []map[string]interface{} `json:"prompt"`
	Model         *string                  `json:"model,omitempty"`
	ThinkingLevel *string                  `json:"thinkingLevel,omitempty"`
	Mode          *string                  `json:"mode,omitempty"`
}

type PromptResponse struct {
	MessageID  string                  `json:"messageId"`
	StopReason string                  `json:"stopReason"`
	Text       *string                 `json:"text,omitempty"`
	Structured *map[string]interface{} `json:"structured,omitempty"`
	Usage      *map[string]interface{} `json:"usage,omitempty"`
}

type PromptAsyncResponse struct {
	MessageID string `json:"messageId"`
	Accepted  bool   `json:"accepted"`
}

type CommandRequest struct {
	Directory string  `json:"directory"`
	Command   string  `json:"command"`
	Arguments *string `json:"arguments,omitempty"`
}

type CommandListResult struct {
	Commands []AvailableCommand `json:"commands"`
}

type ModelEntry struct {
	Provider         string    `json:"provider"`
	ModelID          string    `json:"modelId"`
	Name             string    `json:"name"`
	BaseURL          string    `json:"baseUrl"`
	Protocol         string    `json:"protocol"`
	ContextWindow    *int      `json:"contextWindow,omitempty"`
	MaxTokens        *int      `json:"maxTokens,omitempty"`
	ThinkingLevels   *[]string `json:"thinkingLevels,omitempty"`
	SupportsImages   *bool     `json:"supportsImages,omitempty"`
	Enabled          *bool     `json:"enabled,omitempty"`
	APIKeyConfigured *bool     `json:"apiKeyConfigured,omitempty"`
}

type ConfigResult struct {
	Model  string       `json:"model"`
	Models []ModelEntry `json:"models"`
}

type UpdateConfigRequest struct {
	Model string `json:"model"`
}

type Provider struct {
	ID     string       `json:"id"`
	Name   string       `json:"name"`
	Models []ModelEntry `json:"models"`
}

type ProvidersResult struct {
	DefaultModel string     `json:"defaultModel"`
	Providers    []Provider `json:"providers"`
}

type DiscoverModelsRequest struct {
	BaseURL  string  `json:"baseUrl"`
	Protocol *string `json:"protocol,omitempty"`
	APIKey   *string `json:"apiKey,omitempty"`
	Name     *string `json:"name,omitempty"`
}

type DiscoveredModel struct {
	ModelID        string    `json:"modelId"`
	Name           string    `json:"name"`
	ContextWindow  *int      `json:"contextWindow,omitempty"`
	MaxTokens      *int      `json:"maxTokens,omitempty"`
	SupportsImages *bool     `json:"supportsImages,omitempty"`
	ThinkingLevels *[]string `json:"thinkingLevels,omitempty"`
}

type DiscoverModelsResult struct {
	Provider string            `json:"provider"`
	BaseURL  string            `json:"baseUrl"`
	Protocol string            `json:"protocol"`
	Models   []DiscoveredModel `json:"models"`
}

type TestModelRequest struct {
	ModelID string `json:"modelId"`
}

type TestModelResult struct {
	Success        bool    `json:"success"`
	ResponseTimeMs *int    `json:"responseTimeMs,omitempty"`
	ModelResponse  *string `json:"modelResponse,omitempty"`
}

type ProviderInput struct {
	Name     *string       `json:"name,omitempty"`
	BaseURL  *string       `json:"baseUrl,omitempty"`
	Protocol *string       `json:"protocol,omitempty"`
	APIKey   *string       `json:"apiKey,omitempty"`
	Models   *[]ModelEntry `json:"models,omitempty"`
}

type ModesResult struct {
	Modes []Mode `json:"modes"`
}

type TrustEntry struct {
	Path  string `json:"path"`
	Trust *bool  `json:"trust,omitempty"`
}

type TrustListResult struct {
	Entries []TrustEntry `json:"entries"`
}

type SetTrustRequest struct {
	Path  string `json:"path"`
	Trust *bool  `json:"trust,omitempty"`
}

type PermissionReplyRequest struct {
	OptionID string `json:"optionId"`
}

// DomainEvent is the envelope pigo writes as one SSE data frame.
type DomainEvent struct {
	ID   int64                  `json:"id"`
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
	Time string                 `json:"time"`
}

type ErrorDetail struct {
	Code      string                  `json:"code"`
	Message   string                  `json:"message"`
	RequestID string                  `json:"requestId"`
	Details   *map[string]interface{} `json:"details,omitempty"`
}

type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}
