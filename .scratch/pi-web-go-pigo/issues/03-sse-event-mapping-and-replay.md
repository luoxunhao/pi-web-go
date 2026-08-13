# 03 SSE 事件映射与重连

**Type:** task

## Question

确定 pigo 领域事件（`session.status`、`queue.updated`、`message.part.delta`、`tool.updated`）到 pi-web 现有事件线的映射；设计 `after` 游标透传/重放、断线重连、心跳和事件转换层位置。

**Blocked by:** 01 pi-web API 契约矩阵, 02 pigo 缺口与降级矩阵

## Status

resolved

## Answer

已确定 `session.status`、`queue.updated`、`message.part.delta`、`tool.updated` 到 pi-web wire 的映射并落地 `internal/events` 转换层：`connected` 握手、`?after`/`Last-Event-ID` 游标重放、30s 心跳与串行 SSE 写入均已实现。快照与跨重启游标依赖 issue 04 聚合器。详见 `research/03-sse-event-mapping-and-replay.md`。
