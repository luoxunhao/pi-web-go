# 02 pigo 缺口与降级矩阵

**状态:** 完成
**日期:** 2026-08-13
**基准:** pigo HEAD `fc09a0c`（`E:\project\pigo`）；pi-web API surface（`E:\project\pi-web\app\api`）
**范围:** 只核对代码事实，未修改 pigo / pi-web / pi-web-go 源码。

## 缺口汇总

共 8 个缺口：5 个票点名缺口 + 3 个新增缺口。

| # | 缺口 | 结论 | pi-web-go 降级 | pigo 侧 ticket | 里程碑 |
|---|---|---|---|---|---|
| G1 | HTTP prompt 图片块被丢弃 | 不支持 | 隐藏图片/附件输入，标记 structured unsupported | serve/ACP prompt 透传 image blocks | M3 |
| G2 | `GET /session/{id}/status` 不是真实运行状态 | 运行字段硬编码 | Go 侧 SSE 聚合器提供 running/queued | status 返回真实状态 | M1 靠降级 |
| G3 | 无 files HTTP API | 无任何 `/files` 路由 | Go-native 文件 read/list/watch/upload（F1） | 可选：serve 增加 files API | M1/M2 靠降级 |
| G4 | 无 package/skills HTTP API | 仅 pigo CLI | proxy pigo CLI + Go-native 读目录/lockfile | serve 增加 package/skills HTTP API | M3 |
| G5 | OAuth 不可用，serve 不解析 env 凭据 | 仅 API key 可用 | 只开放 API key，隐藏 OAuth | OAuth 存储/流程 + serve 接 CredentialStore | M3 |
| G6 | `additionalDirectories`/`mcpServers` 被丢弃 | 接收但无效 | 单 cwd 模式，多根只在 Go 白名单实现 | session/new 应用并持久化这两个字段 | M2/M3 |
| G7 | prompt 请求的 `mode` 被忽略 | 契约字段无效 | 先调 `/session/{id}/mode`，隐藏 prompt 级 mode | prompt 支持 mode 或移除字段 | M1 靠降级 |
| G8 | 运行中 serve 不重载模型配置 | 新增模型需重启 | 模型配置保存后提示/重启 pigo | config 写入后热重载模型目录 | M2 |

## 详细证据

### G1 图片附件被 HTTP prompt 丢弃

- `internal/httpapi/gen/api.gen.go` 的 `PromptRequest.Prompt` 是 `[]map[string]interface{}`，本身允许任意块，但 `internal/httpapi/prompt.go:143-150` 只把 `promptText(...)` 的结果放进 `PromptRun.Text`。
- `internal/httpapi/prompt.go:334-345` 的 `promptText` 只提取 `block["text"]`，`type: "image"` 块会被静默丢弃。
- `cmd/pigo/prompt_runner.go:195` 调用 `RunWithTools(ctx, run.Text, nil, ...)`，图片参数固定为 `nil`。
- runtime 本身支持图片：`internal/acp/runner.go:76-80` 会把传入的 image content 追加到用户消息，provider 也有图片 wire 测试（`internal/provider/responses_test.go`）。问题只在 HTTP/ACP 接入层。
- ACP 的 `promptCapabilities.image` 声明为 `true`（`internal/acp/http_adapter.go:112`、`internal/acp/server.go:113`），但 ACP adapter 的 `promptText` 同样丢弃非文本块，声明与实现不一致。

### G2 session status 不是真实运行状态

- `internal/httpapi/sessions.go:315-346` 中 `Status` 返回固定 `Status: "idle"` 和 `QueuedCount: 0`，且不填充 `ContextUsage`；它只从投影读取 model/thinking/lane 等持久信息。
- 真实运行状态在 `PromptManager` 中（`internal/httpapi/prompt.go:135-193` 的 `running`、`queue`、`cancel`），但 `SessionService.Status` 不访问它。
- pi-web 的 `/api/agent/running`（`E:\project\pi-web\app\api\agent\running\route.ts`）需要真实 running 集合；按 map 的 T1/票 04，应由 Go server 消费 pigo SSE 的 `session.status` / `queue.updated` 自行聚合。

### G3 无 files HTTP API

- pigo 当前路由面（`internal/httpapi/gen/api.gen.go:1706-1793`）只有 health、session、events、commands、trust、config、modes，没有任何 `/api/v1/files`、`file-index` 或 watch 端点。
- pi-web 现有 `/api/files/[...path]`、`/api/file-index`、`/api/worktrees`、git status/diff 等能力（`E:\project\pi-web\app\api\...`）。
- pigo agent 内部有文件工具，但未通过 HTTP 暴露；pi-web-go 按票 05 的 F1 白名单用 Go 原生实现 read/list/upload/download/watch/meta/preview 即可绕过该缺口。

### G4 package/skills 管理只有 CLI

