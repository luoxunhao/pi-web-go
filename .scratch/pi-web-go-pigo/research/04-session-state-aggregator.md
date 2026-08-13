# 04 会话状态聚合器

**Type:** task
**Status:** resolved
**Date:** 2026-08-13

## 结论

Go server 自己聚合 pigo 会话运行状态，不依赖 pigo `/status`（issue 02 G2 硬编码 idle）。

## 实现

- `internal/session/manager.go`：`Manager` 以 `sessionId` 为键维护 `Running/Queued/MessageID/LastSeen`。
- 事件驱动：`session.status`、`queue.updated`、`message.part.delta`、`tool.updated`、`permission.asked` 更新状态；idle/cancelled/error 清 running。
- `GET /api/agent/running` 返回 `{runningSessionIds: [...]}`，与 pi-web 快照契约一致。
- `GET /api/agent/running/events` SSE：先发初始快照，再订阅 `Manager.Subscribe()` 推送变化，30s 心跳。
- 超时清理：`Cleanup(now)` 将超过 10 分钟未见事件的 running session 标为 inactive。
- 重连/重启恢复：`Reconcile(ctx, client, directories)` 从 pigo session list 重新水合已知会话；由于 `/status` 不可信，不猜测 running。

## 验证

- `go test ./internal/session/...` 覆盖 running/idle、queue/delta、cleanup、subscribe。
- `go test ./internal/server/...` 覆盖 `/api/agent/running` 快照与 SSE 转换路由。
- `go build ./...` 通过。

## 边界

- `CursorStore` 仍是单进程；跨重启持久游标需要后续把 `Reconcile` 与 `after` 游标一起落盘（M2+）。
- `/api/agent/running/events` 目前不重放历史变化；浏览器断线后以快照为准。
