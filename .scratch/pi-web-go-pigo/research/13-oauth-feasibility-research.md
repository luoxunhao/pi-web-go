# 13 OAuth 可行性研究

> 调研基准：
> - pi-web `E:\project\pi-web`，HEAD `dcb20f6`，package.json `0.8.8`（Next.js）。
> - pi SDK `E:\project\pi`，HEAD `541045ae0`，`packages/ai` 与 `packages/coding-agent` 均为 `0.84.1`（源码即发布包，`pi-web\node_modules` 未安装，直接读 monorepo 源码）。
> - pigo `E:\project\pigo`，HEAD `fc09a0c`。
> 本文件只做事实调查与判定，不修改 pigo / pi-web / pi 源码。

## 1. 结论

- pi-web 的 OAuth / device-code / manual-code 流程全部由 pi SDK 承担，Next.js 层只做 SSE 桥和手动输入回填；这些流程（PKCE、RFC 8628 device code、token exchange、refresh、loopback callback）没有 Node 专属协议依赖，**可以纯 Go 重写**。
- pigo 当前只能“存储并使用静态凭据”：`config.toml` 的 `[[models]]` 只有 `api_key`，HTTP API 的 `ProviderInput` 也只有 `apiKey`；运行时存在未接线的 `TokenSource` / `RegisterOAuth` seam，但没有生产代码注册 OAuth 凭据，也没有持久化 refresh token。
- 按 provider 判定：
  - **OpenRouter**：纯 Go 现在可行；OAuth 产物是永久 API key，pigo 可当作静态 `api_key` 使用。
  - **xAI**：纯 Go 流程可行，access token 可临时作为 pigo 静态 key；但 token 约 1 小时过期，长期可用需要 pigo 增加 refresh 持久化或 pi-web-go 每次刷新后重写配置并重启 pigo。
  - **Anthropic / OpenAI Codex / GitHub Copilot / Kimi / Radius**：登录流程本身可纯 Go，但当前 pigo 无法以正确 wire/auth 形态消费这些凭据，记录为 gap。
- 推荐 pi-web-go 范围：M1/M2 继续 API-key only；OpenRouter OAuth 可作为首个纯 Go 试点，但先落 pigo 配置热加载（或接受登录后重启 pigo）；其余 provider 记 pigo 侧 tickets，不在 pi-web-go 重复实现 LLM provider 调用。

### 1.1 判定汇总

| provider | pi-web 分类 | 登录流 | pigo 当前消费方式 | 判定 |
|---|---|---|---|---|
| `openrouter` | OAuth + API key | PKCE + loopback/manual code，换永久 key | 静态 `api_key`，无需 refresh | **纯 Go 现在可行** |
| `xai` | OAuth + API key | RFC 8628 device code + refresh | 静态 `api_key`（约 1h 过期） | 流程可行，**需 pigo refresh 改动** |
| `anthropic` | OAuth + API key | PKCE + loopback/manual code + refresh | 仅 `x-api-key`，OAuth 需 Bearer + headers | **gap** |
| `openai-codex` | OAuth only | browser/device code + refresh | 无内置 provider、无 headers/account-id | **gap** |
| `github-copilot` | OAuth + API key | device code + Copilot token + refresh | 无内置 provider、无动态 baseURL/headers | **gap** |
| `kimi-coding` | OAuth + API key | RFC 8628 device code + refresh | 仅 anthropic + `x-api-key`，需 Bearer | **gap** |
| `radius` | OAuth + API key | gateway PKCE/device code + refresh | 无 pi-messages 协议 | **gap** |
| `openrouter-images` | OAuth（图片） | 同 OpenRouter | 不在 pigo agent 运行面 | 不适用/gap |

## 2. pi-web 认证表面

需要复刻的 API 面（`E:\project\pi-web\app\api\auth`）：

