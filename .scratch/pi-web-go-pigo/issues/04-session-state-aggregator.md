# 04 会话状态聚合器

**Type:** task

## Question

按 T1 设计 Go server 的会话状态聚合：消费 pigo SSE 的 `session.status`/`queue.updated`，周期性 reconcile，提供 `/api/agent/running`，处理过期/清理和进程重启后的状态恢复。

**Blocked by:** 03 SSE 事件映射与重连

## Status

open
