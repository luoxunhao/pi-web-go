# 01 pi-web API 契约矩阵

**Type:** research

**Status:** open

**Date:** 2026-08-13

**Scope:** 将 pi-web 现有 40 个 `app/api/**/route.ts` 逐条映射到 pigo `serve` HTTP API 或 pi-web-go 的 Go-native 实现，记录响应形状、必填参数（尤其 `directory`）、错误格式、认证差异与 Windows 路径注意事项，并标注 M1/M2/M3 归属。

**Primary sources:**

- pi-web routes: `E:\project\pi-web\app\api\**\route.ts`
- pi-web backend helpers: `E:\project\pi-web\lib\*.ts`
- pigo HTTP API: `E:\project\pigo\api\v1\openapi.yaml`
- pigo handlers: `E:\project\pigo\internal\httpapi\*.go`
- pigo session store: `E:\project\pigo\internal\sessionstore\store.go`
- pi-web-go wayfinder: `E:\project\pi-web-go\.scratch\pi-web-go-pigo\map.md`、`issues\*.md`

## 摘要

- 40 个 route 文件，53 个方法 handler。
- 直接可代理 pigo 的 route：agent 事件、agent 命令、agent new、models、models-config（部分）、project-trust、session 读/改名/删除、state、context、thinking。
- 必须 Go-native 的 route：files、file-index、git、worktrees、cwd、home、app-update、bash-output、export、auto-name、skills、plugins、OAuth、provider catalog。
- pigo 缺口集中点：session 树/leaf/父链、运行态聚合、错误 envelope、模型配置形状、文件/工程能力、skills/plugins HTTP API。

## 里程碑定义

依据 `map.md` 与 issue 01/05/06/09/11/12/14：

- **M1 核心主链路**：agent 会话创建/提示/事件、session 列表/详情/状态、模型配置（models、models-config、API key）、project-trust、thinking 读取。
- **M2 工程能力**：files、file-index、git status/diff、worktrees、cwd browse/validate、default-cwd、home、bash-output、auto-name、export HTML、app-update。
- **M3 生态能力**：skills、plugins、OAuth、provider catalog、models.dev catalog、app-update 发布形态收口。

> 标注为参考性归属；实施时以对应 ticket（05/06/09/11/12/14）的实际排期为准。

## 汇总矩阵

| # | Route | Methods | Owner | 里程碑 |
|---|-------|---------|-------|--------|
| 1 | `/api/agent/[id]/bash-output` | GET | Go-native（文件读取 + session 引用校验） | M2 |
| 2 | `/api/agent/[id]/events` | GET SSE | pigo `/api/v1/events` 代理 + 事件转换 | M1 |
| 3 | `/api/agent/[id]` | GET, POST | pigo session/status/prompt/cancel/command 代理 | M1 |
| 4 | `/api/agent/new` | POST | pigo `/api/v1/session` + prompt/command | M1 |
| 5 | `/api/agent/running/events` | GET SSE | Go 状态聚合器（T1） | M1 |
| 6 | `/api/agent/running` | GET | Go 状态聚合器快照 | M1 |
| 7 | `/api/app-update` | GET | Go-native（npm registry / 配置化 updater） | M2 |
| 8 | `/api/auth/all-providers` | GET | Go-native provider catalog | M3 |
| 9 | `/api/auth/api-key/[provider]` | GET, POST, DELETE | pigo `config/providers` 代理 | M1 |
| 10 | `/api/auth/login/[provider]` | GET SSE, POST | Go-native OAuth（issue 13） | M3 |
| 11 | `/api/auth/logout/[provider]` | POST | Go-native OAuth credential 清理 | M3 |
| 12 | `/api/auth/providers` | GET | Go-native OAuth catalog | M3 |
| 13 | `/api/cwd/browse` | GET | Go-native | M2 |
| 14 | `/api/cwd/validate` | POST | Go-native + 文件 allowlist | M2 |
| 15 | `/api/default-cwd` | POST | Go-native | M2 |
| 16 | `/api/file-index` | GET | Go-native（git ls-files + walk） | M2 |
| 17 | `/api/files/[...path]` | GET, POST | Go-native（F1 files API） | M2 |
| 18 | `/api/git/diff` | GET | Go-native git | M2 |
| 19 | `/api/git/status` | GET | Go-native git | M2 |
| 20 | `/api/home` | GET | Go-native | M2 |
| 21 | `/api/models` | GET | pigo `config/providers` 代理 + 转换 | M1 |
| 22 | `/api/models-config/catalog` | GET | Go-native（models.dev） | M3 |
| 23 | `/api/models-config/discover` | POST | pigo `config/providers/discover` 代理 | M1 |
| 24 | `/api/models-config` | GET, PUT | pigo `config` / `config/providers` 代理 | M1 |
| 25 | `/api/models-config/test` | POST | pigo `config/providers/test` 代理 | M1 |
| 26 | `/api/plugins` | GET, POST | pigo CLI / Go-native pkgmgr（K1） | M3 |
| 27 | `/api/project-trust` | GET, POST | pigo `permission/trust` 代理 + Go 资源扫描 | M1 |
| 28 | `/api/sessions/[id]/auto-name` | POST | Go-native 标题生成 + pigo `/name` | M2 |
| 29 | `/api/sessions/[id]/context` | GET | pigo messages 代理 + Go context 构建 | M1 |
| 30 | `/api/sessions/[id]/entries/[entryId]/thinking` | GET | pigo messages 代理 | M1 |
| 31 | `/api/sessions/[id]/export` | GET | Go-native HTML renderer（issue 12） | M2 |
| 32 | `/api/sessions/[id]` | GET, PATCH, DELETE | pigo load/messages/command/delete + Go 富化 | M1 |
| 33 | `/api/sessions/[id]/state` | GET | pigo status + Go 运行态聚合 | M1 |
| 34 | `/api/sessions` | GET | pigo session list + Go 富化/聚合 | M1 |
| 35 | `/api/skills/check` | POST | Go-native / pigo CLI（K1） | M3 |
| 36 | `/api/skills/install` | POST | pigo CLI `install` / Go-native | M3 |
| 37 | `/api/skills` | GET, PATCH | Go-native skill loader + SKILL.md 编辑 | M3 |
| 38 | `/api/skills/search` | POST | Go-native skills.sh HTTP | M3 |
| 39 | `/api/skills/update` | POST | pigo CLI `update` / Go-native | M3 |
| 40 | `/api/worktrees` | GET, POST, DELETE | Go-native git worktree | M2 |

