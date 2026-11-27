# CHNHCM 工作约定

始终用中文沟通。本仓全新实施独立 China HCM 产品，采用模块化单体，Go 为唯一权威领域内核。保留 CubeBox 体验、业务及验证资产。

- 查共享规则与对象差异，见[按问题查规范](docs/governance/README.md)。
- 查具体业务、页面与技术合同，见[按模块查规格](docs/specs/README.md)。
- 查在办计划、验收记录或历史归档，见[文档首页](docs/index.md)。

## 开始工作

- 先核对当前用户指令、cwd、分支、HEAD、工作区、stash 及[登记表](docs/governance/registry.json)，保护已有修改和未跟踪文件。固定工作树不得切换分支，以本机检出路径为准。[005]
- 默认只检索现行规范。按需查计划及最新交接，区分文档状态、计划执行状态与实际验收结果。旧批次摘要不能重开已结事项或证明当前实现完成。
- **保留用户验收环境**。`wt-dev-d` 日常使用[统一入口 4486](docs/verification/315-department-quota-allocation.md#6-统一验收环境-4486)（API 8486 / DB 54486）。原 45300 及岗位、部门专项库停机保留。`m03-manual-d` 的 4380（API 8380 / DB 54370）、`m03-grade-d` 的 4381（API 8381 / DB 54371）及 `m03-grade-g2-d` 的 4382（API 8382 / DB 54372）仍由本树负责，用户验收结束或下次更新时复查，禁止按自动测试结束回收。4383、4388、[4390](docs/verification/136-actions-publication.md#8-原4390固定环境更新与交接)等持久环境同样保护。[运行接续](docs/verification/307-m03-job-foundation-implementation.md#10-用户授权的独立页面验收环境)

## 页面与交互：实施前必读

- 页面任务先读[012共享体验与表单](docs/specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-expression)、[列表与详情](docs/specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-list)、[动作](docs/specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-actions)、[历史](docs/specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-history)及该对象合同；按日对象同时核对[121接入矩阵](docs/specs/m02/121-dated-object-admission-matrix.md)的适用要求。
- 先确定已符合标准的参考页面和共用实现。复用须覆盖主题、顶栏、控制区、正文滚动、列表/详情、表单、日期维护、操作入口、来源解释、未保存保护及返回上下文；仅使用同名基础组件不代表继承标准。业务差异应在对象合同中说明。
- 直接呈现当前任务所需且有权操作的字段；更正带入原值并自动识别变化。可选修改不默认隐藏，不以展开、勾选或重复确认表达修改意图。辅助信息按需展开；内部机制、正常保证不成为常驻说明或额外步骤。
- 现阶段以桌面浏览器为交付范围。普通页面在 1440×900、1440×540 的 100% 视口对照同构建参考页，实测布局、滚动和键盘操作。关键流程必测原生 200%，400% 按风险或客户要求专项验证。手机窄屏适配仅在明确立项时验收。详见[共享验收规范](docs/specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-viewport-matrix)。
- 业务日期统一`YYYY-MM-DD`，记录时刻按北京时间显示`YYYY-MM-DD HH:mm:ss`并保留原始时刻；使用[026共享日期规范](docs/adr/026-date-display-and-input-format.md)。配置/主数据首次新建默认`1900-01-01`，允许显式修改；已有记录及后续维护不套用，见[202](docs/adr/202-master-data-default-effective-date.md)。

## 领域、数据与资产

- 金额、日期、状态转换、权限及业务不变量由Go判定；前端只负责布局、交互、输入辅助和API调用，不复制业务引擎或引入旧DemoState、transition。
- 按[上下文登记](docs/architecture/contexts.json)和[004目录边界](docs/architecture/004-repository-layout.md)开发：跨域只用公开合同/端口，组合根接线；领域不依赖存储、HTTP或别域聚合。X01仅共享小型值/协议；源域拥有事实与Outbox，消费者拥有采用与Inbox；X04/X05承载设施。M02不维护岗位层级、税费账户或授权决策。
- 同一事实一域负责，模型、存储、API和消息各一套。替换时默认撤下旧接口，例外见[025](docs/adr/025-greenfield-implementation-policy.md#single-current-implementation)。保留持久库、真实历史、不可变修订、准确来源、必要审计、原采用、幂等结果及原验证证据。仅明确可丢弃的数据可重建。
- SQL归表所属上下文，sqlc产物置于对应infra；goose全局补零编号，由upgrader统一迁移，API/Worker不各自迁移。已正式交付的迁移不得原地改写，开发基线整理按025执行。
- 按日有效期与独立修订分开，DB用`daterange [start,end)`；区分`businessAt/sourceVersion/adoptedRef`并按当前权限读取，不用同日序号。范围、默认来源、访问权限分开，不用SetID/TableSet；合同见[120](docs/specs/m02/120-dated-maintenance-technical-contract.md)。
- 体验、业务、验证资产登记到[inventory.json](docs/assets/inventory.json)，保留来源及采用证据。构建/运行/测试必须自足，不依赖相邻仓、外部node_modules、源仓服务或软链接；离线来源不进入构建，许可见[NOTICE.txt](NOTICE.txt)。
- 客户应用、DB及附件独立，厂商平台不集中持有员工/工资明细。不得在仓内（含忽略目录）保存真实凭据、令牌、私钥或敏感数据；外部组织原件只放在仓外私有目录，见[134规则](docs/guides/134-parallel-development-baseline.md)。`.local/`只放非敏感构建/检查输出。

## 验证与交付

- 非平凡业务工作先明确角色、正常菜单、前置、结果、异常、取消、更正及历史。按合同确定独立预期，按实际影响验证；不得以目录占位、mock或健康接口宣称业务完成。
- 按[003](docs/adr/003-implementation-stages.md)和[137](docs/governance/137-actions-and-validation.md)优先在本地发现、修复并复验问题，候选稳定后再推送，减少远程 CI 消耗。按影响运行 make 的 docs、boundaries、check、integration、smoke 目标，交付前 `git diff --check`。远程 CI 默认只跑改动所需检查。DB/API 可按需自动全量。本地按影响实测浏览器。公共仓库自动运行所需远程浏览器前置及分片。
- 如实记录基线、范围、首败、复验及未完成项。产品开发并验证通过后，交付前按[003 验收环境更新规则](docs/adr/003-implementation-stages.md#acceptance-environment-update)更新本工作树负责的手工验收环境，从用户实际地址复查业务并交接。常规更新由固定环境责任方执行。发布须用户授权，合并须最终 PR 的 quality、integration、browser 汇总成功。browser 核对所选远程分片及获准复用的证据，禁止强推。
- 收尾按[003资源规则](docs/adr/003-implementation-stages.md#development-environment-cleanup)和[017运行指南](docs/guides/017-local-bu-development.md#local-runtime-cleanup)执行：回收自有可丢弃临时资源；持久环境`local-down`只停机保留数据；保留运行须记录用途、责任和复查点。
- 本地开发账号为`admin`/`reader`，公开预设密码`admin1234567`；仅限本地开发。数据库随机凭据留仓外，不为登录读取数据库秘密；启动后报告实际profile和端口。

## 入口维护

- 根目录 Markdown 仅此一份，限 **8 KiB**。只列关键约束、任务入口和保护提示。批次进展归计划与验证记录。
- 长期规则归对应合同，此处引用。冲突须明确适用范围和替代关系，新文不自动覆盖旧约定。
- 新文档按[005]先登记，已用号不回收。分类、模板、稳定 ID、生命周期与证据遵循[014](docs/governance/014-document-management.md)。结构检查不能代替语义评审。
- 精简前原文、规则去向及暂存信息见[入口整理记录](docs/verification/005-agents-entry-simplification.md)。

[005]: docs/governance/005-branches-and-documents.md
