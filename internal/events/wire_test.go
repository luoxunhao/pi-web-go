package events

import (
	"testing"

	"github.com/luoxunhao/pi-web-go/internal/pigo"
)

func TestConvertTextDeltaStartsMessage(t *testing.T) {
	c := NewConverter()
	out := c.Convert(pigo.DomainEvent{
		ID:   1,
		Type: "message.part.delta",
		Data: map[string]interface{}{
			"sessionId": "s1", "messageId": "m1", "partId": "text", "delta": "hello",
		},
	})
	if len(out) != 3 {
		t.Fatalf("events = %d, want 3", len(out))
	}
	if out[0]["type"] != "message_start" {
		t.Fatalf("first type = %v", out[0]["type"])
	}
	msg := out[0]["message"].(map[string]interface{})
	if msg["role"] != "assistant" {
		t.Fatalf("role = %v", msg["role"])
	}
	update := out[2]["assistantMessageEvent"].(map[string]interface{})
	if update["type"] != "text_delta" || update["delta"] != "hello" {
		t.Fatalf("update = %#v", update)
	}
}

func TestConvertThinkingAndTextBlocks(t *testing.T) {
	c := NewConverter()
	base := pigo.DomainEvent{Type: "message.part.delta", Data: map[string]interface{}{
		"sessionId": "s1", "messageId": "m1",
	}}
	base.Data["partId"] = "thinking"
	base.Data["thinking"] = "think"
	thinking := c.Convert(base)
	if len(thinking) != 3 {
		t.Fatalf("thinking events = %d", len(thinking))
	}

	base.Data["partId"] = "text"
	base.Data["delta"] = "answer"
	text := c.Convert(base)
	if len(text) != 2 {
		t.Fatalf("text events = %d, want 2", len(text))
	}
	start := text[0]["assistantMessageEvent"].(map[string]interface{})
	if start["type"] != "text_start" || start["contentIndex"] != 1 {
		t.Fatalf("text_start = %#v", start)
	}
}

func TestConvertToolLifecycle(t *testing.T) {
	c := NewConverter()
	start := c.Convert(pigo.DomainEvent{
		Type: "tool.updated",
		Data: map[string]interface{}{
			"sessionId": "s1", "messageId": "m1", "toolCallId": "t1",
			"title": "bash", "status": "pending",
		},
	})
	if len(start) != 2 || start[0]["type"] != "tool_execution_start" {
		t.Fatalf("start = %#v", start)
	}
	done := c.Convert(pigo.DomainEvent{
		Type: "tool.updated",
		Data: map[string]interface{}{
			"sessionId": "s1", "messageId": "m1", "toolCallId": "t1",
			"title": "bash", "status": "completed", "rawInput": map[string]interface{}{"cmd": "ls"},
		},
	})
	if len(done) != 2 || done[0]["type"] != "tool_execution_end" {
		t.Fatalf("done = %#v", done)
	}
}

func TestConvertStatusLifecycle(t *testing.T) {
	c := NewConverter()
	running := c.Convert(pigo.DomainEvent{Type: "session.status", Data: map[string]interface{}{
		"sessionId": "s1", "messageId": "m1", "status": "running",
	}})
	if len(running) != 1 || running[0]["type"] != "agent_start" {
		t.Fatalf("running = %#v", running)
	}
	idle := c.Convert(pigo.DomainEvent{Type: "session.status", Data: map[string]interface{}{
		"sessionId": "s1", "messageId": "m1", "status": "idle",
	}})
	if len(idle) != 2 || idle[0]["type"] != "agent_end" || idle[1]["type"] != "prompt_done" {
		t.Fatalf("idle = %#v", idle)
	}
}

func TestConvertStatusError(t *testing.T) {
	c := NewConverter()
	out := c.Convert(pigo.DomainEvent{Type: "session.status", Data: map[string]interface{}{
		"sessionId": "s1", "status": "error", "error": "boom",
	}})
	if len(out) != 1 || out[0]["type"] != "prompt_error" {
		t.Fatalf("error = %#v", out)
	}
	if out[0]["errorMessage"] != "boom" {
		t.Fatalf("errorMessage = %v", out[0]["errorMessage"])
	}
}

func TestMarshalWithID(t *testing.T) {
	frame, err := MarshalWithID(WireEvent{"type": "agent_start"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	got := string(frame)
	if got != "id: 42\ndata: {\"type\":\"agent_start\"}\n\n" {
		t.Fatalf("frame = %q", got)
	}
}

func TestCursorStoreMonotonic(t *testing.T) {
	store := NewCursorStore()
	store.Set("s1", 5)
	store.Set("s1", 3)
	if got := store.Get("s1"); got != 5 {
		t.Fatalf("cursor = %d, want 5", got)
	}
}
