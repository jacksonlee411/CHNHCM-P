# db/queries/organization

M02 sqlc 查询源；只写本域拥有的表，按公开合同协作。创建/查询的稳定查询源在 queries.sql，根 sqlc.yaml 按上下文生成；事务编排与提交证据读取在本域 infra。

依据与入口：[004 SQL 所有权与生成规则](../../../docs/architecture/004-repository-layout.md) · [013 技术设计预留入口](../../../docs/specs/README.md)。
