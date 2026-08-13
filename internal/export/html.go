// Package export renders self-contained session HTML from pigo messages.
package export

import (
	"html"
	"strings"

	"github.com/luoxunhao/pi-web-go/internal/pigo"
)

func SessionHTML(sessionID string, messages []pigo.Message) string {
	var body strings.Builder
	for _, m := range messages {
		body.WriteString(`<section class="message ` + cssClass(m.Role) + `">`)
		body.WriteString(`<div class="role">` + html.EscapeString(m.Role) + `</div>`)
		for _, block := range m.Content {
			switch block["type"] {
			case "text":
				text, _ := block["text"].(string)
				body.WriteString(`<div class="text">` + html.EscapeString(text) + `</div>`)
			case "thinking":
				text, _ := block["thinking"].(string)
				body.WriteString(`<details class="thinking"><summary>Thinking</summary><div>` + html.EscapeString(text) + `</div></details>`)
			case "toolCall":
				name, _ := block["name"].(string)
				args, _ := block["arguments"].(string)
				body.WriteString(`<div class="tool">` + html.EscapeString(name) + ` <pre>` + html.EscapeString(args) + `</pre></div>`)
			case "image":
				data, _ := block["data"].(string)
				mime, _ := block["mimeType"].(string)
				if mime == "" {
					mime = "image/png"
				}
				body.WriteString(`<img src="data:` + html.EscapeString(mime) + `;base64,` + html.EscapeString(data) + `" alt="">`)
			}
		}
		body.WriteString(`</section>`)
	}
	return `<!doctype html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
  :root { color-scheme: light; }
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; margin: 0; background: #f5f6f8; color: #171717; }
  main { max-width: 860px; margin: 0 auto; padding: 32px 20px; }
  h1 { font-size: 18px; }
  .message { margin: 14px 0; padding: 12px 14px; border: 1px solid #e3e5e8; border-radius: 8px; background: #fff; }
  .message.user { background: #eef4ff; }
  .role { font-size: 12px; color: #6b7280; margin-bottom: 6px; text-transform: uppercase; }
  .text { white-space: pre-wrap; line-height: 1.6; }
  .thinking { color: #6b7280; }
  .tool { margin-top: 8px; font-family: ui-monospace, monospace; font-size: 13px; }
  pre { white-space: pre-wrap; }
  img { max-width: 100%; }
</style>
</head>
<body>
<main>
<h1>Session ` + html.EscapeString(sessionID) + `</h1>
` + body.String() + `
</main>
</body>
</html>`
}

func cssClass(role string) string {
	if role == "user" {
		return "user"
	}
	return "assistant"
}
