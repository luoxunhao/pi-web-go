# 09 认证与请求安全

**Type:** task
**Status:** resolved
**Date:** 2026-08-13

## 结论

按 H3/U1 实现统一认证：浏览器只认证 Go server，Go server 内部携带 pigo 随机凭据转发。

## 实现

- `internal/server/middleware.go`：
  - Host 白名单：默认只允许 `127.0.0.1/localhost/::1`；`web.allowed_hosts` 或 `PI_WEB_GO_ALLOWED_HOSTS` 可扩展。
  - Basic Auth：用户名固定 `pi`，密码来自 `PI_WEB_GO_PASSWORD`/`web.password`；`subtle.ConstantTimeCompare` + sha256 防时序泄露。
  - CORS：loopback Origin 放行，`OPTIONS` 返回 204；开发期 Vite 可跨端口访问。
- `internal/pigo/client.go` 对 pigo 使用 `SetBasicAuth("pigo", password)`；密码由 supervisor 随机生成（S1）。
- 所有路由（含 SSE）都经过统一中间件；静态资源后续同样受保护。

## 验证

- `router_test.go`：无密码 401、正确 Basic 200、SSE 401、CORS preflight 204。
- `go test ./...`、`go build ./...` 通过。