## 逐条契约

### 1. GET `/api/agent/[id]/bash-output`

- **当前行为**：`app/api/agent/[id]/bash-output/route.ts`。`?path=<absPath>` 读取 temp 下 bash output 文件；`download=1` 流式下载；否则返回 `{success:true,data:{output}}`。错误：400 `path required` / `invalid path`，403 `forbidden`，413 带 `{error,data:{size,maxBytes}}`，404 `full output unavailable`。
- **目标 owner**：Go-native。pigo HTTP API 没有 bash-output 端点；pigo 的 `bash_output` 是 agent 工具，读取内存 job buffer（`internal/agenttool/bash_control.go`），不是 temp 文件。pi-web-go 需自行实现 temp 文件写入/读取，或在 pigo 侧补 bash output 文件契约（issue 02 缺口）。
- **必填参数**：`id`、`path`；可选 `download`。
- **关键契约**：Windows `tmpdir` 路径解析必须防穿越；`isBashOutputPathReferencedBySession` 的引用校验需要读取会话转录。

### 2. GET `/api/agent/[id]/events`（SSE）

- **当前行为**：`app/api/agent/[id]/events/route.ts` + `lib/agent-event-stream.ts`。先启动/复用 RPC session，再发 `connected`、`message_start`、`agent_*`/`message_update` 等 pi 事件；30s heartbeat；session 不存在返回 404。
- **目标 owner**：pigo 代理：`GET /api/v1/events?sessionId=<id>&directory=<cwd>&after=<cursor>&types=...`（`internal/httpapi/events.go`），Go 侧转换为 pi-web 事件 wire（issue 03）。
- **必填参数**：`directory`（pigo 过滤事件用）、`sessionId`；`after` 为 pigo 数字 event id。
- **关键契约**：pigo SSE 是命名事件 `event: session.status` 且 `data` 为 `{id,type,data,time}`；pi-web 是无名 `data:` + 客户端事件对象。`EVENT_CURSOR_GONE` 返回 410；pigo 保留 10k 条/24h。认证：浏览器 Basic Auth（pi-web-go H3/U1）；EventSource 同源自动带凭据。

### 3. GET/POST `/api/agent/[id]`

- **当前行为**：`app/api/agent/[id]/route.ts` + `lib/rpc-manager.ts`。POST body `{type,...}` 走 RPC send；成功返回 `{success:true,data}`。prompt 类返回 `data:null`（preflight 通过）；GET 返回 `{running:false}` 或 `{running:true,state}`。prompt 且 session 不存在返回 404 `{error,code:"prompt_rejected",accepted:false}`。
- **目标 owner**：pigo 代理，命令映射：
  - `prompt`/`steer`/`follow_up` -> `POST /api/v1/session/{id}/prompt_async`（body `{directory,prompt:[{type:"text",text}]}`）
  - `abort` -> `POST /api/v1/session/{id}/cancel`
  - `get_state` -> `GET /api/v1/session/{id}/status?directory` + Go 运行态聚合
  - `set_model` -> `PATCH /api/v1/session/{id}` `{directory,model:"provider/modelId"}`
  - `set_thinking_level` -> 同 PATCH `{thinkingLevel}`
  - `set_session_name` -> `POST /api/v1/session/{id}/command` `{directory,command:"name",arguments}`
  - `get_commands` -> `GET /api/v1/commands?directory`
  - `compact` -> `/command` `{command:"compact"}`
  - `fork`/`navigate_tree`/`set_tools`/`reload`/`get_session_stats`/`get_last_assistant_text`/`set_auto_compaction`/`set_auto_retry`/`clear_queue`/`bash`/`abort_bash` 无 pigo HTTP 等价，需降级或 Go 实现（issue 02）。
