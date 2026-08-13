# 14 skills/plugins 委托矩阵

日期：2026-08-13
范围：pi-web-go 如何用纯 Go 后端承接 pi-web 的 skills/plugins 管理 UI；委托策略限定为 Go-native、pigo CLI、skills.sh 直连、unsupported-with-doc 四类。

## 1. 关键事实

### 1.1 pigo 包管理 CLI（`internal/cli/pkgcmd/pkgcmd.go`，`cmd/pigo/main.go:159-174`）

- 子命令：`pigo install`、`pigo list`、`pigo uninstall`、`pigo update`，均为位置参数，不经 pflag。
- 只支持 `npm:` 源（`internal/pkgmgr/ref.go`）；`pigo install npm:<name>[@version]`。
- 拉包走 `npm pack --ignore-scripts --loglevel error`（`internal/pkgmgr/fetch.go`），不安装依赖、不跑脚本；因此 `install/update` 要求本机 `npm` 在 PATH。
- 分类支持 `extension / skill / prompt / theme`（`internal/pkgmgr/classify.go`），一个包可同时是多种类型。
- 落盘位置（`internal/pkgmgr/layout.go` + 各 distribute 文件）：
  - extension → `$PIGO_HOME/plugins`；但 `DistributeExtension` 在 Windows 上直接报 `extension install is not supported on windows yet`（`distribute.go:40`）。
  - skill → `PIGO_SKILLS_DIR`，默认 `~/.agents/skills`，复制为 `<skillsDir>/<name>/SKILL.md` 布局。
  - prompt → `$PIGO_HOME/prompts`（同时兼容旧 `commands/` 目录发现）。
  - theme → `$PIGO_HOME/themes/<name>/`，pigo 尚无 theme 运行时。
- 锁文件：`$PIGO_HOME/packages.json`（默认 `~/.pigo/packages.json`），schema version 1，记录 `name/source/version/types/files`（`internal/pkgmgr/lockfile.go`）。这是 list/uninstall/update 的唯一状态源。
- 没有 JSON 输出模式，没有 package management HTTP API；`pigo serve` 的生成客户端中也没有 packages/install 端点。
- `pigo update` 不带包名时路由到**二进制自更新**（`main.go:163-171`），pi-web-go 委托时永远要传包名。
- CLI 输出：`list` 为 `name\tversion\ttypes\t source` TSV；install/uninstall/update 是自然语言行；错误走 stderr，exit code 0/1/2。

### 1.2 pigo 的 skills/plugins/prompts 发现

- Skills：`runtime.LoadSkillsDir`（`internal/runtime/skills.go:206-245`）扫 `PIGO_SKILLS_DIR`/`~/.agents/skills`，支持 `<name>/SKILL.md` 嵌套布局和根目录 `*.md`，解析 YAML frontmatter 后注册 `/skill:<name>`。
- Plugins：`internal/plugin/manager.go:38-71` 扫 `$PIGO_HOME/plugins`，只发现“直接位于目录内的可执行普通文件”，跳过子目录；Windows 只认 `.exe/.bat/.cmd`；协议为 stdio JSON-RPC 2.0，manifest 提供 tools/commands/modes/events。
- Prompt 模板：`runtime.LoadUserCommandsDir`（`internal/runtime/slashcommand.go:464-500`）非递归扫 `$PIGO_HOME/prompts` 和旧 `commands/` 的 `*.md`。
- `pigo serve` / `pigo acp` 在**进程启动时**只调用一次 `plugin.Discover` 并构建一次 SlashRegistry（`cmd/pigo/serve.go:35-45`、`cmd/pigo/prompt_runner.go:39-65`）；安装新插件/技能后不会热发现，也没有 reload 命令。
- HTTP 侧可查 `GET /api/v1/commands` 拿当前 slash 命令（含 `skill:*` 与插件命令），但只有 name/description，没有文件路径、安装信息、disabled 状态。

### 1.3 pi-web UI 与路由（按钮 → API）

SkillsConfig（`components/SkillsConfig.tsx`）：
- 加载列表 → `GET /api/skills?cwd=`（`app/api/skills/route.ts`）。
- 搜索 → `POST /api/skills/search`（`app/api/skills/search/route.ts`）。
- 安装 global/project → `POST /api/skills/install`（`app/api/skills/install/route.ts`）。
- 可见性开关 → `PATCH /api/skills`（`app/api/skills/route.ts`）。
- 检查更新 → `POST /api/skills/check`（`app/api/skills/check/route.ts`）。
- 更新 → `POST /api/skills/update`（`app/api/skills/update/route.ts`）。

