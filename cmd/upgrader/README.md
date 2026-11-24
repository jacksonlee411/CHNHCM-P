# cmd/upgrader

当前提供受控迁移、企业/本地账号初始化、运行角色配置、账号停用/撤权及提交证据恢复。输入通过 stdin/env，凭据在仓外；API/Worker 不自动迁移。完整离线包、备份和版本切换仍待后续，操作见 [017](../../docs/guides/017-local-bu-development.md)。

依据与入口：[002 交付方向](../../docs/adr/002-technology-stack.md) · [004 迁移调度与依赖](../../docs/architecture/004-repository-layout.md)。
