# tests/integration

运行 `make integration`，使用 Go testing + Testcontainers PostgreSQL 验证 BU创建/查询/资料新增/普通更正的真实约束、事务、并发、当前授权、严格输入、来源历史及当前空库初始化。新增账号边界和重叠授权排序回归；容量样本用 `make load` 单独运行。每个测试自建并销毁隔离数据库，不借用开发库。

依据与入口：[002 数据库验证方向](../../docs/adr/002-technology-stack.md) · [003 阶段口径](../../docs/adr/003-implementation-stages.md) · [013 技术设计](../../docs/specs/m02/013-m02-technical-design.md) · [验证记录](../../docs/verification/README.md)。通过范围以实际报告为准，不能推导停用/重新启用、删除、改日或联合动作已实现。