| 路由 | 行为 | 源码 |
|---|---|---|
| `GET /api/auth/providers` | 返回声明 `auth.oauth` 的 provider（含 anthropic） | `app/api/auth/providers/route.ts:9-12` |
| `GET /api/auth/all-providers` | 返回声明 API-key login 的 provider，排除 `models.json` 自定义 source | `app/api/auth/all-providers/route.ts:9-12` |
| `GET /api/auth/login/[provider]` | SSE：创建 `ModelRuntime.login(provider,"oauth",...)`，把 `auth_url` / `device_code` / `select_request` / `prompt_request` / `progress` 转发给前端；`login` 的 `prompt()` 通过进程内 `__piLoginCallbacks` registry 等待前端 POST | `app/api/auth/login/[provider]/route.ts:45-172` |
| `POST /api/auth/login/[provider]` | body `{token, code}`，用 registry 把 manual code / select 选项喂回 `ModelRuntime.login` | `app/api/auth/login/[provider]/route.ts:18-41` |
| `POST /api/auth/logout/[provider]` | 按 `oauth` 类型删除 `~/.pi/agent/auth.json` 中凭据 | `app/api/auth/logout/[provider]/route.ts:7-23` |
| `GET/POST/DELETE /api/auth/api-key/[provider]` | 查询 auth 状态、调用 `apiKeyAuth.login` 存 key、按 `api_key` 类型删除 | `app/api/auth/api-key/[provider]/route.ts:11-74` |

前端流程（`E:\project\pi-web\components\ModelsConfig.tsx`）：
- `OAuthDetail` 用 `EventSource` 打开登录流，处理 `auth`、`device_code`、`prompt_request`、`select_request` 状态，并把用户输入 POST 回 `/api/auth/login/[provider]`（`ModelsConfig.tsx:1326-1548`）。
- `ApiKeyDetail` 直接 POST / DELETE API key（`ModelsConfig.tsx:1586-1640`）。
- provider 能力分类来自 SDK 的 `provider.auth.apiKey?.login` / `provider.auth.oauth`，不是硬编码 id 列表（`E:\project\pi-web\lib\provider-listing.ts:76-118`、`lib/provider-listing-runtime.ts:8-27`）。
- 凭据文件与 pi SDK 共用：`storeProviderCredential` / `removeStoredCredentialIfType` 操作 `join(getAgentDir(),"auth.json")`，用 proper-lockfile 与 SDK 的 `AuthStorage` 同一把锁（`E:\project\pi-web\lib\provider-credential-store.ts:76-96`）。
- 自定义 provider 的 discover/test 通过临时 `models.json` + `ModelRuntime.getAuth(model)` 解析 `apiKey/headers`（`E:\project\pi-web\lib\model-discovery-auth.ts:24-56`），discover 路由再按 `api` 决定 `x-api-key` / `x-goog-api-key` / `Authorization: Bearer`（`app/api/models-config/discover/route.ts:17-33`）。

## 3. pi SDK 凭据模型与刷新语义

- 凭据类型：`api_key`（`key?` + provider env）或 `oauth`（`access` + `refresh` + `expires`，扩展字段允许如 `accountId`、`enterpriseUrl`、`availableModelIds`）（`E:\project\pi\packages\ai\src\auth\types.ts:24-37`）。
- `AuthStorage` 默认写 `~/.pi/agent/auth.json`，`oauth` 凭据校验必须同时有 `access`、`refresh`、有限 `expires`；写文件 `0600`，跨进程锁用 proper-lockfile（`E:\project\pi\packages\coding-agent\src\core\auth-storage.ts:208-252`、`E:\project\pi\packages\coding-agent\src\config.ts:515-540`）。
- `Models.login()` 把 flow 返回的 credential 通过 `credentials.modify()` 持久化（`E:\project\pi\packages\ai\src\models.ts:565-612`）。
- 请求前解析 `getAuth()`：若 OAuth 剩余有效期不足 5 分钟，在 `CredentialStore.modify()` 锁内调 `oauth.refresh()`，刷新成功后原地持久化再继续；刷新失败不静默回落（`E:\project\pi\packages\ai\src\auth\resolve.ts:119-170`）。
- `OAuthAuth` 接口拆成 `login` / `refresh` / `toAuth`：`toAuth` 把 credential 转成请求级 `apiKey/headers/baseUrl`（`E:\project\pi\packages\ai\src\auth\types.ts:206-229`）。这就是“凭据可以影响请求头/端点”的 seam，pigo 目前没有对应物。

## 4. pigo 当前能力

