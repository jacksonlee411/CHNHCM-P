# CHNHCM-124 按日维护对象接入及容量验证指南

2026-11-15当前接入状态：国家及区划已分别由G2/G3完成真实接入；Location按[204方案](../archive/2026/plans/204-g4-location-implementation.md)第2.2节填写共用规则适用性，专属合同归201，产品实施及限定验收完成，见[204实施记录](../verification/204-g4-implementation.md)。地理路线的规模及20会话500ms目标按[200第4.3节](../plans/200-geography-and-location-roadmap.md#43-验证规模与计时口径)，不将本指南旧BU/部门容量矩阵直接复制为地点门槛。

2026-11-13新对象准入须核对[202默认日期决定](../adr/202-master-data-default-effective-date.md)：配置与主数据首次新建默认1900-01-01，明确输入、原记录及后续操作日期规则保持。当前BU/部门的产品适配及分批回归已由[200 G1报告](../verification/200-g1-default-date.md)持有；原交付证据不改写，未来对象仍须随自身切片验证。

2026-11-14用户明确纠正单对象10000条生效记录的不合理验收场景，正式采用单个BU／部门各20条记录，并保留模型5（m02-compact-5／read-segment-4／query-seal /3，迁移00001—00008）。10000条专用保存P95／分页及其驱动的模型6改造退出当前任务，不再列为待修复容量项；原测量和失败仍保留为历史证据。20条是验收场景，不是产品硬上限，不缩减企业对象总量、完整来源历史或并发规模；原500ms目标及125其他容量P1保持。 本条覆盖下文旧阶段的单对象超长时间线要求，历史数字不回填；见[采纳记录](../verification/135-model5-and-20-record-adoption.md)。

登记信息：[registry.json](../governance/registry.json)。适用范围：Organization中BU和部门已经交付的共用维护机制。上游：[116](../plans/116-m02-shared-dated-maintenance-roadmap.md)、[120](../specs/m02/120-dated-maintenance-technical-contract.md)、[121](../specs/m02/121-dated-object-admission-matrix.md)。

## 1 任务与结果

维护者用本指南核对新对象是否适用现有按日内核、定位真实扩展点、建立独立预期并完成接入验证。BU和部门是原交付示例；国家目录的第三个实际接入及限定能力映射见[200 G2报告](../verification/200-g2-country-catalog.md)。create、pair及selection按对象能力可选，关联区使用当前裁剪的展示资料；指南不把Organization内可复用代码升级为跨域运行平台。

领域规则仍由011、117、118和120拥有，验证参数由023和121拥有。下面的清单引用这些定义，不重新定义同日规则、启停规则或授权。

## 2 接入前核对

先填写[最小接入模板](../templates/dated-object-admission.md)，明确角色、正常菜单、创建前提、当前维护、异常恢复、更正、删除及旧来源。页面部分必须逐类填明[标准继承、业务差异及独立验收](../templates/dated-object-admission.md#page-inheritance)，不能只列组件名。以下每项都要指向本对象代码与真实结果；文件存在、类型通过、另一个对象通过均不能代替该对象验收。

| 核对项 | BU的真实接线 | 部门的真实接线 | 退出要求 |
| --- | --- | --- | --- |
| 稳定身份及类型资料 | [BU Policy](../../internal/contexts/organization/domain/bu_data.go) | [Department Policy](../../internal/contexts/organization/domain/department/data.go) | 编码范围、Clone、EqualData、EqualRevision及缺省/null独立预期；字段不是字符串类型别名的聚合 |
| 当前规则唯一实现 | [dated内核](../../internal/contexts/organization/domain/dated/policy.go)及[BU适配](../../internal/contexts/organization/domain/timeline.go) | Department Kernel使用同一dated内核 | 不复制排序、首条、继承、改日和固定两条；不得恢复任意编排 |
| 可靠提交与恢复 | [共用submission端口](../../internal/contexts/organization/application/submission/transaction.go) | [部门应用端口](../../internal/contexts/organization/application/department/ports.go) | 确认提交才发布终态，原主体/原身份查回；类型命名空间隔离 |
| 独立物理事实 | [BU查询](../../db/queries/organization/queries.sql)、00002/00004 | [部门查询](../../db/queries/organization/department.sql)、00005 | 身份、来源、修订、成员、日志、结果、Outbox独立；FK、封口、只增历史均实测 |
| 动作和字段授权 | [BU应用授权](../../internal/contexts/organization/application/buauth) | [部门授权适配](../../internal/contexts/organization/application/department/authorization.go) | X03当前裁决；显式要求及最终差异均检查；读取与写入、当前与历史分别核验 |
| 真实关系与竞争 | BU反向部门资格 | [部门关系内核](../../internal/contexts/organization/domain/department/hierarchy.go)、[共同守卫](../../internal/contexts/organization/infra/postgres/department_unit.go) | 原新范围及空集合竞争；两次依赖发现与同一预算；不锁全企业来代替关系合同 |
| 当前路径及确切来源 | BU无行政路径，仍有确切来源 | [路径与封存证据](../../internal/contexts/organization/domain/department/path.go) | 当前祖先变化不改子源，显式来源读取原证据；当前引用失权不泄露旧标签 |
| 类型API与结果 | [BU适配器](../../apps/web/src/features/organization/businessUnitAdapter.ts) | [部门适配器](../../apps/web/src/features/organization/departmentAdapter.ts) | OpenAPI/生成类型一致；不能断言转换另一对象DTO或混用路由/恢复空间 |
| 页面与字段发现 | [共用页面](../../apps/web/src/features/organization/DatedPages.tsx)、[共用办理](../../apps/web/src/features/organization/TimelineEditor.tsx) | [部门关系控件](../../apps/web/src/features/organization/DepartmentRelations.tsx)及同一办理 | 直接呈现获权输入，服务端给能力/日期；自动辅助不抢焦点；提交错误准确定位 |
| 选择与导出 | BU四列及自己的查询合同 | 部门八列、路径依赖及当前读权 | 翻页不丢选择，集合变化重选，返回高亮独立；CSV整笔成功 |
| 菜单与分层结果 | [两对象旅程驱动](../../tests/e2e/dated-object-support.ts) | [部门旅程](../../tests/e2e/department-journeys.spec.ts) | Go、真实DB/API、Chromium各有自己的结果；手工预期不调用候选算法计算 |
| 运行与资产 | [017运行指南](017-local-bu-development.md) | 同一运行指南和独占profile | 本仓自足；登记体验/业务/验证资产；保留旧证据，回收自有临时资源 |

## 3 实施与复核顺序

### 3.1 从规则适用性到类型端口

1. 对照121逐项决定适用或不适用。改变同日序列、状态机、审批、采用关系的对象不能以配置开关默认加入现有内核。
2. 定义对象自己的资料和补丁。实现dated.Policy的Clone、Normalize、Apply、EmptyPatch、EqualData、EqualRevision及目录方法。Normalize只处理新输入，不能重归一化旧快照；准确引用与业务值分别比较。
3. 应用层先授权显式要求，再构造候选、解析真实关系、定稿及检查实际影响；保存只接收已授权的最终Write。固定SQL和对象仓储接共用submission/transaction端口，使用自己的结果编解码与公开事件。
4. 对接前端对象、字段、API、列表选择和结果视图。前端展示Go返回的能力和错误，保留缺省/null语义及真实recordId；类型或字段关系不适用时显式拒绝，不能放入通用对象包吞掉差异。

### 3.2 最小测试层次

| 层次 | 可执行例子 | 必须独立断言的结果 |
| --- | --- | --- |
| Go资料与规则 | [candidate_test](../../internal/contexts/organization/domain/dated/candidate_test.go)、[impact_test](../../internal/contexts/organization/domain/dated/impact_test.go) | 输入/旧快照不变；纯资料、纯引用和两者同时变化；首条、位置、继承、固定两条 |
| 应用/事务 | [提交测试](../../internal/contexts/organization/application/submission/transaction_test.go)、[恢复测试](../../internal/contexts/organization/application/submission/recovery_test.go) | 提交确认、拒绝、无变化、未知、迟到响应和当前读权 |
| 真实DB/API | [部门合同](../../tests/integration/department_contract_test.go)、[完整性](../../tests/integration/department_integrity_test.go)、[共同守卫](../../tests/integration/department_guard_test.go) | 39例及24组参数分别映射；独立表、精确历史、关系竞争和取消零部分事实 |
| 组件及类型 | [前端类型反例](../../apps/web/test-fixtures/department-types.ts)、[投影机制](../../scripts/test_dated_projection.mjs) | 路由/字段/结果类型隔离、失权清理、会话归属及草稿保留 |
| 正常菜单 | [双对象验收](../../tests/e2e/department-acceptance.spec.ts)、[引用撤权](../../tests/e2e/department-reference-projection.spec.ts)、[布局](../../tests/e2e/department-layout.spec.ts) | 真实任务路径、直接保存、原结果、上下文返回及已加载样本的窄屏/真实缩放 |

规则和关系发生变化时两个真实对象都执行对应回归。交接执行make check、build、integration及smoke；生成变化时generate并比较，最后docs及diff。分批复验保留每批输入及原失败，不能把多个局部结果合成一次全量通过。

### 3.3 容量验证

容量规模和目标保持120第7节：2000BU、20000部门，每部门20当前成员和50来源，另验深度100、扇出2000、1000来源、137条CSV及20会话。列表/当前读300ms，确切源/普通/固定保存500ms p95；统计使用完整HTTP响应和Go单调时间，保存前读取准备单列。每种缓存条件和负载分开给样本数、错误率、p50/p95/p99，CSV/路径另记资源，不从单次读数推导p95。

当前模型使用[合成容量驱动](../../tests/capacity/storage126)与[万科全量验证驱动](../../tests/capacity/vanke127)。旧R6宽表驱动已随128删除，历史报告中的旧实现按报告登记的基线提交追溯。合成背景样本扩展不代表API创建吞吐，也不能计为原请求重放验收；保存、恢复、关系资格及取消测量使用新的真实HTTP提交。已有持久数据不进入复制或清理目标；临时候选须独立登记归属并在验证后回收。

执行前确认profile归属、容量、磁盘和内存；证据文件使用新路径。禁止清主机全局缓存，冷测只作用于停止后的自有临时数据库文件。运行步骤及最终数据以[本批报告](../verification/116-r6-standards-and-capacity.md)中的实际命令和结果为准，本轮容量实测已交付但未达到全部目标，R6-P01/P02仍为P1；指南可用不等于容量验收通过。

125整改的工具与实际状态见[实施记录](../verification/125-implementation.md)。物理冻结前必须同时检查WSL来宾及其真实宿主盘余量，冻结一次后通过改名保留旧版结果库并消费未写入副本作为候选，不重复建立第三份全库。冷测使用[r6-api.py](../../scripts/r6-api.py)重启固定摘要的测试二进制，不重编译当前工作区替换旧版基线；仍只操作本工作树明确临时的R6 profile。完整负载前先运行oracle-read-check，以真实详情响应验证独立判定器的身份/当前记录/确切源合同。125尚未关闭容量P1，按用户要求在本轮修复提交后暂停。

## 4 失败与恢复

| 现象 | 核对及处理 | 记录边界 |
| --- | --- | --- |
| 已保存但响应未知 | 保留类型/主体/提交号及原内容，按原身份resolve或重试 | 不能换号重发或把503当已拒绝 |
| 权限/关系发生变化 | 用当前授权重读；先清理失权字段与引用，再显示可读草稿 | 旧来源、原成功和旧可见标签均不授权当前读取 |
| 容量超预算或取消 | 核对Go截止时间、DB等待、事务/连接回收和零部分事实；保存首次失败 | 不放宽资格、权限或阈值，不用重试覆盖错误率 |
| 样本准备不完整 | 查数量、身份格式、来源/记录/路径/事件一致性；只重建创建时已明确可丢弃的环境 | 准备失败不记为产品吞吐失败；不改已有持久库 |
| 运行收尾 | 等待驱动结束，destroy自有临时profile并核对容器/网络/卷/配置/监听端口 | local-down仍只停机保留持久数据 |

## 5 跨域适用边界

| 下一对象候选 | 可评估复用部分 | 不能直接继承的语义 | 准入条件及潜在包位置 |
| --- | --- | --- | --- |
| M03职务（Job）基础资料，优先评估候选 | 只在日级资料确实同形时考虑候选、期间比较及恢复机制 | 职位/编制预留、任职占用、组织资格与审批不是部门规则；岗位层级不归M02 | 先由M03合同明确稳定身份、引用和采用；仍在staffing拥有聚合/表/API，两个域证明确需共享后再评估小型日期算法包，不直接导入organization领域包 |
| M04人员任职 | 表单会话、未明结果、当前读权和准确来源的机制 | 同日多动作、任职身份、统计日状态、审批及采用不是单日唯一启停 | 对照106逐项验证同日语义；默认不接dated.Kernel，不将其推广到X01 |
| 其他按日参考对象 | 同日唯一、首条/更正/历史确切引用可能适用 | 是否有状态、关系、字段冻结和消费者采用必须由对象明确 | 先完成接入模板和第二个真实域对象；不得只因日期字段相同建立通用EAV、跨域事务或共享业务表 |

原BU/部门两对象复用成立，后续CountryRegion/GeoArea已分别完成G2/G3限定接入；Location已按204完成实施及限定验收；G5最终联合出口与国家容量仍待接续。跨域采用仍须独立设计与真实验证。X01保持小型值与协议，X04/X05保持设施；本次不创建跨域内核、审批平台或外域消费者。具体容量及交接限制以R6报告为准。

2026-11-15 G5接续：地点及两类目录的首批综合验收、容量测量和固定4383运行交付由[205](../verification/205-g5-implementation.md)完成；产品1392文件与G4一致，复用与本轮执行分开登记。D08/D09非阻断性能项和125其他容量P1保持。

## 6 法律实体接入实例

2026-11-15本轮[206](../plans/206-legal-entity-implementation.md)按本指南形成[207接入合同](../specs/m02/207-legal-entity-design.md)，采用M02内现有按日内核并建立独立类型、表/API、授权与恢复空间。国家采用只依赖窄国家端口，不套用地点四地/区划规则；登记资料只表达基础声明，后续用途/组织绑定与真实M04消费者独立推进。

该对象独立容量为1000背景主体加20个20记录主体，单/20会话各四组100次真实HTTP，并重启核对全部原来源和保存回执；这一输入不替代原BU/部门容量规模。实际验证、首错及206-D01—03等非阻断延迟由[实施记录](../verification/206-legal-entity-implementation.md)持有，D08/D09及125其他容量P1保持。临时环境限定本任务创建的资源，既有持久环境不自动升级。

<a id="page-standard-admission"></a>
## 7 页面标准接入及M03样本

页面合同须列出列表、详情、表单、日期维护、操作菜单、摘要／日志／来源、保存取消及返回定位各自的标准条款、参照页面、公共原语、对象接线和验收。样式与交互由[012](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-expression)、[026](../adr/026-date-display-and-input-format.md)及其明确采用的后续决定持有；本指南维护接入过程，模板维护表式，不另设一套页面标准。

有业务依据的差异须说明适用对象／字段、理由、仍应遵守的标准及验证场景。现有偏差另列整改项；新增字段、引用资格或独立限界上下文不自动豁免公共布局和操作方式。先写标准派生的独立预期，再映射代码及测试，禁止把当前DOM或当前候选结果当期望。业务验收按对象及获权角色取证，核对正常滚轮与键盘可达、保存事实、取消零写及未保存保护。布局沿用共享桌面视口。原生 200% 选择受影响关键流程的代表路径，原生 400% 只覆盖触发专项的路径。记录实际执行范围，应测未运行项记为 not_run。[012 视口与缩放](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-viewport-matrix)

2026-11-17的[308样本](../specs/m03/308-job-foundation-contract.md#m03-page-inheritance)给出11项页面继承、6项业务差异及7项当前偏差，新增[14项页面验收](../specs/m03/308-job-foundation-contract.md#m03-page-acceptance)和28个尺寸角色单元。M03的恢复资料、系列引用和固定选择语义仍由308拥有，不照搬BU业务；Organization中的页面流程目前只是实现参照，中性提炼须由真实M02／M03共同验证。当前仅文档收口，产品整改和自动门禁未实施，见[本轮记录](../verification/308-job-page-admission-contract.md)。

后续308整改、公共列筛选／日志提炼及正式发布已完成，现行参照基线见[发布接续](../verification/308-job-page-standard-implementation.md#m03-308-publication)，上段保留合同编制时点。第二批[310接入合同](../specs/m03/310-grade-ladder-contract.md)分别定义Grade、GradeLadder和Job集合扩展，列明11项页面继承、5项业务差异及独立验收；当前只完成文档，尚无第二批产品通过结果。
