# 03 SSE 事件映射与重连

**Type:** task
**Status:** resolved
**Date:** 2026-08-13
**Baseline:** pigo `serve` SSE（`E:\project\pigo\internal\httpapi\events.go`、`cmd/pigo/prompt_runner.go`、`internal/httpapi/prompt.go`）；pi-web 前端事件线（`E:\project\pi-web\lib\agent-event-stream.ts`、`agent-event-wire.ts`、`streaming-message.ts`、`hooks/useAgentSession.ts`）。
**Scope:** 确定 pigo 领域事件到 pi-web 事件线的映射，设计 `after` 游标透传/重放、断线重连、心跳与事件转换层位置，并落地 `internal/events`。

## 结论

- 转换层放在 pi-web-go 的 `internal/events`，不在 pigo 也不在浏览器：pigo 保持领域事件，浏览器保持 pi-web wire。
- 四类核心事件均完成映射：`session.status`、`queue.updated`、`message.part.delta`、`tool.updated`。
- 浏览器流先发 `connected`，再发转换后的 wire 事件；每帧带 pigo 数字 `id`，支持 `?after=` 与 `Last-Event-ID` 回放。
- 心跳沿用 pi-web 的 30s 注释帧；pigo 上游 15s 心跳由 `internal/pigo.Client.StreamEvents` 忽略。

## 事件线对照

| pigo 事件 | pigo data 关键字段 | pi-web wire 输出 | 说明 |
|---|---|---|---|
| `session.status` | `status=running` | `agent_start` | 一轮 agent 开始 |
| `session.status` | `status=idle\|cancelled` | `agent_end` + `prompt_done` | `agent_end` 触发前端 loadSession，`prompt_done` 收尾 |
| `session.status` | `status=error`, `error` | `prompt_error` | 随后 pigo 仍会发 idle，再走 `agent_end` + `prompt_done` |
| `session.status` | `status=compacting` | `compaction_start` | |
| `session.status` | `status=compacted\|compaction_failed` | `compaction_end` | failed 时带 `errorMessage` |
| `queue.updated` | `queuedCount` | `queue_update` | pigo 只暴露数量；`steering`/`followUp` 降级为空数组 |
| `message.part.delta` | `partId=text`, `delta` | `message_start`（首个）+ `message_update` | `assistantMessageEvent` 依次为 `text_start`、`text_delta` |
| `message.part.delta` | `partId=thinking`, `thinking` | `message_start`（首个）+ `message_update` | `assistantMessageEvent` 依次为 `thinking_start`、`thinking_delta` |
| `tool.updated` | `status=pending\|in_progress` | `tool_execution_start` + `message_update` | `toolcall_start` 带 `contentIndex`/`id`/`toolName` |
| `tool.updated` | `status=completed\|failed` | `tool_execution_end` + `message_update` | `toolcall_end` 带 `toolCall{id,name,arguments}`；`isError` 由 failed 推导 |
| `permission.asked` | `permissionId` 等 | `permission_asked` | 保留透传；浏览器交互流程不在本 issue 范围 |

## Wire 差异

- pigo SSE：`event: <type>` + `data: {id,type,data,time}`，数字 id 单调递增，保留 10k 条/24h，游标过期返回 410 `EVENT_CURSOR_GONE`。
- pi-web SSE：无名 `data:` + 客户端事件对象，先有 `connected` 握手，30s 心跳。
- `internal/events.MarshalWithID` 输出 `id: <pigo id>\ndata: {...}\n\n`，浏览器 EventSource 会自动用最后收到的 id 作为 `Last-Event-ID`。
- 转换器按 session/message 维护流状态：首个 delta 发出带当前 blocks 快照的 `message_start`，后续 delta 只发 `message_update`；idle/cancelled 时清空状态。

## after 游标与重放

- 解析优先级：`?after=` > `Last-Event-ID` > 服务端 `CursorStore`。
- `?after=` 是显式重放意图，用于 Go 状态聚合器（issue 04）或未来前端；`Last-Event-ID` 是浏览器原生重连信号；`CursorStore` 是 M1 单浏览器兜底。
- `CursorStore` 只在整批 wire 帧成功 flush 后更新，避免写入中途断开导致跳过事件。
- 游标过期时 pigo 返回 410，pi-web-go 转成 `startup_error` 并关闭流。pi-web 的 `AgentEventConnection` 对 `startup_error` 停止自动重试，用户刷新页面或显式 `after=0` 可恢复。
- `CursorStore` 是进程内状态，进程重启即丢失；跨重启持久游标归 issue 04 的聚合器。

## 断线重连

- pi-web 的 `AgentEventConnection` 在错误后会 close 旧 EventSource 并创建新 EventSource，因此不一定会带 `Last-Event-ID`；M1 用 `CursorStore` 兜底，并依赖 pi-web 现有 `/api/agent/[id]` reconcile 恢复最终状态。
- 未来 pi-web-go 前端可在 `createSource` 时携带 `?after=`，获得精确重放。
- 路由必须先解析 pi-web session id 到 pigo `sessionId` + `directory`；`directory` 是 pigo 事件过滤的必填参数。
- `connected` 当前固定 `isStreaming:false`：pigo `GET /session/{id}/status` 硬编码 idle（issue 02 G2），无法在连接瞬间拿到真实 streaming 快照；运行态快照由 issue 04 聚合器补齐。

## 心跳

- 浏览器流：30s 发 `:\n\n`，与 pi-web `agent-event-stream.ts` 一致。
- pigo 上游：15s 注释心跳，`Client.StreamEvents` 直接忽略。
- `StreamHandler` 的所有写操作集中在主循环串行执行，事件 goroutine 只投递 frame batch，避免并发写 `http.ResponseWriter`。

## 转换层位置

- `internal/events/stream.go`：SSE HTTP handler、游标解析、心跳、串行写入、错误帧。
- `internal/events/wire.go`：`Converter` 纯事件转换 + `CursorStore` + `Marshal/MarshalWithID`。
- `internal/pigo/client.go`：pigo `/api/v1/events` SSE 客户端，已支持 `sessionId`/`directory`/`after`/`types` 透传。
- 归属：`GET /api/agent/{id}/events` 为 M1 pigo 代理 + 事件转换（research 01 route 2），本 issue 落地转换层；路由接线与 session id -> cwd 映射由后续 server 组装。

## 边界与降级

- 不发 `message_end`：pigo idle 事件不携带最终 assistant 内容，pi-web 通过 `agent_end` 后 loadSession 提交最终消息。
- 不发流式快照 `message_start`（连接时已存在的 streaming message）：依赖 issue 04 聚合器或 pigo status 修复。
- `queue_update` 只有 `queuedCount`，没有 pi-web 期望的 steering/followUp 文案。
- `tool_execution_update` 不转发：pi-web `agent-event-wire.ts` 本身会过滤该事件类型。
- 服务端 `CursorStore` 是单进程、单浏览器假设；多浏览器或跨进程应显式传 `after`。

## 验证

- `go test ./internal/events/...`
- `go build ./...`
- `stream_test.go` 覆盖：`connected` 握手、`id:` 帧、`Last-Event-ID` 透传、`?after=` 优先级、flush 后游标更新。
- `wire_test.go` 覆盖：text/thinking/tool/status 映射、`prompt_error`、`MarshalWithID`、游标单调性。