- 配置模型只有静态 `APIKey` 字段：`config.ModelConfig.APIKey string \`toml:"api_key"\``（`E:\project\pigo\internal\cli\config\providers.go:11-16`）；`MarshalJSON` 不回显 key，只由 HTTP 层给 `apiKeyConfigured`（`E:\project\pigo\internal\httpapi\config.go:235-248`）。
- HTTP API 的 provider 输入只有 `apiKey/baseUrl/models/name/protocol`，没有 headers、oauth、refresh 字段（`E:\project\pigo\internal\httpapi\gen\api.gen.go:238-245`）。
- 运行时解析：`ResolveConfiguredModel` 最终返回 `entry.APIKey`；serve/ACP 的 `RuntimeRunner.ResolveForModel` 也从 `ConfiguredModels.Find()` 拿 `entry.APIKey`（`E:\project\pigo\internal\provider\configresolve.go:54-88`、`E:\project\pigo\internal\acp\runner.go:143-165`）。
- `internal/provider/auth.go` 已有 `TokenSource`（内存 access/refresh/expiry + `Refresh` 回调）和 `CredentialStore.RegisterOAuth`，`GetAPIKey` 优先级是 OAuth > override > env > config（`E:\project\pigo\internal\provider\auth.go:74-161`、`213-252`）。但全仓库 `RegisterOAuth` 只出现在 `auth_test.go`，生产代码没有注册任何 OAuth source；文档把 OAuth 列为“已具备”，实际未接线（`E:\project\pigo\tasks\prd-pigo-agent.md:155-160` 勾选项未完成、`E:\project\pigo\tasks\prd-pigo-parity-features.md:5`、`E:\project\pigo\docs\architecture\03-provider-storage-memory-integration.md:15`、`E:\project\pigo\docs\adr\0001-acp-parity-scope.md:3`）。
- serve/ACP 的 `ConfiguredModels` 只在启动时 `Load()` 一次（`E:\project\pigo\cmd\pigo\prompt_runner.go:94-95`、`E:\project\pigo\internal\acp\models.go:19-31`）；HTTP `/api/v1/config/providers` 由独立 `ConfigService` 写文件，未发现写后 reload 的调用，因此 pi-web-go 通过 HTTP API 更新 `api_key` 后，运行中的 pigo 不保证立即生效（需要 pigo 修复或重启进程）。
- ACP 面：`initialize` 返回 `authMethods: []`，`authenticate` 直接返回 `unknown auth method`（`E:\project\pigo\internal\acp\http_adapter.go:74-75`、`177`、`E:\project\pigo\internal\acp\server.go:140`）。
- 内置 provider registry 没有 `github-copilot`、`openai-codex`、`radius`；`anthropic` 的 env 顺序含 `ANTHROPIC_OAUTH_TOKEN`，但 anthropic wire driver 固定用 `x-api-key`（`E:\project\pigo\internal\provider\registry.go:66-342`、`E:\project\pigo\internal\provider\providers.go:700-721`、`E:\project\pigo\internal\provider\custom_provider.go:54`）。
- `StreamConfig.Extra` 目前只用于 `max_tokens`，没有 header 注入 seam；registry 的 `ExtraHeaders` 注释也写明是 no-op（`E:\project\pigo\internal\provider\providers.go:572-578`、`E:\project\pigo\internal\provider\resolve.go:63-65`）。

## 5. per-provider 判定

### 5.1 OpenRouter —— 纯 Go 现在可行

- pi SDK 流程：PKCE + 临时 loopback callback（随机端口 `/oauth/callback/<uuid>`）+ manual code 兜底；用 `code + code_verifier` 到 `https://openrouter.ai/api/v1/auth/keys` 换**永久 API key**；credential 为 `refresh: ""`、`expires: MAX_SAFE_INTEGER`，refresh 原样返回（`E:\project\pi\packages\ai\src\auth\oauth\openrouter.ts:20-21`、`97-131`、`242-301`）。
- 请求 auth：`toAuth` 返回 `{ apiKey: credential.access }`，走 OpenAI-compatible Bearer（`openrouter.ts:301-310`）。
- pigo：配置 `base_url=https://openrouter.ai/api/v1`、`protocol=openai`、`api_key=<permanent key>` 即可静态消费，不需要 refresh。
- 结论：**可行**。pi-web-go 用纯 Go 实现 flow 后把 key 写入 pigo 配置（或 spawn pigo 前设 `OPENROUTER_API_KEY`）。前置条件是 pigo 配置热加载修复，或 pi-web-go 接受登录后重启 pigo。

### 5.2 xAI —— 流程可行，pigo 需 refresh 改动

