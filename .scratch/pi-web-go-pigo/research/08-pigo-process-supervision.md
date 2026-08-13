# 08 pigo 进程托管

**Type:** task
**Status:** resolved
**Date:** 2026-08-13

## 结论

按 P2/D1/S1 实现 pigo 子进程托管：默认 PATH 发现 `pigo`，`config.toml` 可覆盖 `command/args`；内部凭据默认随机生成，可配置覆盖。

## 实现

- `internal/pigo/supervisor.go`：
  - `Start`：先健康检查；已运行则复用；`auto_start=false` 时仅要求外部 pigo 健康。
  - spawn：`pigo serve --hostname <host> --port <port> --password <random>`，stdout/stderr 透传。
  - 健康轮询 15s 超时，失败则 kill 并返回错误。
  - 崩溃后 `monitor` 自动重启（2s backoff），`Stop` 退出时 kill。
  - 端口占用：启动前 `net.Listen` 预检；若端口被占且健康检查失败，返回明确错误。
- `cmd/server/main.go`：加载配置 → 启动 supervisor → 用 supervisor 的 BaseURL/Password 构造 pigo client → 启动 HTTP server → 优雅关闭。

## 验证

- `internal/pigo/supervisor_test.go`：复用 healthy 外部 pigo、随机密码。
- `go test ./...`、`go build ./...` 通过。

## 边界

- 外部已有 pigo 且配置了密码时，需在 `pigo.password` 提供一致密码，否则健康检查 401。
- 配置热重载/保存模型配置后重启 pigo（issue 02 G8）未在本 ticket 实现，M2 收口。
