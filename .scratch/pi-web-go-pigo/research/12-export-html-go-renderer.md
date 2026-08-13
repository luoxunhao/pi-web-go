# 12 export HTML 纯 Go 渲染

**Type:** prototype
**Status:** resolved
**Date:** 2026-08-13

## 结论

用纯 Go 从 pigo messages 渲染 self-contained 会话 HTML，不依赖 pi SDK/Node。

## 实现

- `internal/export/html.go`：`SessionHTML(sessionID, messages)` 渲染 user/assistant/toolCall/thinking/image 内容块，内联 CSS，无外链资源。
- `internal/server/sessions.go`：`GET /api/sessions/{id}/export`，`inline=1` 内联展示，否则 `Content-Disposition: attachment`。

## 验证

- `internal/export/html_test.go` 覆盖文本与标题。
- `go test ./internal/... ./cmd/...`、`go build ./cmd/... ./internal/...` 通过。

## 边界

- 只导出最近 200 条消息；完整分页导出留待后续。
- 样式为简化 pi-web 风格，不包含原版 SDK 的交互脚本。