PluginsConfig（`components/PluginsConfig.tsx`）：
- 加载列表 → `GET /api/plugins?cwd=`（`app/api/plugins/route.ts`）。
- install / remove / update / disable / enable → 同一个 `POST /api/plugins`，body `{action, source, scope, cwd}`。
- 刷新 → 重新 `GET /api/plugins`。
- Reload session → `POST /api/agent/[id]`，body `{type:"reload"}`（`lib/agent-client.ts`）。

当前 pi-web 后端行为：
- Skills 列表由 `DefaultResourceLoader` + `annotateSkillsWithInstallInfo` 生成（`lib/skills-service.ts`、`lib/skill-lock.ts`）；扫描 `~/.pi/agent/skills`、`.pi/skills`、`~/.agents/skills` 及 settings 显式路径。
- Skill 安装/更新走 `npx skills add ... -y --agent pi [-g]`（`app/api/skills/install/route.ts:39-45`、`lib/skill-updates.ts buildSkillUpdateArgs`），不是直接调 skills.sh HTTP。
- Plugins 走 pi SDK `DefaultPackageManager` + `SettingsManager`（`app/api/plugins/route.ts:205-220,300-370`；SDK 源码 `packages/coding-agent/src/core/package-manager.ts`、`core/settings-manager.ts`）。

### 1.4 skills.sh 外部 API（已用真实请求核对）

- 搜索：`GET {SKILLS_API_URL}/api/search?q=<q>&limit=<n>`，返回 `{"skills":[{"id","skillId","name","installs","source"}],...}`；pi-web 用它作为首选，失败时回退 `npx skills find`。
- 下载/版本快照：`GET {SKILLS_API_URL}/api/download/{owner}/{repo}/{skill}`，返回 `{"files":[{"path","contents"}],"hash"}`；pi-web 的项目 skill 更新检查直接用该端点比对 `computedHash`（`lib/skill-updates.ts:195-210`）。
- 全局 skill 更新检查走 GitHub trees API：`GET https://api.github.com/repos/{owner}/{repo}/git/trees/{ref}?recursive=1`，失败时临时 `git init/fetch/rev-parse` 求 folder tree SHA（`lib/skill-updates.ts:90-180`）。
- 结论：search/check/install 内容下载都可以被纯 Go `net/http` 直连；但 pi-web 现在的 install/update 依赖 `npx skills add` 写锁文件，Go 需要复刻锁文件格式（见 1.5）。

### 1.5 状态/锁文件归属

| 状态 | 所有者 | 默认位置 | 谁写 |
|---|---|---|---|
| pigo 包锁 | pigo | `$PIGO_HOME/packages.json`（默认 `~/.pigo`） | pigo CLI 独占；pi-web-go 只读 |
| pigo 插件/prompt/theme | pigo | `$PIGO_HOME/{plugins,prompts,themes}` | pigo CLI（或手工放置） |
| pigo skill 文件 | pigo + pi-web 共用发现 | `PIGO_SKILLS_DIR`/`~/.agents/skills` | pigo CLI 可写；pi-web-go 可扫描/改 frontmatter |
| pi settings | pi/pi-web | `~/.pi/agent/settings.json`、`<cwd>/.pi/settings.json`（`PI_CODING_AGENT_DIR` 可覆盖） | pi SDK/pi-web；pigo 不读 |
| pi 包目录 | pi/pi-web | `~/.pi/agent/{npm,git}`、`<cwd>/.pi/{npm,git}` | npm/git + pi SDK |
| skills.sh 锁 | skills CLI/pi-web | `~/.agents/.skill-lock.json`（或 `XDG_STATE_HOME/skills/`）、`<cwd>/skills-lock.json` | `npx skills add`；pi-web-go 直接实现时自行写 |

锁文件示例字段（`lib/skill-lock.ts` 与测试）：
- global：`skills.<name> = {source, sourceType:"github", skillPath:"skills/<name>/SKILL.md", skillFolderHash, ref?}`，version 3。
- project：`skills.<name> = {source, sourceType:"github", skillPath, computedHash, ref?}`，version 1。

## 2. 委托矩阵

策略缩写：G=Go-native，P=pigo CLI，S=skills.sh 直连，U=unsupported-with-doc。

