# frontend — pi-web-go 前端（Vite React SPA）

本目录是 pi-web-go 的 Web 前端：由 upstream pi-web 前端迁移而来的 **Vite + React 19 + TypeScript SPA**。这里**没有 Next.js 服务器**——所有 `/api/*` 请求由仓库根的 Go 后端（`cmd/server`）提供，构建产物 `dist/` 由 Go 静态托管。

> ⚠️ 本目录下的 `README.md` / `README.zh-CN.md` 等来自 upstream pi-web，描述旧的 Next.js 架构，仅供对照上游 UI 行为，**不代表本仓库实际架构**。

## 快速开始

前置：Node ≥ 22.19（package.json engines）。依赖用 npm 管理（`bun.lock` 为迁移遗留，勿混用 bun 与 npm）。

```bash
npm install
npm run dev      # Vite dev :5173，/api 代理到 http://127.0.0.1:30141（需先启动 Go 后端）
npm run build    # tsc -b && vite build → dist/，由 Go 静态托管
npm run typecheck
```

开发时在仓库根另开终端跑 `go run ./cmd/server`。**不要 `next build`——本目录不是 Next.js 项目。**

## 架构

```
浏览器 SPA ──fetch/SSE──► Go 后端 :30141（/api/*）
    │                        ├─ /api/agent/{id}/events   会话 SSE 流
    └────────────────────────┴─ 其余 REST：sessions / models / files / git ...
```

- 会话消息走 SSE：`lib/agent-event-connection.ts` 建立连接 → `lib/agent-event-wire.ts` 解析 wire 事件 → `lib/agent-event-stream.ts` 流式聚合
- 页面核心状态在 `hooks/useAgentSession.ts`（约 1900 行）：消息列表、流式渲染、fork / 分支导航、断线重连、运行对账
- 会话包装在 `lib/rpc-manager.ts`：每 session 一个 wrapper、fork 即销毁、启动锁共享

## 文件地图

```
src/main.tsx              Vite 入口：StrictMode + I18nProvider + AppShell
src/next-navigation.ts    next/navigation 兼容 shim（迁移遗留）
app/page.tsx              Next.js 遗留壳，构建不经过它
app/globals.css           全局样式 + CSS 变量（--bg / --accent / --border 等）
components/
  AppShell.tsx            布局 + URL 状态 + Tab 管理
  ChatWindow / ChatInput / MessageView   会话主链路
  SessionSidebar / FileExplorer / FileViewer / TabBar   会话树与文件查看
  ModelsConfig / PluginsConfig / SkillsConfig   配置弹窗
  MarkdownBody / MermaidBlock / ImagePreview / BranchNavigator / ChatMinimap   渲染增强
hooks/
  useAgentSession.ts      消息 + 流式 + SSE + fork/导航/对账（最大文件）
  useI18n / useTheme / useResizablePanel / useAudio / useIsMobile /
  useKeyboardShortcuts / useDragDrop / useViewportHeight
lib/
  agent-client.ts         类型化 fetch 助手
  agent-event-connection.ts / agent-event-stream.ts / agent-event-wire.ts   SSE 连接/聚合/解析
  rpc-manager.ts          AgentSession 封装 + 注册表 + startRpcSession
  session-reader.ts       会话 .jsonl 读取与上下文适配
  model-catalog.ts / models-config-store.ts / model-scope.ts   模型目录与配置
  worktree.ts / git-changes.ts / paths.ts / file-access.ts     工程能力与路径安全
  markdown.ts / ansi.ts / file-fuzzy.ts / file-types.ts         渲染工具
  i18n/                   中英双语（en.ts / zh-CN.ts 各 484 行）
```

## 关键约定与陷阱

- **API 契约 R1**：保持 pi-web 兼容的路径与响应形状；改 `lib/` 前先核对 Go 侧 `internal/server` 路由
- **工具调用字段归一**：pi 存储 `{type:"toolCall",id,name,arguments}`，UI 用 `{toolCallId,toolName,input}`，必须过 `lib/normalize.ts`（文件加载与流式两处都要调）
- **路径比较**：Windows 下用 `samePath()`，禁止 `===`；git 输出 POSIX 路径需 `toNativePath()`
- **文件访问**：受白名单约束，`lib/path-security.ts` 是唯一实现，勿另写一套绕过
- **模型配置**：只读 pigo 代理数据（`/api/models`、`/api/models-config`），前端不落盘 models.json
- **SSE 重连**：页面刷新时若 `state.isStreaming` 为真自动重连；运行中对账用 monotonic run id 防过期事件复活
- **compaction 事件**：兼容 `compaction_start/end` 与旧 `auto_compaction_*` 两套命名
- **i18n**：文案必须走 `lib/i18n` registry，新增 key 同步 en.ts 与 zh-CN.ts

## 测试

前端单测用 Node 内置 test runner（`.test.mjs`，100 个），无需额外框架：

```bash
node --test components/*.test.mjs lib/*.test.mjs hooks/*.test.mjs
```

模式：纯逻辑测试直接 import 模块函数断言；组件测试（如 `AppShell.*.test.mjs`）对组件源码做结构断言（正则匹配关键实现模式），无 DOM 环境。新增逻辑改动须配对应 `.test.mjs`。