- **必填参数**：`directory` 需由 Go server 维护 sessionId -> cwd 映射；prompt body 需要 `prompt`。
- **关键契约**：pigo `prompt_async` 只校验 directory，不校验 session 存在；Go 应先查 status 以复现 404 `prompt_rejected`。pigo `promptText` 只取 text block，图片附件被忽略（issue 02）。pigo 202 返回 `{messageId,accepted:true}`，pi-web 期望 `data:null`。

### 4. POST `/api/agent/new`

- **当前行为**：`app/api/agent/new/route.ts`。body `{cwd,type,message?,provider?,modelId?,toolNames?,thinkingLevel?}`；校验 cwd；创建 runtime session 后发送首个命令。返回 `{success,sessionId,data,model:{provider,modelId},thinkingLevel}`；`ensure_session` 时 `data:null`。
- **目标 owner**：pigo 代理：`POST /api/v1/session`（`directory` 必填，`model` 为 `provider/modelId`），随后 `prompt_async` 或 `/command`；`sessionId` 来自 pigo `Session.sessionId`。
- **必填参数**：`cwd`、`type`；可选 `provider`+`modelId`（必须成对）、`thinkingLevel`、`toolNames`。
- **关键契约**：pigo `NewSessionRequest` 没有 `thinkingLevel`/`toolNames`，需创建后 PATCH 或降级。pigo 返回 400 `MODEL_NOT_FOUND`、409 `MODEL_NOT_CONFIGURED`；Go 需转成 pi-web 的 `{error}` + prompt 场景 `code:"prompt_rejected"`。图片 prompt 同 route 3 缺口。

### 5. GET `/api/agent/running/events`（SSE）

- **当前行为**：`app/api/agent/running/events/route.ts`。SSE 帧 `data: {"type":"running","runningSessionIds":[...]}`，初始快照 + 变化推送 + 30s heartbeat。
- **目标 owner**：Go 状态聚合器（T1，issue 04）。pigo 无 running session 集合端点；需订阅 pigo `session.status`/`queue.updated` 并 reconcile。
- **必填参数**：无。
- **关键契约**：帧格式必须保持 pi-web 前端格式；Go server 重启后需从 pigo session 列表/状态恢复。

### 6. GET `/api/agent/running`

- **当前行为**：`app/api/agent/running/route.ts`。返回 `{runningSessionIds}`，`Cache-Control: no-store`。
- **目标 owner**：Go 状态聚合器快照。
- **必填参数**：无。
- **关键契约**：与 route 5 同源状态；无 pigo 端点。

### 7. GET `/api/app-update`

- **当前行为**：`app/api/app-update/route.ts`。请求 npm `@agegr/pi-web/latest`，12h 缓存、5s 超时；返回 `{currentVersion,latestVersion,updateAvailable,releaseUrl}`；失败 502 `{error}`。
- **目标 owner**：Go-native（npm registry 或配置化 updater）；`map.md` 注明实现细节未定。
- **必填参数**：无。
- **关键契约**：pi-web-go 版本来源为自身版本；release URL 生成逻辑需保持一致。

### 8. GET `/api/auth/all-providers`

- **当前行为**：`app/api/auth/all-providers/route.ts` + `lib/provider-listing.ts`。基于 SDK `ModelRuntime` 返回支持 API key 登录的 provider 列表 `{providers}`。
- **目标 owner**：Go-native provider catalog。pigo `GET /api/v1/config/providers` 只返回已配置 provider，不含内置 catalog 与能力声明。
- **必填参数**：无。
- **关键契约**：pi-web 返回字段含 `supportsOAuth`、`source`；Go catalog 需复刻能力判断，OAuth 能力待 issue 13。

### 9. GET/POST/DELETE `/api/auth/api-key/[provider]`

- **当前行为**：`app/api/auth/api-key/[provider]/route.ts`。GET `{provider,displayName,configured,source,models}`；POST `{apiKey}` 保存 credential；DELETE 删除，OAuth 类型冲突返回 409。
- **目标 owner**：pigo 代理：
  - GET -> `GET /api/v1/config/providers`（models、`apiKeyConfigured`）
  - POST -> `PUT /api/v1/config/providers/{providerId}`（`apiKey` + 现有 models 回填）
  - DELETE -> `PUT ...` 传空 `apiKey` 清 key；不能用 `DELETE /config/providers/{providerId}`，那会删掉整个 provider/models。
- **必填参数**：provider id；POST body `apiKey`。
- **关键契约**：pigo 永远不返回明文 key，只回 `apiKeyConfigured`；无 credential `source`（env vs config）与 catalog displayName。pi-web 的 `auth.json` credential store 不再适用，pi-web-go 必须把 key 写入 pigo 配置。

### 10. GET/POST `/api/auth/login/[provider]`（OAuth）