- pi SDK 流程：RFC 8628 device code（`auth.x.ai/oauth2/device/code` -> `token`），`expires_in` 缺省按 3600 秒，刷新用 `refresh_token`（`E:\project\pi\packages\ai\src\auth\oauth\xai.ts:8-11`、`201-235`）。
- 请求 auth：`toAuth` 返回 `{ apiKey: access }`，OpenAI-compatible Bearer（`xai.ts:235-238`）。
- pigo：access token 可作为静态 `api_key` + `base_url=https://api.x.ai/v1` + `protocol=openai` 使用，token 有效期内可跑。
- 结论：**需要 pigo 改动才长期可用**。纯 Go 流程和静态消费都成立；约 1 小时过期意味着要么 pi-web-go 每次 prompt 前刷新并重写 pigo 配置（受 4. 的 reload/restart 限制），要么 pigo 增加 OAuth 凭据持久化 + refresh 接线（`TokenSource` 已具备但未接线）。

### 5.3 Anthropic —— gap

- pi SDK 流程：PKCE + 固定 loopback `127.0.0.1:53692/callback` + manual code 兜底；token 端点 `platform.claude.com/v1/oauth/token`，含 refresh（`E:\project\pi\packages\ai\src\auth\oauth\anthropic.ts:29-35`、`99-160`、`234-306`、`355-359`）。
- 请求 auth：access token（`sk-ant-oat*`）必须用 `Authorization: Bearer`，并带 `anthropic-beta: claude-code-20250219, oauth-2025-04-20`、`user-agent: claude-cli/...`、`x-app: cli` 等头（`E:\project\pi\packages\ai\src\api\anthropic-messages.ts:843-906`）。
- pigo：anthropic 协议固定 `x-api-key`（`E:\project\pigo\internal\provider\providers.go:700-721`、`custom_provider.go:54`），且没有 per-model headers。
- 结论：**gap**。pigo 需要按凭据选择 auth scheme（Bearer）并支持额外 header 注入，才可能消费 Claude Pro/Max OAuth。

### 5.4 OpenAI Codex —— gap

- pi SDK 流程：登录时可选 browser（PKCE + loopback `127.0.0.1:1455/auth/callback` + manual code 兜底）或 device code（`deviceauth/usercode` + poll + `code_verifier` 交换）；refresh 用 `refresh_token`（`E:\project\pi\packages\ai\src\auth\oauth\openai-codex.ts:26-37`、`427-440`、`445-500`、`515-539`）。
- 请求 auth：`Authorization: Bearer <access>`，还要从 JWT 提取 `chatgpt-account-id`，并带 `originator`、自定义 `User-Agent`；transport 可能是 WebSocket/SSE（`E:\project\pi\packages\ai\src\api\openai-codex-responses.ts:271-294`、`1609-1621`）。
- pigo：没有 `openai-codex` 内置 provider，`ProviderInput` 无法表达额外 header，也没有 credential 派生 header 的机制。
- 结论：**gap**。pigo 需要 openai-codex 专用 provider 或通用 per-model headers + 动态 header 支持；WebSocket transport 是否必须另核。

### 5.5 GitHub Copilot —— gap

- pi SDK 流程：device code（github.com 或 enterprise domain）换 GitHub access token，再用 `copilot_internal/v2/token` 换 Copilot token（`expires_at`），登录后还要 enable 模型并抓 `availableModelIds`；refresh 复用 Copilot token 端点；`toAuth` 从 token 的 `proxy-ep` 推导动态 baseURL（`E:\project\pi\packages\ai\src\auth\oauth\github-copilot.ts:156-225`、`255-285`、`357-426`）。
- 请求 auth：需要 Copilot 专用头（`User-Agent`、`Editor-Version` 等）和请求级动态头 `X-Initiator` / `Openai-Intent` / `Copilot-Vision-Request`（`E:\project\pi\packages\ai\src\api\github-copilot-headers.ts:9-28`、`openai-completions.ts:646-664`、`openai-responses.ts:223-246`、`anthropic-messages.ts:868-882`）。
- pigo：registry 没有 `github-copilot`，且没有动态 baseURL / per-request headers / model enablement。
- 结论：**gap**。

### 5.6 Kimi Coding —— gap

- pi SDK 流程：RFC 8628 device code（`auth.kimi.com/api/oauth/device_authorization` -> token），带 refresh（`E:\project\pi\packages\ai\src\auth\oauth\kimi-coding.ts:13`、`76-161`、`281-305`）。
- 请求 auth：pi 的 `kimi-coding` 是 Anthropic Messages wire，`toAuth` 返回 `Authorization: Bearer`（`kimi-coding.ts:307-310`、`E:\project\pi\packages\ai\src\providers\kimi-coding.ts:7-21`）。
- pigo：registry 把 `kimi-coding` 标成 OpenAI 协议（`E:\project\pigo\internal\provider\registry.go:229-233`），与 pi 实际 wire 不符；即便走 pigo 的 anthropic 协议，也只有 `x-api-key`，无法表达 Bearer。
- 结论：**gap**，同时记录 pigo registry 的 `kimi-coding` 协议描述与 pi 不一致。

