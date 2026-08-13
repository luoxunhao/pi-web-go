# 11 工程能力 Go 实现

**Type:** task
**Status:** resolved
**Date:** 2026-08-13

## 结论

M2 工程能力以 Go-native + git CLI 实现，pigo 只负责 agent 会话数据。

## 实现

- `internal/server/sessions.go`：
  - `GET /api/sessions`、`GET/PATCH/DELETE /api/sessions/{id}`
  - `GET /api/sessions/{id}/context`、`state`、`auto-name`
- `internal/server/engineering.go`：
  - `GET /api/home`
  - `GET /api/cwd/browse`、`POST /api/cwd/validate`、`POST /api/default-cwd`
  - `GET /api/git/status`、`GET /api/git/diff`
  - `GET/POST/DELETE /api/worktrees`
  - `GET /api/file-index`
  - `GET /api/agent/{id}/bash-output`
  - `GET /api/app-update`

## 验证

- `go test ./internal/... ./cmd/...`、`go build ./cmd/... ./internal/...` 通过。

## 边界

- `auto-name` 使用首条用户消息截断标题，不调用 LLM。
- `app-update` 直接查 npm registry，与 pi-web 行为一致但依赖外网。
- git/worktree 输出按 porcelain 解析，Windows 路径经 `filepath` 处理。
