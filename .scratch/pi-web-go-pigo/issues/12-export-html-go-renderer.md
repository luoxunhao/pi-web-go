# 12 export HTML 纯 Go 渲染

**Type:** prototype

## Question

用纯 Go 从 pigo 消息/JSONL 渲染 pi-web 风格的 self-contained export HTML；评估复用现有模板样式、递归树过深处理、`inline=1` 等行为；产出可给用户看的原型。

**Blocked by:** 01 pi-web API 契约矩阵, 10 Vite 前端迁移

## Status

resolved

## Answer

已实现 `internal/export` 纯 Go 会话 HTML 渲染与 `/api/sessions/{id}/export`（含 inline/attachment）。详见 `research/12-export-html-go-renderer.md`。
