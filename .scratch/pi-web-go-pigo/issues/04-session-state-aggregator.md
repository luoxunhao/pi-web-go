# 04 会话状态聚合器

**Type:** task

## Question

按 T1 设计 Go server 的会话状态聚合：消费 pigo SSE 的 `session.status`/`queue.updated`，周期性 reconcile，提供 `/api/agent/running`，处理过期/清理和进程重启后的状态恢复。

**Blocked by:** 03 SSE 事件映射与重连

## Status

resolved

## Answer

已实现 `internal/session/manager.go` 状态聚合：事件驱动更新 running/queued、10 分钟超时清理、`Reconcile` 从 pigo session list 恢复已知会话；`/api/agent/running` 与 `/api/agent/running/events` 已挂载并测试通过。详见 `research/04-session-state-aggregator.md`。
