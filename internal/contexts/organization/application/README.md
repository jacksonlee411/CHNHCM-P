# internal/contexts/organization/application

用例、权限复核、幂等、事务与下游所需端口。领域对象不处理 HTTP 或 SQL。service.go 通过事务/授权端口协调创建与结果恢复，由组合根注入真实适配器。

依据与入口：[004 应用层、端口与事务边界](../../../../docs/architecture/004-repository-layout.md)。
