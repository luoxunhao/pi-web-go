# 06 模型配置代理

**Type:** task

## Question

把 pi-web 的 `/api/models`、`/api/models-config/*`、API key 状态映射到 pigo `config`/`providers`/`discover`/`test` API；不直接读写 pigo `config.toml`；对齐 thinking levels 和默认模型语义。

**Blocked by:** 01 pi-web API 契约矩阵, 02 pigo 缺口与降级矩阵

## Status

resolved

## Answer

已实现 `internal/server/models.go`：`/api/models`、`/api/models-config`、discover/test、`/api/auth/api-key/*`、providers/all-providers 全部代理 pigo config/providers API，不读写 `config.toml`。详见 `research/06-models-config-proxy.md`。