| # | 操作（UI → route） | 当前 pi-web | 策略 | 命令/API | 输出解析与实现要点 | 状态归属 | Windows/风险 |
|---|---|---|---|---|---|---|---|
| 1 | 技能列表加载（`GET /api/skills`） | `DefaultResourceLoader` + skill-lock 注解 | G（+P 可选用 `pigo list`） | 扫描目录；可选 `pigo list` | 扫 `~/.pi/agent/skills`、`.pi/skills`、`~/.agents/skills`、settings 显式路径和 package 资源；解析 SKILL.md frontmatter；用两个 lock 注解 install 信息；pigo lock 的 npm skill 无 skills.sh install 元数据 | 只读 pigo/pi 状态 | 统一 filepath；`PIGO_SKILLS_DIR` 与 `PI_CODING_AGENT_DIR` 都要尊重 |
| 2 | 技能搜索（`POST /api/skills/search`） | skills.sh API，失败回退 `npx skills find` | S | `GET /api/search?q=&limit=` | JSON 解析 `skills[]`，映射 `source@name`、installs、url；纯 Go 不回退 npx | 无 | 无 |
| 3 | 技能安装（AddSkillPanel → `POST /api/skills/install`） | `npx skills add <pkg> -y --agent pi [-g]` | S + G；npm skill 可 P（仅 global） | `GET /api/download/{owner}/{repo}/{skill}`；P: `pigo install npm:<ref>` | 写 `<agentDir>/skills/<name>` 或 `<cwd>/.pi/skills/<name>`；写 `.skill-lock.json`/`skills-lock.json`；`pigo install` 后读 packages.json | skills.sh 锁 + 目录；pigo 锁仅 npm 路径 | 下载内容用 filepath 落盘；锁字段与 `npx skills add` 需做兼容测试 |
| 4 | 技能可见性开关（Toggle → `PATCH /api/skills`） | 手术式改 `disable-model-invocation` | G | 无 | 保留原 YAML 格式，只增删该 key；写前做 file-root/trust 校验；CRLF 友好 | skill 文件 | Windows 行尾；原子写 |
| 5 | 技能更新检查（Check → `POST /api/skills/check`） | GitHub trees API + git fallback；project 用 skills.sh download hash | G（+S） | GitHub trees API；`GET /api/download/...`；可选 `git` | global 比 `skillFolderHash`，project 比 `computedHash`；支持 `GITHUB_TOKEN` | 只读 | GitHub 限流；无 git 时 project 检查仍可走 skills.sh |
| 6 | 技能更新（Update → `POST /api/skills/update`） | `npx skills add`（buildSkillUpdateArgs） | G + S；pigo npm skill 可 P | `GET /api/download/...` 或 GitHub；P: `pigo update <name>` | 重装目标目录并更新锁 hash；复刻 `source/folder#ref --skill <name>` 语义 | skills.sh 锁；pigo 锁仅 npm 路径 | 无 npx 依赖；锁 schema 兼容是重点 |
| 7 | 插件/包列表加载（`GET /api/plugins`） | `DefaultPackageManager.resolve` + settings | G（+P） | 读 `settings.json` 与 `$PIGO_HOME/packages.json`；可选 `pigo list` | 合并 pi settings 包（npm/git/local、scope、filtered、disabled）和 pigo lock 包；统计 extension/skill/prompt/theme 资源；`pigo list` 只用于展示/回退 | 只读 pigo/pi 状态 | 两套状态天然分离，需按 source 去重 |
| 8 | 插件安装（AddPluginPanel → `POST /api/plugins` install） | `installAndPersist`（npm/git/local，global/project） | P（pigo-owned npm）+ G | P: `pigo install npm:<ref>`；G: npm/git 命令 | pigo 委托仅限 global npm 包；project、git、local 和 settings 持久化走 Go-native；Windows 上 extension 分发失败 | pigo 锁（委托）；settings.json（Go 路径） | Windows 的 pigo extension 安装不支持；混合类型包可能部分落盘且不写锁 |
| 9 | 插件移除（Remove → `POST /api/plugins` remove） | `removeAndPersist` | P + G | P: `pigo uninstall <name>`；G: 删 npm/git 目录 + settings entry | 先查 packages.json 判断是否 pigo 管理；成功后读锁确认 | pigo 锁；settings.json | 不要用 `pigo uninstall` 删 pi settings 管理的包 |
| 10 | 插件更新（Update → `POST /api/plugins` update） | `update()` | P + G | P: `pigo update <name>`；G: `npm view/install`、`git fetch/reset` | 永远传包名，避免触发 pigo 二进制自更新；pigo lock 用 name 查 | pigo 锁；settings 包 | Windows 同上 |
| 11 | 插件禁用（Toggle off → `POST /api/plugins` disable） | settings entry 置空数组 | G（pi settings 包）/ U（pigo 包） | 无 CLI | pigo 没有 disabled 概念、无 CLI/API；pi settings 包可直接改 settings.json | settings.json | pigo 包需文档化降级 |
| 12 | 插件启用（Toggle on → enable） | settings entry 还原 source | G / U | 无 CLI | 同上 | settings.json | 同上 |
| 13 | 列表刷新（Refresh → `GET /api/plugins`） | 重新 resolve | G | 同 #7 | 无状态变更 | 只读 | 无 |
| 14 | Reload session（PluginsConfig → `/api/agent/[id]` `{type:"reload"}`） | pi SDK 内 reload | U | pigo 无 reload 命令；`pigo serve/acp` 启动时固定插件与 slash registry | 需要 pi-web-go 托管 pigo 进程并重启，或文档化“新建会话生效” | pigo 进程 | 热发现缺失是最大集成风险 |

