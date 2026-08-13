# Deployment

## Production

1. 构建前端：`cd frontend && npm install && npm run build`。
2. 构建后端：`go build -o pi-web-go.exe ./cmd/server`。
3. 确保 `pigo` 在 PATH 中，或设置 `pigo.command` 指向 pigo 可执行文件。
4. 设置 `PI_WEB_GO_PASSWORD`（可选，推荐）。
5. 运行 `pi-web-go.exe`。

默认监听 `127.0.0.1:30141`；如需局域网访问，设置 `server.hostname = "0.0.0.0"` 并配置 `web.password`。

## Notes

- 当前前端静态文件从 `frontend/dist` 读取；生产 `go:embed` 后续启用。
- pigo 模型/provider 配置全部通过 pigo API 管理，不读写 `~/.config/pigo/config.toml`。
