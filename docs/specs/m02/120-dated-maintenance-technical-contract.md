# CHNHCM-120 M02共用维护与部门技术接入合同

2026-11-16路径显示按[306](../../plans/306-department-path-simplification.md)接续：所有对外路径仅含真实部门，不拼接企业；删除路径内未生效/停用说明，状态与日期独立显示。内部按日解析、历史封存及字段授权保持。

2026-11-15企业按[304](../../plans/304-enterprise-maintenance.md)接入同一dated/submission内核，使用独立企业表组、DTO及实例授权。企业依赖读取只取得身份、有效期间及名称时间线，不读取业务时钟；BU/部门在原取时点以同一个业务日投影名称，写入仍在相关锁全部取得后取时。正反向引用共用企业行共享/排他锁，候选完整启用期间由Go核验。306接续决定：部门对外路径仅包含真实部门，企业名称只用于归属上下文；内部封存证据保持，投影过滤企业节点；列表/树/导航/选择在原语句快照捕获企业当前资料，Go取日投影，路径游标头帧仅传输一次。实际实施及验证见[304记录](../../verification/304-enterprise-maintenance.md)。
2026-11-15地点实际接入：Location复用本合同的日期候选、可靠提交、强类型适配及独立存储；专属全动作采用、地址权限及查询合同由[201](201-geography-and-location-design.md)持有，分批及验证由[204](../../plans/204-g4-location-implementation.md)持有。地点已接入上述共用合同；限定验收已完成，实际范围见[204实施记录](../../verification/204-g4-implementation.md)。部门Hierarchy与BU选择合同保持。

2026-11-13首次新建默认日期按用户确定的[202项目规则](../../adr/202-master-data-default-effective-date.md)统一为1900-01-01，由Go创建上下文提供defaultEffectiveDate；currentDate仍表示今日，明确输入、原日期、后续位置/启停规则及依赖资格保持。默认非法时准确拒绝并允许用户明确改日，不静默换今天或改写父对象。本轮合同已接入现有BU/部门，产品适配及受影响分批回归见[200 G1报告](../../verification/200-g1-default-date.md)，原实施报告不改写。

2026-11-14用户明确纠正单对象10000条生效记录的不合理验收场景，正式采用单个BU／部门各20条记录，并保留模型5（m02-compact-5／read-segment-4／query-seal /3，迁移00001—00008）。10000条专用保存P95／分页及其驱动的模型6改造退出当前任务，不再列为待修复容量项；原测量和失败仍保留为历史证据。20条是验收场景，不是产品硬上限，不缩减企业对象总量、完整来源历史或并发规模；原500ms目标及125其他容量P1保持。 当前工作区已恢复本节模型5合同；135候选机制仅作已退出实验，见[采纳记录](../../verification/135-model5-and-20-record-adoption.md)。

登记信息：[registry.json](../../governance/registry.json)。适用范围：116 R0冻结的共用内核、部门/Hierarchy接线、独立表组及提交恢复合同；产品实现分R1—R6交付。依据[117](../../adr/117-m02-dated-maintenance-and-department-boundary.md)、[118](118-department-business-requirements.md)、[119](119-department-functional-design.md)、[013现行BU](013-m02-technical-design.md)及[004分层](../../architecture/004-repository-layout.md)。

当前日期/记录/路径合同按[123](../../adr/123-current-maintenance-and-dated-query-boundary.md)调整；R4已接通部门及最小Hierarchy后端，实际分层结果见[R4报告](../../verification/116-r4-department-backend.md)，页面与正常菜单已按[R5报告](../../verification/116-r5-department-pages-and-acceptance.md)完成限定验收。

R1起始实施基线为4f4fc0e，继承[122评审修复](../../verification/122-operation-summary-review-fixes.md)与123已实现的BU行为：所选修订操作者摘要独立于日志读权；日志按当前历史/字段权投影；当次资料、删除前原件、固定窗口及失权清理保持；普通维护按所选记录。各批实际复核见[R1报告](../../verification/116-r1-kernel-and-directory.md)，部门后端与正常菜单的对应参数分别由R4/R5报告登记，原分批证据保持。

R4实施设计以0307814及当前未提交合同为基线。原[首轮记录](../../verification/116-r4-contract-review.md)、[关系版本简化](../../verification/116-r4-relation-simplification.md)保持；当前[评审修订](../../verification/116-r4-review-fixes.md)补齐引用查询拟分类、授权与诊断先后顺序。上轮三项问题均已落实设计修订，随后以5fa3275为产品实施基线；真实DB/API结果另见R4报告，原设计报告保持。

2026-11-08合同修订：[128](../../adr/128-authorization-evidence-removal.md)撤除第6节的永久授权过程存证机制与第7.1节READ_PROJECTION_SET完整档案，并解除业务终态对X03历史判定的依赖；按128第2.6节删除关联快捷跳转、canOpen、reference-link及其摘要/缓存/公开节点DTO依赖。删除实施与分层验证已完成，实际结果及容量未通过边界见[128实施记录](../../verification/128-implementation.md)。当前权限、范围事实、操作者身份和业务路径来源保护继续有效，原实现报告保持。

## 1 交付与依赖

Go唯一裁决业务不变量，前端仅布局、输入辅助及类型API调用。公共内核留在Organization域内，domain/public不依赖对方层或infra，不把完整规则放进X01。BU仓储及API继续独立；R1共用Go内核及117目录已接通并完成本批复核，分层结果及复验边界见R1报告。部门与Hierarchy已有独立持久化、类型API及组合根接线，未增加树平台API。

共用包为`organization/domain/dated`，提供强类型`Record[T]`、`Item[P]`、Candidate/Impact及纯计算；对象提供Normalize/Clone/Apply/EmptyPatch/EqualData/EqualRevision及原因目录，不提供可替换的排序、首条或生命周期算法。application提炼命令/预览/事务结果流程；infra实现对象专属仓储和共同事务辅助，bootstrap接线。public只暴露DTO/端口，不导入domain记录或pgx/sqlc类型。

## 2 对象、关系与候选

### 2.1 资料与关系引用

BU资料保持现有name/nameEn。DepartmentData含name、shortName、managementBuId及不可变行政成员采用引用；编码属于稳定身份。行政父边只归Hierarchy，部门修订不再存另一份可写parentId。公开parentDepartmentId是关系命令面，由应用服务委托Hierarchy；关系引用变化参与部门候选的数据/结构比较，即使名称不变也不是no_change。

同部门同日一个当前成员；统一更正仍指向准确记录/修订。停用继承前序全部资料及准确修订引用，重新启用保存独立资料快照。显式null、缺失、空值、声明字段权及NFC等价保持沿013；新字段不能被`map[string]any`吞掉或被名称专用比较遗漏。资料Clone不共享可变slice/map/指针。

R1落实的比较约定：`EqualData`比较业务值（含关系稳定身份），`EqualRevision`独立比较对象自有的修订结构（如准确采用版本）；完整修订相等由内核组合两者与日期、分类、状态和继承元数据。BU没有额外资料引用结构，故该结构比较恒等。对象引用结构变化形成`REFERENCE` facet；即使业务值同时变化也不能省略。新增CHANGE只有业务值与资料引用结构均相等才消除；引用版本单独变化仍产生修订与结构影响。字段缺失由`Item[P].Changes`空指针表示，EmptyPatch区分显式空补丁：CHANGE新增允许空差异并在完整日期/状态守卫后判定no_change，更正或显式修改的重新启用仍拒绝空补丁；新建才调用Normalize，不重写存储快照。

### 2.2 有效父边与启停重算

最小Hierarchy拥有部门行政父边的独立事实；关系聚合按(instance,enterprise,childDepartmentId)建立，完整来源只包含该部门自己的关系时间线，可跨过去/当前/未来BU期间。当前行政森林由各部门关系头在一致快照内组合，不存整BU的完整成员来源。BU/用途DEPARTMENT_ADMIN只确定内部并发守卫，根部门仍展示企业；不开放树平台写API、整树版本发布或客户端树版本参数。每个child在每个启用日恰有一个父部门或企业根，停用期间及首条前无有效父边；有效parent与child同企业/同BU。按全部部门/关系日期边界检验启用覆盖、唯一父和无环。创建开放启用部门时若BU或上级已安排未来永久停用，完整候选不能覆盖则拒绝，不自动缩短部门期间或改动对方安排。

启用修订的行政成员引用须在完整适用启用段指向同child的准确有效关系；停用修订继承最终前序的准确引用作为历史采用证据，该关系可以在停用日结束，不要求覆盖停用区间。引用用途由部门记录状态及读取模式确定，不增加用户可切换的生效标志。每个部门当前关系来源可保留该部门过去有效的边区间；其不可变旧关系来源/成员仍保留。历史引用不作为当前/未来活跃下级枚举的依据。

| 命令与最终状态 | 有效父边重算 | 资格及修订要求 |
| --- | --- | --- |
| 新建/关系变化，生效T | 从T建立新边，归属变化在T结束原边 | 全启用期间同BU、上级启用、无环 |
| 停用S | 原边在S结束，停用段无边 | 反查全部启用下级及未来安排，保留历史采用引用 |
| 恢复R | 从R建立新的有效边及准确成员引用 | 完整默认资料可获权修改；重新核验BU/父，旧引用不代替资格 |
| 删除停用 | 先过共用生命周期，再延展最终启用段所需父边 | 留下无前提REACTIVATE则拒绝；恢复出的每段关系均须合格 |
| 停用改日S到S' | 按新边界延长或缩短父边 | 后移查新增启用覆盖，前移查失去覆盖的下级；保持原相邻位置限制 |
| 固定插入/删除停用期间 | 同一候选一次重算两端与关系期间 | 无中间提交、无任意跨对象联动 |

Go先形成完整部门候选，再确定Hierarchy变化及最终成员引用，最后形成必要的直接/派生部门修订和停用继承引用；引用版本因区间变化而更新属于结构差异，不覆写后继独立资料或增加同日事件。关系及反向资格全部通过后才写入。BU/上级维护仅被实际启用期间的依赖约束，仍须覆盖当前停用子部门的未来恢复。仅有已结束的历史引用不阻止父部门后续转BU，也不得自动改动该子部门资料。

### 2.3 当前路径与来源证据

Hierarchy按部门维护关系来源与边修订，部门资料只采用本部门的确切关系成员。父部门改名不改关系来源；父部门改上级只形成该父部门及其关系的新源，不写后代事实。默认当前读取未携带sourceVersion：列表启用以服务端D为pathBusinessAt，未来首条以首日解析安排路径；停用时定位覆盖D的停用记录S，以S前一日为pathBusinessAt，页面统一标为部门路径，不追加停用说明。路径在同一SQL语句快照内按pathBusinessAt解析各部门及各自关系的最新来源，不沿D直接遍历已结束父边。停用后的祖先变化不带入此前路径，合法回溯更正可改变当前解释。首条前返回no_fact且无今日路径，可另返回首条安排路径，未生效由独立状态字段表达。解析和S前一日计算均在Go，不使用浏览器拼接或DB业务状态机。

每次部门成功写入同时封存该部门完整来源时间线的`path_evidence`：按相关日期边界分段，保存起止、路径语义、各节点的确切关系成员及有序祖先稳定身份/确切资料来源或修订引用；停用段另保存pathBusinessAt＝S前一日，不另维护可编辑的名称串。显式sourceVersion始终读取该来源证据并要求当前history.read，即使所选sourceVersion仍等于当前头；不能因相等退回当前路径或绕过历史权。默认当前路径可因祖先变化而更新，显式来源只读原件。停用改日重建新源的解析日及证据，旧源不改。

详情启用记录返回pathSegments，按所选记录范围与祖先/层级日期边界求交，段中含起止、路径及来源；当前模式读最新关联来源，显式来源裁切封存path_evidence。单段/多段均不新增用户日期输入。停用记录继续使用停用前路径，不按停用段遍历有效父边；权限受限不泄露分段数量或日期。

类型化读取返回pathMode（active、future_first、inactive_before_stop、no_fact、restricted）及独立pathVersion（准确来源依据留在领域历史结构中）；列表等单日投影可读时返回pathBusinessAt，详情启用记录的期间投影返回pathSegments（start、endExclusive、endInclusive和path，开放终点为null）；按128不返回逐节点来源引用或跳转节点数组。future_first只表示未来首条的安排路径，不改变no_fact的今日事实判断；sourceVersion参数是否显式存在决定当前/封存模式。路径还须同时可读departmentPath、effectiveDate和status，缺任一字段返回固定restricted投影；restricted不暴露解析日、节点数或受限来源；历史页面上下文自行说明所选来源，内部引用不作为常驻说明。列表/CSV使用当前模式，停用行输出停用前路径并保留停用状态。pathVersion及查询选择版本覆盖读取模式、解析日、实际授权投影和依赖变化，不冒充department head。只读证据不是第二套父边权威。

<a id="r4-candidate-pipeline"></a>
### 2.4 完整候选、关系计划与修订定稿

R4已将dated.Calculate提炼为同一内核的Prepare/Finalize流程，BU的Calculate也使用该流程；Department在Prepare之后完成关系Plan与准确引用绑定，再由Finalize完成消除、继承和影响。只读引用候选使用独立ReferenceProjection，允许原因尚未选定，不伪造业务原因，也不交给Persist。Department没有复制排序、首条、生命周期或继承算法。

| 顺序 | Go输入与产物 | 不变量 |
| --- | --- | --- |
| 1 授权后准备 | 锁后一次D、当前Department完整时间线及严格类型命令；先完成6.5的本对象读取/动作/声明字段及显式目标预授权，再校验本部门预期并由Prepare产生WorkingCandidate | 锁发现不直接返回来源/资格诊断；Prepare只核本部门身份/版本、日期、分类和生命周期，不调用外部BU/父资格；保留待判断CHANGE，后继独立资料不传播 |
| 2 关系比较及授权 | RelationIntent携带BU/父稳定身份；比较基准与完整WorkingCandidate，形成实际RelationDelta及原/新AffectedScope；立即逐项核验所需reassign/字段/目标引用 | 包括删除/改日等隐式回退；在Hierarchy.Plan前完成，不等完整关系资格通过才授权；只比较稳定身份/期间，不依赖目标是否启用或是否成环；缺权不调用关系资格检查 |
| 3 关系计划 | 在第2步范围授权通过后，Hierarchy.Plan对目标部门启用段按同BU/同父合并区间，得到完整关系候选及MemberBinding | 按全部边界检验BU/父启用、同BU、唯一父和环，反查下级含未来安排；失败按6.5裁剪；不形成其他部门关系新源；名称CHANGE不拆边，首条前/停用无有效边 |
| 4 绑定及定稿 | 把ExistingMemberRef或PlannedMemberRef绑定回完整候选；Finalize重算停用继承、消除无变化CHANGE并调用共用Compare | Existing引用仅因对应关系/期间变化才替换，不因其他部门关系推进重绑成员；Planned引用按确定关系键比较，不以随机UUID制造REFERENCE；单独引用变化形成结构差异 |
| 5 固定点检查 | 若消除CHANGE改变记录边界，用同一关系计划与Finalize再次核对 | 关系区间独立于同值记录切分；循环只能减少冗余CHANGE，不能新建事件或改业务值，次数上界为待消除记录数加一次核验；不能收敛则技术失败、零写入 |
| 6 最终授权复核及物化 | 从最终候选重新构造完整Requirements，复核动作/字段/原新范围后才分配来源/修订/成员ID，替换计划引用 | Finalize不得引入新的BU/父身份或扩大第2步已授权范围；违反即通用技术失败、零业务写入；no_change也不跳过声明字段授权，且不创建关系身份/来源/日志/Outbox |
| 7 证据与写入 | 以最终候选覆盖锁后事实形成路径证据；部门/本部门关系来源、成员、日志、事件及accepted结果同一保存点提交 | 完整证据包含复用旧修订的来源成员；自引用使用本次预分配身份及Go闭包验证，当前头/目录由延迟触发器核验；不以另一对象头代替准确采用 |

DepartmentPolicy的EqualData比较name/shortName/managementBuId和解析后的父稳定身份；EqualRevision比较准确成员采用或确定计划引用。该父身份为不可编辑的内存投影，不能成为第二份持久父边。Clone深复制引用结构。只更换引用版本不追加reassign；真正BU/父变化须按第6节授权。预览使用同一纯Go流程且不物化ID、不写关系或终态；保存不接受预览产物为资格证明。所有候选/关系/定稿循环检查同一context预算；WorkingCandidate与FinalCandidate为不同内部类型，Persist仅接受已授权定稿并完成真实引用绑定的Write，禁止绕过定稿直接持久化Prepare输出。

## 3 独立物理表及生成边界

