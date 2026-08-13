# 14 skills/plugins 委托矩阵

**Type:** research

## Question

按 K1 确定 pigo `install/list/uninstall/update` CLI 与 pi-web skills/plugins 页面的映射；skills.sh 搜索/安装的 Go 直连方式；无法映射项写入能力差距文档。

**Blocked by:** 01 pi-web API 契约矩阵, 02 pigo 缺口与降级矩阵

## Status

resolved

## Answer

14 项 skills/plugins 操作已映射：Go-native 承担目录扫描/frontmatter/settings/disable-enable，skills.sh 直连承担搜索/下载/hash，pigo CLI 只负责 pigo 自有 npm 包生命周期，无法映射项 unsupported-with-doc。主要风险是 packages.json 与 settings.json/skills.sh lock 双轨状态、pigo serve 无热 reload、Windows extension 安装不支持。详见 `research/14-skills-plugins-delegation.md`。
