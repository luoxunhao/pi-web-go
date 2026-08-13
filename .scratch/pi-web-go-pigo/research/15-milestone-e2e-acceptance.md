# 15 里程碑验收与 E2E

**Type:** task
**Status:** resolved
**Date:** 2026-08-13

## 验收清单

### M1 核心主链路
- [x] Go server 启动并托管 pigo（supervisor/健康检查/随机密码）
- [x] 会话创建/列表/删除/状态
- [x] prompt 同步/异步/取消代理
- [x] SSE 事件转换 + `after`/`Last-Event-ID` + 30s 心跳
- [x] running 聚合与 `/api/agent/running` + running/events SSE
- [x] 文件白名单与 `/api/files`（read/list/download/meta/preview/watch/upload）
- [x] 模型配置代理（models/models-config/api-key）
- [x] slash 命令目录与 `/command` 映射
- [x] 统一 Basic Auth + host 白名单 + CORS
- [x] Vite 前端构建 + SPA fallback

### M2 工程能力
- [x] session CRUD/context/state/auto-name
- [x] home/cwd browse/validate/default-cwd
- [x] git status/diff、worktrees、file-index
- [x] bash-output、app-update
- [x] export HTML 纯 Go 渲染

### M3 生态能力
- [x] OAuth 可行性研究（OpenRouter 试点，其余 gap）
- [x] skills/plugins 委托矩阵
- [ ] OAuth 实际登录、skills/plugins UI 真实安装（依赖 pigo 侧能力，见 research 02/13/14）

## 验证记录

- `go test ./internal/... ./cmd/...`：通过
- `go build ./cmd/... ./internal/...`：通过
- `cd frontend && npm run typecheck`：通过
- `cd frontend && npm run build`：通过，产出 `frontend/dist`
- 真实浏览器 E2E：headless Chrome 打开 `http://127.0.0.1:30142/`，DOM 渲染出 AppShell（title 变为 `oh-my-ash - Pi Web`），无 `Uncaught` 控制台错误；截图 `e2e-home.png`
- 自动化覆盖：events 转换、cursor、session manager、files/docx、models proxy、auth/CORS/SSE、supervisor、export HTML

## 剩余风险

- M3 的 OAuth 与 skills/plugins 真实登录/安装被 pigo 缺口阻塞（图片 prompt、OAuth 凭据、package HTTP API、热 reload）；已作为 pigo 侧 tickets 记录。
- `next/navigation` shim 为最小实现，深链接/URL 状态需真实浏览器 E2E 迭代。
