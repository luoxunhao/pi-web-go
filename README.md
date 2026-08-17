# pi-web-go

pi-web 的纯 Go 后端重构：Go server 托管并代理 `pigo serve` 的 HTTP API，前端由 upstream pi-web 迁移为 Vite React SPA。**运行期零 Node 依赖**，交付形态为「单个 Go 可执行文件」——前端产物由 `make build` 复制进 `internal/webui/dist` 并以 `go:embed` 内嵌（E1）；未嵌入时回退到磁盘 `frontend_dir`（开发模式）。

## 特性

- **纯 Go 后端**（chi v5），对齐 pi-web 的 40 个 API route / 53 个 handler 契约（R1）
- **pigo 进程托管**：从 PATH 自动发现并启动 `pigo serve`，随机内部密码、健康检查、崩溃自动重启（P2/D1/S1）
- **SSE 事件转换层**：pigo 领域事件 → pi-web wire 格式，Last-Event-ID 断线重放 + 30s 心跳
- **会话状态聚合**：Go server 服务端聚合 running/queued，`/api/agent/running` + SSE 推送
- **文件安全**：白名单 + realpath 校验（会话 cwd / 项目根 / `~/pi-cwd-*` / 显式 root），纯 Go docx 预览
- **工程能力**：git status/diff、worktree 管理、cwd 浏览/校验、文件索引、app-update、项目信任
- **会话导出**：self-contained HTML（纯 Go 渲染，无 Node）
- **安全**：浏览器 Basic Auth + Host 白名单 + CORS
- **前端**：Vite React SPA（React 19 + TS + Tailwind 4，中英双语 i18n），由 Go 静态托管 + SPA fallback

## 架构

```
┌─────────────┐   HTTP /api/* + SSE    ┌──────────────────────────┐
│   浏览器     │ ◄────────────────────► │   Go Server（chi :30141） │
│  Vite SPA   │                        │  ├─ Security 中间件        │
└─────────────┘                        │  ├─ agent / sessions      │
                                       │  ├─ events 转换层          │
                                       │  ├─ session 状态聚合       │
                                       │  ├─ files / export        │
                                       │  └─ 静态托管 frontend/dist │
                                       └───────────┬──────────────┘
                                                   │ 代理 /api/v1/*
                                       ┌───────────▼──────────────┐
                                       │  pigo serve 子进程 :4096   │
                                       │  Supervisor 托管/重启       │
                                       └──────────────────────────┘
```

关键设计决策见 [`.scratch/pi-web-go-pigo/map.md`](.scratch/pi-web-go-pigo/map.md)（wayfinder map，含 15 个 issue 的决策与落地记录）。

## 快速开始

前置：Go（go.mod 要求 `go 1.27rc1`）、Node ≥ 22.19（仅构建前端用）、`pigo` 可执行文件（在 PATH 或由 `pigo.command` 指定）。

```bash
# 1. 构建前端（产物输出到 frontend/dist）
cd frontend && npm install && npm run build && cd ..

# 2. 构建后端（自动把 frontend/dist 嵌入二进制）
make build        # 等价于：go build -o pi-web-go.exe ./cmd/server

# 3. 运行（pigo 默认自动启动）
./pi-web-go.exe
```

打开 http://127.0.0.1:30141 。

### 开发模式

```bash
# 终端 1：后端
go run ./cmd/server

# 终端 2：前端热更新（Vite :5173，/api 代理到 :30141）
cd frontend && npm run dev
```

## 配置

复制 `config.toml.example` 为 `config.toml`（已 gitignore），或用 `--config` 指定路径。加载顺序：内置默认值 → config.toml → 环境变量覆盖 → 校验。

| 段 | 字段 | 说明 | 默认值 | 环境变量覆盖 |
|---|---|---|---|---|
| [server] | port / hostname | HTTP 监听端口/地址 | 30141 / 127.0.0.1 | PI_WEB_GO_PORT / PI_WEB_GO_HOSTNAME |
| [pigo] | command / args | 子进程命令与参数 | "pigo" / ["serve"] | PIGO_COMMAND |
| [pigo] | host / port / base_url | pigo 监听地址 | 127.0.0.1 / 4096 | PIGO_HOST / PIGO_PORT / PIGO_BASE_URL |
| [pigo] | data_dir | pigo 数据目录（注入 PIGO_HOME，隔离会话库防止与其他 pigo/pi 实例冲突） | 空（不隔离） | — |
| [pigo] | password | pigo 内部 API 凭据；留空则每次进程随机生成 | 随机生成 | PIGO_PASSWORD |
| [pigo] | auto_start | 是否由 Supervisor 自动托管 pigo | true | — |
| [web] | password | 浏览器 Basic Auth 密码（用户名固定 `pi`） | 空 | PI_WEB_GO_PASSWORD |
| [web] | allowed_hosts | 额外允许的请求 Host（防 DNS rebinding） | [] | PI_WEB_GO_ALLOWED_HOSTS |
| [web] | frontend_dir | 静态前端产物目录 | frontend/dist | PI_WEB_GO_FRONTEND_DIR |
| [filesystem] | allowed_roots | /api/files 白名单根目录（realpath 校验） | [] | — |

> agent 相关配置（模型 / provider / mode / slash 命令等）**不在此处**：由 pigo 拥有，通过其 HTTP API 读写（C1）。

## 目录结构

```
cmd/server/         入口：配置加载、pigo Supervisor、HTTP server、优雅关闭
internal/
  config/           TOML 配置 + 环境变量 + 校验
  pigo/             pigo HTTP client + 子进程 Supervisor（自动启动/随机密码/重启）
  events/           SSE 转换层：pigo 事件 → pi-web wire；游标重放；心跳
  session/          running/queued 状态聚合（事件驱动 + 订阅发布）
  files/            文件白名单 + /api/files + docx 预览
  server/           pi-web 兼容 API 路由（agent / sessions / models / engineering / auth）
  export/           会话导出 self-contained HTML
frontend/           Vite React SPA（React 19 + TS + Tailwind 4，详见 frontend/AGENTS.md）
.scratch/           wayfinder map + 15 个 issue 的决策与落地记录
```

## 测试

```bash
make test                                    # Go 单测 + 前端 typecheck
go test ./internal/... ./cmd/...             # Go 测试（10 个测试文件）
cd frontend && node --test components/*.test.mjs lib/*.test.mjs hooks/*.test.mjs   # 前端 100 个单测
```

## 部署

见 [DEPLOY.md](DEPLOY.md)：构建前端 → 构建二进制 → 设置 `PI_WEB_GO_PASSWORD` → 运行。局域网访问需 `server.hostname = "0.0.0.0"` 并配置 `web.password`。

## 里程碑

- **M1 核心主链路** ✅ 会话 / 事件 / 认证 / 文件
- **M2 工程能力** ✅ git / worktrees / export / models 代理
- **M3 生态能力** ⏳ 图片预览、OAuth、package HTTP 等受 pigo 缺口阻塞（见 issue 02）

## License

MIT（见 [LICENSE](LICENSE)）