- **当前行为**：`app/api/auth/login/[provider]/route.ts`。GET SSE 驱动 OAuth，发 `auth_url`/`device_code`/`select_request`/`prompt_request`/`progress`/`success`/`error`/`cancelled`；POST `{token,code}` 回填 code。
- **目标 owner**：Go-native OAuth（issue 13 research）。pigo 无 OAuth credential 链路（issue 02）。
- **必填参数**：provider；POST `token`+`code`。
- **关键契约**：M3；M1/M2 建议禁用该入口而非假成功。SSE 事件格式需保持。

### 11. POST `/api/auth/logout/[provider]`

- **当前行为**：`app/api/auth/logout/[provider]/route.ts`。移除 OAuth credential；类型不匹配 409。
- **目标 owner**：Go-native OAuth credential 清理；无 pigo 端点。
- **必填参数**：provider。
- **关键契约**：M3；与 route 10 一起落地。

### 12. GET `/api/auth/providers`

- **当前行为**：`app/api/auth/providers/route.ts` + `lib/provider-listing.ts`。返回 OAuth provider 列表 `{providers}`。
- **目标 owner**：Go-native OAuth catalog。
- **必填参数**：无。
- **关键契约**：M3；字段 `usesCallbackServer`、`loggedIn`、`supportsApiKey`。

### 13. GET `/api/cwd/browse`

- **当前行为**：`app/api/cwd/browse/route.ts`。`?path=` 列出目录；Windows 无 path 时返回盘符列表；404/400/500。
- **目标 owner**：Go-native（`os.ReadDir`、Windows `GetLogicalDrives` 等价实现）。
- **必填参数**：可选 `path`。
- **关键契约**：Windows 盘符选择器、`resolveDirectory` 行为、parentPath 计算需复刻。

### 14. POST `/api/cwd/validate`

- **当前行为**：`app/api/cwd/validate/route.ts`。`{cwd}` 支持 `~`、相对路径；校验目录并 `allowFileRoot`；返回 `{success,cwd}`。
- **目标 owner**：Go-native + F1 allowlist。
- **必填参数**：`cwd`。
- **关键契约**：错误 400/500；allowlist 注册要与 files route 共享。

### 15. POST `/api/default-cwd`

- **当前行为**：`app/api/default-cwd/route.ts`。创建 `~/pi-cwd-YYYYMMDD` 并 allow；返回 `{cwd}`。
- **目标 owner**：Go-native。
- **必填参数**：无。
- **关键契约**：F1 规定 `~/pi-cwd-*` 自动进 allowlist；目录必须 `MkdirAll`。

### 16. GET `/api/file-index`

- **当前行为**：`app/api/file-index/route.ts`。`?cwd&q`；git `ls-files --cached --others --exclude-standard -z`，非 git 用 BFS walk（深度 8）；上限 5k 客户端 / 200k git / 50k walk；10s 缓存；返回 `{files,truncated}` 或 `{matches}`。
- **目标 owner**：Go-native（exec git 或 go-git + 同 caps）。
- **必填参数**：`cwd` 绝对路径。
- **关键契约**：403 allowlist、404 目录、400 非目录、Windows 绝对路径判断。

### 17. GET/POST `/api/files/[...path]`

- **当前行为**：`app/api/files/[...path]/route.ts`。GET `type=list|read|download|meta|preview|watch`；read 支持图片/音频/文档流式 + Range（206/416），text 预览 256KB 上限，docx preview 生成 HTML；watch 是 SSE。POST `type=upload|upload-check`，multipart，单文件 25MB、总量 100MB、conflict `error|skip|replace`，部分失败返回 207 `{uploaded,skipped,errors}`；POST 先过 `isApiRequestAllowed`。
- **目标 owner**：Go-native files API（F1，issue 05）。
- **必填参数**：URL 路径段拼文件绝对路径（支持 Windows 绝对路径形式）；`directory` 不在此路由，allowlist 隐式来自 sessions/projectRoot/`~/pi-cwd-*`/显式 root。
- **关键契约**：realpath + symlink 防护；Windows 路径段 `C:` 形式、大小写不敏感比较；Range 头、Content-Disposition RFC5987、MIME/语言表、SSE watch 事件、上传限制与 413。pi-web 对 POST 做 host/origin 校验，pi-web-go 统一 Basic Auth + origin 校验。

### 18. GET `/api/git/diff`

- **当前行为**：`app/api/git/diff/route.ts` + `lib/git-changes.ts`。`?cwd&path` 返回 `{supported,status,patch}`；400/403/500。
- **目标 owner**：Go-native git。
- **必填参数**：`cwd`、`path` 绝对路径。
- **关键契约**：只有 `supported:true` 才返回 patch；untracked/added/deleted 分别构造；`LC_ALL=C`、10s timeout。

### 19. GET `/api/git/status`

- **当前行为**：`app/api/git/status/route.ts` + `lib/git-changes.ts`。`?cwd` 返回 `{isGitRepository,repositoryRoot,files,additions,deletions}`；404/400/403。
- **目标 owner**：Go-native git。
- **必填参数**：`cwd` 绝对路径。
- **关键契约**：`git status --porcelain=v1 -z --untracked-files=all` 解析、numstat 行数、untracked 文本行统计、路径在 cwd 内过滤。