- `internal/cli/pkgcmd/pkgcmd.go:18-44` 定义 `install|list|uninstall|update`，`cmd/pigo/main.go` 在 pflag 解析前直接剥离这些子命令；HTTP 路由面中没有对应端点。
- `pigo list` 输出的是人类可读 TSV（`pkgcmd.go:58-60`），不是 JSON；CLI 也没有 `enable/disable/search/check`。
- skills 是 npm package 分发到 skills 目录的产物（`internal/pkgmgr/distribute_skill.go`），没有独立的 skill 管理 CLI 或 HTTP API。
- pi-web 的 `/api/plugins` 支持 install/remove/update/disable/enable 和资源统计（`E:\project\pi-web\app\api\plugins\route.ts`），`/api/skills*` 也有独立搜索/安装/更新面；这些无法全部映射到 pigo CLI。

### G5 OAuth 与凭据链路

- `internal/provider/auth.go` 有内存态 `TokenSource` / `CredentialStore`，支持 OAuth token 自动刷新，但没有持久化、没有浏览器/device flow、没有 HTTP 管理端点。
- HTTP 配置面只接受 `apiKey`（`internal/httpapi/config.go:89-121`），响应只回 `apiKeyConfigured`（`config.go:234-249`），无 OAuth 字段。
- `internal/acp/runner.go:96-105` 把静态 `apiKey` 放进 `LoopConfig.APIKey`，没有设置 `GetAPIKey`；`ResolveForModel` 对配置模型直接返回 `entry.APIKey`（`runner.go:165`）。因此 serve/ACP 不走 REPL 使用的 `CredentialStore` env/OAuth 解析路径。
- `ANTHROPIC_OAUTH_TOKEN` 只是 provider registry 的 env 候选（`internal/provider/registry.go:69`），对 pi-web-go 来说不是可用 OAuth 流程。

### G6 session/new 丢弃 additionalDirectories / mcpServers

- `internal/httpapi/gen/api.gen.go:194-198` 的 `NewSessionRequest` 定义了 `AdditionalDirectories` 和 `McpServers`。
- `internal/httpapi/sessions.go:103-107` 读取 `additional` 后立即 `_ = additional`；`McpServers` 在 `Create` 中从未被读取。ACP adapter 会传 `additionalDirectories`，但 HTTP 层最终丢弃。

### G7 prompt 请求的 mode 被忽略

- `PromptRequest` 有 `Mode` 字段（`internal/httpapi/gen/api.gen.go:215-222`）。
- `internal/httpapi/prompt.go:143-150` 构造 `PromptRun` 时只复制 `Model`、`ThinkingLevel`、`Text`，没有 `Mode`；`prompt_runner.go` 也没有按 mode 调整系统提示/工具集。

### G8 运行中 serve 不重载模型配置

- `cmd/pigo/prompt_runner.go:94-95` 在启动时 `models.Load()` 一次，并在 `runner.ConfiguredModels` 中复用（`prompt_runner.go:138`）。
- `internal/httpapi/config.go` 的 `save` 只写 `config.toml`；没有回调 `ConfiguredModels.Load()` 或发送 reload 通知。
- 因此通过 `PUT /config/providers` 或 `PATCH /config` 新增模型后，会话配置可以立刻看到，但下一次 prompt 仍可能报 `model is not configured`，直到重启 serve。

## 里程碑判定

- **M1 核心主链路**：在采用降级后不被 pigo 缺口阻塞。M1 必须做：图片输入隐藏（G1）、Go SSE 状态聚合器（G2）、Go-native 文件面（G3）、API key 凭据（G5）、prompt 前先 set_mode（G7）。
- **M2 工程能力**：模型配置热重载（G8）、additional dirs/MCP（G6）、prompt 级 mode（G7）需要降级或 pigo 改动；package 管理可先用 CLI proxy 做部分面（G4）。
- **M3 生态能力**：全量 parity 被图片 prompt（G1）、OAuth（G5）、package/skills HTTP（G4）阻塞，必须等 pigo 侧能力或另立 Go-native 实现。

## 建议的 pigo 侧 ticket 清单

- G1: `serve/ACP prompt 透传 image blocks，使 promptCapabilities.image 与实际行为一致`
- G2: `GET /api/v1/session/{id}/status 返回真实 running/queued/contextUsage`
- G3（可选）: `serve 增加 trust 边界一致的 files HTTP API`
- G4: `serve 增加 package/skills 管理 HTTP API（JSON、enable/disable、search）`
- G5a: `pigo 增加 OAuth 登录/登出、token 持久化与 HTTP 管理端点`
- G5b: `serve RuntimeRunner 接入 CredentialStore/env 凭据解析`
- G6: `session/new 应用并持久化 additionalDirectories/mcpServers`
- G7: `prompt 请求支持 mode 覆盖，或从契约移除 mode`
- G8: `config/providers 写入后热重载运行中模型目录`

## 低优先级观察

- `Close` 只校验 session 元数据后返回，不释放运行状态；当前 HTTP server 没有 per-session 长驻 runtime，暂不阻塞，但若后续引入持久 agent 进程需重审。
- session create/load 返回的 `AvailableModes` / mode options 硬编码为 `build`（`internal/httpapi/sessions.go:481-518`），插件 mode 只从 `/api/v1/modes` 暴露；pi-web-go 直接用 `/modes` + `/session/{id}/mode` 即可绕过。
- ACP 声明 `image: true` 但实际丢弃图片，应随 G1 一起修正，避免 pi-web-go 按 ACP 能力声明开放附件入口。
