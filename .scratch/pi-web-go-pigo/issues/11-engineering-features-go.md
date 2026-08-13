# 11 工程能力 Go 实现

**Type:** task

## Question

确定 git status/diff、worktree、file-index、cwd browse/validate、home、bash-output、context/state/auto-name、app-update 在 Go 中的实现方式和 Windows 路径处理；哪些映射 pigo，哪些 Go 自实现。

**Blocked by:** 01 pi-web API 契约矩阵, 05 文件访问安全与 files API

## Status

resolved

## Answer

已实现 session CRUD/context/state/auto-name、home/cwd/git/worktrees/file-index/bash-output/app-update 的 Go 路由。详见 `research/11-engineering-features-go.md`。