### 20. GET `/api/home`

- **当前行为**：`app/api/home/route.ts`。返回 `{home}`。
- **目标 owner**：Go-native `os.UserHomeDir`。
- **必填参数**：无。
- **关键契约**：Windows 用户目录；与 F1 的 `~/pi-cwd-*` 扫描一致。

### 21. GET `/api/models`

- **当前行为**：`app/api/models/route.ts` + `lib/models-cache.ts`。`?cwd` 返回 `ModelsData`：`models` name map、`modelList`、`defaultModel`、`thinkingLevels`、`thinkingLevelMaps`、`thinkingLevelPins`、`modelScopeWarnings`、可选 `modelError`。
- **目标 owner**：pigo 代理：`GET /api/v1/config/providers` + `GET /api/v1/config`。`modelList` 来自 `providers[].models[]`；`defaultModel` 拆分 `providers.defaultModel`；`models` key 为 `provider:modelId`；`thinkingLevels` 来自 `ModelEntry.thinkingLevels`。
- **必填参数**：`cwd`（用于 allowlist/trust gate；pigo config 本身是全局的）。
- **关键契约**：`thinkingLevelMaps`/`thinkingLevelPins`/`enabledModels` glob 作用域/warnings 在 pigo 无对应字段，需降级或 pigo 补 schema。错误形状：pigo `{error:{code,...}}` 要转成 pi-web 的 `{error}` 或安全 `modelError`。

### 22. GET `/api/models-config/catalog`

- **当前行为**：`app/api/models-config/catalog/route.ts`。缓存 models.dev catalog 1h；`?q&provider&baseUrl&limit` 返回 `{models,recommendation,source}`；502。
- **目标 owner**：Go-native（models.dev HTTP + search/recommend 逻辑）。
- **必填参数**：无；可选 `q`/`provider`/`baseUrl`/`limit`。
- **关键契约**：M3；外部网络超时与缓存需复刻。

### 23. POST `/api/models-config/discover`

- **当前行为**：`app/api/models-config/discover/route.ts` + `lib/model-discovery.ts`。body `{providerName,provider:{baseUrl,api,apiKey}}`；直接请求 provider `/models`；返回 `{models,endpoint}`；错误 400/502/504。
- **目标 owner**：pigo 代理：`POST /api/v1/config/providers/discover` `{name,baseUrl,apiKey,protocol}` -> `{provider,baseUrl,protocol,models}`。
- **必填参数**：`providerName`、`provider.baseUrl`；`api` 默认 `openai-completions`。
- **关键契约**：protocol 映射：`openai-completions` -> `openai`，`openai-responses` -> `openai/resp_api`，`anthropic-messages` -> `anthropic`，`google-generative-ai` -> `gemini`。pigo 响应无 `endpoint`，Go 需自行拼 URL 或返回 `baseUrl`。错误 envelope 转换。

### 24. GET/PUT `/api/models-config`

- **当前行为**：`app/api/models-config/route.ts` + `lib/models-config-store.ts`。GET 读 `getAgentDir()/models.json`；PUT 整体写 `{providers:{...}}` 并原子写、清理 cost/空 model id。
- **目标 owner**：pigo 代理：GET `GET /api/v1/config` 转成 `{providers}` 形状；PUT 翻译为 `PATCH /api/v1/config`（默认 model）+ 逐 provider `PUT /api/v1/config/providers/{id}`（删除的 provider 再 `DELETE`）。
- **必填参数**：GET 无；PUT body 完整 models config。
- **关键契约**：pi-web models.json 含 `cost`、provider 级字段，pigo `ModelEntry` 无 cost；PUT 无法 1:1 保真，需记录降级。pigo `PATCH /config` 只接受 `model`；`DELETE /config/providers/{id}` 是整 provider 删除。

### 25. POST `/api/models-config/test`

- **当前行为**：`app/api/models-config/test/route.ts`。body `{providerName,provider,model}`；用临时 models.json 实际 completion；返回 `{ok,latencyMs,status,responseText}`；Content-Type 必须 JSON；20s 超时。
- **目标 owner**：pigo 代理：`POST /api/v1/config/providers/test` `{modelId:"provider/modelId"}` -> `{success,responseTimeMs,modelResponse}`。
- **必填参数**：`providerName`、`provider`、`model`（含 `model.id`）。
- **关键契约**：pigo test 只测**已配置**模型，未保存的 candidate 会得到 `success:false`（200）；pi-web 测的是临时候选。Go 需临时 upsert 再测或列为 pigo 缺口。`status`（上游 HTTP code）pigo 不回；`ok`/`latencyMs`/`responseText` 需字段映射。

### 26. GET/POST `/api/plugins`

