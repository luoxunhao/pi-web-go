# 13 OAuth 可行性研究

**Type:** research

## Question

研究 pi-web 现有 OAuth/device-code/manual-code provider 流程，确认 pigo 能否存储并使用 OAuth token；给出纯 Go 实现方案，或在不支持时记录为明确能力差距。

**Blocked by:** 06 模型配置代理

## Status

resolved

## Answer

pi-web 的 OAuth/device-code/manual-code 流程可纯 Go 重写；pigo 当前只能静态 `api_key`，OpenRouter 现在可行，xAI 需 pigo refresh 改动，Anthropic/OpenAI Codex/GitHub Copilot/Kimi/Radius 记录为 gap。M1/M2 保持 API-key only，OpenRouter 作为首个 OAuth 试点。详见 `research/13-oauth-feasibility-research.md`。