当前采用用户批准的[126方案](../../plans/126-compact-history-storage-research.md)，与[013第3.1节](013-m02-technical-design.md#31-表与约束)共同持有紧凑存储合同。BU与部门仍分别拥有业务表、强类型候选和适配器；不引入通用实体表、EAV、整树版本或旧宽表兼容视图。逻辑来源仍完整，物理上共享重复值与结构。

<a id="r4-storage-layout"></a>
### 3.1 当前物理结构与准确引用

记D＝(instance_id,enterprise_id,department_id)。公开UUID、来源、记录、修订、关系成员及事件身份完整保留，块内用局部整数引用；业务日期字典为daterange [start,end)，时间保持微秒精度。代码及企业归属只在对象原生行维护，强类型时间线引用稳定身份。

| 表或类型 | 保存内容和责任 |
| --- | --- |
| department | D、编码/规范编码、首条、当前来源/序号/块号、当前投影根/局部引用/摘要；企业及编码唯一键；management_bus/parents为当前完整日期轴的依赖索引，随拥有者来源事务更新，不作为另一套业务归属事实 |
| department_history | 按实例＋部门＋块号索引；完整UUID池、来源/主体/提交目录、状态/操作/路径根及字典引用；同013的64来源/256KiB预算、连续追加、封口及头闭包保护 |
| department_dictionary、department_shape | 仅部门类型使用的不可变值字典和强类型内容根；实例/企业范围内摘要去重；引用者封套绑定全部归属与内容 |
| department_terminal | 非accepted终态的原生身份/目标/结果类别及强类型内容；完整身份主键；accepted与部门来源同根保存 |
| department_hierarchy_relation | D及关系头；无整树根；关系头变化必须属于本部门新来源事务并出现在该头块的完整UUID池中 |
| DepartmentState | 本来源完整资料时间线、准确关系来源/编号、边修订/成员引用及各记录原操作者摘要 |
| DepartmentOperation | 来源、原记录时间与身份快照、逐项前后修订/直接派生角色/修改面、删除墓碑、原请求及结果/关闭时刻、资料与关系事件身份、关系前驱及完整边影响 |
| DepartmentEvidence | 本源所有记录期间的冻结路径分段；逐节点名称、当时资料来源/记录/修订与准确关系成员/边引用，包含停用前采用日和未来首条解释 |

旧逐行source_member/path_segment/path_node/path_edge不再作为物理表存在；其全部业务事实由上述强类型来源表示。内部行号和事务物理标记不是新的业务身份。原案例验证转换器只放在容量工具中，产品不理解旧表或旧编码。

130在隔离空库的当前迁移中把department主键保持为(instance_id,department_id)，INCLUDE(enterprise_id,head_source_id)；关系头主键保持为(instance_id,enterprise_id,department_id)，INCLUDE(head_source_id)。两者分别承接原department_list_heads_cover及department_relation_heads_cover，撤除对应独立索引；所有原归属唯一键、FK目标、提交/依赖索引和读取段五类索引保持。覆盖列不参与唯一性判断，不能把index-only scan视为无需实测的保证。

### 3.2 保护映射、终态与事件

| 原约束目的 | 当前实现边界 |
| --- | --- |
| 同实例/企业/部门的来源归属与当前头 | 原生对象/块复合FK、企业FK、连续目录及延迟闭包；读取核对头、来源、首条及强类型对象身份 |
| 资料记录/修订/变化项准确性 | Go校验记录与修订唯一、前后修订号/成员、变化项与HistorySource一致、直接/派生及修改面闭包；活跃成员不得混入删除墓碑 |
| 日期、状态及继承 | 原Go完整候选验证，并在解码后核对连续范围、启停及准确继承；不再宣称块内具有逐行EXCLUDE或复合FK |
| 关系成员、BU与父归属及资格 | 保存前在共同守卫下取得准确依赖快照，由Go验证同企业/BU、有效边、父路径、期间覆盖及无环；依赖数组不能替代这些资格判定 |
| 历史路径采用事实 | Go验证每段覆盖、节点/边归属、准确记录/修订和停用采用日；编码封套及不可变根防止保存后错配。历史读取不访问今天的树重算 |
| 终态与来源/事件/X03决定 | 原完整提交锁、短标签预筛选后的完整UUID/主体核对、accepted与非accepted互斥；Go检查请求摘要、声明、原结果、事件及X03决定绑定；一次提交只形成一个来源及至多一个关系来源 |
| 不可改写历史 | 尾块只增后缀、原根与UUID/值前缀不可变；封口后禁止更新/删除；头/当前投影/关系头与所属新来源事务完成；来源保护不得靠可伪造客户端标识 |

强类型终态仍保留accepted/no_change/rejected/not_applied四种语义、部门request_format_version=1的规范摘要、字段声明、固定两条项、诊断和办理日；不是增加第二种HTTP请求格式。accepted/no_change保存一次执行采样；不存在目标的拒绝不得伪造事实引用。损坏详情读取保持当前权限优先，不把解码失败解释为not_applied。

DepartmentSourceChanged仍使用资料影响、结构变化及准确记录范围；HierarchyRelationSourceChanged仍保存逐边前后修订/BU/父/范围及cause部门来源。类型/schemaVersion由唯一强类型格式定义，事件ID、来源、相关提交及原内容保留；关系号独立递增。只有变化的本部门关系产生新源，不扩写其它部门或整树。消费者/生产投递继续后置，delivered_no不代表本批已经投递。

### 3.3 查询、分页与路径解码

当前列表加载所需当前日期轴的小投影，复用历史UUID值池，不解码全部50或1000份历史。部门依赖由management_bus/parents原生GIN筛选并精确核对；对象身份和编码使用B-tree；提交查回以16位标签GIN筛选后核对完整主体/提交身份，标签不授予权限。GIN关闭待处理列表，避免连续写入使每次精确查回扫描积累的待处理项；其空间全部计入整库预算。

确切sourceVersion先定位所属块，再解码该源状态、操作和必要路径根。变更日志按不可变来源号及固定分页上界解释原前后事实；隐藏来源和缺损来源沿已有权限及错误合同处理。操作者摘要读取成员修订的原形成来源，不能取当前头操作者替代。

编解码仅在各域infra；Go领域层仍拥有日期、资格、状态、影响及最终候选。sqlc只生成当前原生表查询；定型编解码器由Go生成并在make check校验一致。原宽表实现及原验证报告作为历史证据保留，不算当前模型兼容能力。实际故障/容量/浏览器结果由[126实施报告](../../verification/126-compact-history-implementation.md)记录。

<a id="q131-integrity-prototype"></a>
### 3.4 131查询完整性原型与接入约束

本节保留原T00—T03的存储诊断设计及历史证据。2026-11-11用户先采纳行政上级必要职责收敛，随后明确“同样收敛列表和全选”；当前查询分别遵循第3.9和3.10节。本节全企业、过滤前和无结果仍审计的要求不再属于用户查询，原诊断只在integration/query131测试构建中使用。

[131的T00原型](../../verification/131-department-query-integrity-prototype.md)只在自有空库及integration/query131构建中运行。当时采用可查询读取段时固定了以下完整性与捕获约束；本次[全部设计冻结](../../verification/131-t00-design-freeze.md)进一步完成第3.5—3.8、4.6及6.6节，实际公开查询、迁移和页面接入由T01—T03承担。[完成情况复核](../../verification/131-t00-completion-review.md)发现的UUID目录物理形状漏检已由[修复复核](../../verification/131-t00-integrity-fix.md)补齐，Q131-R01保持关闭，原失败证据保留。

派生列由同一Go来源事务生成，并绑定原读取段封套、真实来源/首条/成员/修订、期间、关系头、原字典/结构摘要及全部查询列。数据库仅执行存储字节封装、摘要比对及范围检索，不生成业务状态、日期、权限或父资格。预首条模式与首日生效模式由Go分段，不能修改真实首条日期来适配查询。新增封套不是授权票据，也不能代替来源写入前的Go完整候选校验。

UUID目录须为一维、起始下标1、非空且无NULL元素。封套绑定UUID值及顺序，不能据unnest后的封套相等推断原数组形状合法；物理形状同时由存储约束及过滤前检查保护，独立验证强制损坏时的查询拒绝。原型以CASE先判定维度再检查空元素，不能依赖AND求值顺序调用只支持一维的数组函数；NULL/空目录明确为false，不能借CHECK未知值通过。该约束不改变完整UUID、原封套、业务历史及Go领域职责。

过滤前完整性从可信实例/企业的原生部门身份出发，覆盖缺段、来源绑定、期间连续覆盖和被授权/业务筛选排除的段。原型用单条语句独立聚合全部当前段的完整性，并无条件返回检查结果；正常、零结果、空授权范围、越界页都执行该门槛。任何坏事实均由Go返回503且不输出部分count/items，不能以WHERE校验通过把坏行当成不匹配。读权按当前生效归属由X03决定；早期固定BU集合只用于完整性故障测试，最终计划原型消费第6.6节X03守卫内编译结果，不接收公开范围入参。没有读取入口权时先403，不执行或暴露完整性诊断。

同一语句固定count、本页及Go解码所需原始字典/结构字节，先分页再取本页材料，后续不重读可覆盖的current段。不可变字典仍可能在失去引用后回收，因此“记住旧字典ID”本身不能保证后续读取；使用已捕获字节或另经验证的存储持有协议，不能依赖未声明的永久保留。解码材料是内部事实，核验后移除，不能成为公开DTO。最终原型已覆盖祖先/BU闭包、并发转BU、换父、更名、同会话和撤权屏障；完整产品及容量另由T01—T04验证。

原型仍在PG扫描全范围当前段和相应字典/结构，Go只解码本页；准确总数与完整性成本不得隐藏。其单会话小样本结果不构成300ms容量验收，也不能替代万科与规定合成形状、20会话、长历史和两轮增长测量。

### 3.5 131模型、查询列与索引冻结

当前源码已接入m02-compact-5、read-segment-4、查询封套department-query-131/3的迁移/来源事务写入与模型准入，BU/部门读取规则、生成物及空库夹具同步。原[131 T01基础](../../verification/131-t01-foundation.md)保留阶段结果；正式普通/路径列表已接线，T01功能与工作量验收完成，分轮验证、构建差异及首失败见[实施记录](../../verification/131-t01-list-implementation.md)。[完成情况评审](../../verification/131-t01-completion-review.md)发现的合同收敛P2已由[修订复核](../../verification/131-t01-review-fix.md)关闭；T02完整专项已由[实施记录](../../verification/131-t02-selection-export.md)验证通过；T03父候选期间收敛及V11—V14已由[实施记录](../../verification/131-t03-parent-candidates.md)验证通过；T04/T05及125容量P1保持。本段原T01时固定4390为模型4/read-segment-3；后续已按独立运行记录更新为模型5，本批不操作固定环境。T00当时用模型4真实封套叠加测试列的结果保持；不为旧模型保留双读。原4390/4387/4388只接原匹配程序，禁止升级或用新程序接入。原历史编解码格式、完整UUID、来源与终态不因读取投影格式升级重写。

| 来源或新列 | 冻结含义与生成责任 |
| --- | --- |
| 原生object_id、source_id、slot、span、dictionary_id、shape_id、ids、envelope | 复用真实身份、来源和原读取封套；编码/first_record_id来自部门身份，不额外复制编码文本 |
| q_code_key、q_name、q_name_key、q_short_name、q_short_key | Go原名称与strings.ToLower检索键；简称保留NULL和空值语义，空检索键不伪造可读字段；不使用SQL lower/LIKE来替代Go语义 |
| q_bu、q_parent、q_record、q_revision、q_date、q_end、q_first_date | 实际适用记录、修订、BU/父身份、真实起止和首条日；完整UUID与开放终点保持 |
| q_state、q_path_date、q_relation | Go生成not_effective/active/inactive；active路径日为空表示请求D，未生效取真实首日，停用取停用日前一日；关系来源绑定原关系头 |
| q_owner_seal、q_dictionary_seal、q_shape_seal | Go生成原生身份/编码/首条/关系头绑定、字典和结构内容绑定；用于写入绑定及独立存储诊断，当前查询不再重复核对 |
| q_seal | Go生成/3查询封套，绑定instance/object/source、slot/rule/span、原封套、上述三项绑定、全部查询列及规范UUID目录的有序序列；NULL简称与空串区别编码 |
| q_stored_seal | PG原生GENERATED ALWAYS STORED只做同一/3字节框架与哈希；SQL修改任一输入必重新生成且不能直接赋值，独立存储诊断可与Go的q_seal比较；不推算日期、状态或权限 |

同一来源事务先通过Go完整候选验证，再生成全部段和查询列；任一步失败回滚来源/关系/终态。合并只允许原读取内容及全部查询事实等价的相邻段；真实首日导致状态/路径日变化时必须分段。查询封套绑定最终slot/span，合并后重新生成，不能复制旧段seal。原型故意保留更细分段以验证首日边界，T01整合合并算法时继续受V14保护。

T01完整合成形状首读在5.15秒返回503；原实现每次对424246段重算查询封套并多次物化宽行，独立完整性EXPLAIN耗时26.39秒。故按131第3.2节的条件修订为上述存储生成校验列与窄完整性扫描，初始/2基础报告保留。所有SQL INSERT/UPDATE均由PG重算存储校验，禁用普通触发器也不能绕过；字典/结构本身已有相同的生成摘要，因此被强制修改后仍与Go来源事务保存的绑定不符。上述全范围核对属于T01原合同；当前列表/选择按第3.10节读取必要材料并普通解码，不再扫描全部历史段证明绑定、连续期间或重算封口。行政上级候选按第3.9节直接消费Go生成的查询字段。现有写入格式和存储约束保持。

这里的存储信任边界是当前DDL及PG执行的SQL写入；特权DDL拆除生成属性或物理页静默篡改不属于该证明。生成列不可直接覆盖、禁用触发器后的修改、空范围/无结果/越界页损坏必须实测。它不是缓存或第二套业务引擎，也没有消除全范围扫描；四项新增32字节校验及元组开销需计入空间/写入成本，不能用原小样本性能承诺容量。未交付模型只维护/3当前格式，旧任务候选须离线重建派生索引，固定模型4及更早原库保持。

T00原索引冻结复用读取段主键、非重叠GiST、(instance_id,object_id,coalesce(lower(span),-infinity) DESC)读取索引和字典/结构引用索引，以及身份/企业/关系索引。R69取消每请求全企业存储审计后，按131第3.2节新增日期覆盖索引候选department_list_span_cover：GiST(span)，INCLUDE(instance_id,object_id,q_name,q_name_key,q_short_key,q_bu,q_date,q_state,q_code_key,q_path_date)。它仅加速既有Go事实的存储访问，不改变当前授权、准确count、同一快照或解码所需材料；索引可见性不足时仍由PG回表。R79同完整合成输入三次SQL和74170368字节成本、R80合法字段上限写入证据支持R81接入当前开发迁移，R81合成暖态400次零错误、P95 151.049/515.934ms通过普通列表目标；万科、冷态及两形状最终成本仍须实际验收。名称排序显式COLLATE "C"与Go字节顺序一致、稳定ID打破同名；后续改变索引或完整性门槛须同步计划和成本证据。所有新列/封套的表、TOAST、索引和WAL增量计入T04，两种完整形状均不得缩减。

### 3.6 131单语句捕获端口

列表和选择持X03共享策略守卫，固定服务端业务日D及规范查询，按第3.10节执行单条捕获：当前事实和动作/字段集合 → 显式缺字段筛选拒绝 → 获权文本筛选、准确count及稳定排序 → 当前页或完整匹配集合 → 必要祖先/BU及原始材料。非法字段、操作符和日期先400；自由搜索只用可读编码/名称，隐藏排序键为空串；不能先按隐藏值过滤再判权。路径条件按第4.6节以单声明游标依次取得普通条件剩余的获权候选，由Go原PathAt及引用投影完成路径过滤；取得页大小加一项合格结果即停止，不提供精确count。路径全选必须穷尽匹配身份。

闭包键为(部门身份,事实日期)：本页在D的段，路径日在Go写入时生成的祖先段，每个祖先在D的当前授权事实，及这些段引用BU的相应日期事实。SQL只沿已存父身份和span保守递归，不截断深度；普通页检测重复身份，完整选择以UNION去重保证递归终止。Go按日期分别构造Snapshot并执行原路径及引用权判定，不能把历史路径BU当当前授权BU。真实路径缺依赖或成环须失败，不能因名称无权跳过必要路径结构。

所有后续所需字节在同一语句内固定；捕获后不得再读可替换的当前段或可回收字典。普通全选只捕获固定身份；路径游标按身份帧捕获必要闭包及字典字节，Go逐帧判权和匹配，不在首批前排序或物化全部宽投影。完整导出按固定身份取得同次完整事实。普通编解码读取必要业务事实，来源身份仍匹配；不重算存储摘要、查询封口或逐字段证明派生列。材料缺失、不能解码或必要来源冲突返回503，无部分count/items。内部材料不进入公开DTO，也不跨请求缓存。

选择事务按本次授权/查询使用实际参数计划（事务内force_custom_plan），本轮保留原选择每算子64MiB设置，仅改变读时检查职责；父候选沿第7.1节使用数据库默认设置，两者不混用性能证据。SQL5秒、start/resolve/export/navigation完整读取HTTP10秒保持。接收原生字节后再解码，完整请求包含两阶段耗时、分配及连接成本。共享授权锁保持至响应材料固定及事务结束，会话活动仍在结束前写入。无选择普通列表不搬运全量头目录、不生成集合摘要；全选捕获身份和低命中路径的必要扫描如实记录，不能把FETCH预取量当作Go实际解码量。

### 3.7 131同输入A/B与独立判定

A固定提交3591b0313303ea9bc32579ab0f17759e23e6f011的模型4程序；T00已独立归档Go构建输入并实际构建，摘要见[输入清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/131-t00-design-freeze/ab-inputs.json)。A/B共用130保留的两份不可变来源原件：万科1BU/87343部门，以及合成2023BU/22104部门（包括2000BU/20000核心部门、原20成员50来源及深度/扇出/长线等特殊对象）。两份各20个表文件逐项核对原始解压字节摘要和实际行数，凭据及原件仍在仓外受限目录，仓内只留摘要/统计。

在任务自有空库从同一原件重建A的模型4、B的模型5派生结构；保留全部业务身份、历史、原终态和完整索引，不把原件model3标签误称A的运行模型。使用同样Go业务日、Grant、角色、筛选、20会话/12连接及查询顺序；新会话分别初始化，不复用源会话令牌。B产品构建摘要在T01—T03完成后、T04计时前冻结，T00不伪造尚未存在的B二进制或全量A/B测量。复核脚本为scripts/freeze-query131-inputs.py，拒绝仓内私有输入及覆盖输出。

T00独立小样本以创建输入和字面预期为准：0/1/20/21/137、三种排序/20/50/100页、Unicode与字面%/_、路径日及真实历史；授权另对照既有Decide/DecideReads，期间另对照完整Validate并保留手工反例。T04沿用原130独立空间/历史/语义判定和125请求清单，不能以候选查询生成自己的唯一expected。基线冻结不表示容量通过。

### 3.8 131父候选期间排除证明

定义F为共同Go命令准备得到的完整最终候选，W为F中本次父身份可替换的全部启用区间，包含相交前序记录与所有未来边界。先验证F的不变部分、目标真实首条及实际受影响反向依赖，再对每个可能父身份复用现行ParentChoices.at覆盖/同BU/启用/无环谓词。候选读取按第3.9节取得同快照的必要事实；实际读取失败或已需依赖缺失时拒绝，不能把未加载资料当不存在或从错误中跳过一行。

| 准备形状 | W及必要事实 | 可以排除的部分 |
| --- | --- | --- |
| 创建/恢复开放段 | 从真实新建/恢复日至无穷，包含全部已登记未来停用、转BU、父边及BU边界 | W之前不影响起点覆盖的旧名称/日志与资格重复扫描 |
| 位置新增/历史更正 | 受新父身份影响的最终启用段，继承传播后的相邻段一并纳入 | 不与W相交且已在合法原图及不变候选中证明的期间 |
| 改日 | 按重排后的完整F重算，不能沿旧日或客户端只改一字段的判断 | 只有经差异证明与W及反向依赖均无关的期间 |
| 删除 | 按删除后前序继承扩展的真实期间重算；不得把被删日之后一律排除 | 未受继承/关系改变影响的合法期间 |
| 固定启停期间 | 对最终仍启用且父身份可变的恢复/相邻段取并集；停用段不制造父资格 | 已证明关闭的边；反向引用、环路连接仍验证 |

排除的理由是同一候选之间仅W内父身份不同，W外原始关系保持，名称/修订/日志不参与资格谓词；任何进入W的祖先链仍逐边检查，未来子孙回到目标也会拒绝。不变部分失败则整次请求失败，不能报空候选。T00可执行证明先完整验证原图以建立前提，再收窄期间；720次图形对照、七种准备形状及远未来/一天缺口反例验证等价。此全图前提验证是证明夹具，T03必须用同来源存储保障及必要依赖端口承接，不能照搬为每请求全量加载后宣称减负。无法证明无关的检查继续保留并单列成本。保存仍锁后重读、执行完整最终Validate，不接受候选响应作许可。

### 3.9 行政上级查询的必要检查与关注点分离

2026-11-11用户要求“聚焦必要检查，去掉额外的不必要或过度要求”，并明确坚持关注点分离。行政上级候选的任务是按本次拟编辑关系提供有权且可用的部门；取消每次查询先证明全企业查询数据完整、逐段重算摘要、解码部门历史后比对派生字段的要求。本节局部替代第3.4—3.6及3.8节在行政上级消费者上的全范围或重复证明约束；原T00—T03及R11/R40通过和失败仍作为原合同的历史结果保留，不重写为新实现验证。

Go继续独占命令准备F/W、当前授权、启用及同BU覆盖、目标及反向依赖、无环资格、合格项排序/游标分页与最终保存规则。SQL在单条MVCC语句中按Go给定窗口、X03守卫内计划及保守身份闭包取数；现有范围/状态列只用于不会排除合法候选的取数条件，不在SQL复制最终资格或授权引擎。前端传入生效日期、管理BU及拟编辑上下文，展示服务端结果；日期或范围变化使旧请求失效。

候选查询直接消费Go写入事务生成的部门查询字段，不读取部门字典、历史结构或查询封口作为字段正确性的重复证明。相关BU继续使用既有存储编解码读取实际需要的身份，不为本入口另建一套BU业务模型。来源提交、派生字段和关系仍由同一事务原子保存；DB保留键、非空、引用及范围结构约束，Go写入与生成一致性通过真实回滚/字段对照测试验证。没有新增缓存、异步同步、兼容格式或全库检查平台。

查询只处理实际读取失败、必需列无法解码、已明确需要的依赖不存在及维护上下文来源冲突；不吞掉这些错误返回部分候选。正常提交下准确集合、当前权限和同一快照仍是业务要求。越过正常写入边界，任意修改无关部门、无关期间或未被本次使用的存储结构，不再要求由每次搜索上级发现；其工程诊断与用户查询分开。当前请求的X03计划已允许全部部门引用及编码且无搜索条件时，候选排序仅读取身份、不可变编码及既有管理BU目录；省去全部当前读取段的逐项关联。相关当前标签、完整期间及资格仍在同一游标事实中按需核验；有范围/字段限制或搜索时仍先用必要当前事实确定可见排序/检索键。不得从客户端声明取得此读取计划。当前133-M1取消总数而非返回估计数：每个进入结果或前瞻的候选都经完整Go资格判断；取得pageSize+1个合格项后停止，低命中或零结果继续至真实穷尽，不强制搜索。

保存仍取得共同守卫、锁后重读实际依赖并执行完整最终候选校验；查询结果不是写入许可。验证须保留未来首条、窗口前覆盖、开放期间远未来停用、历史更正闭合期间、转BU、当前撤权、选中标签及候选后变化竞争；另证无关数据问题不阻断正常候选、部门历史解码为零，并分别测两种完整形状的材料字节、资格工作与延迟。当前属于已采纳、实施中，不能引用原R40通过代替新版本结果。

### 3.10 列表与全选的必要职责收敛

2026-11-11用户明确将“只保留必要检查”同样应用于列表和全选。普通列表、路径条件、选择建立/核对及导出共用捕获采用同一边界：按当前授权与合法条件取得准确结果，不在每次请求审计全企业历史读取段、字典/结构摘要、查询封口和连续期间。正常Go来源事务及存储约束维护派生事实；越过这些边界的任意SQL损坏归独立存储诊断，不要求空结果或不相关查询发现它。

普通列表只解码本页及必要祖先/BU的窄段；133-M1普通全选仅捕获身份，路径首屏按64帧游标逐帧解码必要闭包；路径全选和完整导出使用单声明原生材料游标、每批256材料行，共享请求内字典/祖先解码。256材料行不是256个资格候选；路径全选完整穷尽后才建立草稿。现有段包含路径版本所需的准确关系来源，保留一次普通编解码，不为消除解码另建字段或存储模型。查询只检查实际取数/解码失败、明确需要的材料缺失和来源身份冲突；取消解码后再次派生查询列、逐值比对及重算存储摘要。共享材料的身份键用于本请求内复用，不作为新的完整性证明。

当前归属和字段权限、普通列表准确count、同一请求快照及完整路径保持。133-M1以PG固定身份草稿替代部门集合摘要及其失效规则；路径摘要仍服务路径表达，不作为选择或导航的全局版本。导出仍重新核验完整获权结果并在八列CSV全部准备成功后输出，不能删去失权成员后部分成功。保存继续独立取得共同守卫、锁后重读并验证最终候选。

验证覆盖正常写入一致性/回滚、无关存储问题不阻断当前匹配结果、必要依赖失败无部分结果，以及现有授权、准确计数、并发捕获、字典回收、选择失效和完整CSV行为。原T00诊断、T01/T02及R11/R64证据原样保留，不充当本次修改后的通过。必要职责收敛已实施；容量、最终回归及交接状态由131 T04记录持有。2026-11-11用户随后选定R118为本阶段完整选择方案：必要当前日/其他路径日身份分开取数，使用相应覆盖及非空父关系索引，Go逐行原生接收并执行既有编解码。所有结果在语句结束错误和必要材料检查成功后统一返回，不保留完整原生结果的第二份暂存。SQL/HTTP预算、单语句快照及本节职责边界保持；已测暖态单会话仍超目标，方案采纳与最终容量通过分别记录。

## 4 类型API、能力及分类

### 4.1 API与DTO

BU保留当前唯一路径/DTO/格式及已存结果形状。共用前端通过类型适配器映射到带kind的内存ResultView，不在网络响应中并存buId和通用别名。部门路径以`/api/v1/departments`开头，使用departmentId定位；实际OpenAPI/生成客户端在R4同次接入，不在R0发布空端点。

| 端口 | 输入/输出 |
| --- | --- |
| GET /departments；GET /departments/{id} | 与BU相同查询语义；详情返回本对象来源/记录、强类型资料、pathVersion和获权路径及capabilities |
| POST /departments | code/name/shortName?、managementBuId、parentDepartmentId?、effectiveDate、CREATE/原因；省略上级为企业根；请求中不允许身份、状态、路径字符串 |
| POST /departments/{id}/records、correct、delete、redate、deactivate、reactivate、lifecycle-pairs | 继承BU各动作严格字段和本部门版本语义；资料changes扩展name/shortName/managementBuId/parentDepartmentId。保存按最新依赖重新检验资格，不接收整树或关系头版本参数；不开放任意items |
| POST /departments/submissions/{id}/resolve | 无业务正文，CSRF及当前身份；查回/封闭本类型终态 |
| selection/start、selection/resolve、export、navigation | 部门使用第4.6节固定身份草稿和当前锚点协议；CSV按118八列，BU选择继续114/115现行协议 |
| 本域引用查询 | BU/父部门候选按企业、范围、日期资格及最小引用权过滤；未知覆盖失败，不冒用完整详情权 |

新分类以117为唯一目标目录，由同一Go目录驱动候选和准入，按对象可用原因筛选。R1切换BU目录时同步011—013、客户端及测试；保留存储中原分类码/名称快照和精确结果，普通更正/改日/删除不依赖新目录重解释旧分类。Enterprise初始化使用自己的内部建档合同，不因新BU原因目录删除而失效；不得把初始化原因当成BU公开写入的旧目录回退。当前CSV/页面“状态”只是标题变化，JSON键与枚举不变。

### 4.2 能力与错误

各对象适配器给出可写字段、当前可办动作、位置默认/范围、固定两条及修订/关系上下文。关系约束和候选同源计算，保存在当前事务内重检；UI只消费，不重建时间线。新增安全错误为DEPARTMENT_BU_CONFLICT、DEPARTMENT_PARENT_INVALID、DEPARTMENT_HIERARCHY_CYCLE、DEPARTMENT_REFERENCE_CONFLICT、DEPARTMENT_PATH_UNAVAILABLE，分别定位managementBuId、parentDepartmentId或整笔；无权对象不透露存在性。正式code统一以M02_为前缀，损坏路径证据失败关闭而不回退当前树伪造旧路径。

### 4.3 R3前端对象与办理端口

R3以2c8b9c8为基线，前端机制位于`features/organization/dated`。对象定义显式绑定kind、页面根路径、提交集合、类型化resolve及ResultView转换；字段目录通过本对象生成CreateInput的键检查。BU适配器独立拥有字段名称、真实API路径、明确动作联合类型、能力/受限字段投影与原结果链接映射；不把部门结果断言成BU结果，不增加网络别名或空部门端点。现有ObjectForm/ObjectDetailSheet/DataGridPage继续承担共用页面结构，对象字段及关系控件由对象页面组合，不建立第二套领域规则。

SubmissionFlow统一新建、单条与固定两条的本人引用保存、原标识/原正文重试、确定拒绝、结果未明及终态收束。正文只在内存，持久集合仍只含三元引用。事件绑定对象kind及本人引用，回调另核会话代次、挂载/表单和提交归属；同页保存、确认及原内容重试互斥；跨标签的迟到响应仍须服从已确认终态。撤权清理正文不抹掉待确认身份，当前读权恢复后仍可查回。存储写失败不发送新写入，确认结果后的引用清理失败仍保持可查回状态。

useOwnedRead统一上下文、原因、列表及详情的会话/挂载归属；useDraftExit统一路由、注销、浏览器关闭与编辑取消。共用选择/导出和前后对象导航接受对象强类型端口，服务端对象协议和当前权限为准，不在前端推算日期或业务资格。BU保留集合版本及变化重选；部门固定身份与编辑修订按第4.6节。两者保留跨页排除、排序保持、整笔CSV、独立返回高亮及查询/滚动上下文；当前日志筛选仍复用现有表格控件。

放弃确认只授权当前阻塞路由的proceed或当前取消/退出回调，不保存跨动作的全局放行状态。退出失败、请求中断或表单仍挂载时，后续草稿继续按dirty及已保存状态接受取消、路由和beforeunload保护；不能依赖清理字段必然使dirty变为false。R3-09发现及复验见[评审修复](../../verification/116-r3-review-fixes.md)。

R3只接BU真实运行时；合成类型检查或命名空间测试不能计部门接入。公开API、BU表组、原来源与历史终态保持，实际证据见[实施记录](../../verification/116-r3-frontend-workflow.md)。

<a id="r4-typed-ports"></a>
### 4.4 R4类型端口与R5接口清单

部门公开DTO使用Department前缀及departmentId；日期/状态/分类/缺失与null沿现行合同，禁止把部门DTO断言成BU DTO。外部请求不能提供授权事实、操作者、路径、内部计划引用、任意表名或任意items。以下路径均以/api/v1开头；静态路径优先于/{id}，ID须严格UUID。列表/详情/选择/导出拒绝businessAt等额外参数，所有写和只读POST仍核CSRF。写请求submissionId只来自Idempotency-Key头，正文不得再传同名字段；表中的本人提交均指该头，resolve按路径取本人标识。以下为当前端口清单，OpenAPI/生成客户端与已接通实现一致；131调整的查询消费者按第4.6节区分部门与BU协议，接线不代替T02专项验收。

| 方法与完整后缀 | 类型与主要字段 | 当前授权和结果 |
| --- | --- | --- |
| GET /departments/context | DepartmentContext：enterprise、currentDate、defaultEffectiveDate＝1900-01-01、timezone、name/code约束、fieldCatalog、capabilities；默认适配按202及200 G1实施 | 按当前Grant给对象入口能力；创建能力只是提示，保存仍核目标BU及保存后read/history.read；不从bu.read推断部门可见 |
| GET /departments/operation-reasons | operationType；DepartmentReason[]：code/label | 共用117目录按部门过滤；含ORGANIZATION_REASSIGNMENT；已存分类读取不依赖此目录；需要部门上下文可读 |
| GET /departments | DepartmentListInput：q/status/sort/page/pageSize及单列filterField/filterOperator/filterValue；DepartmentList：items/queryKey/businessAt；普通列表count，路径列表hasMore/nextCursor | 同一请求D；授权查询计划由Go构造、固定SQL执行；每行强类型BU/父引用、pathMode/pathVersion及获权路径；无通用查看日期 |
| GET /departments/{id} | DepartmentDetailInput：selectedRecordId?、sourceVersion?；DepartmentDetail：departmentId/code、resolvedSourceVersion、records/selectedRecord、capabilities及路径投影 | 当前read；显式sourceVersion另需history.read，字段交集；缺失显式记录不换条；selectedRecord.operationSummary不靠日志查询取得 |
| GET /departments/{id}/changes | DepartmentChangeQuery：page/pageSize、contextSourceVersion?、throughSourceVersion?、单列筛选三参数；DepartmentChanges沿122分组形状并使用部门字段 | read与history.read交集；首次确定throughSourceVersion后翻页固定上界；当前/历史上下文不混用；返回readFields/detailReadFields，越权筛选拒绝 |
| POST /departments | DepartmentCreateInput：code/name/shortName?、managementBuId、parentDepartmentId?、effectiveDate、operationType/operationReason | create及候选范围read/history.read；新建根与子部门同一入口；关系引用、身份快照及路径证据和首条同事务 |
| POST /departments/{id}/records | DepartmentAddInput：expectedSourceVersion、anchorRecordId、effectiveDate、operationType/operationReason、changes? | 沿所选位置新增/插入；没有实际变化不产新来源；开放字段及分类由服务端判断 |
| POST /departments/{id}/records/{recordId}/correct、/delete、/redate | 独立DepartmentCorrect/Delete/RedateInput：本人提交、expectedSourceVersion/expectedRecordRevision；更正可带changes/newEffectiveDate/classification，改日只带newEffectiveDate，删除无资料正文 | 分类、日期、关系按实际最终差异分别授权；拒绝多余字段；删除/改日隐式归属变化也核reassign |
| POST /departments/{id}/deactivate、/reactivate | 独立DepartmentDeactivate/ReactivateInput：本人提交、expectedSourceVersion、anchorRecordId、effectiveDate、固定类型/原因；仅恢复允许changes | 停用继承、恢复完整独立资料；不创建第二条同日事件；原父不合格不自动清空 |
| POST /departments/{id}/lifecycle-pairs、/lifecycle-pairs/delete | 独立DepartmentInsertLifecyclePairInput（expectedSourceVersion/anchorRecordId/deactivationDate/deactivationReason/reactivationDate/reactivationReason）与DepartmentDeleteLifecyclePairInput（expectedSourceVersion/deactivationRecordId/deactivationRevision/reactivationRecordId/reactivationRevision） | 插入/删除停用期间；两条明确动作一次提交，无任意编排及专用预览API |
| POST /departments/{id}/preview | DepartmentPreviewInput：单条动作的严格判别联合、本部门来源/记录/日期及声明资料；DepartmentPreview：valid/noChange、候选影响及当前readFields；错误沿现行问题定位返回 | 与单条保存同一纯Go候选/关系规则；需read及拟执行动作/字段权；无业务写入/结果表；只供自动辅助，不增加人工预览步骤 |
| POST /departments/references/business-units | DepartmentReferenceQuery：context、q?、page/pageSize；返回items、count、applicable | BU覆盖、最小引用读权与字段裁剪，不授完整详情/历史 |
| POST /departments/references/parents | DepartmentParentQuery：context、q?、cursor?、pageSize；返回items、hasMore、nextCursor?、applicable及selectedReference? | 完整期间父覆盖/同BU/无环，合格页加一早停；最小引用读权与字段裁剪；候选不是提交凭证 |
| POST /departments/selection/start、/departments/selection/resolve、/departments/export、/departments/navigation | 部门请求及DepartmentSelectionPage/DepartmentNavigationPage按第4.6节；八列CSV | 选择使用固定身份草稿，导航按当前锚点查询；导出另核export及字段，全部验证后输出，拒绝半份CSV |
| POST /departments/submissions/{submissionId}/resolve | 无业务正文；DepartmentResult强类型accepted/rejected/no_change/not_applied | 本人、本类型锁及结果表；已有结果按当前read/history.read投影，未知COMMIT不伪造终态 |

DepartmentDetail对显式不存在的selectedRecordId返回200、selectionMissing=true及selectedRecord=null，不静默选择另一条；对象/来源不可见仍按权限或404返回。DepartmentRecord的日期终点和shortName使用可缺省的nullable字段：失权省略，有权的空值/开放终点为null。DepartmentPairDeletionContext.changes只保留获权业务字段实际变化；start/endExclusive受effectiveDate读权控制，无权时两个键均省略，有权的开放endExclusive为null。删除动作权不授予日期读权；deactivation/reactivation及before/after沿用同一记录字段投影。

部门managementBuId筛选统一为equals与完整稳定BU UUID（输入去首尾空白并转小写），编码/名称及contains不作为该字段条件。DepartmentSelectionQuery为部门专属生成合同，字段目录含managementBuId/shortName/departmentPath，不沿用BU字段枚举；GET列表、选择start/resolve、export和navigation调用同一规范化和获权谓词。BU更名保持固定选择身份，导出在当前授权下输出当前名称。无BU标签引用权但仍有managementBuId读权时可按已知ID过滤；失去该字段读权则拒绝筛选。

Preview要求完整拟提交分类及字段，复用保存候选并返回合法性/无变化/影响；能力和位置默认只由详情返回，不在每次预览再次模拟全部动作。预览先取目标、祖先及显式拟选BU/父；Go完整RelationPlan确认有效边覆盖改变时再取完整反向子孙，并复核目标来源。保存仍独立锁后重读和重验；尚未选定原因的关系候选由references端口承担，不借preview伪造原因。

DepartmentReferenceQuery.context采用第4.5节的关闭联合，携带拟执行动作、准确目标、拟日期、拟分类及关系补丁；不能仅凭原记录分类推断正在编辑的启用期间。查询只提供关系候选，不证明整笔资料已可保存。

客户端只承担本部门expectedSourceVersion及必要目标记录修订检查；创建没有尚不存在部门的预期版本。关系选择提交BU/父稳定身份，不要求知道整树或被引用对象的版本。服务器按5.4锁后重读所有实际依赖；相关资格变化按当前授权给出安全业务诊断，无关部门变化不得制造来源冲突。hierarchySources不再作为详情/引用输出，expectedHierarchySources不作为请求或结果字段；额外旧字段按严格DTO拒绝，不保留双格式。确切关系采用仅在已授权资料/路径证据中按节点表示，不用全BU变化计数或哈希替代。

基本资料摘要采用与122一致的{recordId,revision,operationSourceVersion,recordedAt,operator}，operator为M02自己的不可变身份DTO。普通详情仅需department.read；无history.read仍返回所选资料修订的形成操作，隐藏日志/原件能力，不因此允许查询sourceVersion。既有修订被新来源复用时摘要不变；派生修订按真正形成来源归因；两类删除原件均按准确删除前修订归因，删除日志本身显示删除操作人。新部门写入必须读取X03 OperatorSnapshot，失败不得先保存再补身份。摘要不新增权限动作，不把对象头或祖先更名操作者当成本部门记录操作者。

实际应用端口位于application/department/ports.go：Store.WithinDepartment/WithinDepartmentWrite接收Unit；Unit组合Access、Now、Enterprise、submission.Repository[DepartmentResult,Terminal]、Snapshot、PreviewSnapshot、Source、History、CompleteResult、Lock与Persist。ReadSnapshot显式包含当前关系事实、所选准确来源、路径证据及修订摘要；LockRequest携带目标、编码、候选BU/父。Hierarchy的Discover/Plan/Bind/ValidateEvidence由domain/department实现，infra的departmentUnit在同一事务调用固定sqlc查询；不存在仅声明未接线的Repository接口。RelationMemberRef固定为{departmentId,relationSourceId,memberId,managementBuId}，实例/企业由可信上下文绑定；Existing与Planned采用同一部门关系键，PersistPlan仅接受目标部门的计划；BUReferencePort拥有加载资格和反向影响事实；Access.Guard只通过公开DepartmentTarget/ScopeFact/Requirements参与。纯Go的DepartmentPolicy、Hierarchy.Plan、路径解析不导入pgx/sqlc或Access存储类型；infra负责同一事务接线。DepartmentResult里的业务来源/记录/关系证据指原结果准确版本，重放仅按当前权投影，不换成当前关系头；当前授权所需头事实单独读取。能力只说明可办理入口，依赖不足或版本未知时不得返回肯定可办；错误按获权字段定位，隐藏对象不泄露存在性。

<a id="r4-reference-context"></a>
### 4.5 引用查询的拟编辑上下文

DepartmentReferenceQuery.context是关闭的联合：第10节的filter分支只用于列表BU筛选，create/maintain分支如下表。两个引用端点共用同一类型；引用选择实际需要的维护动作仅ADD、CORRECT、REACTIVATE，不为删除/纯改日/停用/固定两条增加无用查询入口。这些动作的保存仍完整核验关系。上下文只投影相应强类型写命令中影响日期、分类和关系的部分；不回传整树版本、客户端计算的期间或授权BU。

DepartmentReferenceQuery可带selectedId，存在时必须是规范UUID，不接受空串、null或其他类型。它只请求当前所选引用的展示投影，不参与资格计算。服务端先校验本次入口、字段和维护来源，再按同一当前授权快照返回selectedReference={id,code?,name?}；读取被拒绝或身份不存在时均只回显请求身份，不泄漏存在性。部分名称/编码读权只输出仍可读部分；取消关联快捷跳转及目标详情权预判；依赖技术错误不能伪装成不可读。未传selectedId时不返回selectedReference。items只包含获权且符合本次候选资格的对象；BU候选保留count，父候选按4.6返回hasMore及nextCursor，selectedReference不代表可保存。

前端按当前查询身份接收selectedReference，不从空候选推断失权，也不回退到已经失权的旧标签。标签读权收缩时先裁剪详情、记录、默认值及固定期间缓存的引用标签和路径，再重读；保持仍可读业务字段、关系稳定身份及提交/来源身份。名称等非关系改动不能被序列化为changes={}，关系补丁不存在时省略；明确parentDepartmentId=null仍保留。异步辅助的错误显示与主动保存的错误聚焦分开，只有主动保存触发字段聚焦。

| 分支 | 必须和可选字段 | 关系期间依据 |
| --- | --- | --- |
| create | mode=create、effectiveDate；查父候选还需managementBuId | 固定CREATE，自effectiveDate起；不要求先填名称、编码或原因 |
| maintain公共部分 | mode=maintain、departmentId、expectedSourceVersion、action | 锁后/一致快照中的本部门完整当前来源；当前read后才核来源/记录，不读取客户端声称的旧事实作当前资格 |
| ADD | anchorRecordId、effectiveDate、operationType；operationReason?、changes? | 与DepartmentAddInput相同位置/拟类型；changes仅允许managementBuId/parentDepartmentId |
| CORRECT | recordId、expectedRecordRevision；newEffectiveDate?、classification?、changes? | 与DepartmentCorrectInput相同目标；classification.operationType决定拟最终状态，不能继续沿原停用状态计算；changes仅允许managementBuId/parentDepartmentId |
| REACTIVATE | anchorRecordId、effectiveDate；operationReason?、changes? | 固定REACTIVATE，与保存相同恢复期间及独立关系资料；changes仅允许managementBuId/parentDepartmentId |

CORRECT的classification缺省表示保留准确目标的原分类；显式存在时operationType必填，operationReason可暂缺仅供选择辅助。已选原因必须一并传入，不从名称、日期或原原因猜新类型；暂缺原因不构造“已有效”的分类，也不伪造替代码。classification=null、空对象、只有reason、非字符串或多余字段均按严格DTO拒绝；关系查询只在类型足够时计算期间，完整保存仍要求合法类型/原因组合及全部授权。显式同一分类仍保留声明及相应字段检查，不能刷新原标签快照。日期缺省继承目标，显式null拒绝；关系changes缺省继承，parentDepartmentId=null明确企业根，managementBuId=null及显式空changes拒绝。

Go从完整命令/查询上下文提取同一个RelationIntent，复用Prepare内部的排序、日期与生命周期期间计算，再调用同一关系资格规则。查询投影是只读类型，不接受Persist，也不通过虚构名称、原因或可保存候选绕过校验；需要扩展内核时同时回归BU。名称/简称及未选原因的准入留完整保存，不建立第二套日期/启停引擎。只在相同相关事实、拟分类/日期/关系输入下比较查询与保存的关系资格；输入不完整不等于整笔可以保存。

查询BU候选时逐一替换该命令的目标BU，仅检查相应BU覆盖，不因尚未选新父而全空；选定BU后的父候选逐一替换拟父，检查同BU、完整启用覆盖及无环等关系约束。已有父关系不得因换BU自动清空。applicable仅表示当前拟动作/类型是否适用本引用选择，不代表写权限或整笔可保存。拟最终DEACTIVATE时返回applicable=false、items=[]；BU候选count=0，父候选hasMore=false；前端结合获权能力关闭关系选择，不自行推导生命周期，也不能把空启用段解释为任意上级都合格。缺引用权的候选在计数和分页前过滤；明确指定的BU上下文不可见/不存在按6.5统一处理。

R5用action、位置/记录、expectedSourceVersion、拟日期、classification及关系补丁（含缺省/null）形成查询身份；相关输入改变即重新查询，旧查询迟到结果不得覆盖新候选或已选关系。恢复原分类也重新计算。服务端判断资格，前端不自行推导启用段；保存必须重新授权、加锁和校验。

### 4.6 R133-M1部门选择、游标与锚点协议

2026-11-15按[302](../../plans/302-detail-navigation.md)补充导航协议：direction支持current/previous/next。普通BU／部门及国家／行政区划共用消费者一次读取返回targetId、previousId/nextId、position及count；位置从1开始，未有邻居省略对应身份。部门普通查询在窄事实排序后仅捕获原锚点、目标及两个邻居的必要资料，由Go继续核验；中间层仅保留排序所需五列，不物化重复宽事实；没有列／状态／树范围筛选时不扫描恒零的筛选字段拒绝计数，当前对象及可见字段权限始终执行；页码不进入权限判定。路径current完整捕获后由Go计算准确匹配及序号，与前后邻居请求独立，不增加草稿或跨请求事实缓存；路径previous/next保持锚点早停和targetId，前端统计失败局部重试。切换成功响应可在同一页面组件内移交一次导航展示，下一次切换重新请求，注销清除；普通返回页由服务端位置确定，不再发列表请求。原M1移除位置的条款以本补充为准，完整读取预算及详情独立可用保持。


本节按132第7节替代原131部门选择摘要、路径精确count及全局导航版本协议。BU选择仍按114，BU与部门共享锚点导航。只保留一套当前DTO，旧部门selectionVersion、expectedSelectionVersion、navigationVersion及全局序号字段严格拒绝；业务日仅由Go固定。接线及验证状态见[本批报告](../../verification/133-m1-implementation.md)，协议实施不等于容量验收完成。

| 入口 | 请求 | 成功响应与边界 |
| --- | --- | --- |
| GET /api/v1/departments | q/status/列条件/sort/page/pageSize；路径条件可带cursor | 普通列表保留准确count；路径列表仅items、hasMore、nextCursor及页面/queryKey/业务日，不返回count；pageSize为20/50/100 |
| POST /api/v1/departments/selection/start | query、selection、page/pageSize、sort及可选路径cursor | explicit.ids最多1024项；allMatching仅允许空excludedIds，完整捕获点击处理时的匹配身份；返回page及summary={draftId,revision,queryKey,selectedCount,canExport} |
| POST /api/v1/departments/selection/resolve | query、draftId、expectedRevision、addIds/removeIds/restoreIds、page/pageSize、sort及可选cursor | 单次编辑总计最多1024身份且正文最多64KiB；只更新命中块及增量，以修订CAS拒绝并发旧意图；page附selectedIds和memberIds，新增匹配对象不加入原全选 |
| POST /api/v1/departments/selection/cancel | draftId、expectedRevision | 只取消归属相符的当前修订；204，无业务来源 |
| POST /api/v1/departments/export | query、draftId、expectedRevision、sort | 对固定所选身份核验当前资料、完整read及export权；全部CSV准备及事务提交成功后发送，失败零附件 |
| POST /api/v1/departments/references/parents | context、q?、selectedId?、pageSize、cursor? | items、applicable、selectedReference?、hasMore、nextCursor?；无count/page；完整期间资格仍由Go判断 |
| POST /api/v1/departments/navigation及/business-units/navigation | query、anchorId、direction=previous或next、sort | targetId或null；按当前锚点与稳定可读排序找最近合格项；锚点失效409，不生成集合摘要或全局序号 |

草稿按实例、主体、会话、业务日及规范查询归属，固定身份不保存业务内容或授权结果。新匹配对象不自动加入；后续更名等资料变化不因全局摘要变化废弃选择。显式新增身份须当时可读且匹配，未知/失权/不匹配统一安全400；草稿归属、业务日、到期或修订冲突安全409。当前入口权仍每次校验；导出任一必需身份不可读、缺失或无export权整笔失败，不静默删项。canExport仅表达入口能力，不能承诺完整成员授权仍然有效。

PG logged草稿表每块最多1024个完整UUID及选择位，最多131072成员、逻辑3MiB，每会话每对象一个、每实例每对象128个，固定30分钟TTL且不滑动延长。新建替换本人旧草稿；取消、注销成功后、API启动和到期回收清理；DB重启保持有效草稿，API启动清空本实例草稿。额度与修订检查同事务，失败不留下部分编辑。到期回收每次最多128项，不能重写业务历史或来源；业务日变化拒绝旧草稿。具体预算及产品生命周期证据分别见132第7.2节和本批报告。

CSV使用0600请求自有临时文件，写入前预留64MiB、最多20个并发及1280MiB预留；超额/写失败/取消整笔拒绝，成功发送结束关闭回收。文件不保存到仓库，不用完整内存缓冲代替预算。共享策略守卫持续至全部成员授权、字段裁剪、完整CSV构造和事务结束；捕获之后的业务变更不混入同一文件，下一次导出读取当前资料。

父候选、路径及部门邻居采用同一READ COMMITTED事务中的单DECLARE语句绑定事实，FETCH每批最多64行；Go取得pageSize+1个合格项后停止，下一页从最后实际返回项继续，额外项不能跳过。签名游标绑定实例/主体/会话/业务日/查询排序及维护目标来源，不能跨请求保留事务或授权。低命中、前置不合格及零结果继续真实穷尽；路径全选穷尽后才建立草稿。关闭游标后更新会话活动再提交，批次间不另读可变事实拼接。普通准确count路径保持原捕获端口。

前端建立期间只发一个start，后续勾选按最新意图增量resolve；分页/排序排队并沿同一草稿继续。取消、条件变化、离页和会话变化使旧响应失效并传播AbortSignal，尽力取消已知草稿；迟到CSV不得下载。跨标签仅传随机失效代次。邻居失败只影响辅助导航，详情与未保存输入保留；返回保留筛选、滚动和稳定身份高亮，失效目标提供安全回退。SQL5秒及既有完整读取HTTP10秒预算、12连接不变，取消须覆盖等待连接、锁及SQL。

草稿准入使用128个持久唯一槽位和逐槽非阻塞事务锁，不用实例级互斥锁串行化20会话的身份块插入。本人替换可复用原槽位，额度满时拒绝新会话；失败回滚保持原草稿。读取锁定本人草稿元数据到事务结束，到期回收使用FOR UPDATE SKIP LOCKED跳过在途草稿，避免摘要与成员块被并发拆开。

相邻对象跳转后，前端最多读取原列表页及相邻一页来定位返回目标；路径使用服务端签名游标及已访问页位置，不计算全局序号。目标仍在该局部范围时返回其实际页、滚动并高亮；并发变化导致无法定位或辅助请求失败时保留原筛选并安全回退，不阻断已经确定的详情目标。

### 4.7 131内部端口冻结

Access.ReadPlanner暴露PlanReads(ctx,enterprise,at,action)与DecidePlan(ctx,plan,ReadIntent)。计划含守卫身份绑定、实例/企业/主体/会话/D/At/动作、Allowed和每字段ReadSet；ReadSet仅All、BU身份集合及部门身份集合，不包含SQL表达式或通用权限DSL。正式强类型接口见access/public/read_plan.go；T01已将计划移出实验标记，普通列表已消费该计划，正式验证见T01实施记录。

Organization.QueryCapture接收已规范的查询、可信实例/企业/D和本请求计划，返回统计、准确count及捕获材料；Go投影只消费该材料与原守卫。普通/路径/完整选择三种消费范围在类型端口显式表达，禁止业务调用者传“跳过完整性/授权”选项。父候选端口接收共用Go准备的F/W、目标准确来源/修订及必要关系范围，同条捕获核对预读目标版本；变更则冲突，不能将旧W套用到新来源。T00提供可执行读取/期间证明，M1以ParentRequest/WalkParents正式接线：输入包括同一Go准备的Base、ParentWindow、目标准确来源、管理BU、当前所选身份、搜索和守卫绑定的DepartmentReference计划；输出分开持有资格事实与D日标签，不能将拟日期归属当当前引用权。WalkQuery为路径及部门锚点提供同声明身份帧；QueryCapture只供普通准确count、本页及完整导出材料，普通全选使用OnlyIDs捕获，不走完整业务投影。

## 5 事务、锁与分表恢复

### 5.1 锁序与原子性

共同顺序固定为X03当前授权共享守卫 → 本类型提交锁 → 企业共享锁 → 全部相关BU头按ID排序 → BU行政关系守卫按GuardKey排序 → 部门头及本部门关系头按部门ID排序。关系/资格读取用足够的共享锁，目标写头用更新锁；一次按最强所需模式取得，不先部门锁再反向补BU/关系守卫。127并发重建修订：创建使用Go新生成且未发布的部门身份，不可能已有入边，可共享BU关系固定身份advisory守卫；仍对全部已有祖先及关系头持共享锁、编码唯一持排他锁，并在持锁后重读来源。已有部门若在锁前完整依赖快照中没有任何日期的入边引用，也可共享范围守卫，但本部门资料头及关系头仍取更新锁，全部祖先仍取共享锁。新增或迁入子节点必须共享锁住这个父节点的同两行，因此不能越过正在维护的叶子；若入边已在目标取锁前提交，锁后完整依赖/来源重读必须释放保存点并重新规划，新的非叶子改取排他范围守卫。除下述完整候选关系不变情形外，已有入边的部门继续排他；BU锁模式按013第9.4节执行；不按客户端声明或仅今日状态认定叶子，不取消任何关系守卫。创建尚无头行用上述编码与范围守卫保护；不建立树根/共享锁表或整树版本计数器。

锁前可发现依赖，持BU/关系守卫后必须重读依赖集合和相关部门/关系来源，变更则重新按统一顺序规划，不在已持有的逆序中加锁。BU变更必须先持相关BU守卫再读取全部当前部门/未来关系，部门写也先持该BU守卫，覆盖空集合到新增关系竞争。上级更名若经完整Go候选证明有效关系覆盖不变，只需目标及完整祖先；覆盖或父关系改变仍覆盖所需后代资格；换BU锁全部受影响原、新BU及其关系守卫。普通读路径/CSV使用一个一致的业务事实快照：可以单条SQL直接读取，或先以单条SQL固定全部资料/关系来源，再只读取这些不可变来源。后者不得重取当前头拼接；部门选择固定身份，BU选择仍按114哈希实际授权投影。列表事实缓存的边界见7.1。

部门业务savepoint覆盖目标部门身份/记录/来源、自己的层级变化/证据、日志、Outbox和accepted结果；X03判定可按013留在外层。任何失败撤销所有业务写入，无法确认回滚或COMMIT未知保持待确认。沿用整体10秒、语句5秒、锁2秒及有界清理2秒，剩余预算取更小值；不以HTTP状态或回调临时结果宣告终态。

只读业务查询也会提交同事务内的会话活动，但不具有业务提交身份。此类查询的外层COMMIT确认失败返回503 M02_DEPENDENCY_UNAVAILABLE及可重试的查询提示，不要求按不存在的业务提交标识查回，不返回暂存的items/count/selectedReference或业务终态。已进入外层提交阶段的读拒绝，仅在提交成功时保持原拒绝；确认失败仍按技术不可用处理。写命令及提交结果查回可能写入终态，其未知COMMIT继续保持原M02_RESULT_PENDING和同标识恢复规则，不能按只读查询降格。

### 5.2 命名空间、摘要与待确认

逻辑提交身份为(instance, subject, kind, submissionId)。BU命名空间继续使用现有三元组JSON的SHA-256前64位锁编码，部门命名空间在同一编码函数的参数序列前加常量`organization.department`；这是两个当前对象的固定命名空间，不存在旧协议试解或回退。两者各自按本人三元主键读取自己表，同一类型重试/resolve使用完全相同锁。跨类型同文本UUID可独立存在，不能互查或互封闭。

内容摘要继续绑定方法/路径、实例主体、全部实际字段和值及存在性；BU原规范化字节不变，部门类型由独立路径绑定。BU resolve仍为现有`/api/v1/organization/submissions/{id}/resolve`且明确只归BU，部门使用上表独立路由，不探遍所有表。原BU摘要及确定结果不改写，已存终态按本人归属、当前读权返回，不重跑资格或当前目录。

浏览器BU集合仍为`chnhcm-pending-submissions`，部门集合为`chnhcm-department-pending-submissions`，每项仍仅存instanceRef/subjectRef/submissionId，类型由受控命名空间给定。共用办理流程接受显式类型适配器；事件和异步回调同时核对kind/实例/主体/提交/会话代次/挂载归属。某类型有待确认时该类型新写先查回，另一类型不受此引用阻塞；失读权保留引用，不能用清空记录解锁，恢复读取权后处理。

resolve取得本类型锁且无终态时，仅向本类型表写not_applied；迟到请求见终态拒绝。accepted/rejected/no_change均提交确认后才解除引用；仅撤写权不阻断获权读取。原主体或类型切换后迟到响应不得广播清理当前表单。

### 5.3 R2共用提交端口与BU适配

R2基线为90dce99，按116的R2-A/R2-B实施。共用应用机制置于`organization/application/submission`，共用PostgreSQL事务置于`organization/infra/postgres/transaction`；不移入X01，不把BU DTO、SQL/sqlc或pgx类型暴露为共用应用端口。

应用层以泛型`Saved[R]`和`Repository[R,T]`描述本类型结果与终态存取，R/T由对象适配器明确绑定。`Run`在外层提交确认前只持有临时Reply，外层失败返回零结果及原错误。BU继续拥有候选准备、完整动作/字段授权、终态/诊断构造和固定格式；accepted写入包含所有业务事实及结果，无变化只写本类型终态，可恢复拒绝只在已确认业务保存点撤销后写终态。

PostgreSQL共用Scope拥有外层事务、明确Confirm、业务保存点、既有2秒清理预算和未知提交错误；BU适配器在保存点内切换自己的事务查询器，离开后恢复外层查询器。只有明确查询拒绝可以在读事务保留安全判定；业务写入不凭HTTP状态提交。保存点回滚未确认必须关闭不可用连接并返回待确认，不在清理context中补写终态。

R2-B的Lookup先取本类型提交锁再读结果，Replay/Resolve统一已有结果读取次序及封闭流程。对象提供强类型结果、读取准备、当前授权/字段投影、重放标记以及not_applied终态构造；共用层固定先身份锁、再读取、原摘要比对、当前授权和存储诊断校验，不能回调重跑旧业务资格。AuthorizeHistory通过X03公开端口逐项决定对象/历史读取交集，返回独立字段副本，不修改授权判定原slice；原写权不作为终态读取条件。对象适配器的访问失败和技术失败分别保留原事务语义，不以反序列化成功绕过授权。

锁键使用同一固定编码函数：BU仍为三字符串JSON的SHA-256前64位，部门按5.2增加固定前缀；未知类型拒绝，不默认落入BU命名空间。类型选择来自受控适配器，不接受客户端对象名，不动态选择结果表。BU请求摘要仍按原实例/主体字节绑定，不改历史摘要。合成端口及真实advisory锁检查只计共用机制验证，部门独立表及API留R4。

<a id="r4-lock-plan"></a>
### 5.4 依赖发现、统一锁计划与单次执行日

一次写请求流程为：认证/严格解析 → X03共享守卫 → 本类型提交锁并Lookup原结果 → 企业共享守卫 → 发现依赖并统一加锁（只记录待处理差异） → 锁后稳定重读及一次Clock采样T/北京时间D → 当前对象可见性、基础动作/声明字段及显式目标预授权 → 本部门预期校验 → Prepare及实际关系差异授权 → 关系Plan/Finalize/最终授权复核 → 身份快照及业务保存点写入 → 外层提交确认。每处失败按6.5返回；原结果查回沿R2，不重跑写资格。发现阶段不正式采样D，也不得先按关系/来源错误返回。

DependencyPlan包含相关BU头及来源、全部受影响BU的GuardKey、相关部门头及其关系头/来源（新部门关系头可不存在）、本部门请求预期值、锁模式和按期间组织的资格事实。既有部门当前授权BU、全部受影响过去/当前/未来启用段的BU、候选原新BU、父链与反向下级均从服务端完整时间线发现；只读取所选记录、客户端关系身份或旧采用行均不足。BU维护先持目标BU更新守卫，再枚举当前部门关系头中的全部启用/未来关系；若涉及其他BU/守卫/部门，则按全局顺序重规划，不持高位锁补低位锁。不存在整树来源或根存在性预期。

| 级别 | 锁对象/模式 | 作用与重新规划条件 |
| --- | --- | --- |
| 1 | X03会话/授权共享守卫；本人类型提交advisory独占锁 | 撤权与写入顺序明确；保留R2原BU键与部门独立键 |
| 2 | 企业共享锁 | 企业身份/开始日读取；不得以企业独占锁串行化所有部门 |
| 3 | 全部BU头按规范UUID排序；资格共享，BU实际写目标更新 | 部门新增先锁BU再读依赖；BU停用在零依赖时也持同一守卫，禁止空集合竞争双成功 |
| 4 | 全部GuardKey按规范元组排序；BU维护及有入边部门对相关BU取事务级advisory独占守卫；新建和经完整快照确认无入边的部门可取共享守卫；普通读取使用一致MVCC快照 | 即使该BU尚无部门关系也使用同一守卫，防止空集合竞争及并发成环；一次取最强模式，原新BU都纳入；无需根行/树版本 |
| 5 | 全部部门按规范UUID排序，每个部门依次取部门头、关系头；资格共享、目标更新；创建使用企业内规范编码advisory独占锁 | 上级/祖先、有效下级及未来恢复参与；单部门关系头缺失由编码及BU关系守卫保护；同编码不同BU创建仍冲突；不先锁部门再回取BU |

GuardKey＝(instanceId,enterpriseId,managementBuId,DEPARTMENT_ADMIN)，全部身份使用规范小写UUID。该键只标识并发范围，不代表版本化业务对象或数据库树根。关系advisory键使用固定数组["organization.department.admin-relations",instanceId,enterpriseId,managementBuId,"DEPARTMENT_ADMIN"]的紧凑UTF-8 JSON，取SHA-256前8字节按大端有符号int64解释；按原规范元组排序而非按哈希值排序。哈希碰撞只增加串行等待，不代替真实对象/来源校验；固定独立向量在R4验证，不调用被测编码反算预期。编码锁含实例/企业/对象类型，不与BU命令锁混用。

对象锁位于可回滚的守卫保存点内，X03/提交锁在其外。持BU/关系守卫后重查依赖和各部门/关系头；依赖新增/消失、需更强模式或相关来源变化时，回滚释放整组对象锁并按同序重取，不在高位锁下补锁。客户端本部门预期失效只记录为内部差异，必须完成6.5相应授权后才能返回来源冲突；无关部门变动不构成业务冲突，不检查整树版本。全流程共用原10秒截止时间，重试不重置。BU/部门的本类型提交身份取锁仍使用Go单调2秒子context；超时的查回仍返回pending且不封闭未知结果。2026-11-15用户授权重新审视并调整守卫后，部门命令预读取及BU维护调用的部门反向依赖准备采用Go单调5秒总预算，包含初始事实、企业/BU/关系/部门锁、锁后重读及保存点重规划，并取原请求剩余预算的较小值；外层预准备与内层发现继承同一更早截止时间。上述显式业务取锁SQL另有累计2秒预算，按单调耗时扣减，只在执行取锁操作时消费；读取准备不消费该较短预算，重规划也不重新分配它。数据库隐式取锁仍受原lock_timeout与阶段截止时间约束。PostgreSQL lock_timeout继续作为数据库侧保护，不能单独承担该阶段时限，因为墙钟回拨可能延迟其触发。阶段成功后业务计算继续使用原请求context；失败后的保存点/外层清理仍共用既有最多2秒清理期限，不产生业务终态。最终D在依赖稳定后只采一次；随后还需重规划则本次不写，走获权安全冲突或通用技术失败，不能混用两天。超时、授权依赖失败和无法确认回滚按R2保留待确认语义，不附隐藏依赖身份、数量、来源或日期，不伪造rejected终态。

读取在X03当前守卫内采一次D，再以一个SQL语句快照读取完整业务/依赖事实；SQL使用Go传入的D和查询计划，不自行调用now/current_date选择另一日。旧sourceVersion只改变被展示事实，不改变用于授权的当前头。路径、列表、选择和CSV共用同一投影；中途权限变化只能在守卫外发生，不输出混合权限的半份结果。

BU反向接线清单覆盖新增/更正、普通/首条/启停改日、各类删除、单条停用/恢复及固定插入/删除停用期间；逐项用BU最终候选检查受影响部门启用段。父部门对应动作同时检查有效/未来下级。纯名称和分类没有资格变化时仍按共同锁序保护读取/路径证据，但不制造关系修订。共同守卫只核验，不替另一对象写事实；拒绝保存后再用补偿任务修复非法关系。

## 6 认证、权限及路径保护

Access继续拥有账号、会话、完整Grant和当前授权判定，并按128记录有明确用途的必要安全事件。判定结果及逐项判权材料只在本请求所需生命周期内存在，不永久保存完整判权过程。R2提炼当前读取编排，支持部门目标、动作和字段的公开Intent由R4扩展，不能直接写Access表或把部门ID填进BUID。部门角色采用企业组织管理、BU组织维护、查询、审计、业务引用的完整授予模式；本批不建设权限管理台。

同会话并发读取沿用013第6.1.2节：进入事务时取得主体共享守卫并重核会话/权限，业务工作结束后由事务提交前钩子调用X03的`RecordSessionActivity`。会话行更新不再提前串行化同一浏览器的列表/context；更新后不再获取业务锁。仍使用原事务、连接、截止时间及原认证时刻，续期失败则整体回滚，撤权排他守卫仍等待整个事务结束。

### 6.1 当前授权归属

每次请求使用一个服务端执行日D；写入沿013锁后采样规则，读取在本次一致快照内确定。M02从当前头来源生成可信的部门范围事实交给X03：D覆盖记录存在时使用该记录的managementBuId（含INACTIVE继承资料）；D早于首条时使用当前来源固定首条最新修订的managementBuId。引用依据包含执行日、当前头、所取记录/修订及current_record或future_first分类；仅作本请求内部类型合同和判权输入，不持久化为授权过程档案，不接受客户端传入授权BU，也不扩大前端业务资料。

scopeKind沿enterprise/bu并增加department。企业与单部门范围按稳定身份匹配；BU范围使用上述事实。停用BU仍可作范围标识，不因业务停用自动撤Grant；实际动作/字段仍由X03判断。businessAt、显式sourceVersion、未来目标BU及创建者身份不能改变当前范围事实。已存在部门的当前头缺失或首条无法解析时安全拒绝，不猜创建者或第一条查询结果所属BU；尚未创建的对象按6.2的候选目标授权。

未来A→B安排在执行日到达转入日后才使授权归属变B；尚未生效首条更正A→B则当前候选提交后立即变B，首条只改日期而BU不变时归属保持。涉及已存在部门的列表/详情/引用/历史/预览/保存/选择/导出/结果读取均按同一规则重判，保存以锁后重读为准；依赖范围变化按5.1重新规划锁。业务日变化拒绝旧草稿；来源或授权范围变化后按当前事实重判，固定选择身份不自动增删，导出任一必需成员失权则整笔失败。不依赖后台任务改写Grant或缓存权限。

### 6.2 动作、字段和候选变化

部门动作使用`department.`前缀，沿BU既有后缀逐项接入：read、history.read、create、record.add、record.correct、deactivate、reactivate、record.delete/redate/reclassify、deactivation.delete/redate/reclassify、reactivation.delete/redate/reclassify、first.redate/reclassify以及export。不授first.delete或退出的timeline.revise。固定插入/删除两条分别检查两个明确动作，不能用通用维护权替代。额外`department.reassign`覆盖BU/上级实际调整；它与本次新增/更正/删除/改日/恢复等实际要求分别完整判定，每个要求由一份完整Grant满足，不拼接不同Grant的字段集合。

读取字段目录为code/name/shortName/managementBuId/parentDepartmentId/departmentPath/status/effectiveDate/operationType/operationReason；引用名称受对应引用权进一步裁剪。创建写字段为code/name/shortName/managementBuId/parentDepartmentId/effectiveDate/operationType/operationReason；之后code、status、departmentPath均不可写，字段缺省/显式空沿严格DTO处理。history.read不扩大readFields，export也不扩大基础读取。创建对候选目标范围分别核验create、保存后read和原结果查回所需history.read；缺任一则拒绝，不先写身份再补权限。

部门基础读取、历史读取、创建/资料维护、各类别改日/删除/分类、启停、归属调整及导出显式授权。基础动作按6.1的当前范围判断，归属调整另核受影响原/新BU范围和对应managementBuId/parentDepartmentId字段权；不能仅检查请求是否显式包含关系字段。按基准/最终候选的实际BU与父身份比较识别删除/改日造成的归属回退，首条首次建立关系走create，不虚构原范围。仅引用版本号变化、纯启停关闭/重开同一关系或同BU首条改日不追加reassign；恢复明确改BU/父则追加。普通更正继承分类、恢复沿REACTIVATE，归属权不代替分类权。

已存结果涉及真实部门时，继续按本人提交及当前read/history.read返回获权内容，不重跑原写权或资格；真实撤权/转出可使其不可读，保留最小待确认引用直到重新具备读取权。没有终态的本人resolve仍按5.2封闭not_applied；创建拒绝且没有业务对象时沿013本人提交读取规则返回安全终态，不要求虚构部门/来源或取得不存在对象的read。不得以曾创建或原accepted绕过当前授权，不以清除引用解除同类型待确认阻塞。

### 6.3 引用与路径保护

新增`bu.reference.read`和`department.reference.read`只给获权稳定引用、编码、名称及必要资格摘要，不等于完整详情/历史。完整读取者也必须满足相应字段和范围，不能从引用端点读回被裁剪名称。departmentPath须有该字段及链上必要名称引用权；任一缺失整条返回受限，链接另需完整详情权。当前路径和封存证据均按当前授权裁剪，日志、计数、集合版本和错误不泄露隐藏字段；不可读不当作null变化。

<a id="r4-authorization-input"></a>
### 6.4 Access目标、范围事实与逐项要求

Access公开Intent增加互斥的DepartmentTarget：{enterpriseId,departmentId?,scopeFact}；BU继续使用原BUID字段且不得同时带DepartmentTarget。scopeFact只接受M02应用端口构造的强类型值，HTTP DTO无该字段；至少含evaluatedAt/businessDate、mode(current_record/future_first/create_candidate)、managementBuId、currentSourceVersion、recordId/revision。create_candidate无既存来源/修订；其余引用必须齐全并与当前锁后头一致。停用属于current_record并保留BU，源/目标范围额外用AffectedScope{managementBuId,range,side}，不是改写当前scopeFact。

X03匹配enterprise/bu/department稳定范围：企业全域、按scopeFact的当前BU、准确部门身份；reassign另对每个受影响原/新BU范围发独立要求。同一BU换父只产生一次该范围要求，仍核parentDepartmentId字段；跨BU还核managementBuId。所有要求分别由一份Grant同时满足scope/action/全部声明字段，不把两份不完整Grant拼接。基础read与history.read分别判定再取字段交集；export不能扩大read，最小reference.read不能扩大完整详情权。创建在候选BU上分别核create、read、history.read，无永久创建者例外。

RequirementsBuilder复用同一动作/字段/范围匹配规则，在6.5规定的各阶段构造要求：入口声明及锁后本对象分类产生基础要求，显式关系目标形成预授权要求，Prepare的实际RelationDelta补齐隐式原/新范围，Finalize后再按完整候选复核。声明同值也不能绕过字段权；删除/改日引起的归属回退不依赖请求是否带关系字段；只有成员版本变化或同关系启停不生成reassign。每项仍由一份完整Grant满足，不把阶段拆分解释为拼接Grant。action/fields/范围及scopeFact保留为本请求内判权材料，向调用方只返回6.5允许的安全诊断，不持久化为逐次判权档案。

R4同时扩展Access的scope_kind、action、read_fields/write_fields允许值和Go匹配函数/SQL输入，新增department范围、6.2完整department动作和两个reference.read动作；动作与字段目录按对象校验，不能因扩展全局集合使bu.*接受部门字段。新部门新建后code不可写，path/status只读，父根由显式null表达；缺失字段不当成null。当前授权判定及必要安全事件仍由Access拥有，Organization只能调用公开Guard；M02不能直接写subject_grant或把关系查询当权限判定。

列表、详情、日志、引用、预览、保存、选择/导出、重放/resolve统一使用此范围事实解析器。引用的BU本身按BU稳定身份授权，父/祖先按各自当前部门范围授权，不能用子部门可读推导祖先可读；历史路径的旧名称仍受当前引用权裁剪。内部为关系安全读取完整事实不向客户端泄露隐藏对象、计数或日期，权限未知和证据损坏均失败关闭。

<a id="r4-authorization-diagnostics"></a>
### 6.5 授权和诊断返回顺序

本节约束新增执行及预览的失败路径；确定结果的重放/resolve继续沿R2按本人及当前read/history.read投影，不重跑原写授权。输入的JSON结构、类型、UUID/日期语法等与存储无关的错误可直接安全拒绝；任何依赖已存业务事实的诊断必须经过以下对应授权。内部依赖发现可以读全量事实，但不能据此提前输出缺失、过期、停用、成环或覆盖区间。

| 次序 | 检查内容 | 对外结果及不得提前披露的信息 |
| --- | --- | --- |
| A 当前对象可见性 | 维护对象使用锁后一次D及当前来源解析scopeFact，再判department.read；不存在或不可见走同一分支 | 404 M02_OBJECT_NOT_FOUND，同一通用消息、无来源/修订/字段诊断；即使请求预期已过期也不得先返回409 |
| B 基础动作及声明字段 | 按当前范围和准确目标原分类、拟分类构造入口动作、原/新类别、日期及完整声明字段要求；创建在请求目标BU稳定身份范围分别核create/read/history.read | 缺权为403 ACCESS_DENIED，无内部失败对象/期间/差异；声明同值也检查；尚无合法目标时不伪造类别授权，只在A通过后返回获权目标诊断 |
| C 显式关系目标预授权 | 先按命令稳定身份及原关系比较，核明显改变涉及的原/新BU范围与关系字段；再对明确选择的BU/父核最小reference.read。范围判断不依赖BU是否启用、父是否合格；create按候选稳定BU范围，不以创建后永久可读替代 | 缺动作/范围为同一403，不先查资格；有范围权但指定引用不存在/不可见统一404 M02_OBJECT_NOT_FOUND，不区分缺失/隐藏/停用/循环；使用通用引用消息且不附名称、状态、源或日期 |
| D 本部门预期及纯候选 | A—C通过后核本部门expectedSourceVersion/目标修订，再Prepare；Prepare仅访问本部门资料和公开企业开始日，不访问外部资格 | 获权的本部门来源/修订冲突可返回409；日期/分类/生命周期错误按自身readFields裁剪，不能夹带外部BU/父事实 |
| E 实际关系差异授权 | Prepare成功后，以完整WorkingCandidate产生原/新AffectedScope，逐项核实际reassign/关系字段及新引入或替换的引用；包含删除/改日隐式回退，显式提交同值不凭猜测追加reassign | 缺权403或引用不可见404优先于任何Hierarchy.Plan诊断；不能先Plan失败后跳过本步；纯候选失败时没有关系资格诊断可输出 |
| F 关系资格及最终复核 | E通过后才执行完整关系资格；Finalize只绑定准确引用/消除冗余，不扩大已授权关系身份/范围；最终完整Requirements再次核验 | 有权查看的目标资格失败才返回相应M02_DEPARTMENT_*诊断；只依赖隐藏祖先/下级的失败统一M02_DEPARTMENT_REFERENCE_CONFLICT及“当前关系不满足保存条件”，不输出隐藏身份、类别、数量、来源或日期 |

创建及显式新BU的范围要求先以服务端解析的稳定身份构造，不需要先查询其启用状态。引用授权对BU使用BU稳定身份，对父/祖先使用各自当前scopeFact；不存在或不可解析的引用与无引用权合并为同一引用不可用结果。纯资料维护及不变历史采用不新增“必须看见所有祖先”的权限前置；内部关系安全仍完整校验，只有输出受限。对于隐式回退，E从准确原成员/完整候选计算范围，不依赖客户端重发关系字段。

上述C仅对原准确目标与显式关系字段已能确定的身份差异生成范围预授权；尚不能确定的期间/隐式变化由E全部补齐，不能据猜测新增reassign。显式选择的引用仍核最小引用权，F复核全部要求；三步使用相同Grant匹配和动作目录。若不能确定某要求，失败关闭，不把关系资格报错当作授权替代。Finalize若产生此前未授权的BU/父身份或扩大范围，属于内核不变量失败，返回通用技术错误且零业务写入，不把已计算诊断先发给客户端。预览与保存执行相同顺序，预览不产生业务终态。

引用查询是只读辅助：maintain先核本部门当前read及上下文字段可见性，再核本部门来源/记录；create只使用获权上下文，不要求先创建对象。候选及明确BU上下文按最小reference.read过滤/拒绝，不能用department.read冒充引用权，也不把查询成功当作create/reassign批准。拟分类用于关系期间计算；无拟写权限的主体不能借preview取得写诊断，但最小引用查询不因此升级为完整详情或写权限。

HTTP错误沿既有Error/Fault结构，不新增诊断正文格式：对象不可见消息固定“部门不存在或当前不可见”，引用不可用固定“引用对象不存在或当前不可见”，403固定“当前权限不允许此操作”。这些通用错误省略field，不新增details或observedSource；已授权业务拒绝的内部failure_details仍沿013固定格式并只含获权内容。

只在已获权目标与字段范围内输出具体诊断；安全错误的HTTP状态、code、消息、field及details形状不能因同一无权目标的存在/启停/父边状态而变化。响应不得附hidden counts、observedSource、内部依赖列表或不可见修订；关联请求标识不参与业务结果等价比较，不承诺恒定执行时长。A—C/E的授权拒绝沿R2授权分支，不写部门command_result，Access不再保存逐次判权过程；确有用途的最小安全事件按128记录；只有通过授权后的业务拒绝才允许保存安全rejected终态。其failure_details仅保存已遮蔽诊断；重放还要按当前权限裁剪，必要时降格为通用关系冲突，只改变响应投影、不修改封存终态；不得保存原始隐藏诊断供未来查回。授权依赖/锁超时/连接失败不能伪装成业务拒绝，沿013/R2技术错误及待确认规则，不记录部分业务来源/关系/Outbox。

### 6.6 131请求内强类型读取计划

X03在已取得共享策略守卫后，从本守卫所持Grant编译计划；企业/BU/部门Grant按现有读取动作的并集及字段并集合并。SQL只对X03输出的身份集合做成员测试，不自行解释Grant。read/history.read/reference.read/export分别编译；写入声明继续原完整Grant校验，不能由字段并集放宽命令权。

计划只在生成它的同一个守卫内使用，不跨请求/会话或日期缓存；即使主体与会话相同，新守卫也拒绝旧计划。DecidePlan验证当前ScopeFact的企业、BU、真实来源/记录/修订、D、At及current_record/future_first模式，再使用集合；旧sourceVersion/路径日期不改变范围。X03现行Decide/DecideReads是等价对照，T00覆盖17280组及手工联合Grant预期，正式接线后继续保留反例。禁止只按角色名、客户端BU或候选历史归属构造计划。原型为内部类型，既不是外部可提交票据，也不持久化为授权过程。

## 7 容量、运行与实施拆包

单对象当前时间线的正式代表场景为20条，BU／部门分别覆盖第1、11、20条普通更正，完整HTTP保存目标P95≤500ms；按日读取、历史、失败回滚及原提交恢复继续核对。20条不限制实际可保存记录数；历史来源1000份、对象总量、深度／扇出、万级选择及1／20会话仍按各自合同，不与记录数混用。原10000条专用目标已按用户需求纠正撤销，不能据此删除既有记录或放宽其他子组。

<a id="department-broad-write-stage-target"></a>
2026-11-15用户针对[125本批代表性大范围写入](../../verification/125-tree-write-remediation.md)明确“阶段性放宽到2000ms”。适用范围为完整万科87343部门、目标全部历史后代含自身9799个的单会话关系变更/恢复，200次完整普通HTTP保存；本阶段P95≤2000ms，原500ms作为长期目标保留。零HTTP/语义错误、完整样本、原子提交、原历史及同标识恢复仍须通过。不推广到普通小范围保存、其他形状、20会话或查询入口；阶段目标采纳时运行时2秒对象准备守卫保持；用户随后授权调整为第5.1节5秒准备/累计2秒显式取锁，5秒SQL和10秒请求预算保持。既有200样本P95为1566.084ms、最大1807.285ms，按相同原件另存阶段/长期双结果，不重发请求、不覆盖原500ms失败。用户随后明确授权重新审视并调整运行守卫，新5秒准备构建另测200次保存零错误，P95为1520.479ms、最大2261.956ms，阶段目标通过而500ms长期目标未达。具体采用和验证由第5.1节及125本批记录持有；验收目标与运行时限分别管理。

版本沿仓内锁定Go1.26、PG18.6、pgx/sqlc/goose和前端锁文件；迁移只由upgrader调度。131 T01当前源码模型标识为m02-compact-5（4390固定运行及130交付为m02-compact-4，125原固定基准为m02-compact-3，128原交付为m02-compact-2，126原固定案例为m02-compact-1），迁移[1,2,3,4,5,6,7]；00002建立BU及部门独立紧凑存储，00004保留账号身份字段，00005登记部门动作及当前模型，00006补充字典引用侧索引以支撑历史无损维护，125本轮00007增加按确切来源的Go分段读取索引。只在隔离当前模型库安装，旧持久库不自动清空或改写，不增加旧模式运行回退。原R4/R6的17张宽表和路径表仅属于原报告的历史受测模型。

R6规定样本为2000BU，部门20000个/每部门20成员与50来源；同BU另有深度100路径、2000个后代扇出及单部门1000来源样本，137条跨页/八列CSV样本和20会话混合负载。沿013读300ms、确切源500ms、普通/固定保存500ms p95目标测量，CSV及路径额外报告耗时/内存/锁/WAL，不凭单次值承诺SLA；R6实测不达标不得借放松资格、历史或权限关闭。该规模是验收设计，不是已验证容量。

131本轮将原提交结果恢复明确按各对象、冷暖及混合组P95≤500ms验收；此处指submissions/{id}/resolve，与部门选择核对selection/resolve按本节规定的目标分别统计。125原轮次仅计查回语义及耗时的证据保持，不据当前判据改写历史通过结论。准备失败、未发出请求、计时HTTP失败和清理失败分别保留，样本不足不得计为完整执行或容量通过。

2026-11-11用户决定：数万部门规模下，新建部门选择管理业务单元后、不输入搜索词首次展开行政上级的正向第一页查询，P95目标按用户最新决定调整为单会话≤300ms、20会话≤1000ms，替代此前3000ms口径。本轮以22104/87343部门两种完整形状分别验证；暖态/冷首/冷首波按各自会话数分别判定，仍要求零HTTP及语义错误。首次展开不等于数据库冷启动；计时仍为完整HTTP及解码，浏览器动作单列。搜索/第二页/maintain父候选、路径筛选、选择建立/核对、BU列表及详情仍沿各自原目标；查询者create空候选不计正向资格。原始失败、样本和摘要保留，另存新判据复评，不重写旧报告；SQL5秒/HTTP10秒、普通列表准确count/当前权/全部期间及历史保护保持；父候选和路径分页按第4.6节取消精确count。

2026-11-11用户随后将部门普通列表P95明确为单会话≤300ms、20会话≤1000ms；两种完整形状和各暖冷组分别应用，完整HTTP及解码、零错误和准确count要求保持。路径条件、选择建立/核对、BU列表、详情和context不随本次调整。125中混合BU/部门列表分别按对象使用原BU300ms及部门20会话1000ms，混合汇总不以单一P95门槛代替两个子组。已有400次普通列表样本按新判据另存复评，不重发请求或覆盖原始300ms判定。

2026-11-11用户进一步将上万组织规模下本次全选目标调整为单会话P95≤1000ms、20会话P95≤5000ms，适用于部门选择建立（显式/全选）及选择核对start/resolve，覆盖本轮22104/87343部门两种完整形状及各暖冷分组。普通列表、首次无搜索行政上级仍为300/1000ms；路径条件、其余父候选、BU读取、详情/context及提交结果恢复不随之调整。完整HTTP及解码、零错误、全部规定样本、当前授权、同一快照和完整CSV要求保持，SQL5秒/HTTP10秒预算不变；既有原始300ms判定保留，按相同样本另存新目标复评，不因调整目标重发请求。

2026-11-12用户在133-M1首轮正式矩阵接近结束时作出阶段决定：其余已测入口暂时接受当前成果，单独将“维护父候选第二页、20会话”的本阶段目标设为P95≤2000ms。该操作仍包含首次取得游标及第二页的全部必需请求和排队；调整时r7为7254.764ms，原结果完整保留；最终r8结果见133-M1报告。原300ms长期目标、其他原目标、失败样本及125整体容量P1均保留，不将阶段接受改写为原目标达标。最终采用构建仍须满足本批完整6400次零错误验收。

2026-11-11后续阶段决定：R65合成首次无搜索父候选的单会话/20会话冷态各100次零错误，P95为1085.897/1910.331ms，用户要求仅作记录、本阶段不再追求解决。该两组不再追加调优或重复压测；目标和未达标结论保留，具体见131第6.2节及T04记录。

<a id="department-path-capacity-target"></a>
2026-11-15用户采纳部门路径文字筛选的阶段目标调整：单会话P95≤300ms；20会话本阶段P95≤1000ms，长期目标P95≤300ms保留。适用于现有完整万科／合成形状、各角色及相应暖冷工况，含部门树范围与路径条件交集；不缩减样本数、对象／来源规模或并发数，不自动重启原后置矩阵。阶段与长期结果分别记录，仍要求正式请求零HTTP／语义错误；原SQL5秒／HTTP10秒及锁预算、API12连接、当前授权、完整成员与历史保护保持。既有原始300ms数值和失败不回写，相同完整样本按新阶段目标另存复评；未测及明显超标项继续由125持有。其余普通读取、父候选、树搜索／定位、选择和保存不随此决定调整。采纳与判定器验证见[本批记录](../../verification/125-path-stage-target-adoption.md)。

<a id="department-large-subtree-stage-target"></a>
2026-11-15用户随后明确采纳：20会话大子树列表／路径的本阶段目标为P95≤1500ms。限定于完整万科／宽深合成形状、各角色的大子树首／末页列表及路径文字交集，即300矩阵的scope-large、scope-large-last、scope-path；原20会话1000ms判据与失败保留，路径长期300ms保持。单会话普通列表和路径仍按300ms正式判定；用户允许评估单会话路径400ms，须单列参考结果，尚未采用为通过门槛。本部门范围、无树范围的普通列表／路径、树节点／搜索／定位、父候选、选择1000／5000ms及写入目标不随之放宽。每组200次、完整对象／来源、20会话／12连接、零HTTP／语义错误及既有运行预算保持；同一样本同时给出阶段、原目标和长期结果，参考400ms不得自动关闭失败。执行和无新增请求的旧样本复评见[本批记录](../../verification/125-query-scope-continuation.md)。

2026-11-16用户“请按以上建议推进”进一步采纳**暖态**大子树路径文字交集及末页（scope-path、scope-large-last）的阶段P95目标：单会话≤400ms、20会话≤2000ms。大子树首屏scope-large仍为300/1500ms；冷态与混合组沿此前300/1500ms，不套用本次暖态放宽。原300/1000ms、路径长期300ms及原失败同时保留；上一阶段结果另列。每组200次、两种完整形状、三角色、1/20会话、API12连接、零HTTP/语义错误及现有运行预算保持。普通列表300/1000ms、选择1000/5000ms、普通保存500ms、其他对象和既有大范围写入限定口径均不改变。采纳及当前构建补验见[本批记录](../../verification/125-target-adoption-and-current-validation.md)；旧样本按新阈值通过不等于最新构建验收，也不关闭125整体容量P1。

实施按可复核交付包估算：R1两个包（资料/候选、影响/继承），R2两个包（事务端口、恢复/授权），R3两个包（共用流程、BU闭环），R4四个包（独立存储/API、最小Hierarchy、路径证据、共同守卫），R5两个包（部门定义、双对象验收），R6两个包（标准、容量/交接）。R1—R6工作量级分别为中/中/中/大/中/中，路径是R4主要增量；未测实施吞吐不编造人日和日历交付日期，首包实际耗时后校准。主代理承担本工作树阶段执行与记录，各阶段报告实际身份和复核范围；人工签收不由代理冒充。

### 7.1 当前读取与守卫优化边界

当前部门查询按消费者区分读取范围，普通列表以第3.6节单语句捕获及第4.6节协议为准。125/130原列表的全量来源头目录、分批加载和森林缓存方案已退出普通列表、选择/导出及详情导航读取链路；其原始测量与失败保留在对应实施报告，不作为当前普通列表的附加步骤。

| 消费者 | 当前读取范围与成本 | 验收边界 |
| --- | --- | --- |
| 无选择普通列表 | 按第3.10节同一语句执行当前授权/筛选及准确count，捕获当前页和必要祖先/BU材料并普通解码；不另取全量头目录、生成集合摘要或审计全企业存储 | T01原结果保留；当前R118构建的R123/R125分轮回归见131 T04，保留首失败，准确count窄扫描成本另报 |
| 路径条件列表 | 同一声明绑定候选及必要闭包，分批FETCH；Go获权匹配到页加一或真实穷尽，无精确count | 133-M1原计划/64帧捕获与解码分别记录；低命中真实扫描保持 |
| selection/start、selection/resolve、CSV及详情navigation | 部门固定身份草稿；显式仅所选身份，全选一次完整身份捕获，后续本页和命中块；CSV完整当前事实及权限；导航当前锚点，无集合版本 | 133-M1完整CSV、真实草稿生命周期及混合读写单列；131原结果保留 |
| 父候选 | 按第3.9节在同一语句取得F/W期间事实及D日标签，直接消费Go生成的部门字段；Go取得页加一真正合格项即停止，稀疏/零结果继续至足够或穷尽 | R64原结果及当前构建R123/R125分轮边界见131 T04；合成R65冷态超标按用户决定本阶段仅记录，万科等完整容量待验，不沿用全企业完整性聚合 |
| 普通详情、写入、预览、显式来源和结果查回 | 使用各自必要事实及完整业务验证快照，不调用普通列表查询替代业务或历史依赖 | 既有历史、守卫及恢复约束保持 |

130在125的T04分段索引上实施相邻等内容合并，保留128强类型历史原件；131 T01将当前存储准入提升至m02-compact-5，迁移集合仍为00001—00007。成功来源在同一事务内生成本对象当前头的read-segment-4派生索引：按完整资料/关系边界由Go划分区间，每段包含真实首条及该段所选的确切成员/关系、来源、字典/结构引用和封口；不保存用户授权或整条路径；部门额外保存Go按期间生成的状态及3.5规定的查询事实。同一对象、确切资料/关系来源、字典和规则中，只合并两侧均非空且正向、结构字节、有序完整UUID目录及全部查询事实完全相同的相邻段；空/反向区间保留原生约束拒绝，不得在合并中吞掉；合并查找区间时保留真实首条日期及成员，重编slot和绑定新区间的封套，状态查询事实由Go按期间生成，并与原成员语义核对；真实首日两侧的未生效/已生效段不合并，不以查找覆盖替代业务适用期间。SQL只按Go传入日期检索这些已生成的范围；普通列表及选择按第3.6/3.10节取得并解码必要事实，不执行全范围历史审计；行政上级按第3.9节直接消费查询列。索引是可重建派生物，来源提交原子替换当前索引，业务历史、提交结果及旧路径证据保持。仍按确切来源分批读取的BU关系候选、详情等快照，在头读取后观察到并发新源时按已固定的旧来源解析历史；普通列表直接消费同语句捕获的原始字节。两者都不得在必要当前材料缺失时回退旧算法或返回部分成功。现有持久环境不自动清空或生成索引；本轮仅在已核验归属且API停止的临时候选中显式重建，正常固定候选从空库初始化。

父候选按第3.9节分离查询与存储检查职责，不再调用全企业完整性聚合或部门历史解码。SQL依据X03计划、可见搜索键及Go给定窗口保守取数，覆盖窗口起点及全部已登记未来边界；目标的反向依赖、不变关系和所需祖先仍供Go一次校验。目标准确来源在同语句内复核，不符返回M02_SOURCE_CONFLICT，不能套用旧窗口。部门材料直接使用现有查询列，相关BU通过原编解码取得所需状态边界；不逐段重算摘要，不重新证明所有读取段连续。当前标签与资格事实分开，未生效标签可使用真实首条，未生效占位段不伪装为实际生效记录。复用Go同事务维护的BU/父身份目录定位保守依赖，并直接复用本次取得的相关期间材料；数据库仍可能扫描目录或相关范围，不能把材料减少写成仅扫描一页。采用实际参数计划、默认关联选择和数据库配置的内存预算，不再强制每算子64MiB；原5秒SQL和10秒HTTP预算保持。实际读取失败、所需依赖缺失和必需列不能解析时正常失败，不输出部分候选；明确BU不存在或无引用权保持404，没有当前部门读取入口权先403，维护先核原对象及上下文字段读权。BU既有按确切来源缓存保持独立，768MiB是其保守保留预算。普通详情、写入、预览、显式来源及结果查回继续使用各自事实端口。

125用途分离：普通详情与显式来源只取目标、必要的当前/历史祖先及源证据引用节点，不扩展无关后代；写依赖发现和锁后重读仍完整包含受影响后代。当前详情路径由Go从一致事实计算，封存路径证据只在显式来源读取；资料操作者摘要在正常详情保留。共同守卫读取完整资格事实/引用，但不读取未使用的展示摘要或封存路径正文；写入的Go候选、关系资格、重新生成的路径证据、锁序和重规划不减。部门列表/选择/导出/导航使用第6.6节的X03强类型读取计划，Go对捕获的当前归属执行DecidePlan；父候选检索使用DepartmentReference强类型计划，返回标签继续按动作批量核验；管理BU候选保持原强类型读取判定。两者保持相同范围匹配和字段合并语义，不保存过程档案，不跨请求缓存。

当前路径投影沿用125 T04的请求内可见路径前缀及定长SHA-256依赖摘要；原优化背景和全量列表测量保留于125报告。普通列表只为本页及必要祖先生成路径投影，路径首屏及邻居按需匹配；路径全选和完整导出承担完整身份范围。v2前缀包含父前缀摘要和节点输入摘要；节点输入摘要仍覆盖节点类型/稳定ID/名称、确切资料来源/记录/修订、关系来源/成员/管理BU。节点输入摘要只压缩单个不可变来源成员，不包含祖先、日期、路径文本、字段掩码或任何用户授权；普通列表在本次捕获内计算和复用，不依赖原森林缓存。每个请求先按当前引用名称权决定路径是否可见，再在请求内组合前缀；以原v1独立算法核对相等/不等关系，版本字节为新算法重新生成。当前列表由捕获SQL按可见编码/名称及稳定ID排序，隐藏字段的排序键为空串；名称使用第3.5节的COLLATE "C"，不使用原森林的编码排序索引。128删除跳转能力位，保留字段仍须有业务消费者；最终路径摘要另包含路径模式、解析日和本记录区间。缺少路径、日期、状态或任一节点名称引用权时仍输出统一restricted投影，不将隐藏依赖写入可见版本。128目标中详情和历史返回Go生成的路径文本及分段日期，不再下发仅为节点跳转使用的nodes/DepartmentPathNode；领域内部确切路径证据保持，不另复制展示节点档案。前缀仅在本次固定快照和授权内复用，不跨请求保存路径或权限。

路径解析器在本请求内共享已验证的不可变父链节点；列表直接计算前缀，详情或封存证据消费者需要时再展开完整节点数组。来源/日期、成环、有效父边与资料一致性校验仍由同一个领域解析器执行。普通列表只装配返回页所需字段；选择/导出/导航按第4.6节各自必要范围投影，不构造未返回的详情结束日、操作分类或父引用DTO。X03批量端口使用仅含目标、动作和用途的ReadIntent，不分配每意图的写入声明空槽；同一部门的不同读取意图可引用同一个本请求不可变ScopeFact，仍分别经过同一范围匹配器，逐项结果仅在本请求内使用。

133-M1已删除部门完整集合摘要及其生产/测试消费者，不保留旧算法回退。固定身份只表达选择意图，编辑revision只解决草稿竞争；祖先或所选资料变更不使身份自动漂移，导出重新获取当前事实及完整权限。原125摘要算法、测量和通过/失败仅作历史证据保存。BU选择仍采用114协议，由T05另行处理。

管理BU引用候选读取完整BU时间线及被维护部门的必要事实，不读取其他部门森林；仍由Go逐候选验证完整BU覆盖、当前引用读权、创建声明及分页前集合。先按可读名称/编码筛选候选，再检验该候选资格，不以提前分页替代完整集合。

行政上级候选复用共用Go命令准备的F/W和ParentChoices.at覆盖、同BU、启用及无环谓词。成功来源写入的合法关系不变量承接T00前提，行政上级的读时职责按第3.9节收敛，不再复用全企业完整性聚合；原候选/祖先W外无关旧名称、修订、日志及重复全历史资格不再装配或扫描，真实首条及不变依赖的必要完整检查除外。共用F不变部分失败返回关系错误，不伪装为成功的空候选。按可见编码/稳定ID依次核验可能父身份，取得pageSize+1个真正合格项即停止；不足则继续至真实穷尽，不计精确总数；selectedReference始终独立按当前权投影，保存仍锁后执行完整最终候选检查。创建与维护、查询者create空候选与maintain可读候选分别验收。无搜索、搜索、第二页及选中标签成本单列；第7节指定的数万部门create首次无搜索第一页按单会话p95≤300ms、20会话p95≤1000ms，其余父候选p95≤300ms；部门普通列表另按第7节的单会话300ms/20会话1000ms。10秒请求/5秒SQL预算内成功也不代表T04容量通过。

X03只读批量端口仅接受object/reference，128撤除reference-link意图及结果分支；动作仍限定为BU/部门基本/引用读取动作，不接收写字段、提交、复合要求或范围变更要求。同一Go匹配器逐意图检验当前范围；精确命中Grant序列相同才可共享字段并集及结果，且共享只存在本次请求。按128撤除READ_PROJECTION_SET的永久事件、input_snapshot、完整Grant/主体/会话快照、targets/requests/outcomes档案、压缩包络及专属格式版本；不把旧档案换一种编码继续保存。列表与选择/CSV的精确集合保护保留，返回页装配、完整集合计算及其临时材料分别测量。原125投影格式与复算结果仅由原实施报告持有；128删除及历史验证结果见128实施记录，当前容量由125本轮记录持有。

需要反向校验的依赖发现先取得真正受影响身份及其全部反向后代，再加入拟选上级、旧/新父和其祖先；加入祖先不重新扩展其无关子部门。BU覆盖变化写仍以所有当前/未来启用使用者为受影响根；部门关系覆盖变化写仍覆盖全部受影响后代、历史/未来父和原新BU，完整关系不变候选按第13节收敛。共享依赖按稳定ID的连续区段批量取事实头与关系头锁，目标更新锁保持原位置。锁后重读、依赖变更重规划、最终全期间资格和历史证据规则不减。

## 8 验证与追踪

规则/输入/独立预期见121及其非敏感JSON样本，R1/R2/R3分别证明纯规则、BU真实事务和菜单，R4/R5证明部门/Hierarchy及两对象；R6固化标准并执行容量。原BU历史样本与初始未提交资料已由R0保存，不能用新合同重写旧报告。

状态称谓改变仅运行受影响检查；R1共用规则及目录已有本批执行证据；部门真实关系、路径、分表和命名空间条件已有R4真实DB/API证据，详见121及R4报告；菜单已有R5实际结果，R6容量已执行且未达标，整改仍由116持有。文档通过不是部门实现、DB并发或路径生产能力通过。

## 9 待决事项

原R0业务范围和单部门关系来源简化保持。上轮两项剩余意见已分别在4.5及6.5明确合同，121映射R4-V21—V24，见本轮范围评审；本轮范围内无未关闭设计项。R4产品实现和真实DB/API已逐批执行，结果见116 R4报告及121实际映射；R5菜单实际结果见121第10节及R5报告，R6容量已实测且未通过，实际结果及P1遗留见[R6报告](../../verification/116-r6-standards-and-capacity.md)。R4原完整森林读取已按7.1优化；写守卫仍按相关BU保护并发，不锁全企业。原137行及H45样本不能证明20000部门容量或生产SLA。


## 10 R5页面接入合同增量

对象页面定义持有状态的可选读权字段映射；部门为status，BU无独立状态字段权。共用展示投影依据服务端readFields同步清理详情status、所选记录/时间轴/defaults中的state，并在详情渲染时复核该映射；此操作只裁剪已加载值，不自行推导业务状态或授权。单条辅助/保存、固定期间403及关系候选读取拒绝使用同一清理入口，未提供readFields时按空字段范围清理受保护投影；关系编辑输入、基线和父详情必须在重读返回前完成清理，后续重读失败不得恢复旧值。日志changedField=state选项使用同一对象状态映射，但字段范围取日志响应readFields，不使用detailReadFields或当前详情缓存；后端继续执行独立历史字段权校验。新建部门适配器省略未选择的parentDepartmentId，只有已选稳定身份才进入创建正文；更正等changes中的显式null继续表示清除。

R5共用展示端口仅承载Go返回的记录、能力、结果、选中状态与字段输入。BU及Department各自将本对象DTO映射到展示模型，并按本对象类型显式构造创建、维护、自动辅助、日志、选择/导出和查回请求；不得将部门DTO强转BU DTO、根据标签还原关系或在前端重算资格。正常菜单分别为设置/组织配置/业务单元及部门，待确认引用按对象类型、实例和主体隔离，两个类型均有待确认时明确显示对象名称。

关系候选控件将401/403/404拒绝限定于当前会话主体、对象维护上下文及候选种类；详情重读造成的disabled/enabled切换不自动重发同一被拒绝上下文。显式重试或查询上下文改变允许重新请求，成功后清除拒绝；候选仍由Go判断，不在前端复制资格或字段授权规则。迟到响应继续受会话及查询身份保护。

部门详情返回同一Go名称策略的nameGuidance。DepartmentPathSegment新增只读endInclusive（开放终点为null）；原DepartmentPathNode/canOpen及祖先链接目标按128退出公开合同；详情直接显示获权路径文本和分段日期，不为跳转下发逐节点源/记录/修订/父边身份。显式来源仍由领域内部原封存证据生成路径；当前名称读权继续裁剪。缺路径所需字段权时不返回路径段、段日期或数量。ChangeDiff/ChangeValue类型目录包含部门关系字段及reference值；日志使用可读标签，名称不可读时不把稳定ID包装成名称。

引用查询增加封闭的context={mode:filter}，仅用于管理BU列筛选，不带日期、维护动作或关系补丁。它要求部门入口和managementBuId字段读权，并逐项执行BU最小引用权；结果eligibility为readable，不声称维护资格。create/maintain继续返回eligible及applicable，由Go计算完整候选。filter允许按稳定ID解析当前可读标签，保持分页/搜索；不要求create或完整BU详情权。前端采用等于UUID谓词，标签变化不改变筛选身份。新增真实DB/API参数见121第10节。

部门页面只接受准确来源和所选记录；包括adoptedRef在内的未知参数必须统一拒绝并提供返回当前资料入口，不能忽略后展示当前事实。该边界与后台严格参数解析一致。

## 11 R133主线读取协议的实施输入

133-M0冻结的下一批读取协议及原型边界见[132第7节](../../adr/132-query-and-validation-boundary.md#73-单请求游标与一致事实)和[第一批记录](../../verification/133-m0-implementation.md)。采用同一READ COMMITTED事务的单声明PG游标，当前策略守卫保持，业务目标、候选、完整期间、必要祖先/BU和字典内容在同一语句快照捕获；批次之间禁止重新拼接可变事实，资格仍由Go决定。关闭游标后再更新会话活动和提交；不跨页面保持快照。

M1已按第4.6节接线；原M0证据不替代产品验证。仍须以本批实际结果验证SQL不预先物化全体宽投影、权限/期间/来源及字典回收并发、取消释放、密集21次资格早停和低命中真实穷尽。08首错的本批改动不改变来源事务、历史和共同守卫。

R133-M2同模型算法：摘要按(recordId,revision)一次索引，确切来源及删除原件/原结果投影使用Source端口而不枚举此前全部操作；完整日志仍用History。日期窗口只借用有序目录，返回的记录资料独立复制；读取段沿已校验有序非重叠边扫描，空档及真实首条保留。SQL字典/结构摘要改为有序聚合，计数、长度、UTF-8、原字节顺序及NULL传播保持，不删除摘要或改变存储模型。实际验收见[133-M2记录](../../verification/133-m2-implementation.md)。

长线减负仍保持当前格式：结构字典先查已有、批量插入缺项，并发冲突后按完整摘要取得实际身份；已封存/已满块在编码前决定换块。BU完整读取段仍逐段计算新来源、字典、结构、UUID及封套；相同slot和完整范围更新全部payload，变更或消失范围先删除、新范围后插入，所有工作与来源和头在同一事务提交。上述两条批量语句采用参数描述缓存及本次参数规划，二进制参数和SQL语义不变，其余语句沿原计划策略。头守卫对完整UUID目录只读取一次长度，全部引用逐项上下界检查不减。单块hash/合法性快捷处理与多块聚合必须保持相同摘要、NULL及数组边界；[补验](../../verification/133-long-save-overhead.md)持有实际结果，不宣称500ms达标。

M2预览的反向依赖由Go完整RelationPlan决定：起止、停启覆盖、BU或父关系改变均须读取全部保守子孙；仅身份差异不足以排除覆盖改变。关系覆盖完全不变时只读取目标/祖先及显式拟选引用，扩展读取后目标来源须仍匹配输入。所有分支仍校验当前动作/字段权、输入、目标和必要关系；Prepare结果不能直接保存，资格与Finalize保持。


## 12 R129共用边界与日志按页装配

BU覆盖不变维护的锁序及锁后重验以013第9.4节为唯一协议；部门关系写仍使用同BU范围守卫，覆盖变化保存保留全期间依赖验证。BU完整预览采用132第10节的全部削减期间存在性检查，命中反例须由Go校验完整事实，无命中须穷尽范围；纯扩张不引入新的引用冲突。预览不产生保存许可，当前授权、来源、必要完整性及取消语义保持。BU预览普通读取目标P95≤300ms，按动作、形状及会话独立统计。部门业务模型及其预览DTO不因BU精简而同步删字段。

两个对象原因目录逐项返回Go决定的default；共用表单只应用当前请求的服务端结果，保持有效用户选择、原分类、权限清理和会话代际。提交已知成功不因本地引用清理异常变成未知，发送前身份持久化及待确认任务阻断保持。

无筛选操作日志保留精确total、页码/大小、来源上界、全部筛选及当前读权与历史读权交集。原来源仍完整解码核验，仅当前页投影业务diff、标签和前后值；有筛选仍按原完整语义计算。BU连续来源号及总数必须相符，损坏不能以分页跳过。没有新日志表、索引、游标或跨请求投影缓存。

## 13 关系不变保存的必要依赖

客户实库案例的组织明细由私有原仓保存。

预读取与锁后稳定重读共用第5.1节Go单调5秒准备预算，显式取锁SQL累计2秒，重试均不续期。取锁继续沿5.1顺序，BU关系范围守卫可共享，目标事实头及关系头取更新锁，所需祖先头共享锁；并发新增/迁入子节点也须共享锁住同一父头，因此与更名串行。锁后再次读取依赖和来源，变化则释放保存点重规划；正常授权及请求来源/修订核对、完整Prepare/资格/Finalize继续执行。持久化端再次要求最终候选关系不变且原来源与缩小范围时完全相同，不能将该范围用于关系变化保存。停启、首日调整及BU/父变更继续完整反向资格检查。当前后代路径读取新名，原来源封存路径和后代自身来源不改写。

本批实现与实际页面、并发及历史验证见[修复报告](../../verification/133-property-rename.md)。用户明确将冷态及10000记录保存性能列为可并行整改事项，不作为新功能阻断；该决定不改写500ms目标、原失败或125容量P1。

共同基线后本轮采用实现进一步收敛初始规划：固定停启追加及固定期间插入/删除直接采用完整依赖路线，不为这些操作先试算缩小范围。关系不变保存可复用本请求已取得、范围和目标来源完全匹配的前向预备快照，仅作为第一次取锁的规划输入；该快照只消费一次，发生重规划必重新发现。锁后的完整依赖读取、全体来源比较、授权、最终候选/资格及持久化端来源绑定不变，预读和重试仍共用原Go单调预算。不复用跨请求快照，不将预备快照直接当作锁后事实。采用决定及实测结果由[本轮记录](../../verification/133-performance-continuation.md)持有。

本轮发布阶段进一步采用[133同模型减负](../../verification/133-cold-publication.md)：部门来源头、BU/父索引及当前投影在同一带原来源条件的UPDATE发布；读取段仍完整写入，历史、关系头、原回执与全部原生约束继续在原事务中闭合。固定期间结果需要逐项投影时，只复用本事务刚验证并发布的确切对象/新来源、锁后依赖及历史事实；当前读权和引用字段投影照常执行，取消检查不省略，保存点失败清除复用值，只有外层提交确认后才能返回accepted。新请求、独立读取和恢复继续从数据库读取验证，不使用跨请求缓存。

BU/部门历史追加接收同一对象写守卫下已经读取并核验的末历史块，或离线导入在本事务上次追加实际返回的块；新对象显式建立带完整归属的空首块。不得传入跨请求缓存的可变末块。去掉重复LoadChunk不改变前缀不可变、来源/提交唯一、连续编号、64条换块、归属、完整UUID、原回执及头闭合检查。内部history调用的三个现行消费者同步修改，领域和公开接口、模型5/read-segment-4/query-seal /3及迁移1—8保持。

<a id="department-tree-contract"></a>
## 14 部门树与范围查询合同

本节冻结部门树接口及范围字段；同批更新Go、HTTP、OpenAPI、客户端和全部消费者。

| 端口 | 输入 | 输出与边界 |
| --- | --- | --- |
| GET /api/v1/departments/tree/nodes | parentId省略表示根（空字符串非法）、pageSize、cursor；定位首窗可用aroundId，与cursor互斥 | items包含稳定id、获权code/name、parentId（真实根为null）、hasChildren；businessAt、hasMore/nextCursor、hasPrevious/previousCursor；不含总数、员工数、完整时间线 |
| GET /api/v1/departments/tree/search | q、pageSize、cursor | 今日有效且获权的匹配节点、businessAt及分页信息；不是维护行政上级候选，不接受拟操作日期、原因或资格窗口 |
| GET /api/v1/departments/{id}/tree-location | 稳定部门id | 从根到目标的当前获权链、businessAt；链项区分kind=node／context，只有node含hasChildren和可用同级锚点，context仅含获准的路径身份／标签／相邻关系；只返回必要链，不含旧来源 |
| 现有部门查询族 | scopeDepartmentId可空；有范围时includeDescendants默认true | 列表、selection/start、selection/resolve、export和navigation共同绑定；缺少id却提供包含下级参数、非法UUID或不支持的布尔值明确拒绝 |

无范围时两字段均省略，不能用空ID／null／字符串false替代合法类型；有范围而省略includeDescendants与显式true规范化为同一个查询身份。JSON用布尔值，URL只接受true／false两个字面值，其他值拒绝。排序、页码与树的展开／折叠不改变选择查询身份。路径文字条件与结构范围同时存在时分别执行其语义并取交集，分页仍走现有路径游标；不得把路径包含改成子孙关系匹配。

树游标沿用现有签名和归属保护原则，绑定对象种类、实例／企业、主体／会话、D、父节点、搜索或定位上下文、页大小、方向和稳定排序键；逐请求重新核权。进程重启、错父节点、跨日、篡改或跨会话使用明确拒绝并允许重新读取，不携带可复用授权结论。同一有效游标允许重复读取，不增加一次性游标或消费记录；事实及授权不变时同页内容一致，事实变化后返回当前获权内容，不保证历史页重放。

普通首窗从最小排序键开始；aroundId首窗以目标为首项向后取满pageSize，较早兄弟由previousCursor到达，不要求居中补满。previous／next均以当前窗边界作严格排除，方向编码在游标内，客户端回传cursor即可；前向／后向读取均按统一升序返回，旗标为false时对应游标缺省。aroundId必须仍是parentId的获权直接子节点；失配统一返回窗口不可用，不回退到第一页或泄露新父节点。定位取得的context祖先不产生可操作窗口请求。

各HTTP只承诺本次事实一致。客户端按会话代次、D、父身份和分支请求代次接收页面，合并时每个UUID只保留一个位置；观察到新父边后移除旧挂载，废弃受影响窗口及经过该节点的旧定位链，推进旧分支代次并取消其请求。维护返回或定位窗口失配时重新取得目标链；同一用户动作最多自动重定位一次，仍冲突则就地提示重试，不循环遍历兄弟。尚未观察到的并发变化可能使旧窗口暂时过时，刷新／重新展开恢复；静态全量可达与并发最终刷新分别验收。正常树分页和右侧列表分页是两套位置状态，不互相覆盖；未加载同级和失败同级不得伪装为没有同级。

部门固定草稿响应新增明确区分的当前页状态：共同返回本人已核验的summary（含queryKey）及businessAt；pageState=available时必须含page、selectedIds、memberIds；pageState=scope_unavailable时不含这三项，只有安全的范围不可用说明，不伪造count=0。该形状同步更新部门Go DTO、OpenAPI、生成客户端和页面适配器；共用钩子通过对象端口处理，BU维持自身无范围页状态的合同。

| 调用／失败 | 服务端结果与事务边界 | 页面行为 |
| --- | --- | --- |
| 普通列表、navigation或start时范围不可用 | 安全的范围不可用业务错误；start不建立新草稿 | 清理该范围标签和列表，保留其他条件，允许清除范围 |
| 已有草稿的空编辑resolve，范围不可用 | 先核本人草稿／D／queryKey／revision，再返回summary及pageState=scope_unavailable；不改成员、不消耗修订 | 显示已选汇总，允许导出和主动清除；没有当前页时禁用新增选择、翻页和逐行操作；导出不依赖page或旧标签 |
| 带add／remove／restore的resolve，范围不可用 | 明确拒绝，本事务成员和修订均不提交；只读resolve可再核验原草稿 | 不将未确认选择显示为成功；恢复确认状态，不自动重新start |
| 只读resolve的503／断连／取消 | 保留真实技术错误，不包装成scope_unavailable或成功摘要 | 保留不透明草稿引用，汇总标为待核验、暂停导出；成功的只读resolve后恢复，不能用旧受保护标签填回 |
| 会话／业务日／草稿失效或修订冲突；编辑结果无法确认 | 沿用现行拒绝／取消边界，不增加新的恢复协议或自动重放编辑 | 清理或要求重新选择，不自动建立新成员集；迟到结果不恢复选择／下载 |

范围不可用只隔离辅助当前页。上述成功摘要不代表全部成员已通过导出核验；export仍在完整生成CSV并核完当前权后才交付，现有必要权限、来源完整性及零文件保证保持。

树与带范围的普通页／身份查询复用参数描述并按当前参数规划，不复用跨根、宽分支和深链的通用执行计划。范围内六字段检查仅在同类身份集合可证明覆盖全部可见对象时省略重复判断；必要祖先的引用权限及完整性仍独立核验。成员数组表达身份集合，页面items继续持有显示顺序。

300同模型读取增量使用00010的department_tree_parent索引：启用读取段按instance_id、q_parent、COLLATE C的q_code_key及完整object_id定位，包含span、q_bu、q_path_date。当前及未来生效仍由Go在来源事务内发布读取段，索引由同事务维护，不增加业务表、整树缓存、封套或模型版本。根空父与指定父使用可索引的NULL／等值条件；同一SQL捕获页、必要祖先和子项存在性，按父索引证明存在后即可停止，不枚举宽分支只为显示箭头。X03计划仍是唯一权限输入；Go仅以同类身份集合的保守包含证明消除重复字段检查，不根据BU推断部门成员，不能证明时继续逐对象核对。SQL只传本树使用的六个字段及最小引用字段，应用层保留原始当前计划。00009由c国家目录占用；本分支采用1—8、10的精确迁移集合，合并时统一接续，不更新固定环境。实际成本、迁移和历史核对由[300实施记录](../../verification/300-department-tree-implementation.md#33-父节点索引诊断)持有。

## 15 301用户反馈接续

2026-11-14用户授权实施[301方案](../../plans/301-user-feedback-and-recovery.md)。第3节规定本轮目标：保存请求结束未知后至多自动确认一次，手动确认与同内容重试互斥；刷新只恢复身份；会话校验与业务权限分流；局部失败保留合法条件和可读输入。错误分类、筛选／选择恢复及提示生命周期按301同步实施；本节替代旧长提示、强制返回列表和旧统一错误分流要求。Go业务规则、当前授权及不可变历史保持。实际完成范围见[实施记录](../../verification/301-user-feedback-implementation.md)，未完成验收不作已交付声明。

2026-11-15 G5接续：地点及两类目录的首批综合验收、容量测量和固定4383运行交付由[205](../../verification/205-g5-implementation.md)完成；产品1392文件与G4一致，复用与本轮执行分开登记。D08/D09非阻断性能项和125其他容量P1保持。

## 16 法律实体基础维护接入

LegalEntity的差异、国家采用矩阵和L2/L3边界由[207](207-legal-entity-design.md)持有；实施及实际验证见[206](../../verification/206-legal-entity-implementation.md)。本类型采用既有dated/submission，独立Data/Patch、授权动作字段、表组/封套/回执及公开API；追加00014，不改变模型5/read-segment-4/query-seal /3及原类型编号。

登记成对约束评价完整补丁后的最终资料；只改名称、类别或登记不重新采用未变国家。新采用在国家共同锁内读取并核当日资格，旧采用按准确来源及当前历史权限显示。编辑候选的可选集合不能证明旧标签读取权限；不变引用的编辑上下文须另外投影原采用标签并报告受限状态，失权清理已显示的旧标签，候选失败或不再可选本身不清空稳定身份和其他输入。