- **当前行为**：`app/api/plugins/route.ts`。GET `?cwd` 返回 `{packages,totals,diagnostics,projectResourcesLoaded}`；POST `{action:install|remove|update|disable|enable,source?,scope?,cwd}` 后返回同样形状；项目 scope 需 trust；请求需 origin/JSON 校验。
- **目标 owner**：pigo CLI / Go-native pkgmgr（K1，issue 14）。pigo HTTP 无 plugin/package 端点；CLI 面是 `pigo install|list|uninstall|update`（`cmd/pigo/main.go`）。
- **必填参数**：`cwd`；POST 另需 `action`，install/remove/disable/enable 需 `source`。
- **关键契约**：M3；CLI 文本输出需解析成结构化 PluginsResponse 或降级；pigo 无 `enable/disable` 独立动作等价。

### 27. GET/POST `/api/project-trust`

- **当前行为**：`app/api/project-trust/route.ts` + `lib/project-trust.ts`。GET `?cwd` 返回 `{requiresTrust,trusted}`；POST `{cwd}` 写入 trust 并销毁该 cwd 的 RPC session；无 trust 资源或 busy 返回 409。
- **目标 owner**：pigo 代理：`GET /api/v1/permission/trust`（查 path decision）+ `POST /api/v1/permission/trust` `{path:cwd,decision:"trusted"}`；`requiresTrust` 需 Go 扫描项目资源（`.pi/extensions`、project settings、`.agents/skills`）。
- **必填参数**：`cwd`。
- **关键契约**：pigo trust 是全局 `~/.pigo/trust.json` 决策表，不理解 pi 的 "resources require trust"；销毁 sessions 需 Go 聚合器逐个 `/cancel`/`/close`，pigo 无 "destroy by cwd" 端点。

### 28. POST `/api/sessions/[id]/auto-name`

- **当前行为**：`app/api/sessions/[id]/auto-name/route.ts` + `lib/session-title.ts`。用 shadow agent 生成标题，写入 session name；返回 `{title,usage}`；404/409/500。
- **目标 owner**：Go-native 标题生成（可调 pigo 侧会话/模型或本地启发式）+ `POST /api/v1/session/{id}/command` `{command:"name",arguments:title}`。
- **必填参数**：`id`。
- **关键契约**：pigo 无 auto-name 端点；`/name` 只改名。`usage` 字段 pigo `/command` 响应无，需 Go 记录或省略。

### 29. GET `/api/sessions/[id]/context`

- **当前行为**：`app/api/sessions/[id]/context/route.ts` + `lib/session-reader.ts`。`?leafId&deferThinking&deferMedia` 返回 `{context:{messages,entryIds,thinkingLevel,model}}`；基于 SessionManager 树构建。
- **目标 owner**：pigo 代理 `GET /api/v1/session/{id}/messages?directory` / `load` + Go context builder。
- **必填参数**：`id`、`directory`（pigo 必填）。
- **关键契约**：pigo `Message` 无 parent/leaf/tree 信息（`internal/httpapi/messages.go` 只输出 id/role/timestamp/content），无 `branch_summary`/`custom_message` 类型；`leafId`、fork 分支、deferThinking/deferMedia 无法完整复现，属 pigo schema 缺口（issue 02）。

### 30. GET `/api/sessions/[id]/entries/[entryId]/thinking`

- **当前行为**：`app/api/sessions/[id]/entries/[entryId]/thinking/route.ts`。`?blockIndex` 返回 `{thinking}`；400/404。
- **目标 owner**：pigo 代理 messages，按 entryId + blockIndex 找 `type:"thinking"` block。
- **必填参数**：`id`、`entryId`、`blockIndex`；`directory` 由 Go 查 session 映射。
- **关键契约**：pigo `Message.content` 的 thinking block 字段为 `{"type":"thinking","thinking":...}`，可直接映射。

### 31. GET `/api/sessions/[id]/export`

- **当前行为**：`app/api/sessions/[id]/export/route.ts`。`?inline=1` 用 pi CLI 导出 self-contained HTML 并 patch 深层树递归；返回 HTML + CSP/Disposition 头；404/500。
- **目标 owner**：Go-native HTML renderer（issue 12）。数据源可走 pigo `POST /api/v1/session/{id}/command` `{command:"export",arguments:<tmp>.jsonl}`（pigo 写 JSONL）或 messages API。
- **必填参数**：`id`；可选 `inline`。
- **关键契约**：M2；pigo `/export` 是文本命令（写文件、回文本），不是 HTML/HTTP 下载；Go 需处理大文件树、inline 与 attachment 头、CSP。

### 32. GET/PATCH/DELETE `/api/sessions/[id]`

- **当前行为**：`app/api/sessions/[id]/route.ts`。GET 返回 `{sessionId,filePath,info,leafId,tree,context,totalActiveMs}`；PATCH `{name}` 改标题；DELETE 删除并重挂子 session 到父 session，返回 `{ok:true}`。
- **目标 owner**：
  - GET -> pigo `GET /api/v1/session/{id}/load`（`directory` 必填）+ `GET /messages` + Go 富化；tree/leaf/filePath/totalActiveMs 无 pigo 等价。
  - PATCH -> pigo `/command` `{command:"name",arguments}`，包 `{ok:true}`。
  - DELETE -> pigo `DELETE /api/v1/session/{id}?directory`（204）。pigo `Store.Delete` 只删该 session 及依赖，不重挂子 session（`internal/sessionstore/store.go`），pi-web 的手工 re-parent 逻辑在 pigo 无对应 API，需 pigo 补或 Go 直连 store（违反纯 API 代理原则）。
