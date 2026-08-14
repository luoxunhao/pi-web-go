// Package events converts pigo's domain SSE events into the event wire
// pi-web's useAgentSession already understands.
package events

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/luoxunhao/pi-web-go/internal/pigo"
)

// WireEvent is one data frame sent to the browser. It mirrors the loose JSON
// events emitted by pi-web's agent event stream.
type WireEvent map[string]interface{}

// CursorStore tracks the last pigo event id per session so a browser
// reconnection can replay from the next event.
type CursorStore struct {
	mu      sync.Mutex
	cursors map[string]int64
}

func NewCursorStore() *CursorStore {
	return &CursorStore{cursors: make(map[string]int64)}
}

func (s *CursorStore) Get(sessionID string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cursors[sessionID]
}

func (s *CursorStore) Set(sessionID string, id int64) {
	if sessionID == "" || id <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if id > s.cursors[sessionID] {
		s.cursors[sessionID] = id
	}
}

// Converter keeps per-message streaming state while translating pigo events.
type Converter struct {
	mu      sync.Mutex
	streams map[string]*streamState
}

type streamState struct {
	messageID   string
	started     bool
	blocks      []map[string]interface{}
	textIndex   int
	thinkIndex  int
	toolIndexes map[string]int
}

func NewConverter() *Converter {
	return &Converter{streams: make(map[string]*streamState)}
}

func streamKey(sessionID, messageID string) string {
	return sessionID + "\x00" + messageID
}

func (c *Converter) state(sessionID, messageID string) *streamState {
	key := streamKey(sessionID, messageID)
	st, ok := c.streams[key]
	if !ok {
		st = &streamState{
			messageID:   messageID,
			textIndex:   -1,
			thinkIndex:  -1,
			toolIndexes: make(map[string]int),
		}
		c.streams[key] = st
	}
	return st
}

func (c *Converter) clear(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.streams {
		if key[:len(sessionID)+1] == sessionID+"\x00" {
			delete(c.streams, key)
		}
	}
}

// Convert translates one pigo DomainEvent into zero or more browser wire
// events. A nil result means the event should not be forwarded.
func (c *Converter) Convert(ev pigo.DomainEvent) []WireEvent {
	sessionID, _ := ev.Data["sessionId"].(string)
	messageID, _ := ev.Data["messageId"].(string)
	switch ev.Type {
	case "session.status":
		return c.convertStatus(sessionID, ev.Data)
	case "queue.updated":
		return []WireEvent{{
			"type":        "queue_update",
			"steering":    []interface{}{},
			"followUp":    []interface{}{},
			"queuedCount": ev.Data["queuedCount"],
		}}
	case "message.part.delta":
		return c.convertDelta(sessionID, messageID, ev.Data)
	case "tool.updated":
		return c.convertTool(sessionID, messageID, ev.Data)
	case "permission.asked":
		wire := WireEvent{"type": "permission_asked"}
		for k, v := range ev.Data {
			wire[k] = v
		}
		return []WireEvent{wire}
	default:
		return nil
	}
}

func (c *Converter) convertStatus(sessionID string, data map[string]interface{}) []WireEvent {
	status, _ := data["status"].(string)
	switch status {
	case "running":
		return []WireEvent{{"type": "agent_start"}}
	case "idle":
		c.clear(sessionID)
		// Match pi SDK terminal order: agent_settled clears the streaming flag
		// before prompt_done, so the web UI settles immediately.
		return []WireEvent{{"type": "agent_end"}, {"type": "agent_settled"}, {"type": "prompt_done"}}
	case "cancelled":
		c.clear(sessionID)
		// Same terminal order for aborted turns.
		return []WireEvent{{"type": "agent_end"}, {"type": "agent_settled"}, {"type": "prompt_done"}}
	case "error":
		wire := WireEvent{"type": "prompt_error"}
		if msg, ok := data["error"].(string); ok {
			wire["errorMessage"] = msg
		}
		return []WireEvent{wire}
	case "compacting":
		return []WireEvent{{"type": "compaction_start"}}
	case "compacted":
		return []WireEvent{{"type": "compaction_end"}}
	case "compaction_failed":
		wire := WireEvent{"type": "compaction_end"}
		if msg, ok := data["error"].(string); ok {
			wire["errorMessage"] = msg
		}
		return []WireEvent{wire}
	default:
		return nil
	}
}

