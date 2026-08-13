# 06 模型配置代理

**Type:** task
**Status:** resolved
**Date:** 2026-08-13

## 结论

模型/provider 配置全部由 pigo 持有，pi-web-go 只做 HTTP 代理与响应形状转换，不直接读写 `config.toml`。

## 实现

- `internal/server/models.go` 提供：
  - `GET /api/models`：聚合 pigo `config` + `config/providers`，输出 pi-web `ModelsData` 形状。
  - `GET/PUT /api/models-config`：GET 合成 providers 视图；PUT 逐 provider 调 pigo `UpsertProvider`，`defaultModel` 调 `PATCH /api/v1/config`。
  - `POST /api/models-config/discover`：映射 pigo `config/providers/discover`，protocol 按 pi-web `api` 字段转换。
  - `POST /api/models-config/test`：映射 pigo `config/providers/test`，返回 `ok/latencyMs/responseText`。
  - `GET/POST/DELETE /api/auth/api-key/{provider}`：只透传 `apiKeyConfigured`，写 key 走 pigo `UpsertProvider`；不回显明文。
  - `GET /api/auth/providers`：OAuth 目录留空（issue 13 后置）。
  - `GET /api/auth/all-providers`：从 pigo providers 生成 API-key provider 列表。

## 验证

- `internal/server/models_test.go` 用 fake pigo 验证 `/api/models` 的 `modelList/defaultModel` 转换。
- `go test ./...`、`go build ./...` 通过。

## 边界

- `models-config/catalog`（models.dev）属 M3，未在本 ticket 实现。
- pigo 写配置后不热重载（issue 02 G8），pi-web-go 需在保存模型配置后触发 pigo 重启或等待 pigo 修复。