### 5.7 Radius —— gap

- pi SDK 流程：gateway 动态发现 `GET /v1/oauth`，可选 browser PKCE（loopback `127.0.0.1:1456/oauth/callback`）或 device code，token/refresh 走 gateway `/v1/oauth/token`；provider 使用 pi-messages 协议并从 gateway 拉模型目录（`E:\project\pi\packages\ai\src\auth\oauth\radius.ts:27-35`、`220-260`、`305-330`、`363-398`、`E:\project\pi\packages\ai\src\providers\radius.ts:20-68`）。
- pigo：没有 pi-messages 协议、没有 radius 内置 provider、没有动态 gateway 模型目录。
- 结论：**gap**。

### 5.8 其他

- `openrouter-images` 复用 OpenRouter OAuth，但属于图片模型，不在 pigo coding-agent 运行面内，记为不适用/gap（`E:\project\pi\packages\ai\src\providers\openrouter-images.ts:11-16`）。
- 纯 API-key provider（deepseek、groq、mistral、google、openrouter 等）不涉及本 ticket：pigo 已有 `api_key` 存储与 `apiKeyConfigured`，pi-web-go 可直接代理。

## 6. pigo 侧改动清单（按优先级）

1. **配置热加载（P1）**：`httpapi.ConfigService` 写 `config.toml` 后让 `ConfiguredModels` reload，否则 pi-web-go 通过 pigo API 写 key/模型不生效；当前 workaround 是 pi-web-go 重启 pigo。
2. **provider 元数据 API（P1）**：pigo 目前只暴露“已配置模型”，没有 pi-web 所需的“哪些内置 provider 支持 OAuth / API key、模型数、显示名”目录；pi-web-go 要复刻 `/api/auth/providers` 与 `/api/auth/all-providers`，需要 pigo 增加只读 provider catalog/auth 能力端点，或允许 pi-web-go 维护本地静态目录（与 map “agent 配置全从 pigo 获取”冲突，故优先 pigo 端点）。
3. **OAuth 凭据持久化 + refresh 接线（P2）**：把 `internal/provider/auth.go` 的 `TokenSource` 用起来：增加 auth.json 或 config 内 OAuth 字段、每 provider refresh 实现、serve/ACP 启动时 `RegisterOAuth`，并让 `GetAPIKey` 在过期时刷新。
4. **per-model auth scheme / headers / 动态 baseURL（P2）**：这是 Anthropic、OpenAI Codex、GitHub Copilot、Kimi 共用的最小能力：配置项或凭据能指定 `x-api-key` / Bearer / 额外 header，credential 能覆盖 baseUrl。
5. **专用 provider（P2/P3）**：`openai-codex`、`github-copilot`、`radius`、anthropic Claude Code headers、Kimi anthropic-Bearer 各按需要落地。

## 7. pi-web-go 建议范围

- **M1/M2**：保持“API key only”。实现 `/api/auth/api-key/[provider]`（写入 pigo `api_key`）与 provider 列表的 API-key 子集；OAuth 路由可以先返回能力未启用，不复制 pi-web 的 OAuth UI。
- **首个 OAuth 试点**：OpenRouter。纯 Go 实现 PKCE/manual-code 流程，成功后把永久 key 写入 pigo 配置；由于 pigo 热加载未定，试点应同时包含 pigo 改动 1（或 pi-web-go 登录后重启 pigo 的验收路径）。
- **xAI**：当 pigo 改动 1 + 3 落地后再启用；在此之前不实现，避免每 1 小时重启 pigo 的体验。
- **Anthropic / OpenAI Codex / GitHub Copilot / Kimi / Radius**：不在 pi-web-go 实现 provider 调用；只把“pigo 不支持消费该凭据”明确暴露在 ModelsConfig 对应入口，并记 pigo tickets。
- **自定义 provider discover/test**：pigo `discover` 只接受 `apiKey`；pi-web 的 `resolveModelDiscoveryAuth` 还能解析任意 `headers`，pi-web-go 复刻时要么在 Go 侧自己发 discover/test 请求（保留 headers），要么给 pigo discover 增加 headers 参数。
