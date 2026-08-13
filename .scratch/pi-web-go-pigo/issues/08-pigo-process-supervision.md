# 08 pigo 进程托管

**Type:** task

## Question

按 P2/D1/S1 实现 pigo 子进程托管：PATH 发现 + 配置覆盖、启动参数、随机内部密码、健康检查、崩溃重启、退出清理、端口冲突处理。

**Blocked by:** 01 pi-web API 契约矩阵

## Status

resolved

## Answer

已实现 `internal/pigo/supervisor.go` 与 `cmd/server/main.go`：P2 托管、D1 PATH+配置覆盖、S1 随机密码、健康检查、崩溃重启、端口冲突报错、优雅退出。详见 `research/08-pigo-process-supervision.md`。
