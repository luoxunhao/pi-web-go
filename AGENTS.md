# AGENTS.md — pi-web-go 协作指南

给在 pi-web-go 仓库工作的 AI 代理 / 开发者的指引。开始任何工作前先读本文件与 wayfinder map。

## 1. 首先阅读

- [`.scratch/pi-web-go-pigo/map.md`](.scratch/pi-web-go-pigo/map.md) — wayfinder map：目标、决策（Decisions so far）、未决事项（Not yet specified）、范围外（Out of scope）的**权威来源**
- [`README.md`](README.md) / [`DEPLOY.md`](DEPLOY.md) — 构建、配置、部署
- 实施某个 issue 前，先读对应 `.scratch/pi-web-go-pigo/issues/NN-*.md` 与 `.scratch/pi-web-go-pigo/research/NN-*.md`

## 2. 项目是什么

pi-web 的纯 Go 后端重构（module `github.com/luoxunhao/pi-web-go`）：

- Go server（chi）提供 pi-web 兼容的 HTTP API，托管并代理 `pigo serve` 子进程的 `/api/v1/*`
- 前端是 upstream pi-web 迁移来的 Vite React SPA，由 Go 静态托管
- 运行期零 Node；构建期允许 Node（仅前端构建）

## 3. 硬性约束（违反即破坏项目）

| 编号 | 约束 |
|---|---|
| T3 | 纯 Go 全量重实现，不保留 Node sidecar；运行期零 Node |
| R1 | API 路径与响应形状保持 pi-web 兼容（40 routes / 53 handlers，契约矩阵见 `.scratch/research/01`） |
| C1 | agent 配置（模型 / provider / mode / slash / skills 等）全部从 pigo API 获取，**禁止**在 pi-web-go 内重写或写入 config.toml |
| F1 | 文件白名单：会话 cwd / 项目根 / `~/pi-cwd-*` / 显式 root + realpath；不自动放行 `$HOME` |
| U1/H3 | 浏览器 Basic Auth（用户 `pi`，密码 `PI_WEB_GO_PASSWORD`）+ Host 白名单 |
| N1 | 前端构建产物 `frontend/dist` 由 Go 静态托管，SPA fallback 到 index.html |
| M1 | 事件格式：Go server 转换 pigo 事件为 pi-web wire 格式（`internal/events`），不要改变 wire 契约 |

## 4. 代码地图

| 路径 | 职责 | 注意 |
|---|---|---|
| `cmd/server/main.go` | 装配：config → Supervisor → client → 各 handler | 依赖注入集中在 `server.Dependencies` |
| `internal/config` | TOML + 环境变量 + 校验 | 只含 server / pigo / web / filesystem 四段 |
| `internal/pigo` | client.go HTTP 客户端；supervisor.go 子进程托管；types.go 领域类型 | 随机密码、健康检查、崩溃重启 |
| `internal/events` | wire.go 转换器（含流式状态）；stream.go SSE handler + 游标 + 心跳 | 断线重放靠 CursorStore（Last-Event-ID） |
| `internal/session` | Manager 聚合 running/queued，ObserveEvent 事件驱动 + 订阅发布 | pigo `/status` 硬编码 idle，勿依赖它 |
| `internal/files` | access.go 白名单（realpath）；handler.go `/api/files`；docx.go 预览 | **安全边界所在**，改动需谨慎 |
| `internal/server` | router.go 路由 + agent / sessions / models / engineering / middleware | 每组 handler 一个 `*Handler` struct |
| `internal/export` | 会话 → self-contained HTML | 纯 Go 渲染 |
| `frontend/` | Vite React SPA | 见 `frontend/AGENTS.md` |

## 5. 常见陷阱

- **不要在 config.toml 加 agent 配置**（C1）——模型 / provider 由 pigo 管理，走 `/api/models` 等代理
- **不要改变 /api/\* 的响应形状**（R1）——前端契约依赖
- **pigo 事件是领域格式**，浏览器只认 wire 格式；新事件类型要过 `internal/events` 转换层，别绕过
- **文件访问必须过 internal/files 白名单**（realpath + case-folding），禁止自行 `os.ReadFile` 任意路径
- **Windows 路径**：git 输出 POSIX 路径，比较一律规范化 / realpath，勿用字符串相等（`samePath()` 模式）
- **前端产物**：修改前端后需 `npm run build` 重新生成 `frontend/dist`，Go 侧才生效
- **SSE 心跳**：30s 一次 `:` 注释帧；浏览器断线重连靠游标重放，改 events 时保持兼容
- **会话状态是事件驱动**：pigo `/status` 硬编码 "idle"（issue 02 G2），聚合逻辑以 SSE 事件 + 会话列表对账为准
- **前端 `app/page.tsx` 与 next-config 相关文件是迁移遗留**，构建走 Vite（`src/main.tsx`），不要在此基础上加 Next.js 代码
- **pigo 数据隔离**：本机可能存在其他 pigo/pi 实例（如 DSH）独占 `~/.pigo/sessions.db`，导致本实例只能只读打开、写入报 `readonly database`；部署时应设置 `[pigo] data_dir`（supervisor 自动注入 `PIGO_HOME`）隔离数据

## 6. 构建与测试

```bash
make build           # go build -o pi-web-go.exe ./cmd/server
make build-frontend  # cd frontend && npm run build
make dev             # 前端热更新（:5173，/api 代理 :30141）
make test            # go test ./internal/... ./cmd/... + npm run typecheck

# 前端单测（Node 内置 test runner，无需额外框架）：
cd frontend && node --test components/*.test.mjs lib/*.test.mjs hooks/*.test.mjs
```

约定：后端改动保持每个包自带 `*_test.go`（现有 10 个）；前端纯逻辑进 `lib/` 并配 `.test.mjs`（现有 100 个）。

## 7. 提交约定

- 提交信息用中文，简洁说明改动
- 默认不 push（除非明确要求）
- 行为变化或决策新增时，同步更新 `.scratch/pi-web-go-pigo/map.md` 的 Decisions
