# 15 里程碑验收与 E2E

**Type:** task

## Question

为 M1/M2/M3 定义可验收清单：Windows 浏览器端到端验证、Go handler/SSE/安全测试、前端 typecheck/test、打包形态（D1 PATH 依赖 vs 是否捆绑 pigo 二进制）。

**Blocked by:** 08 pigo 进程托管, 10 Vite 前端迁移, 11 工程能力 Go 实现, 12 export HTML 纯 Go 渲染

## Status

resolved

## Answer

验收清单、部署文档、配置示例与 Makefile 已落地；Go 测试/构建、前端 typecheck/build、真实启动冒烟与 headless Chrome DOM 渲染全部通过（AppShell 渲染无未捕获错误，截图 `e2e-home.png`）。M3 剩余 OAuth/skills 真实安装依赖 pigo 侧能力，已在 research 02/13/14 记录。详见 `research/15-milestone-e2e-acceptance.md`。