- **必填参数**：`id`、`directory`。
- **关键契约**：GET 的 `info` 包含 parentSessionId、messageCount、firstMessage、transient 等，pigo summary 只有 title/updatedAt，需按 session 拉 messages 富化。

### 33. GET `/api/sessions/[id]/state`

- **当前行为**：`app/api/sessions/[id]/state/route.ts`。返回 `{running:true,state}` 或 `{running:false}`；404。
- **目标 owner**：pigo `GET /api/v1/session/{id}/status?directory` + Go 运行态聚合。
- **必填参数**：`id`、`directory`。
- **关键契约**：pigo `SessionService.Status` 硬编码 `status:"idle"` 且缺 `isStreaming`/`isBashRunning`/`isCompacting`/queuedMessages/systemPrompt/extensionStatuses；running 判定必须由 Go 从 SSE 聚合（issue 04）。

### 34. GET `/api/sessions`

- **当前行为**：`app/api/sessions/route.ts` + `lib/session-reader.ts`。`?force=1` 返回 `{sessions,runningSessionIds}`；sessions 含 path/id/cwd/name/created/modified/messageCount/firstMessage/parentSessionId/projectRoot/worktreeBranch。
- **目标 owner**：pigo `GET /api/v1/session`（不带 directory 列全部，或按 directory 过滤）+ Go 富化 + 运行态聚合。
- **必填参数**：无；可选 `force`。
- **关键契约**：pigo `SessionSummary` 只有 `sessionId,directory,title,updatedAt`；`parentSessionId`/`messageCount`/`firstMessage`/projectRoot 需额外拉取/本地缓存，列表页会变慢。pigo 分页 `before`/`limit` 是 offset 数字，不是 pi-web 的 force 缓存语义。

### 35. POST `/api/skills/check`

- **当前行为**：`app/api/skills/check/route.ts` + `lib/skill-updates.ts`。`{cwd,package?,scope?}` 检查已装 skill 更新，返回 `{updates}`。
- **目标 owner**：Go-native / pigo CLI 更新检查（K1，issue 14）。
- **必填参数**：`cwd`；`package`+`scope` 需成对。
- **关键契约**：M3；GitHub token 环境变量与 update args 需复刻。

### 36. POST `/api/skills/install`

- **当前行为**：`app/api/skills/install/route.ts`。`{package,scope,cwd?}` 执行 `npx skills add <pkg> -y --agent pi`；project scope 需 trust；返回 `{success,output}`。
- **目标 owner**：pigo CLI `install <pkg>`（pkgmgr 会分发 skills）或 Go-native 安装器。
- **必填参数**：`package`；project scope 需 `cwd`。
- **关键契约**：M3；无 pigo HTTP 等价；输出 ANSI 清理与成功判定需复刻。

### 37. GET/PATCH `/api/skills`

- **当前行为**：`app/api/skills/route.ts` + `lib/skills-service.ts`。GET `?cwd` 返回 skills + install info；PATCH `{filePath,disableModelInvocation}` 编辑 SKILL.md frontmatter，允许 agent dir / `~/.agents/skills`。
- **目标 owner**：Go-native skill loader + frontmatter 编辑。
- **必填参数**：GET `cwd`；PATCH `filePath`、`disableModelInvocation`。
- **关键契约**：M3；pigo slash registry 把 skills 当命令暴露（`GET /api/v1/commands`），但没有结构化 skills 列表/元数据端点。

### 38. POST `/api/skills/search`

- **当前行为**：`app/api/skills/search/route.ts`。`{query,limit}` 先请求 `SKILLS_API_URL`（默认 `https://skills.sh`），失败 fallback `npx skills find`；返回 `{results}`。
- **目标 owner**：Go-native skills.sh HTTP；npx fallback 可选保留。
- **必填参数**：`query`；可选 `limit`（1-50）。
- **关键契约**：M3；输出排序/格式化需一致；外部 API 失败返回空 results 或 500。

### 39. POST `/api/skills/update`

- **当前行为**：`app/api/skills/update/route.ts`。`{cwd,package,scope}` 校验已装 skill 后可更新，执行 npx；返回 `{success,skill,output}`。
- **目标 owner**：pigo CLI `update <pkg>` / Go-native。
- **必填参数**：`cwd`、`package`、`scope`。
- **关键契约**：M3；`canCheckForUpdates` 判定需 Go 侧维护。

### 40. GET/POST/DELETE `/api/worktrees`

