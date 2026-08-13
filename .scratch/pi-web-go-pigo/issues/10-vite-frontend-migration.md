# 10 Vite 前端迁移

**Type:** task

## Question

基于 upstream `agegr/pi-web` 最新版把前端迁入 `frontend/`：移除 Next.js 特定代码、API routes 和 `process.env.NEXT_PUBLIC_*` 依赖，改为 Vite 构建；确定静态产物如何交给 Go server（E1），保持组件接口不变。

**Blocked by:** 01 pi-web API 契约矩阵, 03 SSE 事件映射与重连

## Status

resolved

## Answer

upstream pi-web 已迁入 `frontend/`，Next.js 依赖/API routes 已移除，Vite + React 构建通过，`AppShell` 作为入口，Go server SPA fallback 托管 `dist`。详见 `research/10-vite-frontend-migration.md`。
