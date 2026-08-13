# 01 pi-web API 契约矩阵

**Type:** research

## Question

列出 pi-web 现有 40 个 API route，逐条映射到 pigo HTTP API 或 Go-native 实现；记录响应形状、必填参数（尤其 `directory`）、错误格式、认证差异，产出可供 M1/M2/M3 直接引用的契约矩阵文档。

**Blocked by:** none

## Status

resolved

## Answer

已映射 40 个 route 文件、53 个方法 handler：可直接代理 pigo 的约 12 组，Go-native 承担 files/git/worktrees/cwd/home/app-update/bash-output/export/auto-name/skills/plugins/OAuth/provider catalog；M1/M2/M3 归属与逐条契约已落在 `research/01-pi-web-api-contract-matrix.md`。Top 风险：session 树/运行态 schema、SSE wire 转换、错误 envelope 不兼容、models.json 与 pigo config 形状差异、pigo DELETE 不重挂子 session。