- **当前行为**：`app/api/worktrees/route.ts` + `lib/worktree.ts`。GET `?cwd` 返回 `{projectRoot,isGit,isTopLevel,currentWorktreePath,worktrees}` 并 allow 各 worktree；POST `{cwd,branch}` 建 `<repo>-worktrees/<dir>`；DELETE `{cwd,path,force?}` 删除，dirty 409 `{error,dirty:true}`。
- **目标 owner**：Go-native git worktree（exec git 或 go-git）。
- **必填参数**：`cwd`；POST `branch`；DELETE `path`。
- **关键契约**：M2；`git worktree list --porcelain`、`rev-parse --git-common-dir`、Windows `toNativePath`、`samePath` 大小写不敏感、`LC_ALL=C` 错误匹配、allowFileRoot 扩展。

## 顶层契约风险（Top 5）

1. **Session 树/状态 schema 缺口**：pigo `Message` 无 parent/leaf/tree，`/status` 硬编码 `idle`；`/api/sessions/[id]`、`/context`、`/state`、`/sessions` 无法纯代理复现 pi-web 的树、分支、running flags、parentSessionId 与 totalActiveMs，必须先定 pigo schema 扩展或 Go 直读 sessionstore（issue 02/04）。
2. **SSE wire 与运行态聚合差异**：pi-web 消费 pi 的 `connected`/`message_start`/`message_update`/`agent_*` 事件和 running 集合；pigo serve 发出 `session.status`/`queue.updated`/`message.part.delta`/`tool.updated`/`permission.asked` 且 `data` 是 DomainEvent 包。需要事件转换层 + `after` 游标透传 + 断线重连 + 状态聚合器（issue 03/04）。
3. **错误与响应 envelope 不兼容**：pigo 统一 `{error:{code,message,requestId}}`，pi-web 多为 `{error:string}`，prompt 场景另有 `code:"prompt_rejected",accepted:false`，model test 用 `{ok:false,...}`。Go 代理必须逐 route 转换状态码与 body，否则前端错误处理直接坏。
4. **模型配置/auth 形状不对齐**：pigo 是全局 `config.toml` 扁平 model 条目（`provider/modelId`），pi-web 是 `models.json` provider 分组 + cost + displayName + source + thinkingLevelMaps/Pins；pigo `providers/test` 只测已配置模型、API key 删除需回填 models、`source` 不暴露。M1 模型配置代理需降级矩阵（issue 06）。
5. **文件/工程/生态能力无 pigo 端点**：files、file-index、git、worktrees、cwd、home、bash-output、export、skills、plugins、OAuth 全部要 Go-native；其中 `/api/files` 的 Windows 路径/realpath/上传/SSE/Range、`/api/worktrees` 的 dirty 409、DELETE session 的 child re-parent（pigo `Store.Delete` 不重挂）都是高复杂度高安全风险项（issue 05/09/11/12/14）。

## 附录：pigo HTTP API 端点清单（依据 openapi.yaml + server.go）

- `GET /api/v1/health`、`GET /api/v1/openapi.json`、`GET /api/v1/doc`
- `POST /api/v1/session`、`GET /api/v1/session`
- `DELETE /api/v1/session/{id}`（query `directory` 必填）
- `PATCH /api/v1/session/{id}`
- `POST /api/v1/session/{id}/mode`
- `POST /api/v1/session/{id}/load`
- `POST /api/v1/session/{id}/close`（query `directory` 必填）
- `GET /api/v1/session/{id}/status`（query `directory` 必填）
- `GET /api/v1/session/{id}/messages`（query `directory` 必填）
- `GET /api/v1/events`
- `POST /api/v1/session/{id}/prompt`、`POST /api/v1/session/{id}/prompt_async`
- `POST /api/v1/session/{id}/cancel`
- `GET /api/v1/commands`、`POST /api/v1/session/{id}/command`
- `GET/POST/DELETE /api/v1/permission/trust`
- `POST /api/v1/session/{id}/permissions/{permissionId}/reply`
- `GET/PATCH /api/v1/config`
- `GET /api/v1/config/providers`
- `PUT/DELETE /api/v1/config/providers/{providerId}`
- `POST /api/v1/config/providers/discover`
- `POST /api/v1/config/providers/test`
- `GET /api/v1/modes`

pigo 错误 envelope（`internal/httpapi/errors.go`）：`{"error":{"code","message","requestId","details?"}}`；已定义 code：`INVALID_PARAMS`、`UNAUTHORIZED`、`NOT_FOUND`、`INTERNAL`、`DIRECTORY_INVALID`、`MODEL_NOT_FOUND`、`MODEL_NOT_CONFIGURED`、`MODE_NOT_FOUND`、`QUEUE_FULL`、`PERMISSION_EXPIRED`、`EVENT_CURSOR_GONE`、`UNKNOWN_AUTH_METHOD`、`SESSION_NOT_FOUND`。

认证（`internal/httpapi/middleware.go`）：配置 password 后，pigo 对所有 `/api/v1/*` 要求 Basic Auth（user `pigo`）；CORS 放行 loopback/same-origin 与 `AllowedOrigins`。pi-web-go 按 H3/U1 对浏览器统一 Basic Auth（user `pi`，`PI_WEB_GO_PASSWORD`），内部携带 pigo 随机凭据（S1）转发。
