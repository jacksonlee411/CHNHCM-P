# packages/api-client

从唯一 OpenAPI 3.1 合同用 openapi-typescript 生成类型，openapi-fetch 执行请求；index.ts 仅作错误/会话适配，不持有业务规则。make generate 重建，不引入手写业务 DTO 或旧 DemoState。

依据与入口：[当前 API 契约](../../api/openapi.yaml) · [004 API 与生成边界](../../docs/architecture/004-repository-layout.md) · [013 生成工具预留入口](../../docs/specs/README.md)。
