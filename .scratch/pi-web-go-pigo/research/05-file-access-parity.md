# 05 文件访问安全与 files API

**Type:** task
**Status:** resolved
**Date:** 2026-08-13

## 结论

按 F1 实现 pi-web 风格的文件白名单与 `/api/files` Go-native 接口。

## 实现

- `internal/files/access.go`：`Access` 线程安全白名单，支持配置 roots、会话 cwd 动态加入；词法 + realpath 双重校验，Windows 大小写不敏感；不自动放行 `$HOME`。
- `internal/files/handler.go`：`GET /api/files/*` 支持 `list/read/download/meta/preview/watch`，`POST` 支持 `upload/upload-check`；文件类型按扩展名映射，文本预览 256KB 上限，上传单文件 25MB/总量 100MB。
- `internal/files/docx.go`：纯 Go docx preview，zip + XML 解析，段落/文本/换行/图片 data URI，self-contained HTML 与 pi-web 同款 CSP。
- 路由：`internal/server/router.go` 挂载 `/api/files/*`。

## 验证

- `go test ./internal/files/...`：白名单拒绝越界、list/read、docx preview（含图片）。
- `go test ./internal/server/...`、`go build ./...` 通过。

## 边界

- `watch` 使用 1s 轮询而非 OS watcher；满足前端文件视图刷新，但大目录下开销较高。
- docx preview 覆盖常见段落/文本/图片；复杂表格、嵌套样式、脚注等渲染为简化 HTML。
