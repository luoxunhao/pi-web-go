# 16 · Win7 迁移落地清单（Go 1.20 + Chrome 109）

> 状态：proposal（未开工）
> 背景：pi-web-go 已实现「运行期零 Node + 单 Go 可执行文件（E1 go:embed）」。要让整条链路（pi-web-go.exe + 内嵌前端 + pigo serve 子进程）在 Windows 7 上运行，剩余工作与依赖锁定版本如下。依据：pigo 仓库 `docs/reports/win7-compat.md`（第 8 节方案评估 + 8.7 嵌入 dist）。

## 目标形态

```
Win7 机器：pi-web-go.exe（内嵌前端，Go 1.20 构建）
              └─ Supervisor 托管 pigo.exe（Go 1.20 兼容分支构建）
浏览器：Chrome 109 / Firefox 115 ESR（Win7 上限）
```

## 1. 后端降 Go 1.20（pi-web-go 侧，工作量：小）

go.mod `go 1.27rc1` → `go 1.20`。**自身代码零 1.21+ 特性**（已 grep 验证：无 min/max/slices/for range n/errors.Join），无需改代码。

依赖锁定（当前版本均要求 go 1.23，实测 module cache）：

| 依赖 | 当前 | 需锁定 | 说明 |
|---|---|---|---|
| github.com/go-chi/chi/v5 | v5.3.1（go 1.23） | **v5.0.8**（2023-02，go 1.17） | API 兼容，`go mod tidy` 验证 |
| github.com/BurntSushi/toml | v1.6.0（go ≥1.21） | **v1.2.1**（2023-01，go 1.16） | API 兼容 |
| golang.org/x/*（如有传递依赖） | 2026 版本 | 2023 年初快照 | `go mod tidy` 后按 go.mod 报错逐个降 |

执行：
```bash
go mod edit -go=1.20
go get github.com/go-chi/chi/v5@v5.0.8 github.com/BurntSushi/toml@v1.2.1
go mod tidy && go build ./... && go test ./...
```

注意：Go 1.20 已 EOL（最后补丁 1.20.14，2024-02），安全修复停止——需在 README 声明。

## 2. 前端降级到 Chrome 109（构建期一次性，工作量：中）

实测 dist 产物（`frontend/dist/assets/index-BVD0gndo.css`，44KB）：

| CSS 特性 | 出现次数 | Chrome 109 支持 | 处理 |
|---|---|---|---|
| `oklch` | 1 | ❌（111+） | Tailwind 4 调色板 hex 化 |
| `color-mix` | 46 | ❌（111+） | Tailwind 4 调色板 hex 化 |
| `@property` | 35 | ✅（85+） | 无需处理 |
| `@layer` | 5 | ✅（99+） | 无需处理 |
| `:has` | 2 | ✅（105+） | 无需处理 |

JS 侧：Vite 6 默认 target `'modules'`（≈chrome87+），Chrome 109 基本兼容；建议显式 `build.target: 'chrome109'` 锁定。

执行：
```bash
# frontend/vite.config.ts
build: { outDir: "dist", target: "chrome109" }
# tailwind 调色板 hex 化（Tailwind 4 需覆盖 --color-* CSS 变量或构建后检查无 oklch/color-mix）
# 验证：构建后 grep -c "oklch\|color-mix" dist/assets/*.css 应为 0
```

## 3. pigo 主仓库 Go 1.20 兼容分支（工作量：大，前置依赖）

pi-web-go 是壳，壳内跑的 `pigo serve` 子进程在 Win7 上起不来（Go 1.21+ 运行时强制加载 Win10 专属 `bcryptprimitives.dll` → `runtime: bcryptprimitives.dll not found`）。详见 pigo 仓库 `docs/reports/win7-compat.md` §1/§8：

- 依赖整体锁定 2023 年初快照（x/sys ~v0.7、x/net ~v0.8、modernc sqlite ~v1.21-1.23、kin-openapi ~v0.117）
- openai-go（go 1.21）替换为自写 OpenAI 兼容客户端
- pigo 自身 <20 处改写（min/max/slices/for range n）
- 初始化调用 `crypto/x509.SetFallbackRoots()`（Go 1.20 特性）绕开 Win7 根证书库过期
- 该分支产物 `pigo.exe` 放入 pi-web-go 的 `bin/` 或 PATH

## 4. 验证清单（Win7 VM 冒烟）

```powershell
# 1. 前端产物无 oklch/color-mix（步骤 2 的构建后检查）
# 2. pi-web-go.exe 直接运行（frontend_dir 指向不存在目录，证明走 embed）
.\pi-web-go.exe --config .\bin\win7.toml
# 3. 浏览器 Chrome 109 打开 http://127.0.0.1:30141
#    - 首页渲染、颜色正常（无花屏/丢失样式）
#    - 新建会话 → 发 prompt → SSE 事件流 → 会话完成
#    - 文件浏览 / git status / 会话导出 冒烟
# 4. 确认 Supervisor 拉起的 pigo.exe 版本为 1.20 分支构建（--version）
# 5. 确认 PIGO_HOME 数据隔离生效（bin/ 下 sessions.db 可写）
```

## 5. 风险

| 风险 | 说明 |
|---|---|
| Go 1.20 / Node 18 / Win7 三层 EOL | 2024-02 / 2025-04 / 2020-01 后无安全修复；依赖锁在 2023 年初 |
| pigo 分支长期维护 | 模型 API 协议演进需自适配（reasoning、流式格式） |
| mermaid 体积 | dist 60+ chunks 与 Win7 无关，但首屏偏重，可后续懒加载优化 |
| 若不做 Win7 | 维持现状（最低 Win10 1809+），本文档作废 |

## 6. 决策建议

1. 若 Win7 仅为个别老旧设备：先做 **spike**（Go 1.20 + 最小链路），确认模型调用/会话/工具全通再正式维护分支；
2. pi-web-go 侧步骤 1-2 可在任意时间点独立完成（不依赖 pigo 分支），作为「最低系统要求 Win10」的降级准备也成立；
3. E1（go:embed）已落地，单文件交付与 Win7 目标无冲突。
