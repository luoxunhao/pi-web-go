package export

import (
	"strings"
	"testing"

	"github.com/luoxunhao/pi-web-go/internal/pigo"
)

func TestSessionHTML(t *testing.T) {
	out := SessionHTML("s1", []pigo.Message{
		{Role: "user", Content: []map[string]interface{}{{"type": "text", "text": "hello"}}},
		{Role: "assistant", Content: []map[string]interface{}{{"type": "text", "text": "world"}}},
	})
	if !strings.Contains(out, "hello") || !strings.Contains(out, "world") {
		t.Fatalf("missing content: %s", out)
	}
	if !strings.Contains(out, "Session s1") {
		t.Fatalf("missing title: %s", out)
	}
}
