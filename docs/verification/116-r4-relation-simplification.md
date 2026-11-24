# CHNHCM-116 R4关系版本简化复核

登记信息：[registry.json](../governance/registry.json) `governance.verification_records`。
适用范围：用户采纳的取消客户端整树版本及全树成员快照；仅修订R4设计及验收映射。
上游依据：[116](../plans/116-m02-shared-dated-maintenance-roadmap.md)、[117](../adr/117-m02-dated-maintenance-and-department-boundary.md)、[118](../specs/m02/118-department-business-requirements.md)、[119](../specs/m02/119-department-functional-design.md)、[120](../specs/m02/120-dated-maintenance-technical-contract.md)、[121](../specs/m02/121-dated-object-admission-matrix.md)。

## 1 验证对象与受测基线

2026-11-05，CHNHCM-wt-dev-b固定分支wt-dev-b，HEAD为03078144666924896077471c290c25dff6c44c5d。开始时存在上一轮未提交R4设计、登记/导航及证据；本轮在其上修订，不清理工作区或修改索引。[本轮基线](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-relation-simplification/baseline.json)保存修改前4548个已跟踪及未忽略未跟踪文件的摘要、状态及索引摘要。

用户同意去掉过度设计，并询问整套组织结构发布的用途；这授权本轮简化设计，不代表批准R4产品实施、组织调整方案平台或Git提交。

## 2 环境与前置条件

主代理完成文档/合同复核，非独立评审或人工业务验收。本轮只使用Python文档工具和只读Git核对；没有启动数据库、产品进程、容器、浏览器或持久profile。临时编辑脚本和检查过程文件属于本轮，最终结论与非敏感证据保存到仓内；003运行环境启停不适用。

原[首轮设计报告](116-r4-contract-review.md)及其data/116-r4-design原件保持。原报告“未发现尚未关闭设计问题”只描述该轮自查；后续评审提出整树版本、分类查询上下文和授权错误顺序三项P2，不能继续用原结论代表当前合同已全部通过。

## 3 本轮修改与验证结果

| 对象 | 修改及复核口径 | 实际结果 |
| --- | --- | --- |
| 117/118/119 | 明确Hierarchy仍唯一持有父边；来源按单部门，客户端只承担本部门版本，不接受无关树变化造成的冲突 | 本轮限定范围语义复核通过，产品not_run |
| 120存储/候选 | 单部门关系身份、来源、边修订、成员及事件；跨BU仍一个关系新源；原成员/停用采用与路径证据保留；无全树来源/根表 | 精确合同已同步，未运行DDL/DB |
| 120类型/并发 | 移除整树请求、响应和终态字段；稳定BU范围GuardKey、锁后重读及完整资格校验保留 | 合同已同步，未运行API/锁测试 |
| 121与023 | 原39例/预期保持；原16组按新粒度映射，追加R4-V17—V20 | 当前映射共20组，产品全部not_run |
| 文档/保护/格式 | make docs、原件与产品输入保护、git diff --check及新增文件格式 | 首轮make docs通过73项脚本检查及239份Markdown；55项静态核对修正包装后通过，最终收尾另存 |

当前[验收映射](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-relation-simplification/acceptance-map.json)保留原映射摘要及全部业务预期引用；另列单部门关系准确性、隐藏无关部门变化、真实来源/资格冲突、跨BU单源及并发环/空集合反例。新GuardKey向量为独立Python标准库计算，未调用待实现产品代码，不冒充产品锁验证。

## 4 评审发现及处理边界

| 发现 | 本轮处理 | 状态 |
| --- | --- | --- |
| 整树版本可暴露隐藏节点变化，并扩大局部编辑冲突 | 去掉客户端整树版本、全树成员快照及树头；保留部门自己的关系来源和实际获权投影 | 设计修订完成，产品验证not_run |
| RelationQuery.context缺拟更正classification | 本轮未处理；120第9节和当前映射明确保留，不能凭此次类型清单修改判为解决 | open，R4开工前补齐 |
| 关系/来源提前失败与追加授权之间的错误优先级 | 本轮未处理；当前授权、原新范围和安全诊断要求保持，具体优先级须另行冻结 | open，R4开工前补齐 |

本轮取消整树来源不取消关系历史；不把parentId复制到Department，也不把“锁住树”的内部守卫升级成可发布业务版本。全树版本、关系历史和锁范围分别处理，不能因前端少一个参数就删除共同守卫。

首轮静态检查将oracle_expected_sha256误按整条案例计算，39项摘要比对失败；原文件和原映射完整相等，未修改任何业务预期。核实原约定只哈希expected后修正检查脚本，55项复验通过。[首次失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-relation-simplification/contract-check.json)与[复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-relation-simplification/contract-check-retest.json)及两版脚本均保留；这是检查包装错误，不是产品回归。

收尾首轮将已授权修改的verification/README导航也纳入“历史验证原件”集合，保护项误报失败；实际变化仅该导航，所有原报告和证据字节保持。保留[首轮收尾](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-relation-simplification/closing-check-first.json)，按报告/证据与导航的范围区别修正后重核；不扩大允许变更的12份文件集合。

## 5 未覆盖范围与限制

R4迁移、Go实现、OpenAPI/客户端及真实DB/API，R5菜单/视觉与R6容量全部未运行；未执行make check或integration。当前39例及20组参数都是待执行设计，不能算通过数。另两项前轮设计意见仍未关闭，不声明R4整体设计全部通过或产品可用。

“按编号发布、审批、采用整套组织结构”属于后续业务：明确调整范围/用途、候选版本、审批和生效，以及消费者固定采用。编号本身不自动更改任职、预算或权限；这些需要独立合同。本轮只解释用途，不增加此平台、发布表或消费流程。行业参考：[Oracle树及版本](https://docs.oracle.com/en/cloud/saas/human-resources/faucf/guidelines-for-managing-trees-and-tree-versions.html)、[部门树启用与引用](https://docs.oracle.com/en/cloud/saas/human-resources/fahbo/guidelines-for-loading-department-trees.html)，2026-11-05在线核对；示例中的人工审批和预算冻结为业务解释，不宣称Oracle具体模块全部提供。

## 6 结论与后续事项

用户采纳的简化已写入当前合同；9项限定范围合同复核、73项脚本检查/239份Markdown以及修正后的55项映射/保护静态核对通过，最终文件与格式核对见收尾证据。4536项既有输入保持，修改限12份合同/导航/登记文件；索引、HEAD和stash保持。后续先处理另两项设计意见，再决定R4产品实施；实际产品逐包遵循116/121。原R0—R3及本轮修改前的R4证据保持，未提交或推送。任务临时目录中的脚本及基线副本在仓内证据齐全后回收；没有操作既有持久环境。

## 7 证据保存

[修改前基线](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-relation-simplification/baseline.json)、[当前验收映射](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-relation-simplification/acceptance-map.json)。[合同复核](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-relation-simplification/semantic-review.json)、[首次文档检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-relation-simplification/docs-validation.txt)、[首轮静态检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-relation-simplification/contract-check.json)、[修正复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-relation-simplification/contract-check-retest.json)、[首轮检查脚本](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-relation-simplification/contract-check-first-script.txt)、[修正检查脚本](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-relation-simplification/contract-check-script.txt)、[最终输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-relation-simplification/final-inputs.json)、[最终文档检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-relation-simplification/docs-close.txt)、[收尾核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-relation-simplification/closing-check.json)。原证据留在原目录，不用新结果覆盖旧结果。
