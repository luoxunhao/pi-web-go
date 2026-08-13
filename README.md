# pi-web-go

pi-web 的纯 Go 后端重构：Go server 托管并代理 `pigo serve` HTTP API，前端由 upstream pi-web 迁移为 Vite React SPA，运行期不依赖 Node。

## Quick start

```bash
go build -o pi-web-go.exe ./cmd/server
cd frontend && npm install && npm run build
cd ..
./pi-web-go.exe
```

打开 `http://127.0.0.1:30141`。`pigo` 默认从 PATH 发现并自动启动（`pigo.auto_start=true`）。

## Config

复制 `config.toml.example` 为 `config.toml`，或通过 `--config` 指定。浏览器密码优先用环境变量 `PI_WEB_GO_PASSWORD`；pigo 内部凭据默认随机生成。

## Architecture

- `cmd/server`：入口，进程托管 + HTTP server。
- `internal/pigo`：pigo HTTP client 与子进程 supervisor。
- `internal/events`：pigo SSE → pi-web wire 转换、游标、心跳。
- `internal/session`：running/queued 状态聚合。
- `internal/files`：文件白名单与 `/api/files`。
- `internal/server`：pi-web 兼容 API 路由。
- `frontend`：Vite React SPA。
