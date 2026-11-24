# CHNHCM-001 本地建仓交付记录

登记信息：[registry.json](../governance/registry.json)。适用范围：2026-10-30 的独立建仓交付及未结事项交接。本文保留该次交付范围；后续 M02 任务由 [008](008-next-steps.md) 组织，切片实施范围由预留的 010 承接，路由见[规格入口](../specs/README.md)。

## 1 建仓来源与选择

建仓采集时，CubeBox 的 origin/main 为 `2deaf9276086f68e07aaa0c2cf7f35025046a9ee`。它是交互原型；Go、PostgreSQL、pgx/v5、sqlc、goose，以及客户独立部署、Compose、本地升级、pgBackRest、Testcontainers 是技术方向，不是已验收的后台能力。本仓按该提交保存来源，没有以本机固定工作树的 `074fe044` 代替来源基线。

China HCM 来源为 FSCR 的 `d88c993d13d325b18943549f120c5f03e7864205`：20 个业务模块、6 个横向能力、17 个限界上下文、53 张目标图、131 项对标概念、154 组对象归属及 437 个术语。对象组不是聚合根数或功能完成数。术语数在 [009](../adr/009-mainland-organization-terms.md) 中按来源 JSON 核对：R25 冻结版及源完成报告中的 405 项不代表 d88 输入；后续本仓新增词条由修订链记录，不改写初始统计。

R25 冻结输入通过了术语及相邻合同的范围评审；采集的 d88 提交有 26 个输入文件与冻结版本不同，源记录为 `in_progress` / `matches=false`。建仓按用户当时决定采用这些命名增量，保留待整合验证状态。R25 结论不扩张到后续字节或产品验收。精确来源、初始摘要及采集日期见[基线](../reference/baselines.json)，本仓变更见[修订链](../reference/local-revisions.json)。

## 2 交付范围与证据

| 建仓交付物 | 范围及证据位置 |
| --- | --- |
| 独立 Git 历史、main 与六个固定开发分支 | 号段、工作树职责及登记见 [005](../governance/005-branches-and-documents.md) |
| React/MUI 导航壳和 Go 健康服务 | 构建、健康接口及正常导航结果见[建仓验证](../verification/bootstrap.md) |
| 自足的工程分层 | 当前目录及依赖方向见 [004](../architecture/004-repository-layout.md)；编译不读取相邻仓库 |
| 离线目标设计与来源包 | 图、模块/DDD 合同及来源摘要见[来源入口](../reference/README.md) |
| 三类优先资产保存 | 范围与摘要见 [inventory.json](../assets/inventory.json)，采用及重新验证按 [006](../assets/006-preservation-contract.md) |
| M02 选定与四份文档预留 | 切片边界见 [007](../architecture/007-m02-entry.md)，编写顺序及准备事项见 [008](008-next-steps.md) |

建仓报告记录了资料检查、前后端构建、健康端点和 M02 导航入口验证。它没有完整受测工作树指纹，本轮保留该历史限制；不把文档修订后的提交或新检查计数补写为旧受测对象。原交付未设置远程、推送、创建 PR 或部署。

## 3 未实施范围及交接

建仓未实施 M02 命令、聚合、表结构、正常业务页面及真实鉴权，其余上下文也未进入业务实现；生产 Compose、Worker、upgrader、厂商平台、pgBackRest 恢复和 Testcontainers 集成测试只保留职责。资产提取不表示全部页面行为已经保留或旧案例已在新仓通过。

| 未结事项 | 影响 | 后续承接位置与负责角色 |
| --- | --- | --- |
| 命名增量整合验证 | 可能影响 M02 术语及直接依赖的规格解释 | [008 准备事项](008-next-steps.md)，Organization lead 协同相关设计维护者界定受影响范围 |
| 来源行为与目标合同差异 | 旧页面或测试可能隐含未提炼的业务预期 | [006](../assets/006-preservation-contract.md) 的采用流程及 008，领域维护者逐项提炼 |
| 首批可办任务与验收范围 | 目录和目标图不能替代可交付用户任务 | 008 组织 010—013，Organization lead 负责推进 |
| 正式工具版本与运行组合 | 建仓工具可用不等于生产支持 | [002](../adr/002-technology-stack.md) 的版本锁定要求，由后续 013 承接 |

## 4 长期约定与保存

技术方向由 002、阶段完成口径由 [003](../adr/003-implementation-stages.md)、依赖方向由 004、协作与编号由 005、资产采用由 006、M02 边界由 007 维护。本文只记录建仓交付，不另行维护这些合同或持续产品进度。

登记中的执行关闭依据仍为原建仓范围的交付记录。本次仅完善交接和格式，未重新执行旧构建/浏览器验收。本文作为交付记录原位保留；以后评估归档时，按 [014](../governance/014-document-management.md) 核对有效规则、未结事项和全部入口的承接，再登记 archive 并迁移，不以 completed 自动移动文件。
