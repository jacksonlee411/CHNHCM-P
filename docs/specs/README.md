# 业务与功能规格

「人员 → 员工」按部门筛选各份任职，点行直达职务信息。0 号主岗及各正号兼岗分别列行，再聘在原 0 号追加新任期。「人员 → 人事事务 → 办理记录」按笔跨员工查询，两个入口都能定位同一原办理。[506 主兼任职与记录操作](m04/506-job-information-records.md) · [507 职务信息](m04/507-job-information-tab.md#job-tab-layout) · [507 办理记录列表与筛选](m04/507-job-information-tab.md#personnel-action-register)

普通地点更改从任职直接保存，已履职的派岗可从原任职补录。同日转正、晋升、调动分别保留原件，任职按业务日展示一份适用状态。[地点更正](m04/506-job-information-records.md#m04-ac-010) · [实况补录](m04/507-job-information-tab.md#actual-duty-entry) · [晋升、调动与撤销表单](m04/507-job-information-tab.md#same-day-job-form) · [403 事务与历史](m04/403-hire-technical-contract.md#same-day-action-write)

直接入职由人事从已登记的原人办理。组织用途与岗位政策先按期间准备，实际入职按模式签收。[401 业务规则与验收](m04/401-direct-hire-requirements.md) · [402 页面合同](m04/402-hire-interactions.md) · [403 技术合同](m04/403-hire-technical-contract.md) · [400 实施方案](../plans/400-hire-readiness.md)

[310职级／阶梯对象接入](m03/310-grade-ladder-contract.md)：同企业Job可用职级与Ladder顺序、完整日期引用、页面继承及18项新验收；[309方案](../plans/309-m03-grade-ladder.md)分G1／G2，产品未开始。

[308职务系列／职务页面标准](m03/308-job-foundation-contract.md#m03-page-inheritance)：逐页继承与业务差异、独立场景及14项新增验收；页面整改及28个尺寸角色单元已完成，后续正式发布与4380交接见[308实施记录](../verification/308-job-page-standard-implementation.md#m03-308-publication)。

法律实体基础维护：[207合同](m02/207-legal-entity-design.md)，按[206计划](../plans/206-legal-entity-implementation.md)实施首阶段；用途/绑定/M04后续接续。

2026-11-15 G4/G5集成接续：[性能及集成记录](../verification/205-g5-publication.md)登记20会话14组中的6组达标、8组超标，D08/D09及125其他容量P1保持；用户授权提交、同步main、push/PR及最终三项CI成功后合并。已同步e359f741并补齐地点302导航，3个DB/API根及26条浏览器同轮通过；PR #16最终三项CI已通过并合入d810757b；随后按用户要求更新4383，原38表与全部历史保持，147项API及双角色8组导航通过，见[运行更新](../verification/205-g5-publication.md#7-合并构建的4383运行更新)。

2026-11-15 G5实施、主代理复评及首批限定验收完成：[205记录](../verification/205-g5-implementation.md)持国家400次、区划3600次零错误与31条浏览器结果；G4完整证据按1392个不变产品文件复用。4383的m02-current-vanke-c已追加00013，原30表、132284份组织来源及目录/身份保持，147项API和双角色菜单/地点未保存草稿通过。200首批关闭；20会话地点D08、区划D09及125其他容量P1保持，自有临时资源已回收。

2026-11-15 G4地点实施及限定验收完成：方案已提交da164345，迁移00013、四地地址、完整共用维护、独立表组/API、固定选择及CSV已交付；主代理复评通过，251个DB/API根及354个浏览器用例分轮通过，8组1600次HTTP零错误、400份回执与全部原历史重启核验通过，详见[实施记录](../verification/204-g4-implementation.md)。G4限定验收已完成，自有临时环境已回收；20会话四组P95仍超500ms，按非阻断口径由200-D08接续。该批G5及运行更新尚未执行，当前结果见上条；原失败和125其他容量P1保持。

2026-11-15按用户要求核对最新main并制定[G4地点实施方案](../plans/204-g4-location-implementation.md)：完整共用维护、四地地址、逐动作采用、企业授权、固定选择与CSV及可靠提交；三轮主代理文档评审通过，3项P1、3项P2文档层关闭；状态同步见[记录](../verification/204-g4-plan-review.md)。G0/G1完成、G2功能交付、G3限定验收及后续4383运行已交付；原G2三条缩放已有G3后续通过证据并关联，国家容量待补。该次方案编制时G4/G5尚未实施；当前结果见上述实施记录，125其他容量P1保持。

[201国家地区行政区划与地点建设方案](m02/201-geography-and-location-design.md)持有下一批三类对象的四地范围、共性接入、来源与验收设计，已完成[本轮评审](../verification/200-geography-design-review.md)；分批任务见[200](../plans/200-geography-and-location-roadmap.md)。配置与主数据首次新建默认1900-01-01按用户决定由[202](../adr/202-master-data-default-effective-date.md)持有，G1已完成企业初始化及BU/部门默认日期适配，G2国家目录已交付；G3当前结果由[实施记录](../verification/203-g3-implementation.md)持有，G4地点实施及限定验收完成，见[204实施记录](../verification/204-g4-implementation.md)。

[文档入口](../index.md) · [文档管理](../governance/014-document-management.md) · [登记表](../governance/registry.json) · [模板](../templates/README.md)

本目录维护当前需求、功能交互和技术合同；阶段结果见实施报告，来源原件见reference，不以旧通过替代新规则验收。

当前109仅交付固定插入/删除启停对，AC020/022与通用编排案例退出；原历史继续读取，原T07/104—108结果保持时点。

## M01 人员登记、名录与个人资料

[505 个人资料九栏目维护合同](m01/505-personal-profile-contract.md)规定 T01 之后的资料集合、获权维护、保存及验收，交付次序见 [504 实施方案](../plans/504-personal-profile-extension.md)。

501–503 规定人员登记只按现用主要证件号码判断重复，以及同一张登记单再次办理时如何查回结果。[501 业务与验收](m01/501-person-registration-requirements.md) · [502 页面交互](m01/502-person-directory-interactions.md) · [503 技术合同](m01/503-person-registration-design.md) · [方案评审](../verification/500-registration-without-name-prompts.md) · [实施与试办](../verification/500-personnel-registration-trial.md)

## M03职务基础

313 规定后台部门总额、岗位定额、按日分配与调剂。315 先独立签收岗位核编和部门汇总，再签收部门额度与调剂。招聘预留、任职占编与调动有各自的后续验收条件。[313 编制管理合同](m03/313-position-capacity-contract.md) · [部门分配与减编](m03/313-position-capacity-contract.md#m03-br-040) · [315 后台实施计划](../plans/315-staffing-quota-administration.md) · [招聘与任职接入](m03/313-position-capacity-contract.md#m03-br-027)

[312岗位定义与岗位层级](m03/312-position-contract.md)：岗位三个必填引用、可空职级及全期间资格；岗位管理汇报按日维护并保留不可变发布版本。

[308 职类、职务系列与职务基础合同](m03/308-job-foundation-contract.md)持有首批业务、正常菜单、企业日期保护及页面标准。[独立职类](m03/308-job-foundation-contract.md#m03-job-function)按[314 路线图](../plans/314-m03-job-function-roadmap.md)实施，职务分类和编制管理首批分别签收。原首批已按[307 方案](../plans/307-m03-job-foundation.md)关闭，页面整改、发布与 4380 环境见[308 实施交接](../verification/308-job-page-standard-implementation.md#m03-308-publication)。

## M02当前规格

[123普通维护日期调整](../adr/123-current-maintenance-and-dated-query-boundary.md)：今日列表、未来首条可发现、按记录详情、跨日选择/导出与部门路径合同；实际检查按123报告记录。

当前已接通BU创建/查询、资料维护、启停、删除、改日与固定启停期间维护。101取消手动预览门槛；104将名称/日期/分类合并为更正记录；103/105按所选位置提供新增/插入及默认；107取消首条专用补全入口，首条更正后后续资料按需独立新增。长期规则均在下表维护，此前维护复核见108，当前启停专用维护及多轮检查由[109](../verification/109-lifecycle-pair-maintenance.md)记录。

| 文档 | 维护内容 |
| --- | --- |
| [010本版实施计划](../plans/010-m02-scope-and-plan.md) | 当前范围和办理路径；T07原批次及后续增量分别留证 |
| [011业务需求](m02/011-m02-business-requirements.md) | SCN001—018/027、BR001—023/034—035、AC001—033/041—044；122已接线，实际分层结果见实施报告；选择/导出按115实际分层证据读取 |
| [012功能交互](m02/012-m02-functional-and-interaction-design.md) | 列表详情、当前菜单/表单、自动辅助、直接保存及未保存保护 |
| [013技术合同](m02/013-m02-technical-design.md) | Go模型、唯一DTO/API、来源/结果、授权、事务及测试分工 |
| [023时间轴案例](m02/023-m02-timeline-test-cases.md) | 394个稳定编号：350现行、44退役；TL-367—386是122新增20例，本轮执行见122实施报告；123新增TL-387—394，BU按123报告分层记录，部门TL-394后端及菜单分别按R4/R5报告记录；其中TL-349—366由115按限定参数分层验证；110/111新增12例执行见[111 T03](../verification/111-community-t03.md)；参数及主要/补充层，未运行或退役不计通过 |

[113 Community决定](../adr/113-bu-list-community-adoption.md)及[111计划](../plans/111-bu-list-experience-upgrade.md)已完成T01/T02源码及页面接线，采用部门样板、摘要列和原生列设置，取消拖动/固定；[本轮报告](../verification/111-community-t01-t02.md)持有类型/构建、实际组件与合成HTTP检查。TL-323—334真实业务及七组视口已由[111 T03](../verification/111-community-t03.md)独立留证；后续单列与启停案例由各报告持有。新增[114决定](../adr/114-bu-query-selection-and-export.md)与[115计划](../plans/115-bu-query-selection-and-export.md)承接全结果选择/CSV，已接线并按[115报告](../verification/115-query-selection-implementation.md)分层验证，不提升全案例或全M02结论。

## M02部门及共用接入合同

以下合同已在116 R0冻结，产品能力留R1—R6实施；不替代011—013当前BU的实际范围和原证据。

| 文档 | 维护内容 |
| --- | --- |
| [117决定](../adr/117-m02-dated-maintenance-and-department-boundary.md) | 独立物理表、多对一BU归属、不跨BU共享、状态称谓、最小行政路径及新分类目录目标 |
| [118部门业务](m02/118-department-business-requirements.md) | SCN019—026、BR024—033、AC034—040；完整期间、父边与历史路径、权限和列表CSV |
| [119部门交互](m02/119-department-functional-design.md) | 正常菜单、字段/关系分组、能力、路径读取、历史和结果恢复 |
| [120技术接入](m02/120-dated-maintenance-technical-contract.md) | 强类型资料、Hierarchy唯一父边、路径证据、独立表/API、锁序、提交命名空间及容量样本 |
| [121验证矩阵](m02/121-dated-object-admission-matrix.md) | 共用规则与部门差异、BU保护基线；原39例手工预期保持；R4后端实际结果另存，R5菜单实际检查见[R5报告](../verification/116-r5-department-pages-and-acceptance.md) |

[116评审修订](../verification/116-r0-authorization-and-hierarchy-review-fixes.md)补齐未来首条/停用部门的授权归属、有效父边与历史采用、停用前路径及来源证据口径，相关更正/恢复继续继承合法分类。117—121为当前合同，原R0报告保留当时仅文档的范围；后续产品执行按R4/R5独立报告读取。

## 实施与验证去向

- [131部门查询减负计划](../plans/131-department-query-reduction.md)的[T00全部冻结](../verification/131-t00-design-freeze.md)完成：C01—C06落118—121/125，模型5/read-rule4、X03计划、完整捕获、期间依据、API消费者与同输入基线固定；13项DB及5项授权/期间原型通过。T01普通/路径列表功能与工作量验收已完成，见[列表实施](../verification/131-t01-list-implementation.md)和[合同复核](../verification/131-t01-review-fix.md)，原[基础阶段](../verification/131-t01-foundation.md)保持；T02[选择与导出](../verification/131-t02-selection-export.md)已完成V07—V10/V15，最终smoke272项及完整万科CSV通过，开发模式模块加载失败、固定构建尝试中断、稳定环境复验、分包与预算调整单列。[T03父候选](../verification/131-t03-parent-candidates.md)V11—V14及主代理评审验证完成，分包DB/API516项、五专项另验、smoke274项及完整万科候选/历史保护通过。固定4390本任务不切换，T04/T05及125容量P1保持，原证据不改写。

- [122实施计划](../plans/122-bu-operation-summary-and-change-log.md)已获批准并接通记录摘要、身份快照及分组日志，实际检查与资源结果见[122实施报告](../verification/122-operation-summary-implementation.md)。116后续阶段及部门不在本批范围。当前权限缓存、焦点与S/P/F删除补验见[122评审修复](../verification/122-operation-summary-review-fixes.md)。

- [122在123后的复核](../verification/122-post123-contract-review.md)：明确删除原件摘要与删除证据修订的区别、未来首条和空态，补充直达及受限权限参数；合同阶段20例未运行；本轮执行另见122实施报告，394/350/44总计不变。

- [116共用按日维护路线图](../plans/116-m02-shared-dated-maintenance-roadmap.md)承接BU/部门两对象的后续实现复用；物理表组及提交结果各自独立。R0/R1已完成，R2事务/恢复提炼已完成并复核通过，R3前端流程与BU接入已完成，R3-09 P2已修复并通过定向复核，R4部门后端按[本次报告](../verification/116-r4-department-backend.md)独立记录；R5部门页面与双对象限定验收已完成并复核通过，状态读权及根部门创建评审修复见[首轮修复](../verification/116-r5-review-fixes.md)，后续关系候选及日志筛选授权复修见[本轮记录](../verification/116-r5-authorization-followup.md)；R6容量未开始，现有BU结果或来源部门样板不计部门实现。决定与执行分别见[117](../adr/117-m02-dated-maintenance-and-department-boundary.md)和[116 R0报告](../verification/116-r0-contract-preparation.md)。

- [104统一更正](../verification/104-unified-record-correction.md)、[105位置新增及覆盖补全](../verification/105-positioned-addition-and-coverage.md)、[107首条通用维护](../verification/107-first-record-maintenance.md)分别记录对应增量的实际范围。
- [T07综合验收](../verification/010-t07-acceptance.md)保留原257案例基线；T04/T05/T06/T08及第一批报告由[验证索引](../verification/README.md)导航。真实消费者及生产能力继续后置，不声明全M02完成。
- [020](../adr/020-effective-record-delete-and-redate.md)、[021](../adr/021-bu-lifecycle-and-record-maintenance.md)、[022](../adr/022-joint-timeline-revision.md)的有效领域约束保持；已被101/104/107替代的交互在各决策内注明。024/025分别持有验证收敛和全新实施边界。

## 历史编写记录

[008编写计划](../plans/008-next-steps.md)及011—013末章保留起草过程。原文档验证记录只证明当时的文档检查；其中“未实现”不作为当前运行状态。各定稿报告不回写，当前已执行及未执行参数按对应后续报告读取。

116 R1共用Go内核、BU适配及117目录已完成，执行与复验边界见[R1报告](../verification/116-r1-kernel-and-directory.md)。R2已完成并复核通过，分批结果见[R2记录](../verification/116-r2-transaction-and-recovery.md)；R3前端流程与BU接入已完成，R3-09 P2已修复并通过定向复核，R4部门后端当前结果见本次实施报告；原39例设计样本保持not_run，实际DB/API另存映射，R5菜单实际结果进入本轮报告；R6容量仍not_run。

R3及R3-09修复已完成；原两批实施与复核见[实施记录](../verification/116-r3-frontend-workflow.md)；单轮187项Chromium及21组视口通过，部门真实接入仍后置。

2026-11-05 R3-09评审修复：已采纳退出失败后新草稿保护缺失的P2意见，移除跨动作持续放行。修复前反例重现，修复后12+7项定向Chromium通过，check及前端构建通过；原187项、DB及21组视口保留修复前证据，本轮未重跑全套。实际范围见[修复记录](../verification/116-r3-review-fixes.md)。

[116 R4设计](../verification/116-r4-contract-review.md)：120补齐精确表/键/证据、完整R5端口、两阶段候选及Access/锁计划；121映射原39例和16组技术参数，含122摘要分权。仅合同设计，产品全部not_run。

[116 R4关系版本简化](../verification/116-r4-relation-simplification.md)保留单部门关系历史及共同守卫；[后续评审修订](../verification/116-r4-review-fixes.md)补齐拟分类查询、授权及安全诊断，121使用当前39例及24组参数。原报告和映射保留，当前合同三项发现已修订，原设计时点产品not_run；当前后端实施结果见下列R4报告。

[116 R4实施及分批复核](../verification/116-r4-department-backend.md)：独立表组、全标准动作、路径/摘要/日志、选择导出和共同守卫已接线；当前39例及24组技术参数的后端结果、首次失败和R5接口清单单独登记，原设计报告保持。

2026-11-06 R5完成六轮评审与修复，关闭1项P1及5项P2（含修复回归和布局验证缺口）；最后两轮32项、18项分别通过，无新增主要问题。完整DB/API的外层异常和单项耗时失败、随后3次定向通过分别保留，不称单轮全绿。实际结果、当前输入和资源收尾见[多轮报告](../verification/116-r5-multiround-review.md)，R6仍未开始。

[G3必要行政区划接入方案](../plans/203-g3-geo-area-admission.md)已制定，细化[201](m02/201-geography-and-location-design.md)的固定结构、分地区发布、分页候选与准确采用合同；该次仅方案及代码同步；当前G3接续见[实施记录](../verification/203-g3-implementation.md)。

[G3专项评审](../verification/203-g3-plan-review.md)已关闭2项P1、3项P2，201合同及203实施方案在该次文档范围采纳；不替代后续[实施验收](../verification/203-g3-implementation.md)。
