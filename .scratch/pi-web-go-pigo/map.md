# WAYFINDER MAP: pi-web-go 纯 Go 后端 + pigo

**Status:** active

## Destination

把 pi-web 从 Next.js + Node 后端重构为独立仓库 `pi-web-go`：Go server 托管并代理 `pigo serve` HTTP API，纯 Go 实现 pi-web 现有功能（T3），运行期零 Node，前端迁为 Vite React SPA 并由 Go 托管静态产物。

## Notes

域：Go（chi）+ React SPA + pigo HTTP serve。每 session 先读本 map；实施前先解决 open ticket 对应的决策。Windows 为主运行环境；提交信息中文，默认不 push。关键约束：agent 相关配置（模型/provider/mode/thinking/slash 等）全部从 pigo 获取，不在 pi-web-go 重复实现；pigo 缺口记 pigo 侧 tickets（P1）。

## Decisions so far

- 产物形态：独立新仓库 `E:\project\pi-web-go`，module `github.com/luoxunhao/pi-web-go`
- 接入方式：Go server 调用 `pigo serve` HTTP API（`/api/v1/*`）
- 功能范围：S3 全量对齐 pi-web 40 个 API route
- 冲突优先级：T3 纯 Go 全量重实现，不保留 Node sidecar
- Node 边界：N1 运行期零 Node，构建期允许 Node
- 前端构建：V1 Vite React SPA
- API 契约：R1 保持 pi-web 现有前端 API 路径和响应形状
- 仓库布局：L1 `frontend/` + `cmd/server` + `internal/`
- Go 框架：G2 chi
- 配置：C1 TOML + 环境变量；agent 相关配置全部从 pigo API 获取
- 认证：H3 Go server 统一认证；U1 浏览器 Basic Auth（用户 `pi`，密码 `PI_WEB_GO_PASSWORD`）
- pigo 进程：P2 Go server 托管子进程；D1 PATH 优先 + config 覆盖
- pigo 凭据：S1 随机生成内部密码，可配置覆盖
- 静态托管：E1 生产 `go:embed`，开发读磁盘
- 文件白名单：F1 对齐 pi-web（会话 cwd/项目根/`~/pi-cwd-*`/显式 root + realpath），不自动放行 `$HOME`
- 会话状态：T1 Go server 自己聚合 SSE 状态
- 事件格式：M1 Go server 转换 pigo 事件为 pi-web 事件格式
- [01 pi-web API 契约矩阵](issues/01-pi-web-api-contract-matrix.md) — 40 routes/53 handlers 映射完成；pigo 直代理与 Go-native 归属已定
- slash 命令：C1 只暴露 pigo 支持的命令；缺失命令记录到对齐文档
- 模型配置：R1 先代理 pigo config/providers/discover/test；OAuth 后置
- skills/plugins：K1 委托 pigo CLI，外部搜索 Go 直连
- OAuth：O1 先 API key；OAuth 可行性开 research
- pigo 缺口：P1 记 pigo 侧 tickets，pi-web-go 先降级
- 里程碑：M1 核心主链路；M2 工程能力；M3 生态能力
- [02 pigo 缺口与降级矩阵](issues/02-pigo-gap-and-degradation-matrix.md) — 8 个缺口；M1 可降级通过，M3 被图片/OAuth/package HTTP 阻塞
- [03 SSE 事件映射与重连](issues/03-sse-event-mapping-and-replay.md) — 四类事件映射与 `internal/events` 转换层落地；after/Last-Event-ID 游标、30s 心跳；流式快照依赖 issue 04
- [04 会话状态聚合器](issues/04-session-state-aggregator.md) — Go server 聚合 running/queued；`/api/agent/running` 与 running/events SSE 已落地
- [13 OAuth 可行性研究](issues/13-oauth-feasibility-research.md) — 流程可纯 Go；pigo 仅静态 API key，OpenRouter 先试点，其余 gap
- [14 skills/plugins 委托矩阵](issues/14-skills-plugins-delegation.md) — 14 项操作映射 Go-native/skills.sh/pigo CLI/unsupported-with-doc；双轨状态与无热 reload 是主要风险

## Not yet specified

- export HTML 模板是否复用 pi SDK 样式（依赖 export ticket）
- 发布形态是否捆绑 pigo 二进制（D1 后置，M3 收口）
- app-update 的 Go 实现细节（npm registry 直连 vs pigo self-update）

## Out of scope

- 在 pi-web 仓库内保留 Node 后端（已选独立仓库）
- 运行期保留 Node sidecar（T3 否决）
- pi-web fleet/多机远程会话（沿用旧 wayfinder 的 out-of-scope）
- 在 pi-web-go 内重写 pigo 的模型/provider 配置逻辑（必须走 pigo API）
