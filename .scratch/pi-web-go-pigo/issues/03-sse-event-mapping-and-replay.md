# 03 SSE 事件映射与重连

**Type:** task

## Question

确定 pigo 领域事件（`session.status`、`queue.updated`、`message.part.delta`、`tool.updated`）到 pi-web 现有事件线的映射；设计 `after` 游标透传/重放、断线重连、心跳和事件转换层位置。

**Blocked by:** 01 pi-web API 契约矩阵, 02 pigo 缺口与降级矩阵

## Status

open
