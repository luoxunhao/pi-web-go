# 05 文件访问安全与 files API

**Type:** task

## Question

按 F1 实现文件白名单：会话 cwd、项目根、`~/pi-cwd-*`、显式 roots，realpath 校验，Windows 路径大小写；覆盖 pi-web `/api/files` 的 read/list/upload/download/watch/meta/preview 行为。

**Blocked by:** 01 pi-web API 契约矩阵

## Status

resolved

## Answer

已实现 `internal/files`：F1 白名单（realpath + Windows 大小写）、`/api/files` 的 list/read/download/meta/preview/watch/upload/upload-check，以及纯 Go docx preview。测试与 `go build` 通过。详见 `research/05-file-access-parity.md`。