func (c *Converter) convertDelta(sessionID, messageID string, data map[string]interface{}) []WireEvent {
	if sessionID == "" || messageID == "" {
		return nil
	}
	partID, _ := data["partId"].(string)
	delta, _ := data["delta"].(string)
	if partID == "thinking" {
		if v, ok := data["thinking"].(string); ok {
			delta = v
		}
	}
	if delta == "" {
		return nil
	}

	c.mu.Lock()
	st := c.state(sessionID, messageID)
	out := make([]WireEvent, 0, 3)
	if !st.started {
		st.started = true
		out = append(out, WireEvent{
			"type": "message_start",
			"message": map[string]interface{}{
				"role":    "assistant",
				"content": []interface{}{},
			},
		})
	}

	switch partID {
	case "thinking":
		if st.thinkIndex == -1 {
			st.thinkIndex = len(st.blocks)
			st.blocks = append(st.blocks, map[string]interface{}{"type": "thinking", "thinking": ""})
			out = append(out, WireEvent{
				"type": "message_update",
				"assistantMessageEvent": map[string]interface{}{
					"type":         "thinking_start",
					"contentIndex": st.thinkIndex,
				},
			})
		}
		out = append(out, WireEvent{
			"type": "message_update",
			"assistantMessageEvent": map[string]interface{}{
				"type":         "thinking_delta",
				"contentIndex": st.thinkIndex,
				"delta":        delta,
			},
		})
	case "text", "goal":
		if st.textIndex == -1 {
			st.textIndex = len(st.blocks)
			st.blocks = append(st.blocks, map[string]interface{}{"type": "text", "text": ""})
			out = append(out, WireEvent{
				"type": "message_update",
				"assistantMessageEvent": map[string]interface{}{
					"type":         "text_start",
					"contentIndex": st.textIndex,
				},
			})
		}
		out = append(out, WireEvent{
			"type": "message_update",
			"assistantMessageEvent": map[string]interface{}{
				"type":         "text_delta",
				"contentIndex": st.textIndex,
				"delta":        delta,
			},
		})
	}
	c.mu.Unlock()

	// Refresh the snapshot carried by message_start so streamReducer has a
	// consistent initial message even when the first event is a thinking delta.
	if len(out) > 0 && out[0]["type"] == "message_start" {
		c.mu.Lock()
		out[0]["message"] = map[string]interface{}{
			"role":    "assistant",
			"content": st.blocks,
		}
		c.mu.Unlock()
	}
	return out
}

func (c *Converter) convertTool(sessionID, messageID string, data map[string]interface{}) []WireEvent {
	if sessionID == "" || messageID == "" {
		return nil
	}
	toolCallID, _ := data["toolCallId"].(string)
	toolName, _ := data["title"].(string)
	status, _ := data["status"].(string)
	if toolCallID == "" || toolName == "" {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	st := c.state(sessionID, messageID)
	idx, seen := st.toolIndexes[toolCallID]
	out := make([]WireEvent, 0, 2)
	switch status {
	case "pending", "in_progress":
		if !seen {
			idx = len(st.blocks)
			st.blocks = append(st.blocks, map[string]interface{}{
				"type": "toolCall", "toolCallId": toolCallID, "toolName": toolName,
			})
			st.toolIndexes[toolCallID] = idx
			out = append(out, WireEvent{
				"type":       "tool_execution_start",
				"toolCallId": toolCallID,
				"toolName":   toolName,
			})
			out = append(out, WireEvent{
				"type": "message_update",
				"assistantMessageEvent": map[string]interface{}{
					"type":         "toolcall_start",
					"contentIndex": idx,
					"id":           toolCallID,
					"toolName":     toolName,
				},
			})
		}
	case "completed", "failed":
		if !seen {
			idx = len(st.blocks)
			st.blocks = append(st.blocks, map[string]interface{}{
				"type": "toolCall", "toolCallId": toolCallID, "toolName": toolName,
			})
			st.toolIndexes[toolCallID] = idx
		}
		rawInput, _ := data["rawInput"].(map[string]interface{})
		if rawInput == nil {
			rawInput = map[string]interface{}{}
		}
		out = append(out, WireEvent{
			"type":       "tool_execution_end",
			"toolCallId": toolCallID,
			"toolName":   toolName,
			"isError":    status == "failed",
		})
		out = append(out, WireEvent{
			"type": "message_update",
			"assistantMessageEvent": map[string]interface{}{
				"type":         "toolcall_end",
				"contentIndex": idx,
				"toolCall": map[string]interface{}{
					"id": toolCallID, "name": toolName, "arguments": rawInput,
				},
			},
		})
	}
	return out
}

// Marshal renders one wire event as an SSE data frame without an id line.
func Marshal(ev WireEvent) ([]byte, error) {
	return MarshalWithID(ev, 0)
}

// MarshalWithID renders one wire event as an SSE data frame with an optional
// id line so EventSource can resume with Last-Event-ID.
func MarshalWithID(ev WireEvent, id int64) ([]byte, error) {
	b, err := json.Marshal(ev)
	if err != nil {
		return nil, err
	}
	var prefix []byte
	if id > 0 {
		prefix = []byte(fmt.Sprintf("id: %d\ndata: ", id))
	} else {
		prefix = []byte("data: ")
	}
	return append(append(prefix, b...), '\n', '\n'), nil
}
