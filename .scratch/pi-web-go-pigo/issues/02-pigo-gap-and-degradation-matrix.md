# 02 pigo 缺口与降级矩阵

**Type:** research

## Question

盘点 pigo serve 当前缺口：图片附件（`prompt` 图片块被忽略）、`/status` 硬编码 idle、无 files HTTP API、无 package management HTTP API、无 OAuth 凭据链路。对每个缺口确定 pi-web-go 的降级行为，并给出 pigo 侧 ticket 清单。

**Blocked by:** 01 pi-web API 契约矩阵

## Status

resolved

## Answer

共 8 个缺口（图片 prompt、status 硬编码、无 files HTTP API、package/skills 仅 CLI、OAuth 仅 API key、additionalDirectories/mcpServers 丢弃、prompt mode 忽略、运行中配置不热重载）。M1 在采用降级后不被阻塞；M3 全量 parity 被图片、OAuth、package/skills HTTP 阻塞。详见 `research/02-pigo-gap-and-degradation-matrix.md`，pigo 侧 ticket 清单已列在文件内。
