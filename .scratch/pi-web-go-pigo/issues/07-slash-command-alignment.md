# 07 slash 命令对齐

**Type:** task

## Question

按 C1 从 pigo `GET /api/v1/commands` 动态生成命令目录，映射 `/command` 调用；处理 fork/clone/tree/name/compact 等；把 pi-web 自带但 pigo 缺失的命令记录到对齐文档。

**Blocked by:** 01 pi-web API 契约矩阵, 02 pigo 缺口与降级矩阵

## Status

resolved

## Answer

已实现 `internal/server/agent.go`：`get_commands` 动态来自 pigo，prompt/abort/set_model/thinking/name/compact/fork/clone/navigate_tree 均映射到 pigo HTTP API；pi-web 独有命令显式 unsupported 并记录。详见 `research/07-slash-command-alignment.md`。