## 3. 推荐委托策略

1. pigo CLI 只负责 pigo 自己拥有的 npm 包生命周期：`install/list/uninstall/update`；成功后以 `$PIGO_HOME/packages.json` 为结构化事实源，CLI 文本只做错误展示。
2. skills.sh 直连承担搜索、下载、hash 检查；install/update 的锁文件写入由 Go 实现，目标是兼容 `npx skills add` 的 global/project lock schema。
3. Go-native 承担：settings.json 读写、资源扫描与 frontmatter 解析、disable/enable、project/git/local 包管理、GitHub 更新检查、文件路径与 trust 校验。
4. unsupported-with-doc：pigo 包 disable/enable、pigo serve 热 reload、Windows pigo extension 安装、pigo project-scope 安装；UI 需显式显示原因而不是假成功。

## 4. 顶层委托风险

- 状态双轨：pigo 的 `packages.json` 与 pi 的 `settings.json`/skills.sh lock 互相不知道对方；同一个“插件”可能只在一侧可见，remove/update 走错所有权会留下孤儿目录或假成功。
- 热发现缺失：`pigo serve` 启动后新装的 plugin/skill/prompt 不会出现在 slash/命令面，必须重启 pigo 进程；当前没有 reload 命令。
- Windows 缺口：pigo extension 安装直接不支持；`pigo install` 对“extension+skill/prompt”混合包可能部分落盘但不写锁，后续无法用 lockfile 精确清理。
- CLI 输出不可结构化：`pigo list/install/uninstall/update` 只有人类文本，多包、ANSI、编码差异都容易破坏解析；应把 packages.json 当唯一结构化输出。
- skills.sh 安装协议：pi-web 现在用 `npx skills add` 写锁，Go 直连只能拿到文件与 hash；锁 schema、`skillFolderHash` vs `computedHash`、带 `ref` 的 update 语义需要一次性兼容实现与真实 CLI 对拍。
- 发现目录不一致：pigo 用 `$PIGO_HOME` + `~/.agents/skills`，pi-web 用 `~/.pi/agent` + `.pi`；pi-web-go 必须同时尊重两套 env 覆盖和默认值，不能只按一个 home 展开。

## 5. 主要资料来源

- pigo：`cmd/pigo/main.go`、`internal/cli/pkgcmd/pkgcmd.go`、`internal/pkgmgr/*`、`internal/runtime/skills.go`、`internal/runtime/slashcommand.go`、`internal/plugin/manager.go`、`cmd/pigo/serve.go`、`internal/httpapi/commands.go`。
- pi-web：`app/api/skills/*`、`app/api/plugins/route.ts`、`components/SkillsConfig.tsx`、`components/PluginsConfig.tsx`、`lib/skill-updates.ts`、`lib/skill-lock.ts`、`lib/skills-service.ts`、`lib/agent-client.ts`。
- pi SDK：`packages/coding-agent/src/config.ts`、`core/package-manager.ts`、`core/settings-manager.ts`、`core/resource-loader.ts`。
- skills.sh：2026-08-13 实测 `GET /api/search?q=commit&limit=2` 与 `GET /api/download/juliusbrussee/caveman/caveman-commit`。
