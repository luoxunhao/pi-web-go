# 07 slash 命令对齐

**Type:** task
**Status:** resolved
**Date:** 2026-08-13

## 结论

按 C1 只暴露 pigo 实际支持的命令；pi-web 独有且 pigo 无法映射的命令显式返回 `unsupported`，并记录在本文件。

## 实现

- `internal/server/agent.go` 新增 `POST /api/agent/{id}` 命令映射：
  - `get_commands` → `GET /api/v1/commands`，转成 pi-web `SlashCommandInfo`（`source: "prompt"`）。
  - `prompt/steer/follow_up` → `prompt_async`；图片附件显式拒绝（issue 02 G1）。
  - `abort` → `/cancel`。
  - `set_model` / `set_thinking_level` → `PATCH /session/{id}`。
  - `set_session_name` → `/command name`。
  - `compact` → `/command compact`。
  - `fork` → 按 entryId 定位用户消息序号，再 `/command fork <n>`。
  - `clone` → `/command clone`。
  - `navigate_tree` → `POST /session/{id}/load {leafId}`。
  - `get_state/get_session_stats/get_last_assistant_text/get_tools` 由 pigo status/messages 合成。
- session → cwd 映射：`internal/session.Manager` 新增 `SetDirectory/Directory`，`/api/agent/new` 创建后登记。

## pi-web 独有、pigo 暂不支持的命令

`reload`、`set_tools`、`set_auto_compaction`、`clear_queue`、`bash`、`abort_bash`、`abort_compaction`、`extension_ui_*` 返回 `{code:"unsupported"}`，不在前端命令目录中暴露。

## 验证

- `go test ./...`、`go build ./...` 通过。
