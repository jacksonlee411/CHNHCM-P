# CHNHCM-116 R0授权与层级合同评审修订

登记信息：[registry.json](../governance/registry.json)的governance.verification_records。适用范围：用户采纳的未生效授权归属、有效父边/历史采用及启停路径具体方案，以及相应更正/恢复分类继承一致性。上游为[116](../archive/plans/116-m02-shared-dated-maintenance-roadmap.md)、[117](../adr/117-m02-dated-maintenance-and-department-boundary.md)及[118—121入口](../specs/README.md)。本轮只调整文档和独立预期，不启动产品实现。

## 1 验证对象与受测基线

日期2026-11-04；固定工作树`/home/vivobook/projects/CHNHCM-wt-dev-b`、分支wt-dev-b，基线`6e39d4ba2bc85b6ef6668a62e94f6cd1b5088a78`。开始时工作区/暂存区干净，stash为空；122资料已在该基线提交。本次文档与样本为基线外dirty增量；原输入及53份历史保护文件摘要见[before-inputs.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r0-review-fixes/before-inputs.json)。

原[116 R0报告](116-r0-contract-preparation.md)、原19个样本、原语义复核及122交付均保留原字节。原R0“准入已闭合”是当时结论，本次评审识别的缺口由现行117—121和本报告承接，不回写旧结果。

## 2 环境与前置条件

实际执行者为Codex主代理；用户已采纳具体规则并授权调整、检查和本地提交。使用本工作树Python标准库、Git及文档检查器。正常业务入口仍为设置→组织配置→部门；下述菜单/DB/API均为未来R4/R5目标，本轮未运行。

本轮不启动产品服务、数据库、浏览器、容器或本地profile；无任务运行端口、数据卷或服务需要回收，现有持久环境保持。资源收尾不适用，不以检查脚本的模拟输出作为实际容器清理证据。

## 3 合同调整与独立预期

| 评审问题 | 采纳规则及合同位置 | 必须保留的边界 |
| --- | --- | --- |
| 未生效对象无执行日BU可用于授权 | 118 BR030及120第6节：有资料取执行日BU，停用保留BU，首条前取当前来源固定首条；创建需read/history.read，所有读取/查回按当前授权 | 不提前产生业务事实，不新增可写授权归属，不保留创建者永久权限 |
| 停用历史引用与有效父边未区分 | 118 BR027/028及120第2节：有效父边只覆盖启用段，停用保留准确成员引用；恢复/删除/改日按最终候选重算及双向资格 | 未来恢复仍受守卫，零部分业务写入，无跨对象自动搬迁 |
| 停用路径与确切来源解释不清 | 118 BR029及119第3.3节：当前停用前路径按S前一日最新事实，显式来源按封存证据，即使仍等于头；路径依旧按当前权限裁剪 | 旧源及子部门头不被祖先变化重写，首条前无当日路径 |
| 新规则所需首条归属纠错/恢复与原因目录冲突 | 117第2.4节及118 BR026：新增CHANGE归属变更用专属原因；普通更正继承，首条保持CREATE，恢复保持REACTIVATE | 显式分类权独立，旧分类快照及原终态保持，目录切换仍留R1 |

[补充样本](data/116-r0-review-fixes/contract-fixtures.json)新增D20—D39共20例，连同原D01—D19为39例，映射见121。包含12份按动作/范围/字段明确的Grant profile、固定日期、受控Clock及每例before/command/expected；预期手工给定，不调用被测算法生成。所有actual_result仍为not_run。

重点复核了未来首条创建后读取/丢响应查回、缺读取权拒绝、首条改日/BU纠错、未来转入跨日及旧源不扩权、停用BU范围、停用后父部门转BU与未来恢复、恢复重选、删除/前移/后移停用、固定两条、当前/显式头来源路径分离、回溯更正、祖先失权、归属回退专权和结束父边中途失败。

## 4 检查过程与结果

| 检查 | 实际结果 | 证据与边界 |
| --- | --- | --- |
| 主代理合同语义复核 | 通过 | [semantic-review.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r0-review-fixes/semantic-review.json)，仅合同采纳范围 |
| 首次make docs | 失败 | [docs-first.log](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r0-review-fixes/docs-first.log)：73项脚本测试通过，但新报告登记缺少results |
| 补齐登记后make docs | 通过 | [复验输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r0-review-fixes/docs-after-metadata-fix.log)：73项脚本测试及仓库文档检查通过 |
| 固定分支及文档引用 | 通过 | [refs-first.log](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r0-review-fixes/refs-first.log)，`python3 -B scripts/check-repository.py --check-refs` |
| 独立预期及历史保护 | 通过 | [design-and-protection.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r0-review-fixes/design-and-protection.json)：39例编号唯一、新增20例未运行、授予/日期/引用及53份原材料保护 |
| 差异及新增文本空白检查 | 通过 | [diff-final.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r0-review-fixes/diff-final.json)，只检查文档差异及文本格式 |
| 新增20例部门业务验证 | 未运行 | [补充样本](data/116-r0-review-fixes/contract-fixtures.json)，不把静态预期检查记为产品通过 |

首次失败为报告登记缺项，补齐真实结果项后复验通过；保留失败输出，不以复验覆盖。定稿检查命令/退出码见[validation.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r0-review-fixes/validation.json)，受测文档及样本摘要见[reviewed-inputs.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r0-review-fixes/reviewed-inputs.json)。

## 5 修订复核与历史保护

主代理按评审触发场景核对117决定、118规则/验收、119页面、120技术和121矩阵的一致性；这不是独立评审或具名人工业务签收。新增样本的静态检查只证明日期、引用、授予与设计资产可核对，不证明未来算法、权限服务或数据库已实现。

本轮不更改产品源码、迁移、OpenAPI/客户端、原BU规格011—013/023、122合同、来源包或资产实现/验证状态。116保持in_progress，R1—R6保持未开始；原状态称谓产品验证只引用原报告，不计作本轮重跑。

## 6 未覆盖范围与后续事项

部门、Hierarchy、当前授权范围适配、路径API和页面均未实现或运行；20个新增案例及原19个部门案例全部not_run。未执行make check/build/integration/smoke，不声明全M02、容量、消费者或生产验收。

R1落实共用分类继承及强类型引用的资料/结构区别；R2/R4落实X03范围事实、创建/结果读取、独立存储、期间重算与共同守卫；R5执行正常菜单和路径/选择/撤权旅程。本轮合同与设计资产检查通过，可作为对应阶段实现输入；用户授权本地提交，不包含推送、PR、合并或部署。

## 7 证据保存

本次证据目录为[data/116-r0-review-fixes](data/116-r0-review-fixes/)。保存原输入/保护摘要、补充预期、语义复核及实际检查输出；最终受测摘要覆盖文档、登记和样本，不包含自身和会递归变化的检查输出。原R0及122报告、样本、原始输出保留不变。
