# 10 Vite 前端迁移

**Type:** task
**Status:** resolved
**Date:** 2026-08-13

## 结论

基于 upstream `agegr/pi-web@0877bff` 把前端迁入 `frontend/`，Vite 构建通过，`AppShell` 作为入口直接渲染，Go server 静态托管 SPA。

## 实现

- 从 upstream 拉取源码，删除 `app/api`、`instrumentation.ts`、`next.config.ts`、`proxy.ts`。
- `frontend/package.json`：移除 `next`/`eslint-config-next`，改为 Vite scripts，保留 React/pi SDK 生态依赖。
- `frontend/vite.config.ts`：`@` alias 指向仓库根、`next/navigation` 指向本地 shim、dev proxy `/api → 127.0.0.1:30141`。
- `frontend/src/main.tsx`：直接渲染 `AppShell`。
- `frontend/src/next-navigation.ts`：提供 `useRouter/useSearchParams/usePathname` 最小 shim，替换 Next 路由依赖。
- `frontend/src/env.ts`：`APP_VERSION/PI_VERSION/IS_PROD` 替代 `process.env.NEXT_PUBLIC_*` 与 `NODE_ENV`。
- Go server：`Dependencies.StaticDir` + SPA fallback，`cmd/server/main.go` 默认托管 `frontend/dist`。

## 验证

- `npm run typecheck` 通过。
- `npm run build` 通过，产出 `frontend/dist`。
- `go test ./internal/... ./cmd/...`、`go build ./...` 通过。

## 边界

- `next/navigation` 是路由 shim，深链接/URL 状态完整复刻归 ticket 15 E2E 迭代。
- 生产 `go:embed` 尚未启用，当前按 E1 开发路径读磁盘 `frontend/dist`。
