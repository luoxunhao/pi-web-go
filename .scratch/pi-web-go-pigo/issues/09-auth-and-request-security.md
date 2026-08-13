# 09 认证与请求安全

**Type:** task

## Question

按 H3/U1 实现统一认证：浏览器 Basic Auth（用户 `pi`，密码 `PI_WEB_GO_PASSWORD`），请求来源安全校验，Go server 内部携带 pigo 随机凭据转发；确认对 SSE/静态资源的认证行为。

**Blocked by:** 08 pigo 进程托管

## Status

resolved

## Answer

已实现统一 Basic Auth（用户 `pi`）、host 白名单（含 env 覆盖）、loopback CORS 和 SSE 认证测试；pigo 内部凭据由 supervisor 随机生成并走 Basic。详见 `research/09-auth-and-request-security.md`。
