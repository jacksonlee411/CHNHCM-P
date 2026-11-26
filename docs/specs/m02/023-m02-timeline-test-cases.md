# CHNHCM-023 M02 时间轴全面测试案例

登记信息：[registry.json](../../governance/registry.json)。本文是[技术设计模板](../../templates/technical-design.md)裁剪形成的测试设计，持有可执行的测试输入、步骤和独立预期；业务规则仍由[011](011-m02-business-requirements.md)持有，交互由[012](012-m02-functional-and-interaction-design.md)持有，协议、事务及迁移由[013](013-m02-technical-design.md)持有，范围和批次由[010](../../archive/plans/010-m02-scope-and-plan.md)持有。

下列现行页面案例按[012 视口与缩放](012-m02-functional-and-interaction-design.md#m02-ui-viewport-matrix)选取共享桌面视口、关键流程的原生 200% 和触发专项的原生 400%。各批执行交接中的倍率与结果按当时记录。

当前普通查询按[123](../../adr/123-current-maintenance-and-dated-query-boundary.md)执行。旧案例中的任意businessAt事实断言改由内部按日解析或受控服务端时钟验证；查看日期控件的旧交互预期由第26节明确替代，旧报告保持原结果。

当前交互按[101直接保存](../../adr/101-direct-save-and-form-assistance.md)执行：全部场景取消手动预览及预览许可门槛，填写后直接提交；只读计算用于自动默认/必要影响，Go在保存事务内重新核验。下文“预览”在仍有计算含义处指只读辅助，不再指用户按钮或必经步骤；历史实施记录保留原时点。


103文档采纳时的记录（2026-11-02）：[103位置新增/插入](../../adr/103-positioned-record-addition.md)：第5.1节追加24项待验案例，旧SCN003案例补充所选位置与显式锚点输入。规则/案例已编写不代表产品或调用已实现；原T07的253例及旧报告保持各自当时合同，当时新24例和受影响旧参数留待实施；后续已由105接线验证，108补验其明确留出的参数，实际范围分别见对应报告。


当前范围按[109](../../archive/plans/109-lifecycle-pair-maintenance.md)收窄：BR019只公开插入停用与重新启用、删除停用与重新启用；旧通用请求/预览、项目编辑、混合办理、多条分类纠错和日期交换退出。带“109退役”的行保留原输入/预期作历史，不执行、不计当前通过。未退役的共用原子性、权限/并发/来源/恢复断言按固定两请求或合法普通单条入口承接；原报告不回写。

2026-11-08按[128](../../adr/128-authorization-evidence-removal.md)修订TL-178/TL-212及反例14的预期，编号不变；取消逐次完整判权存证断言，改验当前权限、原结果独立恢复及过程档案删除。新参数尚未运行，旧报告结果保持，不能沿用为新参数通过。

## 1 范围与证据边界

覆盖011所持BU的19个场景、25项规则及37个验收编号（AC020/022退出，35项现行；含122新增SCN027、BR034/035、AC041—044，部门118编号不重占）中与BU时间轴及列表有关的正常、边界、异常和恢复路径，承接[018](../../adr/018-effective-records-and-source-history.md)、[020](../../adr/020-effective-record-delete-and-redate.md)、[021](../../adr/021-bu-lifecycle-and-record-maintenance.md)、[022](../../adr/022-joint-timeline-revision.md)及[114](../../adr/114-bu-query-selection-and-export.md)。包括创建首条、查询、三类日期新增、更正、删除、改日、多次启停、固定启停期间、首条两步维护、分类纠错、快照、引用、权限、并发、故障和全结果选择/导出。后两项已按115增量实施，实际分层与参数范围见本轮报告。

本文持有测试设计，实际执行按各批报告登记。按[025](../../adr/025-greenfield-implementation-policy.md)保留原257个编号，103追加TL-258—281共24例，104追加TL-282—293共12例，105追加TL-294—301共8例，107追加TL-302—306共5例并退役原保留选择TL-132，109追加TL-307—322共16例并退役通用编排39例；110/111追加TL-323—334共12例列表目标，本次单列条件增量追加TL-335—342共8例，单条启停追加TL-343—348共6例，114/115追加TL-349—366共18例选择/导出案例，122追加TL-367—386共20例摘要/日志目标案例，123追加TL-387—394共8例，合计394个编号，其中**350例现行、44例退役**（并行旧编号映射见[023记录](../../verification/023-parallel-case-identifiers.md)）。新增12例当前执行见[111 T03](../../verification/111-community-t03.md)；现行案例按各批实际参数及分层证据验收，未运行参数继续保留，退役不计通过。已有创建/查询报告保留其原基线和范围，不能直接给新案例标通过。这里的“全面”指覆盖现行有限合同的等价类、边界与明确组合；不是枚举所有可能日期、任意长度时间线或任意故障交错的数学证明。超出范围的请求作为拒绝或能力未接入案例，不据此扩展业务。

`TL-001`等是本文固定的测试案例标识，显式锚点为`m02-tl-001`；不新增或复制BR/SCN/AC定义。后续只追加未占用编号，删除的编号不复用，不随章节或表格调整重排。参数化实例用`.P/.T/.F`或明确坐标后缀单独报告，不把一个编号的部分参数通过写成整组通过。

## 2 执行约定与独立预期

### 2.1 环境、角色和正常入口

使用隔离的合成数据实例I、唯一企业E，E从2025-01-01起有效且终点开放；默认受控业务日D＝2026-06-15，时区Asia/Shanghai。案例中的月日均为2026年，另写年份者除外。时间通过可注入Clock控制，不修改主机时间；跨午夜和时钟回退例外在步骤中明确。

默认角色A具有目标BU及历史读取、案例所需动作/字段权限；R只有读取，CN缺英文读写，EW可读英文但无英文写权，L只有所述生命周期专用权及必要读取/字段权，X无目标BU权限。每个授权子项独立配置，不能靠“管理员”名称替代授权。另建BU2和实例I2用于隔离反例，身份UUID均不同。

正常入口固定为“设置 → 组织配置 → 业务单元 → 列表 → 目标BU详情”。 普通更正的B层路径统一为“生效日期轴选择记录 → 标题旁更多操作 → 更正记录”；普通维护不提供查看日期。菜单迁移不改变本表的业务输入、主要层、独立预期和原编号。每章注明后续菜单动作；标有B的案例必须从该入口真实点击办理、刷新并读取持久结果。A层反例可在同一合法会话下直调接口，以证明隐藏按钮不能代替服务端守卫；未知/无权对象返回安全信息，不要求页面提供非法入口。行内菜单步骤说明业务上下文，G通过Go例表、D通过真实存储/应用事务、A通过实际接口承接相同输入；只有B实际点击菜单，不能为执行非法参数而要求页面开放非法操作。

本批UI补充断言映射AC001/AC004/AC007/AC008/AC010及012：普通列表和详情无查看日期入口；关闭维护菜单后焦点回触发处；详情按所选记录表达；历史模式保持只读，只读主体不出现写入项或空更多菜单；编辑操作集中且未保存保护保持；菜单/弹窗覆盖既有视口和真实缩放条件。用例位于tests/e2e/detail-actions.spec.ts；独立观测不增加一套TL业务参数矩阵，不将本次UI通过提升为全部322例通过。两个返回入口区分未实施，原历史保护继续回归。

对象表单补验映射AC002/AC003/AC004/AC007及[012统一外观](012-m02-functional-and-interaction-design.md#m02-ui-object-form)：用真实新建表单作为同环境对照，测量新增/更正的文字、输入高度、下划线及标签位置，核对选择器/控制栏密度、桌面两列与窄屏单列；切换尺寸保留输入，取消保持保护与焦点。直接保存、字段错误、显式清空、权限及历史仍由原业务旅程回归，外观通过不代替业务验收或新增TL编号。

按[页面表达原则](012-m02-functional-and-interaction-design.md#m02-ui-expression)补验：新增/更正没有字段提交勾选或常驻机制说明，直接编辑后仅发送用户编辑字段，英文清空和同值结果保持原协议；取消预览按钮及独立拟保存时间线；必要信息自动带出。字段错误、未保存、冲突、权限和待确认反馈仍验证原结果，不因精简文案移除保护。

103普通单条ADD的共同输入要求：每个正例都明确选择目标日前在T0中紧邻的现有记录，手写其recordId映射作为anchorRecordId，与expectedSourceVersion一同提交；原表只写日期的例子按该固定T0手工补全此输入，不调用被测默认/查找函数生成独立预期。旧日期非法反例也携带明确现有锚点，保持原拒绝主因；锚点自身错误另由TL-275承接。停用、重新启用和联合项不添加此字段；首条更正后另次普通新增仍必须携带锚点。B层根据所选位置进入新增记录或插入记录，旧新增生效记录文案仅在旧报告作为历史保留。

### 2.2 合成资料与时间线夹具

资料值是完整快照：A＝甲/Alpha，B＝乙/Beta，C＝丙/Gamma，Z＝新/New；A2＝甲2/Alpha、B2＝乙2/Beta、C2＝丙2/Gamma。斜线前后分别为中文/英文，`null`为明确无英文。`S`表示DEACTIVATE，`R`表示REACTIVATE，其他记录的类型显式写出。f始终为固定首条，c/s/r为各自稳定recordId别名；执行时换为真实不同UUID并保存映射。T0是该例办理前完整来源，T1是本次成功来源；一例内多次成功用T2/T3，不以记录时间推断顺序。

| 夹具 | T0完整成员，日期均为当月1日 | 用途 |
| --- | --- | --- |
| F0 | E已初始化，无目标BU | 创建、身份和协议 |
| F1 | f：1月CREATE A | 唯一首条、末尾新增、未来安排 |
| F2 | f：1月CREATE A → c1：6月CHANGE B → c2：10月CHANGE C | 插入、后继保持、普通重排 |
| F3 | f：1月CREATE A → s1：4月S → r1：8月R B → c1：10月CHANGE C → s2：12月S | 当前停用、启停后继和跨段 |
| F4 | f：1月CREATE A → s1：4月S | 开放停用段 |
| F5 | f：1月CREATE A → s1：4月S → r1：5月R B | 末尾恢复和独立快照 |
| F6 | f：7月CREATE B | 今日NO_FACT、未来首条 |
| F7 | f：6月CREATE B → s1：7月S → r1：8月R C → c1：10月CHANGE Z | 有后继的首条历史补全 |
| F8 | f：1月CREATE B → c1：3月CHANGE B → s1：5月S → r1：7月R B | 先以首条A建立后继，再更正首条为B形成T0；c1不能因当前同值被自动删除 |
| F9 | f：1月CREATE A → c1：3月CHANGE B → c2：5月CHANGE C → c3：7月CHANGE Z | 分离影响、多个普通记录 |
| F10 | f：1月CREATE A → s1：4月S → r1：5月R B → s2：8月S → r2：9月R C → c1：11月CHANGE Z | 多轮启停及间接依赖 |
| F11 | f：1月CREATE 甲/Nroth → c1：6月CHANGE 乙/Nroth → s1：7月S → r1：8月R 乙/Nroth | c1当初仅输入中文，r1确认默认；验证后来英文更正不追溯传播 |

创建默认原因NEW_ORGANIZATION，CHANGE为NAME_ADJUSTMENT，S为BUSINESS_CLOSURE，R为BUSINESS_RESUMPTION；每个新业务请求携带明确原因；独立HTTP夹具直接提供原因，页面唯一默认按117及129由Go给出。当前CHANGE原因纠错另用OTHER_DATA_ADJUSTMENT，停用另用ORG_RESTRUCTURING；旧目录原件单独用受控历史夹具。所有夹具先核验完整来源、准确修订和独立快照；建立方式及最终内容须留证。真实页面验收须通过已接通命令准备或使用可核验种子；直接SQL夹具只计存储/当前初始化验证，不能计对应业务流程已运行。

### 2.3 共同业务预期与各层断言

| 结果标记 | 适用的独立断言（观测责任见第2.4节） |
| --- | --- |
| W，真实保存 | 同一BU头只推进一次，恰一份完整新来源、汇总业务审计和Outbox；新记录获新ID，更新保留ID，删除原件保留。每个变化记录最多一个合并修订，未变成员复用原修订；联合source_change_item仅列直接实际变化项，派生变化另在完整来源/影响中核对 |
| N，无变化 | 技术结果为no_change，业务身份/修订/来源/审计/事件零新增；授权、日期、准确目标及引用仍先校验。当前固定两条不产生no_change；旧通用项no_change只读核验，不重开写入口 |
| R，业务拒绝 | 旧头、身份集合、所有旧来源/成员/业务日志/事件原样，无部分成功；可信请求可按013保存唯一拒绝及安全诊断。格式不可信、未认证等错误不编造终态；400/403/503本身不证明持久rejected或未提交 |
| U，结果待确认 | 保留原实例/主体/提交引用，不能换新身份盲重试；按同锁resolve及当前授权确定accepted/rejected/no_change/not_applied。只有可确认终态才解除待确认，权限失效不等于未保存 |
| H，历史与时间 | 每次写后读取T0及T1：在合法日期域内的各变化边界前一天、边界当天、下一边界前一天及D核对；最小/最大日期不得为取样而溢出。旧recordId/revision/source/分类名称/期间不被覆盖；recordedAt为实际时间，source_no按提交递增。原采用不重算，无真实消费者时只验合同或返回未接入 |
| UI，交互 | W定位实际目标及服务端结果：新增/改日定位新日，普通更正保持目标，删除展示对应原件及删除后事实，固定两条在详情直接展示正式节点，原件由日志定位。N说明无变化；R定位安全错误并保留可见输入；U提供查询结果。保留筛选/排序/分页/滚动/返回焦点，受限正文按当前授权清理 |

W/N/R/U及H表达各例的业务预期，UI仅由浏览器验证承接。Go验证候选、影响及应用结果；DB及实际API验证真实持久化、身份和历史；浏览器验证交互及通过服务端读回的事实。不能要求Go用例操作页面，也不能由Go返回正确结果推导DB原子性。实际执行按主要/补充层分配观测断言，跨例复用须注明同基线和实际输入/断言，不要求每层重复整组共同检查。

### 2.4 主要验证层、补充断言与证据

按[024](../../adr/024-timeline-implementation-simplification.md)为每例标记一个主要层（主）及有独有风险的补充层（补）；条件F表示需控制故障、时钟或并发，不是另一套完整执行层。未标出的层不列为该例待办，也不能记为已通过。主要层覆盖该行全部参数，补充层按下表的代表输入和特有断言执行。

| 层 | 观测责任 | 作为补充层时的范围 |
| --- | --- | --- |
| G | Go领域/应用独立例表，完整日期/状态/分类/权限选择及排列矩阵 | 验证特有计算，不重复DB/API往返 |
| D | 真实PostgreSQL的事务、约束、历史及锁；可通过应用服务驱动 | 首个实际变化及本行首个拒绝或无变化输入（若有），核对落库/回滚及旧源；日期矩阵取当天，不重跑全部日期格 |
| A | 连接真实服务/存储的API，严格DTO、当前授权及结果恢复 | 首个合法及首个非法输入（若有），核对协议/权限及绑定；具体缩放见下文 |
| B | 正常菜单桌面Chromium的办理、返回上下文及错误/恢复反馈 | 本行首个合法流程及与该UI风险直接相关的反馈；多输入例按下文明确，不重跑纯规则矩阵 |
| M | 当前模型的实际安装、受控初始化及启动失败，包含所需DB/API观测 | 只执行表内现行步骤，不另复制D/A矩阵；旧版本专项退役 |
| P | 记录环境的容量及取消测量，包含真实DB/API指标 | 只执行规定的规模/压力样本 |

补充范围的明确例外：TL-007的D覆盖两个合法日期极值；TL-011的B验证200码点含扩展汉字完整输入/回读、201码点反馈及创建/维护emoji拒绝；TL-013的B验证Go唯一默认、直接保存及原因名称快照；缺失/错配仍由A核验；TL-149的A取一个合法改型及缺原/新类别权各一例。TL-008的B覆盖换日前后两次显示；TL-168的A按四种专用权各取匹配成功及一种不匹配拒绝，TL-169按三种权同样取代表，并复用TL-075首条拒绝；完整交叉矩阵仍在G。TL-062的B仅补重新启用编辑中字段撤权、缓存清理和旧预览迟到；主G仍持有部分字段矩阵。TL-173的B只验省略英文可恢复。TL-174的B只验accepted丢响应后撤写权查回；读权收窄由TL-218的A核验。TL-201/202的B各验一个迟到未知响应。TL-226的B只验联合写跨午夜显示，单条/联合时钟组合在A。TL-248的A按行内三个等价入口和一个非法后继执行。

API或浏览器证据包含实际DB断言时，可供相关D案例引用，须能定位同一基线、输入和观测；不强制为同一断言重跑一次测试。参数和重复覆盖可引用同基线完整例表的具体行，不能只引用总通过率。不能用mock端点替代A/B，F中的引用替身仍只证明端口合同，未接消费者不计真实集成。

P0保护事实、权限、历史及恢复，P1覆盖交互和容量。案例关闭要求主要层全部参数与明确补充断言通过；切片关闭仍需各责任层真实证据，未运行/中断不算通过。每次记录基线、夹具、输入、独立预期、实际结果及层次；角色/时钟/UUID/菜单/事务前后等按该例实际风险记录，可共享有版本的夹具及证据，不逐例复制完整环境清单。

### 2.5 日期与转换矩阵的展开

标“[三时]”的行固定业务日期t及时间线，分别令D＝t后一天（.P）、t（.T）、t前一天（.F）。主要层独立准备三次，事实预期相同，今日状态和办理说明按各自D。更正/删除覆盖三时以防回到仅未来可维护；补充层按第2.4节代表输入，不自动再展开三倍。

改日“[九格]”固定D＝2026-06-15，旧日06-05/06-15/06-25，新日06-10/06-15/06-20，展开P→P、P→T、P→F、T→P、T→T、T→F、F→P、F→T、F→F。TL-090—093各在Go完整执行9格，仅T→T为N，其余W。首条用唯一首条，CHANGE用1月首条＋普通尾条，S用1月首条＋停用尾条，R用1月首条＋4月S＋恢复尾条。旧来源仍按旧日、新来源按新日解释，不能据现在状态决定可办。日期入库/查询由TL-017/230，四类菜单代表由TL-160及TL-095的B覆盖，不在DB/API/浏览器重复36格。

转换矩阵按日期前状态验证全部12格：NO_FACT只允许CREATE；ACTIVE只允许CHANGE、DEACTIVATE；INACTIVE只允许REACTIVATE。其余8格均拒绝；CREATE还必须是固定首条。日期唯一、企业范围与首条约束先于无变化消除，不能用同值项绕过同日占用。完整矩阵由对应案例展开，不能只测四条成功边。

<a id="tl-create"></a>
## 3 日期、身份与创建首条

A从列表“新建业务单元”办理；无特别说明均F0，编码使用该例独立值。 本章默认P0。

| 案例 | 前置与输入 | 操作步骤 | 明确预期（并执行第2.3节共同断言） | 验收依据 · 主层/补层 |
| --- | --- | --- | --- | --- |
| <a id="m02-tl-001"></a>TL-001 首条三类日期 | F0；首日03-01，[三时] | 选CREATE/NEW_ORGANIZATION，填A后保存并刷新 | W；只有固定首条，03-01前NO_FACT，自该日ACTIVE A；登记时间不倒填。 | [AC002](011-m02-business-requirements.md#m02-ac-002)、[AC013](011-m02-business-requirements.md#m02-ac-013)；主G |
| <a id="m02-tl-002"></a>TL-002 企业起点边界 | F0；首日2025-01-01及2024-12-31两例 | 分别新建 | 起点当天W；前一天R，无BU半成品；不自动改日期。 | [AC002](011-m02-business-requirements.md#m02-ac-002)、[AC003](011-m02-business-requirements.md#m02-ac-003)；主G |
| <a id="m02-tl-003"></a>TL-003 缺企业和有限企业 | 未初始化、歧义/有限终点上下文分别隔离准备 | 尝试初始化/启动，分别核对未初始化Open拒绝、歧义/有限企业DDL拒绝 | 阻断；不选第一条、不造匿名企业、不把开放BU期间放到有限企业外。当前模型在业务API开放前失败关闭；不禁用约束伪造运行上下文。 | [AC002](011-m02-business-requirements.md#m02-ac-002)、[AC011](011-m02-business-requirements.md#m02-ac-011)；主M |
| <a id="m02-tl-004"></a>TL-004 未来首条可发现 | F0；首日07-01，D=06-15 | 创建B，返回默认列表；受控服务端时钟分别取06-15/07-01读取 | W；默认列表可见未生效B，06-15无适用事实，07-01启用；浏览器无查询日期输入。 | [AC001](011-m02-business-requirements.md#m02-ac-001)、[AC002](011-m02-business-requirements.md#m02-ac-002)、[AC007](011-m02-business-requirements.md#m02-ac-007)；主B |
| <a id="m02-tl-005"></a>TL-005 闰日与大小月 | 首日2028-02-29、2026-02-29、2026-04-31 | 每个日期独立创建 | 仅闰年真实日W，后两者400拒绝；不归一到下月。 | [AC002](011-m02-business-requirements.md#m02-ac-002)、[AC003](011-m02-business-requirements.md#m02-ac-003)；主G；补B |
| <a id="m02-tl-006"></a>TL-006 日期格式拒绝 | 2026-6-1、时间戳、数字、首尾空白、空串、null、缺日期、带时区字符串 | 逐值直调并核对页面输入 | 400且业务零写入；不取浏览器今日、不截断时间字符串。 | [AC002](011-m02-business-requirements.md#m02-ac-002)、[AC003](011-m02-business-requirements.md#m02-ac-003)、[AC013](011-m02-business-requirements.md#m02-ac-013)；主A |
| <a id="m02-tl-007"></a>TL-007 日期上下界 | E起点0001-01-01；首日0001-01-01或9999-12-31；另传0000及10000年 | 分别新建并读首日和开放末段 | 合法极值W，开放上界无界；不生成10000年或将9999-12-31当有限终点；非法年拒绝。 | [AC002](011-m02-business-requirements.md#m02-ac-002)、[AC003](011-m02-business-requirements.md#m02-ac-003)、[AC014](011-m02-business-requirements.md#m02-ac-014)；主G；补D |
| <a id="m02-tl-008"></a>TL-008 时区换日 | 数据库时刻2026-06-14T15:59:59Z及16:00:00Z | 默认查询并获取日期约束 | 业务日分别06-14/06-15；浏览器不同时区结果一致，不用UTC日替代。 | [AC001](011-m02-business-requirements.md#m02-ac-001)、[AC003](011-m02-business-requirements.md#m02-ac-003)、[AC024](011-m02-business-requirements.md#m02-ac-024)；主G；补B；条件F |
| <a id="m02-tl-009"></a>TL-009 编码规范与同名 | 编码“ Ab_1-2 ”及“ab_1-2”；另用不同编码和同名A | 先保存首个，再试后两者；停用/删非首条后再试重复编码 | 首个去空白后W，大小写重复409；不同编码同名W；停用/删记录也不释放编码。 | [AC002](011-m02-business-requirements.md#m02-ac-002)；主G；补D |
| <a id="m02-tl-010"></a>TL-010 编码边界 | 1/40/41字符、非法首字符、中文、内部空白、空值 | 逐项创建 | 仅符合ASCII格式且1—40字符通过；错误定位编码，不截断或重新编码。 | [AC002](011-m02-business-requirements.md#m02-ac-002)；主G |
| <a id="m02-tl-011"></a>TL-011 名称码点与归一 | NFC后1/200/201码点、扩展汉字、重音组合、多语言/标点、emoji序列及文字连接上下文 | 创建/ADD/CORRECT共用Go；G手写全表，A校验三入口/预览及零业务写入，B覆盖菜单代表与历史 | 合法文字按NFC保存/回读，200通过201拒绝；普通数字与4字节汉字允许；emoji/旗帜/肤色/键帽/非法不可见/纯标点拒绝，合法ZWNJ/virama/文字变体保留，NFC等价更正无新事实；原历史不清洗。 | [AC002](011-m02-business-requirements.md#m02-ac-002)、[AC004](011-m02-business-requirements.md#m02-ac-004)；主G；补A、B |
| <a id="m02-tl-012"></a>TL-012 英文空值与非法文本 | nameEn缺席/null/空串/仅Zs空格；首尾/内部制表换行、Zl/Zp、无效UTF-8及非法名称字符 | 分别保存或直调 | 可选空值归null且只显示回落，不持久中文为英文；非法文本拒绝，无截断修复。 | [AC002](011-m02-business-requirements.md#m02-ac-002)、[AC004](011-m02-business-requirements.md#m02-ac-004)、[AC013](011-m02-business-requirements.md#m02-ac-013)；主A；补D |
| <a id="m02-tl-013"></a>TL-013 分类必选与入口一致 | F0；缺原因、未知原因、NAME_ADJUSTMENT原因、operationType=S | 逐项尝试，再用Go默认的NEW_ORGANIZATION直接创建 | 错配/缺失拒绝；NEW_ORGANIZATION合法W。单项目录按Go默认，多项不擅选；服务端生成名称快照。 | [AC002](011-m02-business-requirements.md#m02-ac-002)、[AC013](011-m02-business-requirements.md#m02-ac-013)；主A；补B |
| <a id="m02-tl-014"></a>TL-014 伪造服务器字段 | 提交BU ID、企业、recordId、state、firstRecordId、recordedAt、名称快照或note | 逐字段向创建DTO追加 | 400，业务零写入；不能创建直接停用的首条或修改归属。 | [AC002](011-m02-business-requirements.md#m02-ac-002)、[AC013](011-m02-business-requirements.md#m02-ac-013)；主A |
| <a id="m02-tl-015"></a>TL-015 首次创建原子性 | F0；事实/判定/事件写入分别失败 | 提交后重启，按编码、身份及提交查回 | 失败无无首条BU；成功必须身份/首条/来源/结果一致，重试不重复创建。 | [AC002](011-m02-business-requirements.md#m02-ac-002)、[AC010](011-m02-business-requirements.md#m02-ac-010)、[AC012](011-m02-business-requirements.md#m02-ac-012)、[AC025](011-m02-business-requirements.md#m02-ac-025)；主D；条件F |
| <a id="m02-tl-016"></a>TL-016 初始化幂等与歧义 | 相同实例企业/管理员初始化输入；另改企业起点或编码 | 重复受控初始化后访问菜单 | 相同输入返回原身份，异内容拒绝；不复制企业、不覆盖BU上下文。 | [AC002](011-m02-business-requirements.md#m02-ac-002)、[AC011](011-m02-business-requirements.md#m02-ac-011)；主M |

<a id="tl-query"></a>
## 4 按日查询、来源历史与只读定位

A/R从今日列表进入详情，选择生效记录或确切来源；读取不产生业务来源。需要断言任意日期事实的G/D案例调用内部按日解析，A/B边界案例使用受控服务端时钟，不恢复任意日期HTTP或页面入口。本章默认P0。

| 案例 | 前置与输入 | 操作步骤 | 明确预期（并执行第2.3节共同断言） | 验收依据 · 主层/补层 |
| --- | --- | --- | --- | --- |
| <a id="m02-tl-017"></a>TL-017 半开期间与停启边界 | F3 | 依次查03-31/04-01/07-31/08-01/09-30/10-01/12-01 | 分别ACTIVE A、INACTIVE A、INACTIVE A、ACTIVE B、ACTIVE B、ACTIVE C、INACTIVE C；边界无空隙或双命中。 | [AC001](011-m02-business-requirements.md#m02-ac-001)、[AC008](011-m02-business-requirements.md#m02-ac-008)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G；补D |
| <a id="m02-tl-018"></a>TL-018 首条前无事实 | F6 | 查06-30和07-01，选择未来首条 | 前者NO_FACT且未来B明确标记；后者ACTIVE B；NO_FACT不是INACTIVE或不存在BU。 | [AC001](011-m02-business-requirements.md#m02-ac-001)、[AC008](011-m02-business-requirements.md#m02-ac-008)、[AC014](011-m02-business-requirements.md#m02-ac-014)；主A |
| <a id="m02-tl-019"></a>TL-019 今日状态筛选 | F3及F6 | 普通列表切全部/启用/停用/未生效；受控服务端D在06-15、08-15补验边界 | 状态按服务端D及当前来源判断，停用对象默认可见；计数与分页不含越权BU。 | [AC001](011-m02-business-requirements.md#m02-ac-001)、[AC008](011-m02-business-requirements.md#m02-ac-008)、[AC009](011-m02-business-requirements.md#m02-ac-009)；主B |
| <a id="m02-tl-020"></a>TL-020 选中记录与办理定位 | F2，D=06-15，显式选中10月c2 | 打开新增/改日表单后取消 | c2名称/状态/期间统一表达该记录；普通新增默认10-02，改日仍带原10-01；不能改为今日适用c1。仅初始带值取消不确认，实际编辑才保护。 | [AC001](011-m02-business-requirements.md#m02-ac-001)、[AC003](011-m02-business-requirements.md#m02-ac-003)、[AC007](011-m02-business-requirements.md#m02-ac-007)；主B |
| <a id="m02-tl-021"></a>TL-021 未来首条旧来源 | F6的B创建为T0，再更正为C得到T1 | 查T0与T1在06-15的未来展示 | T0未来首条仍B，T1为C，两者当日事实均NO_FACT；不混入当前值。 | [AC008](011-m02-business-requirements.md#m02-ac-008)；主D |
| <a id="m02-tl-022"></a>TL-022 更正删除多来源答案 | F1新增03-01 B得T1，更正为C得T2，删除得T3 | 各来源查03-15及原记录 | T0为A、T1为B、T2为C、T3为A；被删原件仍可在原来源只读定位，分类按该版解释。 | [AC004](011-m02-business-requirements.md#m02-ac-004)、[AC005](011-m02-business-requirements.md#m02-ac-005)、[AC008](011-m02-business-requirements.md#m02-ac-008)、[AC013](011-m02-business-requirements.md#m02-ac-013)；主D |
| <a id="m02-tl-023"></a>TL-023 默认来源与确切来源 | F2后更正c1为Z | 同一次页面分别查看默认当前和T0 | 当前06-15为Z，T0为B；返回当前需显式重读权限/版本，不把旧版变可编辑。 | [AC008](011-m02-business-requirements.md#m02-ac-008)、[AC009](011-m02-business-requirements.md#m02-ac-009)；主A |
| <a id="m02-tl-024"></a>TL-024 来源不存在与对象越权 | 已授权BU内随机来源；X读取同一BU | 分别请求确切来源 | 授权者得到M02_SOURCE_NOT_FOUND；X安全404且不泄露版本/名称，均不回退最新。 | [AC008](011-m02-business-requirements.md#m02-ac-008)、[AC009](011-m02-business-requirements.md#m02-ac-009)；主A |
| <a id="m02-tl-025"></a>TL-025 查询格式不允许降级 | sourceVersion空串/重复/非法UUID、日期空值、畸形转义 | 逐个请求并观察页面缓存 | 400；不能删掉错误参数后展示当前来源，旧正文清空并提供显式返回入口。 | [AC008](011-m02-business-requirements.md#m02-ac-008)；主A |
| <a id="m02-tl-026"></a>TL-026 退役截点与采用未接入 | 请求knownAt或adoptedRef | 分别通过API及旧链接访问 | knownAt非法输入；adoptedRef明确能力未接入，不拿当前来源或空值冒充采用结果。 | [AC008](011-m02-business-requirements.md#m02-ac-008)、[AC011](011-m02-business-requirements.md#m02-ac-011)；主A |
| <a id="m02-tl-027"></a>TL-027 一次响应来源一致 | F2查询过程中另一连接更正c1 | 以屏障安排详情/列表读写交错 | 每个响应的成员、事实、能力及计数来自一致快照；只见完整旧版或新版，无混版。 | [AC001](011-m02-business-requirements.md#m02-ac-001)、[AC008](011-m02-business-requirements.md#m02-ac-008)、[AC010](011-m02-business-requirements.md#m02-ac-010)；主D；条件F |
| <a id="m02-tl-028"></a>TL-028 审计时刻相同或回退 | 连续三次合法写，Clock给同刻再回退 | 分页读取日志及确切来源 | source_no/前驱维持提交顺序；时间如实展示，不据时钟拒绝合法保存或重排历史。 | [AC008](011-m02-business-requirements.md#m02-ac-008)、[AC012](011-m02-business-requirements.md#m02-ac-012)、[AC024](011-m02-business-requirements.md#m02-ac-024)；主D；条件F |
| <a id="m02-tl-029"></a>TL-029 原记录从历史返回当前 | F2在T0选c1，当前已删除c1 | 返回当前再点维护 | 不对已删除目标办理；安全说明差异，读取当前可见记录；原来源链接仍准确。 | [AC005](011-m02-business-requirements.md#m02-ac-005)、[AC007](011-m02-business-requirements.md#m02-ac-007)、[AC008](011-m02-business-requirements.md#m02-ac-008)；主B |
| <a id="m02-tl-030"></a>TL-030 排序搜索与字段权限 | 多BU同名、含%/_的名称及200码点名称；CN缺英文读权 | 搜索字面值、翻页、排序；尝试受限字段搜索 | 字面子串不变LIKE模式；页码/顺序稳定，400码点查询上限；无权字段不参与搜索、排序或提示。 | [AC001](011-m02-business-requirements.md#m02-ac-001)、[AC009](011-m02-business-requirements.md#m02-ac-009)；主A；补B |
| <a id="m02-tl-031"></a>TL-031 保存不等于未来到日 | F1已排07-01 S、08-01 R | 只推进Clock跨两边界并重查 | 状态按日自然变停用/启用；无第二次写入、来源或事件，不需要到日任务。 | [AC001](011-m02-business-requirements.md#m02-ac-001)、[AC008](011-m02-business-requirements.md#m02-ac-008)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主B；条件F |
| <a id="m02-tl-032"></a>TL-032 读取失败与历史权限撤销 | F2旧来源读取中断；另撤销history权 | 重试读取并恢复权限 | 暂不可读不回退最新；撤权后清正文，恢复后按原确切来源读，原件与业务结果不变。 | [AC008](011-m02-business-requirements.md#m02-ac-008)、[AC009](011-m02-business-requirements.md#m02-ac-009)、[AC010](011-m02-business-requirements.md#m02-ac-010)；主B；条件F |

<a id="tl-add"></a>
## 5 资料新增、追加与追溯插入

A先从当前详情选择位置，再通过新增记录或插入记录办理，类型固定CHANGE，原因初始为空待选NAME_ADJUSTMENT；服务端按该位置自动带出资料及日期。普通ADD显式锚点按第2.1节共同输入补齐；本章默认P0。

| 案例 | 前置与输入 | 操作步骤 | 明确预期（并执行第2.3节共同断言） | 验收依据 · 主层/补层 |
| --- | --- | --- | --- | --- |
| <a id="m02-tl-033"></a>TL-033 开放尾段追加 | F1；03-01新增B，[三时] | 确认日期及名称并保存 | W；01-01—02-28 A，03-01起B；旧T0始终A，只有新记录ID，首条区间派生修订。 | [AC003](011-m02-business-requirements.md#m02-ac-003)、[AC013](011-m02-business-requirements.md#m02-ac-013)；主G |
| <a id="m02-tl-034"></a>TL-034 已结束区间插入 | F2；03-01只改中文为新，[三时] | 预览并保存 | W；03-01—05-31 新/Alpha；6月B/Beta、10月C/Gamma保持，不以今日B作基准。 | [AC003](011-m02-business-requirements.md#m02-ac-003)、[AC019](011-m02-business-requirements.md#m02-ac-019)；主G |
| <a id="m02-tl-035"></a>TL-035 当天与未来中段 | F2；分别06-15、08-01新增Z | 各自从T0独立保存并查边界 | W；新日到09-30为Z，10月C不变；08月新增不提前改变06-15的B。 | [AC003](011-m02-business-requirements.md#m02-ac-003)、[AC024](011-m02-business-requirements.md#m02-ac-024)；主G |
| <a id="m02-tl-036"></a>TL-036 当前停用但历史合法 | F3；03-01新增B | 保存并查3月、4月、8月 | W；3月B，4—7月停用继承B；8月恢复B独立、10月C保持，不因今日停用禁用整个入口。 | [AC003](011-m02-business-requirements.md#m02-ac-003)、[AC006](011-m02-business-requirements.md#m02-ac-006)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G |
| <a id="m02-tl-037"></a>TL-037 恢复后再次更名 | F3；09-01新增Z | 保存并读各段 | W；8月B、9月Z、10月C；曾有停用不是统一拒绝理由。 | [AC003](011-m02-business-requirements.md#m02-ac-003)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G |
| <a id="m02-tl-038"></a>TL-038 停用段内新增 | F3；06-01新增Z | 预览及强制正式提交 | 预览invalid/不可保存；正式R M02_TIMELINE_INVALID，无隐式恢复或改状态。 | [AC003](011-m02-business-requirements.md#m02-ac-003)、[AC006](011-m02-business-requirements.md#m02-ac-006)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G |
| <a id="m02-tl-039"></a>TL-039 首条与占用日 | F2；2025-12-31、01-01、06-01；F3的04-01/08-01 | 逐日新增实际差异 | 首条前/首条日/已有资料或启停日全部R；不生成同日序号。 | [AC003](011-m02-business-requirements.md#m02-ac-003)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G |
| <a id="m02-tl-040"></a>TL-040 同值新增不占日期 | F2；03-01显式输入A | 提交后以新身份同日输入Z | 首次N无新recordId，第二次W；不是删除或恢复虚构记录。 | [AC003](011-m02-business-requirements.md#m02-ac-003)、[AC016](011-m02-business-requirements.md#m02-ac-016)、[AC023](011-m02-business-requirements.md#m02-ac-023)；主G |
| <a id="m02-tl-041"></a>TL-041 同值也执行守卫 | F2在06-01同值B；另用旧来源、越权字段同值 | 逐项独立提交 | 分别同日冲突/版本冲突/权限拒绝；不能先做no_change短路。 | [AC003](011-m02-business-requirements.md#m02-ac-003)、[AC009](011-m02-business-requirements.md#m02-ac-009)、[AC010](011-m02-business-requirements.md#m02-ac-010)、[AC023](011-m02-business-requirements.md#m02-ac-023)；主G |
| <a id="m02-tl-042"></a>TL-042 字段省略与清空 | F2；03-01只改中文，另例显式nameEn=null | 分别保存 | 省略英文继承Alpha，显式null清空；后继6月Beta保持，中文完整非空。 | [AC003](011-m02-business-requirements.md#m02-ac-003)、[AC019](011-m02-business-requirements.md#m02-ac-019)；主G |
| <a id="m02-tl-043"></a>TL-043 缺changes与受限输入 | 缺changes、空对象、name=null、带state/note/类型名称快照 | 逐项直调 | 400拒绝；不把隐藏字段当空值、不忽略额外字段后成功。 | [AC003](011-m02-business-requirements.md#m02-ac-003)、[AC009](011-m02-business-requirements.md#m02-ac-009)、[AC013](011-m02-business-requirements.md#m02-ac-013)；主A |
| <a id="m02-tl-044"></a>TL-044 补登记的真实保存时间 | F2；D=11-01时补03-01 Z | 查新source_no、recordedAt及旧来源 | W；业务始于3月但登记在11月，原采用和旧来源保持；不伪造过去登记。 | [AC003](011-m02-business-requirements.md#m02-ac-003)、[AC008](011-m02-business-requirements.md#m02-ac-008)、[AC024](011-m02-business-requirements.md#m02-ac-024)；主D；条件F |
| <a id="m02-tl-045"></a>TL-045 插入前后引用范围 | F2；插03-01 Z；受控依赖位于04-15及未变11月 | 分别允许/阻断所需字段或资格 | 校验完整03-01—06-01影响；仅历史快照不阻断，真正不满足的引用R；替身仅计合同证据。 | [AC003](011-m02-business-requirements.md#m02-ac-003)、[AC011](011-m02-business-requirements.md#m02-ac-011)、[AC021](011-m02-business-requirements.md#m02-ac-021)；主G；条件F |

<a id="tl-correct"></a>
<a id="m02-tl-position-addition"></a>
### 5.1 按所选位置新增及插入增量

以下24项均关联[AC030](011-m02-business-requirements.md#m02-ac-030)、[BR005](011-m02-business-requirements.md#m02-br-005)、[页面位置规则](012-m02-functional-and-interaction-design.md#m02-ui-position-addition)及[技术合同](013-m02-technical-design.md#m02-tech-position-addition)。105已接入产品并分层执行本组主要位置、输入及历史参数，逐项覆盖及未穷举参数见[105验证](../../verification/105-positioned-addition-and-coverage.md)；不沿用旧TL或102结果宣称通过，既有253例保留原历史执行状态。层次采用第2.4节G/D/A/B约定，独立时钟与日期预期不可由被测逻辑生成。

| 案例 | 前置与输入 | 操作 | 独立预期 | 追踪与层次 |
| --- | --- | --- | --- | --- |
| <a id="m02-tl-258"></a>TL-258 位置与标签 | F2；普通进入后分别选f/c1/c2；另最后更正f | 从正常菜单观察按钮及打开表单 | f/c1为插入记录，c2为新增记录；不按今日适用位置或f最新登记时刻判断；表单标题一致 | AC030；主B |
| <a id="m02-tl-259"></a>TL-259 过去末条默认 | F1，D=06-15；锚点f=01-01 | 打开新增，再手改日期03-01并填Z/选择原因保存 | 默认06-15而非01-02；合法过去追加仍成功，选中结果记录 | AC030；主B；补G核验max公式 |
| <a id="m02-tl-260"></a>TL-260 当天末条默认 | 独立样本末条06-15；D=06-15 | 准备并保存次日Z | 默认06-16、最早也是06-16；不占用同日 | AC030；主G；补A核验上下文/保存同源 |
| <a id="m02-tl-261"></a>TL-261 未来末条与未来首条 | 末条07-01及独立首条2028-02-29，D仍2026-06-15 | 分别打开新增 | 默认07-02及2028-03-01；未到首条日仍NO_FACT，未把默认新增当作已保存 | AC030；主B；补G核验日期 |
| <a id="m02-tl-262"></a>TL-262 插入区间及默认 | F2选择c1=06-01，R=c2=10-01 | 准备上下文，选择06-02/09-30分别独立保存Z | 默认06-02，闭区间06-02—09-30；中文乙/英文Beta来自c1，后继C保持 | AC030；主G；补B核验首屏预填和控件边界 |
| <a id="m02-tl-263"></a>TL-263 无空档只禁用 | 两条当前记录06-01及06-02，选择前条 | 正常页面鼠标/键盘访问插入入口 | 插入记录仅禁用；无原因文字、tooltip、弹窗或全灰日历，不进入编辑、不产生预览/写入；读权与历史不扩大 | AC030；主B |
| <a id="m02-tl-264"></a>TL-264 仅一个可用日 | 两条当前记录06-01及06-03，选择前条 | 打开并填Z、选原因后保存 | 只允许06-02且默认该日，其他日期手输亦不能保存 | AC030；主B；补A核验绕过UI |
| <a id="m02-tl-265"></a>TL-265 日历边界 | 分别令L为2026-01-31、2026-12-31、2028-02-28、2028-02-29；插入R取L之后第三日，追加D不晚于L | 独立核验默认及前后边界 | 次日依次02-01、2027-01-01、2028-02-29、2028-03-01；上界为R前一日；使用日历而非客户端24小时；无可表示次日不可用 | AC030；主G；补A取一例回读 |
| <a id="m02-tl-266"></a>TL-266 资料默认与分类排除 | F1首条CREATE及F3的08-01 REACTIVATE分别作为锚点 | 打开表单再修改一个名称保存 | 带出对应A/B，类型固定CHANGE，原因空且未选不得提交；不复制CREATE/REACTIVATE原因、ID、修订或时刻；保存有真实新身份 | AC030；主B；补A核验请求白名单 |
| <a id="m02-tl-267"></a>TL-267 所选位置决定默认 | F2，D=06-15，选c1=06-01 | 打开插入、取消，切f后再选c1重进 | 资料为B、默认06-02，范围至09-30；不取今日或末条C/10-02 | AC030；主B |
| <a id="m02-tl-268"></a>TL-268 初始带值取消 | 两种入口；等待准备和自动辅助到达 | 直接取消、返回列表、退出；分别独立样本 | 没有未保存输入弹窗或beforeunload保护；无事实写入，原查看/返回/焦点保持 | AC030；主B |
| <a id="m02-tl-269"></a>TL-269 实际编辑与恢复初值 | 两种入口的初始日期/名称/空原因 | 逐一编辑日期、名称、原因，取消选择继续；再恢复全部初始值后退出 | 真实修改受保护且输入保留，完全恢复后可直接退出；不把默认或自动加载当dirty | AC030；主B |
| <a id="m02-tl-270"></a>TL-270 迟到默认不覆盖输入 | 准备/自动辅助响应受控延迟；另切所选位置 | 先手填名称/日期，再放行响应；切换位置处理真实未保存输入 | 不覆盖手工值、不把旧来源默认带回新位置；选择继续留原处，放弃后才加载新位置 | AC030；主B；条件F |
| <a id="m02-tl-271"></a>TL-271 复制但无变化 | F2选f及独立选c1；原因明确选择，资料不改；另日期改为区间内其他合法日 | 提交changes={}及显式同值两种输入；另验证英文清空 | 守卫通过后N，无新ID/修订/来源；仅原因或日期不同不制造记录；空原因仍拒绝，英文原非空后清空属于实际变化 | AC030；主A；补G核验域内no_change |
| <a id="m02-tl-272"></a>TL-272 字段和位置权限 | CN/EW/X及缺日期读权角色；当前/历史分别读取 | 准备默认、撤读权后继续保存；伪造显式受限字段 | 不可读英文/日期/范围不泄露；可读不可写字段只读带出且未编辑不伪造写入声明；撤权清缓存，服务端拒绝越权 | AC030；主A；补B核验清理和受限表单 |
| <a id="m02-tl-273"></a>TL-273 日期空档与状态前提 | F3选04-01停用、选01-01启用、选08-01恢复；另F6首条之前 | 尝试普通新增；X/R及旧来源另核对入口 | 停用段有空档也不能CHANGE；此前启用段可插入，恢复后可新增；首条之前先更正首条，后续资料另次新增/插入，旧来源/R不开放写入 | AC030；主G；补B核验禁用与只读入口 |
| <a id="m02-tl-274"></a>TL-274 日期及企业边界不可绕过 | F2选c1；按当前支持企业范围及日期域 | 手输/直调API提交L、R、区间外、已占日、首条之前、企业范围外或非法日历日期 | 即使日期在BU另一启用段也拒绝该位置越界；无部分事实、不自动改位置；安全字段错误仍可读 | AC030；主A |
| <a id="m02-tl-275"></a>TL-275 显式锚点强制 | 有权BU当前来源；缺失/畸形/随机/其他BU或实例/已删除的anchorRecordId | 用合法日期和资料分别调用准备及保存；无权父BU另测 | 不容忍旧无锚点格式或按日期猜锚点；非本BU当前成员统一安全400定位anchorRecordId，父BU越权安全404，不泄露外部对象存在性 | AC030；主A |
| <a id="m02-tl-276"></a>TL-276 并发插入改变后继 | F2选f，A/B都持T0；A先在03-01保存Z | B仍按T0在04-01提交；随后显式重读 | B来源冲突，无第二次落库或静默挪段；重读后f的R变03-01，新合法范围/默认一致，用户核对后另次办理 | AC030；主D；补B核验重读后的输入保护 |
| <a id="m02-tl-277"></a>TL-277 并发改日或删除锚点 | F2选c1；另一会话分别将c1改日、删除c1或改动c2日期 | 原表单带T0提交并尝试只换版本，随后显式重读 | 原T0拒绝；新来源下锚点不存在/范围不符仍拒绝，不自动选另一条；仍有权输入保留，重读后需用户核对 | AC030；主D |
| <a id="m02-tl-278"></a>TL-278 幂等与原结果定位 | 在合法位置完成ADD，持原key及正文，后续来源已推进 | 原内容重试/resolve；同key只换anchorRecordId或日期 | 原内容返回原终态/新记录定位，不重新计算新默认；异内容M02_REQUEST_MISMATCH且无新事实 | AC030；主A |
| <a id="m02-tl-279"></a>TL-279 跨午夜保留原日期 | 准备时D=06-15、末条更早；带出06-15并选原因输入Z | 未接受请求跨至06-16提交；已成功原key再跨日重试 | 仍允许原06-15过去日期，不强制换为新D；原接受说明及结果保持，只产生一次事实 | AC030；主D；补G受控Clock |
| <a id="m02-tl-280"></a>TL-280 后继与停用继承 | F3选f，于03-01插入Z | 保存后读新来源03月、04月停用、08月恢复、10月资料，再读T0 | 新增Z独立快照；04月停用正确继承Z且保留停用分类；08月B、10月C独立快照保持，T0原答案/引用保持 | AC030；主D；补B正常菜单读回 |
| <a id="m02-tl-281"></a>TL-281 命名定位及可达 | 两种入口，共享桌面视口。选位置后编辑、保存或取消的关键流程实测原生 200% | 从菜单选择位置、编辑、取消/保存，刷新及返回；查看日志及原结果 | 普通按钮/标题/帮助一致新增记录或插入记录，生效日期/适用期间保留；事件仍ADD/CHANGE；选中新记录、列表上下文保持，键盘滚轮可达，无额外预览/确认 | AC030；主B；补A核验结果/事件语义 |

<a id="m02-tl-unified-correction"></a>
### 5.2 104统一更正记录增量

承接[AC031](011-m02-business-requirements.md#m02-ac-031)，保留所有旧ID及原证据。104原批次曾把跨节点成功参数移至通用联合；109进一步退出该能力，TL293明确退役，普通表单/API仅原位置范围。104原批次不含103，位置新增后由105接线；当前参数分别见105/108。统一更正原分层范围见[104报告](../../verification/104-unified-record-correction.md)。

| 案例 | 前置及步骤 | 独立预期 | 主要层 |
| --- | --- | --- | --- |
| <a id="m02-tl-282"></a>TL-282 统一入口及初值 | 首条及F2选中c1，从正常菜单打开更正记录，直接读取类型/原因后取消；补目录缺原值/加载失败参数 | 旧改日/分类菜单及业务分类折叠按钮移除；分类仅一组操作类型/操作原因，无原/新重复标签；有权字段直接可达并保留原值，固定类型或仅有读权时只读、无读权逐字段隐藏；初值及目录加载不算修改，无离开确认；目录缺原值或失败不阻断仅名称保存 | B |
| <a id="m02-tl-283"></a>TL-283 组合一次保存 | F2非首条c1名称改B2、日期07-01、CHANGE原因由NAME_ADJUSTMENT改OTHER_DATA_ADJUSTMENT | 同recordId一次CORRECT来源/事件，首条A/后继C保持，日志实际三面差异，旧T0保持 | D/A/B |
| <a id="m02-tl-284"></a>TL-284 日期边界 | F2 c1日期01-02或09-30可选，01-01/10-01占用或11-01越界；首条企业下界、末条开放、闰日 | 普通日期保持原邻居位置，原日可保留；非法组合无任何部分保存，日期字段可定位 | G/A/B |
| <a id="m02-tl-285"></a>TL-285 仅原日可选 | 06-01/02/03相邻，选中06-02 | 日期只读而名称可保存，无全灰日历或额外确认 | B |
| <a id="m02-tl-286"></a>TL-286 单面权限 | 无联合权，仅名称/日期/分类之一及其字段写权 | 有相应更正入口，该面成功；其他面403，不借合并入口提权 | A/D |
| <a id="m02-tl-287"></a>TL-287 名称与类型转换 | CHANGE先改名称再转DEACTIVATE；原DEACTIVATE转CHANGE | 前者明确移除名称或保留草稿，最终停用继承前序；后者默认原目标自身解析资料并可显式调整；旧源保持 | A/B |
| <a id="m02-tl-288"></a>TL-288 分类及依赖整笔核验 | 首条仅CREATE原因；非首条禁CREATE；S/R中单改S为CHANGE | 原/新类别权及字段权核验，非法链整笔拒绝；不提供关联入口或携草稿编排修复 | A/B |
| <a id="m02-tl-289"></a>TL-289 初值同值及英文清空 | 仅默认/目录加载、类型及原因改后恢复、同值命令、显式清空英文；恢复分类后仅更正名称 | 默认不算修改；无变化不建业务来源，恢复原分类可直接退出，仅名称请求不含classification且无分类变化日志；不根据名称或日期匹配分类；真实日期/分类仍算变化，英文清空不覆盖后继快照 | A/B |
| <a id="m02-tl-290"></a>TL-290 冲突重读 | 名称草稿期间他人更正日期/分类，提交冲突后重读 | 保留用户名称，未编辑日期/分类更新到新源，下一次仅提交名称，不回写旧值 | B |
| <a id="m02-tl-291"></a>TL-291 动态撤权 | 日期草稿撤读权、撤写权但仍可读 | 撤读清除日期输入/辅助/父缓存；撤写保留草稿，明确移除该面后仍可保存名称 | B |
| <a id="m02-tl-292"></a>TL-292 丢响应恢复 | 三面CORRECT实际保存后阻断响应，重试或刷新查回 | 原submission/source/revision和原新日期，不重复来源；后续演进不改变原结果 | D/A/B |
| <a id="m02-tl-293"></a>TL-293 保留跨节点联合（109退役） | F2 c1改11-01普通拒绝，携相同草稿到现有联合UPDATE | 联合允许合法重排为A/C/B，c1身份及B快照保持，旧源A/B/C保持；联合入口文案/通用项交互不变 | A/B；109退出通用编排，本例退役，不执行或计当前通过 |

<a id="m02-tl-correction-coverage"></a>
### 5.3 105统一更正覆盖补全

根据[105覆盖评估](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/105-positioned-addition-and-coverage/initial-coverage-audit.json)追加以下独立预期；不重用旧通过记录填充未执行参数，实际结果另见[105验证](../../verification/105-positioned-addition-and-coverage.md)。103的TL-258—281在同批实施，104原始证据保持。

| 案例 | 前置与操作 | 独立预期 | 主要层 |
| --- | --- | --- | --- |
| <a id="m02-tl-294"></a>TL-294 更正闭区间及历法落库 | F2中段分别改到前条次日/后条前日；首条到企业起点、末条远未来；闰日/月末/跨年及越界分别独立验证 | 范围端点本身可用，邻居当日及范围外拒绝；同recordId、旧源及后继保持，不仅检查控件min/max字符串 | G/A |
| <a id="m02-tl-295"></a>TL-295 七种更正组合无需联合权 | 撤去联合入口权，保留实际名称/日期/分类权；三面所有非空子集分别提交 | 7种均成功，每次一来源/事件；未选修改面保持，独立后继和旧源保持 | D/A |
| <a id="m02-tl-296"></a>TL-296 同值声明仍需字段权 | 名称草稿额外声明原日/原分类，分别撤相应写权；日期草稿额外声明同值名称另测 | 显式同值也403且无部分事实；省略无权面后合法修改可保存，不按实际差异缩减授权 | A |
| <a id="m02-tl-297"></a>TL-297 新修改面严格协议 | 空/缺失修改面、null对象、缺类型或原因、分类额外/重复键、非法日期、CORRECT混入锚点 | 400；无新来源、业务修订或事件，不把空对象与缺席混同 | A |
| <a id="m02-tl-298"></a>TL-298 组合辅助故障直接保存 | 正常菜单改CHANGE记录名称、日期和原因，辅助接口受控503 | 无预览按钮，保存仍可用；最终Go命令验证后一次成功，旧源保持；不靠辅助许可解锁 | B/A |
| <a id="m02-tl-299"></a>TL-299 单面授权页面可达 | 分别只有名称、日期、分类写权且无联合入口权，从正常菜单办理 | 对应面可编辑，其余只读/不显示；一处更正入口可保存，请求只包含实际面，无关联入口 | B/A |
| <a id="m02-tl-300"></a>TL-300 分类动态读写撤权 | 三面草稿后撤分类读权/写权，保持名称及日期权；补普通新增的原因选择保护 | 撤读清分类输入和父缓存；撤写保留可读草稿供明确移除，类型转换草稿的标签必须是真实新类型；仅一项有读权时只读显示该项；其余名称/日期仍可保存，未回写旧分类；分类候选清理不得清空普通新增有权选择的原因目录 | B/A |
| <a id="m02-tl-301"></a>TL-301 日期分类草稿冲突重读 | 用户改日期/分类，他人仅改名称；原命令冲突后重读 | 用户日期/分类保留，未编辑名称更新到最新；下次只提交日期/分类，不覆盖他人名称；与TL-290双向互补 | B/A |

### 5.4 107首条通用维护增量

按[107决策](../../adr/107-first-record-maintenance.md)追加以下参数，并迁移TL-129—131/251为普通入口；旧TL-132保留原意并退役，不计通过。测试设计与实际执行分别由本稿及[107报告](../../verification/107-first-record-maintenance.md)持有。

| 案例 | 前置与步骤 | 独立预期 | 主层/补层 |
| --- | --- | --- | --- |
| <a id="m02-tl-302"></a>TL-302 专用入口与选择移除 | 从正常菜单选唯一首条及有后继的首条，进入更正，前移并改名 | 无补全/补齐首条历史菜单，无保留原日期资料输入；直接CORRECT且不自动添加原日节点 | B |
| <a id="m02-tl-303"></a>TL-303 两次独立保存 | 原6月B；无联合权先CORRECT到1月A，再ADD6月B；有无后继分别核验 | 第一步成功即正常可查，第二步另来源/事件/提交，首条ID固定且新6月新ID；旧源和中间源保持；按钮分别新增/插入 | D；补B |
| <a id="m02-tl-304"></a>TL-304 第二步取消与同值 | 第一步已保存，再打开第二步编辑取消；另提交同值名称 | 取消不写事实；同值no_change不建记录；第一步仍为当前且可结束，不要求继续或回滚 | B；补A |
| <a id="m02-tl-305"></a>TL-305 第二步拒绝及撤权 | 第一步已保存，第二步非法名称、无新增权分别提交，修正/恢复权后继续 | 各次拒绝无业务变化，不回滚第一步；恢复后只新增第二步，后继及停用继承准确 | A；补B取非法名称 |
| <a id="m02-tl-306"></a>TL-306 两步原结果独立恢复 | 两步成功后分别原内容重试/resolve；第二步真实保存响应丢失 | 第一结果仍指中间来源及首条，第二结果指新来源/新记录；仅查回丢失的第二步，不重复第一步或新节点 | A；补B |

## 6 普通更正、停用继承与后继快照

A选中明确资料记录后“更正记录”；本章只编辑名称、不编辑日期/分类，无处理原因。 本章默认P0。

| 案例 | 前置与输入 | 操作步骤 | 明确预期（并执行第2.3节共同断言） | 验收依据 · 主层/补层 |
| --- | --- | --- | --- | --- |
| <a id="m02-tl-046"></a>TL-046 首条名称更正 | F1；f改中文为新，[三时] | 保留日期保存，读取T0/T1 | W；f ID/01-01/CREATE原因保持，英文Alpha保持；旧T0仍A。 | [AC004](011-m02-business-requirements.md#m02-ac-004)、[AC013](011-m02-business-requirements.md#m02-ac-013)、[AC019](011-m02-business-requirements.md#m02-ac-019)；主G |
| <a id="m02-tl-047"></a>TL-047 中段更正只到独立后继 | F2；c1改为Z，[三时] | 保存并查6月/10月 | W；6—9月Z，10月C不变，c1日期/类型保持，无新增生效日。 | [AC004](011-m02-business-requirements.md#m02-ac-004)、[AC019](011-m02-business-requirements.md#m02-ac-019)；主G |
| <a id="m02-tl-048"></a>TL-048 重新启用名称纠错 | F10；r1改中文为新，[三时] | 预览、更正并读取s2、r2 | W；预览和保存均含s2的INHERITANCE，预览afterRevisionId为空、保存为准确新修订；r1新/Beta，s2继承新/Beta；r2仍C、11月仍Z，不能借更正取消启停作用。 | [AC004](011-m02-business-requirements.md#m02-ac-004)、[AC015](011-m02-business-requirements.md#m02-ac-015)、[AC019](011-m02-business-requirements.md#m02-ac-019)；主G |
| <a id="m02-tl-049"></a>TL-049 直接继承首条的停用变化 | F4；f的A改为C | 预览及保存后读s1及T0 | W；预览提前报告s1的INHERITANCE，保存s1继承C并引用f新修订；T0仍停用A，不另造一次停用业务。 | [AC004](011-m02-business-requirements.md#m02-ac-004)、[AC006](011-m02-business-requirements.md#m02-ac-006)、[AC019](011-m02-business-requirements.md#m02-ac-019)、[AC021](011-m02-business-requirements.md#m02-ac-021)；主G |
| <a id="m02-tl-050"></a>TL-050 曾部分输入的后继不传播 | F11；只更正f英文为North | 预览并保存 | W；1—5月North；6月c1、7月s1、8月r1仍Nroth，中文各段保持；提示后继保留旧值。 | [AC004](011-m02-business-requirements.md#m02-ac-004)、[AC019](011-m02-business-requirements.md#m02-ac-019)；主G；补B |
| <a id="m02-tl-051"></a>TL-051（109退役）明确联合后继更正 | F11；联合更正f和c1英文North | 一次保存，未选r1 | W；1—7月North，8月r1仍Nroth；s1派生引用c1新修订，不能自动加入r1。 | [AC016](011-m02-business-requirements.md#m02-ac-016)、[AC019](011-m02-business-requirements.md#m02-ac-019)、[AC021](011-m02-business-requirements.md#m02-ac-021)；主G |
| <a id="m02-tl-052"></a>TL-052 受限字段保持 | F2；CN只改c1中文 | 保存并由有权A读取 | W；Beta未回传仍完整保留，CN响应/差异不显示英文；不能把隐藏值写成null。 | [AC004](011-m02-business-requirements.md#m02-ac-004)、[AC009](011-m02-business-requirements.md#m02-ac-009)、[AC019](011-m02-business-requirements.md#m02-ac-019)；主D |
| <a id="m02-tl-053"></a>TL-053 英文显式清空 | F2；c1 nameEn=null；另一例只改中文 | 独立保存和回读 | 前者英文null，后者Beta；T0英文仍Beta，不能混淆缺席和null。 | [AC004](011-m02-business-requirements.md#m02-ac-004)、[AC019](011-m02-business-requirements.md#m02-ac-019)；主G |
| <a id="m02-tl-054"></a>TL-054 同值更正 | F2；c1显式B/Beta及等价首尾空白 | 各以独立身份提交 | N；不追加修订或刷新分类名称；显式英文仍须写权。 | [AC004](011-m02-business-requirements.md#m02-ac-004)、[AC013](011-m02-business-requirements.md#m02-ac-013)、[AC023](011-m02-business-requirements.md#m02-ac-023)；主G |
| <a id="m02-tl-055"></a>TL-055 停用名称不可直接更正 | F3；选择s1 | 检查能力并直调correct提交name | 统一入口可办理获权日期/分类，名称只读；只带name的API拒绝，不生成停用独立资料；应通过其明确前序或分类纠错办理。 | [AC004](011-m02-business-requirements.md#m02-ac-004)、[AC006](011-m02-business-requirements.md#m02-ac-006)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主A |
| <a id="m02-tl-056"></a>TL-056 更正严格白名单 | F2；correct夹带effectiveDate、顶层operationType/operationReason、state、note | 各字段独立请求，包括照抄原值 | 400；不自动丢字段或转成联合命令，原分类/日期/身份保持。 | [AC004](011-m02-business-requirements.md#m02-ac-004)、[AC013](011-m02-business-requirements.md#m02-ac-013)、[AC014](011-m02-business-requirements.md#m02-ac-014)、[AC018](011-m02-business-requirements.md#m02-ac-018)；主A |
| <a id="m02-tl-057"></a>TL-057 历史或删除目标不可更正 | F2后删c1；另以T0旧revision调用 | 分别从历史与API尝试 | 历史只读；已删/失效目标或过期依据拒绝，不恢复recordId。 | [AC004](011-m02-business-requirements.md#m02-ac-004)、[AC008](011-m02-business-requirements.md#m02-ac-008)、[AC010](011-m02-business-requirements.md#m02-ac-010)；主D |
| <a id="m02-tl-058"></a>TL-058 原因下架不妨碍普通更正 | F2；受控目录移除/更名原NAME_ADJUSTMENT | 更正c1中文，再查原结果和T0 | W且继承原代码和名称快照；不靠当前目录补历史，新业务仍按当前目录。 | [AC004](011-m02-business-requirements.md#m02-ac-004)、[AC013](011-m02-business-requirements.md#m02-ac-013)；主G；补D；条件F |

<a id="tl-lifecycle"></a>
## 7 新增停用、重新启用与多轮启停

A从详情标题旁更多操作中的“停用业务单元”或“重新启用业务单元”办理；判断目标日期前状态，不以D的状态替代。 本章默认P0。

| 案例 | 前置与输入 | 操作步骤 | 明确预期（并执行第2.3节共同断言） | 验收依据 · 主层/补层 |
| --- | --- | --- | --- | --- |
| <a id="m02-tl-059"></a>TL-059 末尾停用三类日期 | F1；04-01 S，[三时] | 显式选BUSINESS_CLOSURE保存 | W；03-31启用A，04-01起停用A；s无独立更名，未来停用不提前生效。 | [AC006](011-m02-business-requirements.md#m02-ac-006)、[AC013](011-m02-business-requirements.md#m02-ac-013)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G |
| <a id="m02-tl-060"></a>TL-060 开放停用段恢复 | F4；05-01 R B，[三时] | 确认完整资料和BUSINESS_RESUMPTION | W；4月仍停用A，5月起启用B；原BU/编码不变，新r ID，不自动恢复人员、岗位或权限。 | [AC006](011-m02-business-requirements.md#m02-ac-006)、[AC013](011-m02-business-requirements.md#m02-ac-013)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G |
| <a id="m02-tl-061"></a>TL-061 同名恢复仍是事实 | F4；05-01 R省略changes | 确认默认A后保存 | W而非N；新独立A快照，恢复状态变化真实；以后更正f不传播至r。 | [AC015](011-m02-business-requirements.md#m02-ac-015)、[AC019](011-m02-business-requirements.md#m02-ac-019)；主G |
| <a id="m02-tl-062"></a>TL-062 恢复仅改部分字段 | F4；05-01 R只改中文新，另例显式清空英文 | 分别保存 | 省略英文得新/Alpha，显式null得甲/null；均为独立完整快照，不显示无权默认值。补B：取得默认后收窄英文读写权，预览拒绝并重读；默认、输入、详情及日志缓存均不得残留英文，旧预览迟到不能恢复；中文输入保留，重直接保存后英文原值仍在。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC015](011-m02-business-requirements.md#m02-ac-015)、[AC019](011-m02-business-requirements.md#m02-ac-019)；主G；补B；条件F |
| <a id="m02-tl-063"></a>TL-063 今日启用仍可排未来恢复 | F1先排07-01 S | D=06-15时再排08-01 R B | W；当前仍A，7月停用A，8月B；按钮与API不能因今日启用拒绝未来合法恢复。 | [AC006](011-m02-business-requirements.md#m02-ac-006)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G；补B |
| <a id="m02-tl-064"></a>TL-064 补历史整段使用固定插入 | F2；拟加03-01 S或04-01 R | 先分别提交，再用固定插入停用与重新启用 | 单S使6月CHANGE非法、单R缺停用均R；固定两条一次保存，不能自动补另一项。 | [AC006](011-m02-business-requirements.md#m02-ac-006)、[AC015](011-m02-business-requirements.md#m02-ac-015)、[AC016](011-m02-business-requirements.md#m02-ac-016)；主G |
| <a id="m02-tl-065"></a>TL-065 停用后有普通后继 | F2；05-01 S | 预览并提交 | R指向6月后继依赖；不自动删c1/c2，不制造隐式恢复。 | [AC006](011-m02-business-requirements.md#m02-ac-006)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G |
| <a id="m02-tl-066"></a>TL-066 恢复后原恢复会重复 | F3；05-01 R B | 提交并检查8月r1 | R；05月恢复使原8月R缺停用前提，不能只校验新日期。 | [AC015](011-m02-business-requirements.md#m02-ac-015)；主G |
| <a id="m02-tl-067"></a>TL-067 重复停用 | F4；05-01 S | 直接提交 | R M02_TIMELINE_INVALID，无新增停用，无“已停用所以成功”或N。 | [AC006](011-m02-business-requirements.md#m02-ac-006)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G |
| <a id="m02-tl-068"></a>TL-068 重复恢复 | F5；06-01 R | 直接提交 | R M02_TIMELINE_INVALID；相同名称不改变此结论。 | [AC015](011-m02-business-requirements.md#m02-ac-015)；主G |
| <a id="m02-tl-069"></a>TL-069 首条之前与同日启停 | F1在2025-12-31 S/R；F4在04-01 R；F5在05-01 S | 逐例独立尝试 | 首条前或已有日期占用均R；不使用生效序号，不能当天先停再启。 | [AC003](011-m02-business-requirements.md#m02-ac-003)、[AC006](011-m02-business-requirements.md#m02-ac-006)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G |
| <a id="m02-tl-070"></a>TL-070 三类状态全部转换 | NO_FACT=F6的06-01，ACTIVE=F1的03-01，INACTIVE=F4的05-01 | 每状态分别尝试CREATE/CHANGE/S/R；CREATE通过新建或非法第二首条路径区分 | 12格仅NO_FACT新对象CREATE、ACTIVE CHANGE/S、INACTIVE R合法；已有BU不能再CREATE，其余8格R。 | [AC002](011-m02-business-requirements.md#m02-ac-002)、[AC003](011-m02-business-requirements.md#m02-ac-003)、[AC006](011-m02-business-requirements.md#m02-ac-006)、[AC015](011-m02-business-requirements.md#m02-ac-015)、[AC016](011-m02-business-requirements.md#m02-ac-016)；主G |
| <a id="m02-tl-071"></a>TL-071 停止类型禁止资料夹带 | F1 S带changes/name/nameEn/state；R带空changes或name=null | 各独立请求 | 400；停用不能同时更名，恢复存在changes时须非空且合并中文有效。 | [AC006](011-m02-business-requirements.md#m02-ac-006)、[AC013](011-m02-business-requirements.md#m02-ac-013)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主A |
| <a id="m02-tl-072"></a>TL-072 类型原因全目录 | S分别ORG_RESTRUCTURING/BUSINESS_CLOSURE，R为BUSINESS_RESUMPTION；交换原因或缺原因 | 合法及非法组合独立办理 | 适用原因W并固化标签；缺失、错配、客户端标签拒绝；唯一原因按Go默认，多项不擅选，不允许自由文本。 | [AC013](011-m02-business-requirements.md#m02-ac-013)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G；补A |
| <a id="m02-tl-073"></a>TL-073 多轮完整链 | 从F1依次加04-01 S、05-01 R B、08-01 S、09-01 R C、11-01 CHANGE Z；[三时]按每次目标日运行 | 每次从最新来源办理，逐段读取 | 形成F10；两停用段和三个独立启用快照准确，既有停用不妨碍恢复后更名；每次来源仅增1。 | [AC003](011-m02-business-requirements.md#m02-ac-003)、[AC006](011-m02-business-requirements.md#m02-ac-006)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G |
| <a id="m02-tl-074"></a>TL-074 过去新增与原采用保持 | F1；D=11-01固定成对插入或分合法步骤登记过去启停 | 读旧来源、源事件及采用入口 | 新来源反映过去期间，旧来源仍连续启用；无消费者时明确未接入，不能声称已重算下游。 | [AC008](011-m02-business-requirements.md#m02-ac-008)、[AC011](011-m02-business-requirements.md#m02-ac-011)、[AC015](011-m02-business-requirements.md#m02-ac-015)、[AC016](011-m02-business-requirements.md#m02-ac-016)；主D |

<a id="tl-delete"></a>
## 8 删除各类记录与完整依赖

A选择当前记录“删除此记录”，确认原类型/日期、全期间和今日状态；不填新日期、分类或删除原因。 本章默认P0。

| 案例 | 前置与输入 | 操作步骤 | 明确预期（并执行第2.3节共同断言） | 验收依据 · 主层/补层 |
| --- | --- | --- | --- | --- |
| <a id="m02-tl-075"></a>TL-075 首条始终不可删除 | F1/F2/F6分别唯一/有后继/未来首条；[三时] | 检查菜单能力并直调删除f | 均R FIRST_RECORD_PROTECTED，BU/编码/首条保持；不提供并列撤销或整BU删除入口。 | [AC002](011-m02-business-requirements.md#m02-ac-002)、[AC005](011-m02-business-requirements.md#m02-ac-005)、[AC014](011-m02-business-requirements.md#m02-ac-014)、[AC017](011-m02-business-requirements.md#m02-ac-017)；主G；补A |
| <a id="m02-tl-076"></a>TL-076 普通中段删除 | F2；删除c1，[三时] | 确认影响后保存 | W；1—9月A、10月C；旧T0的6—9月B及删除原件保留，删除后选中适用前段。 | [AC005](011-m02-business-requirements.md#m02-ac-005)、[AC008](011-m02-business-requirements.md#m02-ac-008)、[AC013](011-m02-business-requirements.md#m02-ac-013)；主G |
| <a id="m02-tl-077"></a>TL-077 普通尾条删除 | F2；删除c2，[三时] | 保存并读10月以后 | W；6月B延续至开放终点；c1准确期间修订，旧T0仍C。 | [AC005](011-m02-business-requirements.md#m02-ac-005)、[AC008](011-m02-business-requirements.md#m02-ac-008)；主G |
| <a id="m02-tl-078"></a>TL-078 停用前普通删除 | F3；删除c1即10月C | 读10月、12月s2和8月r1 | W；8月起明确B，s2停用继承B及准确r1修订；早期s1/r1独立性保持。 | [AC005](011-m02-business-requirements.md#m02-ac-005)、[AC006](011-m02-business-requirements.md#m02-ac-006)、[AC015](011-m02-business-requirements.md#m02-ac-015)、[AC021](011-m02-business-requirements.md#m02-ac-021)；主G |
| <a id="m02-tl-079"></a>TL-079 末尾停用删除 | F4；删s1，[三时] | 确认从原停用日起重新解释为启用后保存 | W；4月起A连续ACTIVE，原T0仍INACTIVE；这是删除错误停用，不新增REACTIVATE。 | [AC005](011-m02-business-requirements.md#m02-ac-005)、[AC006](011-m02-business-requirements.md#m02-ac-006)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G |
| <a id="m02-tl-080"></a>TL-080 停用有恢复后继 | F3或F5；只删s1 | 单删预览及提交 | R，准确指出r1失去停用前提；不把r1自动转CHANGE或级联删掉。 | [AC005](011-m02-business-requirements.md#m02-ac-005)、[AC006](011-m02-business-requirements.md#m02-ac-006)、[AC015](011-m02-business-requirements.md#m02-ac-015)、[AC016](011-m02-business-requirements.md#m02-ac-016)；主G |
| <a id="m02-tl-081"></a>TL-081 末尾重新启用删除 | F5；删r1，[三时] | 确认影响并保存 | W；4月起停用A延伸开放，旧T0的5月起仍B启用；不新增停用业务。 | [AC005](011-m02-business-requirements.md#m02-ac-005)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G |
| <a id="m02-tl-082"></a>TL-082 恢复后有普通记录 | F5加06-01 CHANGE C；只删r1 | 提交 | R，6月CHANGE将落入停用；c1/r1/s1均不变。 | [AC005](011-m02-business-requirements.md#m02-ac-005)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G |
| <a id="m02-tl-083"></a>TL-083 恢复后有再次停用 | F10；只删r1 | 提交 | R，8月S将成为重复停用；不因11月普通记录离目标较远而漏扫。 | [AC005](011-m02-business-requirements.md#m02-ac-005)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G |
| <a id="m02-tl-084"></a>TL-084 成对删除保留正确后继 | F3；专用操作删除s1和r1 | 确认整段恢复连续启用后保存 | W；1—9月A、10月C、12月停用C；两个旧原件留T0，一份新来源。 | [AC005](011-m02-business-requirements.md#m02-ac-005)、[AC015](011-m02-business-requirements.md#m02-ac-015)、[AC016](011-m02-business-requirements.md#m02-ac-016)；主G |
| <a id="m02-tl-085"></a>TL-085 删除后同日重新新增 | F2删c1后，以新提交06-01 ADD Z | 逐次保存并读各来源 | 新c拥有新ID，旧c1不复活；原编码仍保留，原删除结果仍定位原删除修订。 | [AC003](011-m02-business-requirements.md#m02-ac-003)、[AC005](011-m02-business-requirements.md#m02-ac-005)、[AC008](011-m02-business-requirements.md#m02-ac-008)、[AC020](011-m02-business-requirements.md#m02-ac-020)；主D |
| <a id="m02-tl-086"></a>TL-086 重复删除与原结果重放 | F2删除c1成功；原请求重试及新身份再删c1 | 分别提交 | 原身份原内容返回原accepted不再写；新请求已删目标R，不制造第二删除或恢复。 | [AC005](011-m02-business-requirements.md#m02-ac-005)、[AC010](011-m02-business-requirements.md#m02-ac-010)；主A |
| <a id="m02-tl-087"></a>TL-087 删除输入和准确目标 | DELETE夹带日期/原因/changes；另用过期revision或跨BU目标 | 每种独立提交 | 额外字段400，版本/范围错误安全拒绝；不按当前目录或另一BU同名记录替换目标。 | [AC005](011-m02-business-requirements.md#m02-ac-005)、[AC009](011-m02-business-requirements.md#m02-ac-009)、[AC010](011-m02-business-requirements.md#m02-ac-010)、[AC013](011-m02-business-requirements.md#m02-ac-013)；主A |
| <a id="m02-tl-088"></a>TL-088 同值前序删除仍更新继承 | F8；删c1 | 读f、s1继承引用与v2事件 | W；业务值集合可为空，但f期间延长、c1删除、s1改指f新修订；不能复用s1原引用或返回N。 | [AC005](011-m02-business-requirements.md#m02-ac-005)、[AC021](011-m02-business-requirements.md#m02-ac-021)；主G；补D |
| <a id="m02-tl-089"></a>TL-089 仅历史引用不阻断 | F4或F5目标有原来源/审计证据；另设阻断点引用 | 分别删末尾S或R | 只有历史证据时合法W；删除R使引用日失去启用则R；旧采用不改写，真实消费者未接入只计合同。 | [AC005](011-m02-business-requirements.md#m02-ac-005)、[AC011](011-m02-business-requirements.md#m02-ac-011)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G；条件F |

<a id="tl-redate"></a>
## 9 全类别改日与日期重排

A选择当前记录“更正记录”内生效日期，只改日期及准确来源/修订；跨节点97/98通过原有显式联合UPDATE办理，普通CORRECT/仅日期API拒绝；首条和各生命周期目标使用专用动作权。 本章默认P0。

| 案例 | 前置与输入 | 操作步骤 | 明确预期（并执行第2.3节共同断言） | 验收依据 · 主层/补层 |
| --- | --- | --- | --- | --- |
| <a id="m02-tl-090"></a>TL-090 首条新旧日期九格 | 第2.5节唯一首条样本，[九格] | 逐格改日并查原新日、D | 同一f，首条前NO_FACT，新日起A；仅T→T N，其余W，原来源仍旧起点。 | [AC014](011-m02-business-requirements.md#m02-ac-014)、[AC024](011-m02-business-requirements.md#m02-ac-024)；主G |
| <a id="m02-tl-091"></a>TL-091 普通尾条新旧日期九格 | 1月f A＋旧日c1 B，[九格] | 逐格改日 | 新日前A、新日起B，原新日之间只按相应来源解释；同值N，身份/分类保持。 | [AC014](011-m02-business-requirements.md#m02-ac-014)；主G |
| <a id="m02-tl-092"></a>TL-092 停用新旧日期九格 | 1月f A＋旧日s1，[九格] | 逐格改日 | 新日前ACTIVE A，新日起INACTIVE A；今日可在启用/停用间改变，不能据原已生效拒绝。 | [AC014](011-m02-business-requirements.md#m02-ac-014)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G |
| <a id="m02-tl-093"></a>TL-093 恢复新旧日期九格 | 1月f A＋4月S＋旧日r1 B，[九格] | 逐格改日 | 4月至新日前停用A，新日起B启用；独立B不重新取前序A；原来源仍原恢复日。 | [AC014](011-m02-business-requirements.md#m02-ac-014)、[AC015](011-m02-business-requirements.md#m02-ac-015)、[AC019](011-m02-business-requirements.md#m02-ac-019)；主G |
| <a id="m02-tl-094"></a>TL-094 首条前移与后移 | F2；f分别改2025-12-01及02-01 | 独立预览/保存 | 前移延长A；后移1月NO_FACT且2—5月A；c1/c2内容不变，不能声称自动补出不同早期资料。 | [AC014](011-m02-business-requirements.md#m02-ac-014)、[AC017](011-m02-business-requirements.md#m02-ac-017)；主G |
| <a id="m02-tl-095"></a>TL-095 唯一首条移到未来 | F1；f改07-01 | D=06-15保存并查当前/旧来源 | W；当前NO_FACT并显示未来首条A，T0今日ACTIVE A；编码/身份不变。 | [AC001](011-m02-business-requirements.md#m02-ac-001)、[AC014](011-m02-business-requirements.md#m02-ac-014)；主G；补B |
| <a id="m02-tl-096"></a>TL-096 首条越过后继拒绝 | F2；f改06-01、06-02；另改到2024-12-31 | 逐值提交 | 同日、首条不再最早、企业范围外均R；不换first_record_id。 | [AC014](011-m02-business-requirements.md#m02-ac-014)、[AC017](011-m02-business-requirements.md#m02-ac-017)；主G |
| <a id="m02-tl-097"></a>TL-097 普通记录向前重排（109退役） | F9；c2从05-01改02-01 | 显式联合UPDATE保存并查各段；同参数普通入口拒绝 | W；1月A、2月C、3—6月B、7月Z；c2仍完整C，不从新前序重新继承。 | [AC014](011-m02-business-requirements.md#m02-ac-014)、[AC019](011-m02-business-requirements.md#m02-ac-019)；主G；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-098"></a>TL-098 普通记录向后重排（109退役） | F2；c1从06-01改11-01 | 显式联合UPDATE保存；同参数普通入口拒绝 | W；1—9月A、10月C、11月起B；c2内容/日期保持，旧来源6月仍B。 | [AC014](011-m02-business-requirements.md#m02-ac-014)；主G；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-099"></a>TL-099 普通改日到停用段 | F3；c1从10-01改06-01 | 提交 | R，CHANGE落在INACTIVE；不能自动移s1/r1或变成恢复。 | [AC014](011-m02-business-requirements.md#m02-ac-014)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G |
| <a id="m02-tl-100"></a>TL-100 普通移到首条前或占用日 | F2；c1改2025-12-31/01-01/10-01 | 逐值提交 | 均R，当前一日一条且非首条晚于f；不腾挪未指定记录。 | [AC014](011-m02-business-requirements.md#m02-ac-014)；主G |
| <a id="m02-tl-101"></a>TL-101 停用合法前后移 | F3；s1分别改03-01、05-01 | 独立保存并读边界 | W；停用分别从3月或5月起至8月，r1的B及后继保持，旧来源仍4月起停用。 | [AC014](011-m02-business-requirements.md#m02-ac-014)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G |
| <a id="m02-tl-102"></a>TL-102 停用越过普通或恢复 | F3另在03-01加CHANGE Z；s1改02-01；原F3另试08-01/09-01 | 逐例提交 | 分别使3月CHANGE失去前提、同日冲突或8月孤立R，全部R。 | [AC014](011-m02-business-requirements.md#m02-ac-014)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G |
| <a id="m02-tl-103"></a>TL-103 恢复合法边界和后继拒绝 | F3；r1改09-01可行，04-01/03-01/11-01不可行 | 各从T0独立提交 | 09月W且停用延长、B保持；其余同日/无前提/10月CHANGE依赖导致R。 | [AC014](011-m02-business-requirements.md#m02-ac-014)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G |
| <a id="m02-tl-104"></a>TL-104 同值改日仍需权限版本 | F3 r1改原08-01；另缺日期权或用过期来源 | 分别办理 | 合法N；无权或失效R；不能以新旧日期相同绕过守卫。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC010](011-m02-business-requirements.md#m02-ac-010)、[AC014](011-m02-business-requirements.md#m02-ac-014)、[AC023](011-m02-business-requirements.md#m02-ac-023)；主G |
| <a id="m02-tl-105"></a>TL-105 改日前序同值仍换继承 | F8；c1从03-01改04-01 | 保存并读s1引用及影响 | W；业务值无差异但日期/期间/继承有结构变化，s1准确引用c1新修订；不是N。 | [AC014](011-m02-business-requirements.md#m02-ac-014)、[AC021](011-m02-business-requirements.md#m02-ac-021)；主G；补D |
| <a id="m02-tl-106"></a>TL-106 改日DTO及目标资格 | redate夹带oldEffectiveDate/effectiveDate/changes/分类；已删目标及旧来源另测 | 逐项直调 | 白名单400，失效目标/来源拒绝；correct也必须执行日期专用权、原位置及版本核验，不能用旧日期覆盖当前。 | [AC008](011-m02-business-requirements.md#m02-ac-008)、[AC013](011-m02-business-requirements.md#m02-ac-013)、[AC014](011-m02-business-requirements.md#m02-ac-014)；主A |
| <a id="m02-tl-107"></a>TL-107 改日结果在再次维护后保持 | F2 c1改07-01得T1，后再改08-01或删除 | 按第一次提交重试/resolve | 原结果仍同一c1及06-01→07-01/T1，不返回后来8月或假称不存在；当前授权继续生效。 | [AC008](011-m02-business-requirements.md#m02-ac-008)、[AC010](011-m02-business-requirements.md#m02-ac-010)、[AC014](011-m02-business-requirements.md#m02-ac-014)；主A |

<a id="tl-joint"></a>
## 10 有限联合修订与最终候选

A从详情联合编辑区明确ADD/UPDATE/DELETE项；默认F2，直接提交时按最终完整候选整笔核验。 本章默认P0。

| 案例 | 前置与输入 | 操作步骤 | 明确预期（并执行第2.3节共同断言） | 验收依据 · 主层/补层 |
| --- | --- | --- | --- | --- |
| <a id="m02-tl-108"></a>TL-108 已有后继补整段启停 | F2；固定插入03-01 S和04-01 R，恢复继承A | 从专用入口明确停用/恢复两日及原因，一次保存 | W；1—2月A启用、3月A停用、4—5月A启用、6月B、10月C；c1/c2独立快照保持，原T0连续启用。 | [AC016](011-m02-business-requirements.md#m02-ac-016)、[AC019](011-m02-business-requirements.md#m02-ac-019)；主G；补B |
| <a id="m02-tl-109"></a>TL-109 成对删除启停 | F3；DELETE s1/r1 | 同时移除并保存 | W；1—9月A，10月C，12月停用C；两原件仍在T0，不能先提交单删结果。 | [AC015](011-m02-business-requirements.md#m02-ac-015)、[AC016](011-m02-business-requirements.md#m02-ac-016)；主G |
| <a id="m02-tl-110"></a>TL-110 两个普通记录交换日期（109退役） | F2；c1→10-01，c2→06-01 | 同一候选保存，再交换项顺序重放 | W；6—9月C，10月起B；两个ID随各自日期保留，不按旧落点冲突拒绝。 | [AC014](011-m02-business-requirements.md#m02-ac-014)、[AC016](011-m02-business-requirements.md#m02-ac-016)；主G；补D；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-111"></a>TL-111 三个日期循环置换（109退役） | F9；c1→05-01，c2→07-01，c3→03-01 | 全部明确后一次保存 | W；1—2月A、3—4月Z、5—6月B、7月C；无临时占用或中间来源。 | [AC014](011-m02-business-requirements.md#m02-ac-014)、[AC016](011-m02-business-requirements.md#m02-ac-016)；主G；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-112"></a>TL-112 混合删新同日及更新（109退役） | F9；删c1，03-01 ADD B2，c2改06-01且中文丙2 | 一次保存 | W；1—2月A、3—5月B2、6月丙2/Gamma、7月Z；新3月新ID，c2仅一个合并修订，旧c1留删除证据。 | [AC016](011-m02-business-requirements.md#m02-ac-016)、[AC020](011-m02-business-requirements.md#m02-ac-020)；主G；补D；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-113"></a>TL-113 去掉混合删除项（109退役） | 上例但保留原c1 | 预览及提交 | 同日冲突R，ADD/UPDATE均不落库，不让删除遗漏被自动补齐。 | [AC016](011-m02-business-requirements.md#m02-ac-016)、[AC020](011-m02-business-requirements.md#m02-ac-020)；主G；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-114"></a>TL-114 一个项合并多修改面（109退役） | F2；c2改11-01、中文新、CHANGE原因保持 | 单UPDATE含日期和名称保存 | W；c2保留ID且只追加一个直接/期间合并修订，显式字段/日期权均检查。 | [AC014](011-m02-business-requirements.md#m02-ac-014)、[AC016](011-m02-business-requirements.md#m02-ac-016)、[AC020](011-m02-business-requirements.md#m02-ac-020)；主G；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-115"></a>TL-115 最终非法的启停交换（109退役） | F5；仅交换s1/r1日期但保持原类型 | 预览及正式提交 | R：4月R没有停用前提，不能把同日唯一当全部合法性。 | [AC015](011-m02-business-requirements.md#m02-ac-015)、[AC016](011-m02-business-requirements.md#m02-ac-016)；主G；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-116"></a>TL-116 首条顺序及身份保护（109退役） | F2；把f移至11月，或ADD CREATE，或DELETE f | 分别与其他合法项组合 | 均整笔R，不因联合允许删首条、更换first_record_id或让其他项部分成功。 | [AC016](011-m02-business-requirements.md#m02-ac-016)、[AC017](011-m02-business-requirements.md#m02-ac-017)；主G；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-117"></a>TL-117 重复itemId和目标（109退役） | 两个项同itemId；同c1两个UPDATE；c1 UPDATE加DELETE | 三组完整DTO独立提交 | M02_REVISION_ITEM_CONFLICT；多个修改面应合并单项，不能最后一个覆盖前一个。 | [AC016](011-m02-business-requirements.md#m02-ac-016)、[AC020](011-m02-business-requirements.md#m02-ac-020)；主G；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-118"></a>TL-118 新增项不可再次更新（109退役） | 同笔ADD的itemId被另一UPDATE/DELETE当recordId | 直接提交 | R；新身份仅服务器保存时分配，不把itemId当已存在记录或接受嵌套脚本。 | [AC016](011-m02-business-requirements.md#m02-ac-016)；主A；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-119"></a>TL-119 跨BU与非基准成员（109退役） | 含BU2目标、已删除目标或仅旧来源存在的revision | 与合法ADD组合提交 | 安全拒绝整笔，BU1/BU2均无变化；不构造跨BU事务。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC016](011-m02-business-requirements.md#m02-ac-016)；主D；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-120"></a>TL-120 空集合与空更新（109退役） | items=[]；UPDATE无修改面；classification缺type或reason；DELETE带changes | 逐例预览/正式提交 | 合法缺项预览只incomplete，正式400；协议非法仍400，不保存草稿或部分业务。 | [AC016](011-m02-business-requirements.md#m02-ac-016)、[AC018](011-m02-business-requirements.md#m02-ac-018)、[AC022](011-m02-business-requirements.md#m02-ac-022)；主A；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-121"></a>TL-121 同值ADD先验同日唯一（109退役） | F1；同日03-01 ADD A及ADD Z | 一次提交 | R同日冲突，即使A拟no_change也不能先去重放行；两项都不建身份。 | [AC016](011-m02-business-requirements.md#m02-ac-016)、[AC023](011-m02-business-requirements.md#m02-ac-023)；主G；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-122"></a>TL-122 ADD默认来自最终前序（109退役） | F1；UPDATE f中文新；ADD03-01只改英文Beta | 输入ADD先于UPDATE后保存 | W；1—2月新/Alpha，3月新/Beta；不取旧T0甲作为新增默认，顺序无意义。 | [AC016](011-m02-business-requirements.md#m02-ac-016)、[AC019](011-m02-business-requirements.md#m02-ac-019)；主G；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-123"></a>TL-123 同值ADD消除后继续扫描（109退役） | F1；f更正为B，ADD03-01 B，ADD05-01 C | 整笔保存 | W；3月项no_change无ID，1—4月B、5月C；5月前序是实际f，不引用被消除的虚构成员。 | [AC016](011-m02-business-requirements.md#m02-ac-016)、[AC023](011-m02-business-requirements.md#m02-ac-023)；主G；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-124"></a>TL-124 无变化项仍有派生期间（109退役） | F2；f UPDATE同值A，ADD03-01 Z | 一次保存 | W；f项no_change无直接source_change_item，但f因期间变化有派生修订；显示输入无变化与整体影响各自含义。 | [AC016](011-m02-business-requirements.md#m02-ac-016)、[AC021](011-m02-business-requirements.md#m02-ac-021)、[AC023](011-m02-business-requirements.md#m02-ac-023)；主G；补D；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-125"></a>TL-125 全项同值（109退役） | F2；c1同值名称、c2同日改日、03-01 ADD A | 正式提交后resolve | 整笔N；ADD无ID、无新来源/项/事件，技术结果完整列各itemId；同值声明权限仍核验。 | [AC016](011-m02-business-requirements.md#m02-ac-016)、[AC023](011-m02-business-requirements.md#m02-ac-023)；主G；补D；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-126"></a>TL-126 纯结构变化和分离影响（109退役） | F9；f→甲2、c2→丙2；另例只改f原因NEW_UNIT | 分别预览/保存 | 前例数据区间仅[01-01,03-01)和[05-01,07-01)；后例数据集合空而分类结构非空仍W，未变间隙不扩大。 | [AC016](011-m02-business-requirements.md#m02-ac-016)、[AC021](011-m02-business-requirements.md#m02-ac-021)；主G；补D；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-127"></a>TL-127 显式删除全部非首条（109退役） | F10；明确列出s1/r1/s2/r2/c1五个DELETE | 一次直接保存 | W；仅f从1月连续ACTIVE A；保留BU编码及全部旧原件；这不是整BU删除或隐式级联。 | [AC005](011-m02-business-requirements.md#m02-ac-005)、[AC015](011-m02-business-requirements.md#m02-ac-015)、[AC016](011-m02-business-requirements.md#m02-ac-016)；主G；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-128"></a>TL-128 任一条失败整笔保护 | 固定插入或删除停用/恢复两条；分别破坏一方权限/修订/日期，注入引用故障或写阶段失败 | 从固定请求提交并查回 | 全笔拒绝或待确认按原终态处理，无半份来源、孤立身份或单条成功；旧来源保持。 | AC009/010/011/016/025；主D；条件F |

<a id="tl-backfill-class"></a>
## 11 首条资料维护与分类录错处置

首条使用更正记录前移并改名，按需另次新增/插入；必要关联分类纠错可明确联合。每次新增须选择原因，不设置首条保留开关；第2.3节W按每次独立保存核验。自愿联合的单次原子性继续由第10章验证。本章默认P0。

| 案例 | 前置与输入 | 操作步骤 | 明确预期（并执行第2.3节共同断言） | 验收依据 · 主层/补层 |
| --- | --- | --- | --- | --- |
| <a id="m02-tl-129"></a>TL-129 唯一首条前移及独立新增 | 单6月f B；前移01-01并更正A，再明确6月B | CORRECT保存T1，再普通ADD保存T2 | 两次各W；T1只有1月A，T2为1—5月A/6月B新ID，f身份固定；T0/T1保持。 | [AC017](011-m02-business-requirements.md#m02-ac-017)；主A |
| <a id="m02-tl-130"></a>TL-130 已有后继时独立插入原日资料 | F7；前移f至1月A，再新增6月B | 更正保存T1后，在1月与7月之间普通插入T2 | 各W；T1七月停用继承A；T2为1—5月A、6月B、7月停用B、8—9月C、10月Z，恢复/资料快照保持，T0/T1保持。 | [AC017](011-m02-business-requirements.md#m02-ac-017)、[AC019](011-m02-business-requirements.md#m02-ac-019)、[AC021](011-m02-business-requirements.md#m02-ac-021)；主D；补B |
| <a id="m02-tl-131"></a>TL-131 已有后继时直接更正不新增原日 | F7；前移f至1月A | 普通CORRECT保存，不办理第二步 | W；1—6月A、7月停用A、8月C、10月Z；6月无新记录，无保留选项，s1继承f新修订。 | [AC017](011-m02-business-requirements.md#m02-ac-017)、[AC019](011-m02-business-requirements.md#m02-ac-019)、[AC021](011-m02-business-requirements.md#m02-ac-021)；主D；补B |
| <a id="m02-tl-132"></a>TL-132 保留选择尚未确认（107退役） | 原F7专用引导要求选择保留/不保留 | 原步骤保留作历史：尝试保存并定位必选项 | 107取消该输入和入口，本例不再执行或计通过；入口及无选择的新预期由TL-302持有。 | [AC017](011-m02-business-requirements.md#m02-ac-017)；退役 |
| <a id="m02-tl-133"></a>TL-133 单条前移与不同资料补全区分 | 单6月f B；只改日01-01 | 通过单条改日保存 | W仅延长B，不新增6月、不变成A；页面不宣称完成A→B补全。 | [AC014](011-m02-business-requirements.md#m02-ac-014)、[AC017](011-m02-business-requirements.md#m02-ac-017)；主G |
| <a id="m02-tl-134"></a>TL-134 补全边界与不自动改原因 | F7；f前移到企业外；另合法前移而未声明分类 | 分别提交 | 企业外R；合法项继承f原CREATE原因，不因“补登记”自动换原因；原首条始终不可删除。 | [AC013](011-m02-business-requirements.md#m02-ac-013)、[AC017](011-m02-business-requirements.md#m02-ac-017)；主G |
| <a id="m02-tl-135"></a>TL-135 补全与既有字段冻结引用 | F7；普通更正前移改名后06-15由B变A；受控引用冻结该日名称 | 分别仅历史证据、真正字段冻结两例 | 仅历史证据W；真实冻结条件不满足则全笔R，不能只检查新增早期范围。 | [AC011](011-m02-business-requirements.md#m02-ac-011)、[AC017](011-m02-business-requirements.md#m02-ac-017)；主G；条件F |
| <a id="m02-tl-136"></a>TL-136 首条原因纠错 | 受控F1历史原NEW_UNIT；显式改NEW_ORGANIZATION | 只改分类原因并保存 | W；日期/资料/ID不变，新来源读新原因，旧来源与原提交仍旧原因；数据影响空但结构有差异。 | [AC013](011-m02-business-requirements.md#m02-ac-013)、[AC018](011-m02-business-requirements.md#m02-ac-018)、[AC021](011-m02-business-requirements.md#m02-ac-021)；主G；补D；当前新建单项同值为N，历史夹具不计旧目录公开写入通过 |
| <a id="m02-tl-137"></a>TL-137 CHANGE改停用 | F2；c2的10月CHANGE改DEACTIVATE/BUSINESS_CLOSURE | 不提交changes，保存 | W；10月起停用继承6月B，c2原ID保留；T0仍C启用，各来源使用本版修订类型，不用最初类型覆盖。 | [AC018](011-m02-business-requirements.md#m02-ac-018)、[AC021](011-m02-business-requirements.md#m02-ac-021)；主G |
| <a id="m02-tl-138"></a>TL-138 停用改CHANGE | F4；s1改CHANGE/NAME_ADJUSTMENT | 确认基准解析A作为独立快照 | W；4月起ACTIVE A且保留s1身份；虽同名仍类型/状态变化，非N。 | [AC018](011-m02-business-requirements.md#m02-ac-018)、[AC019](011-m02-business-requirements.md#m02-ac-019)；主G |
| <a id="m02-tl-139"></a>TL-139 停用改重新启用（109退役） | F4；s1改R，同时03-01 ADD S | 一次保存 | W；3月停用A，原s1在4月恢复独立A；没有新增前置S的单项版本R。 | [AC015](011-m02-business-requirements.md#m02-ac-015)、[AC018](011-m02-business-requirements.md#m02-ac-018)；主G；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-140"></a>TL-140 重新启用改CHANGE（109退役） | F5；r1改CHANGE，同时DELETE s1 | 一次保存 | W；1—4月A、5月B启用；r1保留B完整快照；未删除S时单项R。 | [AC015](011-m02-business-requirements.md#m02-ac-015)、[AC018](011-m02-business-requirements.md#m02-ac-018)、[AC019](011-m02-business-requirements.md#m02-ac-019)；主G；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-141"></a>TL-141 重新启用改停用（109退役） | F5；r1改DEACTIVATE，同时DELETE s1 | 不提交名称，保存 | W；1—4月启用A，5月停用A；原r1的B在T0保留，当前停用按最终f继承。 | [AC015](011-m02-business-requirements.md#m02-ac-015)、[AC018](011-m02-business-requirements.md#m02-ac-018)、[AC019](011-m02-business-requirements.md#m02-ac-019)；主G；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-142"></a>TL-142 CHANGE改重新启用（109退役） | F2；c2改REACTIVATE，同时09-01 ADD S | 一次保存 | W；9月停用B，10月原c2恢复独立C；缺新增S时单项R。 | [AC015](011-m02-business-requirements.md#m02-ac-015)、[AC018](011-m02-business-requirements.md#m02-ac-018)、[AC019](011-m02-business-requirements.md#m02-ac-019)；主G；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-143"></a>TL-143 停用恢复互换类型及日期（109退役） | F5；s1改05-01 R，r1改04-01 S | 两个UPDATE保存 | W；4月停用A，5月恢复A；两个ID分别随新日移动，恢复默认取原s1基准A，不误用原r1的B。 | [AC014](011-m02-business-requirements.md#m02-ac-014)、[AC018](011-m02-business-requirements.md#m02-ac-018)、[AC019](011-m02-business-requirements.md#m02-ac-019)；主G；补B；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-144"></a>TL-144 分类日期字段同时纠错（109退役） | F5；f英文West，s1→03-01 CHANGE仅中文甲2，r1→06-01 CHANGE仅中文乙2 | 三项一次保存 | W；1月甲/West、3月甲2/Alpha、6月乙2/Beta，全程ACTIVE；原停用默认不取最终前序West，每记录仅一合并修订。 | [AC014](011-m02-business-requirements.md#m02-ac-014)、[AC018](011-m02-business-requirements.md#m02-ac-018)、[AC019](011-m02-business-requirements.md#m02-ac-019)、[AC020](011-m02-business-requirements.md#m02-ac-020)；主G；补B；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-145"></a>TL-145 单改前项造成孤立恢复 | F5；只将s1改CHANGE | 预览及正式提交 | R定位5月r1无停用前提；不能只按目标新类型合法就保存。 | [AC015](011-m02-business-requirements.md#m02-ac-015)、[AC018](011-m02-business-requirements.md#m02-ac-018)；主G |
| <a id="m02-tl-146"></a>TL-146 首条与非首条类型限制 | f分别改CHANGE/S/R；非首条改CREATE；已删记录分类纠错 | 分别与合法项组合 | 全部R；首条仅可改CREATE原因，不换指针、不恢复删除目标。 | [AC017](011-m02-business-requirements.md#m02-ac-017)、[AC018](011-m02-business-requirements.md#m02-ac-018)；主G |
| <a id="m02-tl-147"></a>TL-147 转停用禁止名称面 | F2 c2转S且changes同值C或nameEn=null | 逐例提交 | 400整笔拒绝，不能先丢弃名称面再成功；停用只从最终前序继承。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC018](011-m02-business-requirements.md#m02-ac-018)；主G |
| <a id="m02-tl-148"></a>TL-148 同分类及目录变化 | F4；显式相同S原因，目录仅改标签或移除旧项；另改真实新原因 | 分别提交和查回 | 同代码N且不刷新历史标签；真实代码变化按当前目录验证并W或R；普通更正/改日/删除不要求重新选择。 | [AC013](011-m02-business-requirements.md#m02-ac-013)、[AC018](011-m02-business-requirements.md#m02-ac-018)、[AC023](011-m02-business-requirements.md#m02-ac-023)；主G；条件F |
| <a id="m02-tl-149"></a>TL-149 双侧分类与修改面权限 | 仅对能独立合法办理的单条类型转换移除原类、新类、日期或显式名称权；需要多目标修复的旧参数随139—144退出 | 每次仅移除一项所需授权 | 整笔安全拒绝；更正记录仍须各实际修改面权限，默认继承未写字段不要求额外写权。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC018](011-m02-business-requirements.md#m02-ac-018)、[AC023](011-m02-business-requirements.md#m02-ac-023)；主G；补A |

<a id="tl-ui"></a>
## 12 联合预览、页面状态与返回保护

从正常详情进入编辑；本章均要求实际浏览器和服务端合同联验，不用前端模拟期间结果。 本章默认P1。

| 案例 | 前置与输入 | 操作步骤 | 明确预期（并执行第2.3节共同断言） | 验收依据 · 主层/补层 |
| --- | --- | --- | --- | --- |
| <a id="m02-tl-150"></a>TL-150 缺项继续编辑（109退役） | F2；空items、ADD缺日期/原因、UPDATE缺修改面 | 逐次预览后补齐 | 200 incomplete、canSave=false、candidateTimeline/impact=null；仅给可确定默认，无持久草稿/提交结果。 | [AC016](011-m02-business-requirements.md#m02-ac-016)、[AC022](011-m02-business-requirements.md#m02-ac-022)；主B；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-151"></a>TL-151 非法集合补成合法（109退役） | F2；先加03-01 S，再加入04-01 R A | 逐步预览 | 先invalid且无最终期间，再valid且canSave=true；单项依赖不能封死加入关联项入口。 | [AC016](011-m02-business-requirements.md#m02-ac-016)、[AC022](011-m02-business-requirements.md#m02-ac-022)；主B；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-152"></a>TL-152 未知分类与已知越权区分（109退役） | ADD尚未选类型；另一项已明确越权字段 | 分别预览 | 未选类别awaiting_input，不伪称已授权；已知越权立即安全拒绝并清理，不以缺项掩盖。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC018](011-m02-business-requirements.md#m02-ac-018)、[AC022](011-m02-business-requirements.md#m02-ac-022)；主A；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-153"></a>TL-153 缺席与非法输入区分（109退役） | 预览日期缺席、空串、null、非法日；重复JSON键/额外属性 | 逐个调用 | 缺席可incomplete；非法格式仍400，不把任意JSON当宽松草稿；正式提交不放宽必填。 | [AC013](011-m02-business-requirements.md#m02-ac-013)、[AC022](011-m02-business-requirements.md#m02-ac-022)；主A；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-154"></a>TL-154 能力加入与保存分离（109退役） | F5；单删s1/r1依赖冲突；F1首条删除 | 检查单条动作、canInclude和canSave | 关联启停可以明确加入联合，补齐才保存；首条删除/历史编辑始终不可加入，不能拼各行能力得到可保存。 | [AC005](011-m02-business-requirements.md#m02-ac-005)、[AC016](011-m02-business-requirements.md#m02-ac-016)、[AC017](011-m02-business-requirements.md#m02-ac-017)、[AC022](011-m02-business-requirements.md#m02-ac-022)；主B；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-155"></a>TL-155 预览默认失效（109退役） | F1先预览f改B及3月英文更正；再删除前序必填值或改日期 | 连续编辑并延迟旧响应 | 旧默认和完整候选失效，迟到响应不得覆盖新输入；不沿用先前B当未知前序。 | [AC019](011-m02-business-requirements.md#m02-ac-019)、[AC022](011-m02-business-requirements.md#m02-ac-022)；主B；条件F；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-156"></a>TL-156 预览合法后外部变化（109退役） | F2合法预览后另一会话改来源、撤权或新增阻断引用 | 分别尝试保存 | 正式提交重新校验并R，不把预览当许可；来源变化重核全部项，越权清受限值。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC010](011-m02-business-requirements.md#m02-ac-010)、[AC011](011-m02-business-requirements.md#m02-ac-011)、[AC022](011-m02-business-requirements.md#m02-ac-022)；主B；条件F；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-157"></a>TL-157 整笔同值可明确结束（109退役） | F2合法全同值联合 | 预览、保存、查回 | valid且提示全同值；保存N不制造版本，返回清楚的无变化结果。 | [AC016](011-m02-business-requirements.md#m02-ac-016)、[AC022](011-m02-business-requirements.md#m02-ac-022)、[AC023](011-m02-business-requirements.md#m02-ac-023)；主B；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-158"></a>TL-158 影响展示完整且可见 | 固定删除s1/r1，恢复名称不同且后有继承停用及独立恢复；另例隐藏英文读权 | 比较Go确认内容、保存结果与旧来源 | 仅展示实际可读变化；包含后续停用继承范围，独立快照保持，隐藏字段不产生泄露值的差异行。旧任意多项分离更正参数退出。 | AC016/019/021；主B；补D |
| <a id="m02-tl-159"></a>TL-159 每个入口取消无写入 | 创建/新增/更正/日期/单条删除/启停及两个固定弹窗 | 输入后取消，切对象/记录/历史模式分别选继续及放弃 | 继续保留可见输入，放弃回原上下文；无来源/事件/原因强填，也不删除已保存事实。 | [AC007](011-m02-business-requirements.md#m02-ac-007)、[AC013](011-m02-business-requirements.md#m02-ac-013)、[AC016](011-m02-business-requirements.md#m02-ac-016)；主B |
| <a id="m02-tl-160"></a>TL-160 保存和删除后的定位 | F2在有筛选/分页条件下新增、改日、更正、删除；固定两条另例 | 完成后返回列表与详情 | 按共同UI断言选中实际结果，保留原查看/筛选/排序/页码；对象不再匹配或页码超界有说明，不静默丢上下文。 | [AC007](011-m02-business-requirements.md#m02-ac-007)、[AC014](011-m02-business-requirements.md#m02-ac-014)、[AC016](011-m02-business-requirements.md#m02-ac-016)；主B |
| <a id="m02-tl-161"></a>TL-161 未知结果离开后恢复 | 固定两条保存响应丢失，关闭编辑/刷新/重开并重新登录 | 按原提交引用查回 | 不把关闭当取消保存，不保存表单正文/凭据；原结果确定前不创建新身份，原表单卸载后迟到响应不导航。 | [AC007](011-m02-business-requirements.md#m02-ac-007)、[AC010](011-m02-business-requirements.md#m02-ac-010)、[AC026](011-m02-business-requirements.md#m02-ac-026)；主B；条件F |
| <a id="m02-tl-162"></a>TL-162 旧会话迟到响应 | 原会话保存/resolve未返回，切换账号或新会话；另一查回已终结 | 释放旧响应及错误 | 不清新会话数据、不重置下一次提交、不重新置U；无权旧正文和内存重试包清理，待确认引用按原主体隔离。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC010](011-m02-business-requirements.md#m02-ac-010)、[AC026](011-m02-business-requirements.md#m02-ac-026)；主B；条件F |
| <a id="m02-tl-163"></a>TL-163 固定拒绝与刷新定位 | 同一空档内恢复日不晚于停用日；或所确认配对已失效，丢弃拒绝响应 | 仅原提交引用查回 | 返回原安全拒绝，不重跑当前校验、不恢复输入；固定日期字段或整体配对定位，不支持数组重排。 | [AC027](011-m02-business-requirements.md#m02-ac-027)；主A；补B；条件F |
| <a id="m02-tl-164"></a>TL-164 超限和服务不可用反馈 | 合法预览超时；写入超时；结果格式暂不可读；初始化未就绪 | 分别恢复后重试读取 | 自动辅助失效清理旧结果但不禁止直接保存，最终Go命令校验；写/结果U保留引用；不以旧缓存冒充成功/未提交，恢复后重新授权查回。 | [AC028](011-m02-business-requirements.md#m02-ac-028)、[AC029](011-m02-business-requirements.md#m02-ac-029)；主B；条件F |
| <a id="m02-tl-165"></a>TL-165 长编辑区与键盘可达 | 固定插入与含名称变化的删除弹窗，桌面及窄屏 | 真实滚轮/键盘滚动、错误聚焦、确认关闭和返回 | 完整时间轴/日志/保存按钮可达，焦点回触发处；状态非仅颜色表达，不通过DOM改状态替代办理。 | [AC007](011-m02-business-requirements.md#m02-ac-007)、[AC016](011-m02-business-requirements.md#m02-ac-016)；主B |

<a id="tl-access-reference"></a>
## 13 当前权限、真实引用与隔离

默认从正常菜单读取有权BU，再按案例收窄授权或直调；引用端口替身必须单独标记，不能计M03/M04已接入。 本章默认P0。

| 案例 | 前置与输入 | 操作步骤 | 明确预期（并执行第2.3节共同断言） | 验收依据 · 主层/补层 |
| --- | --- | --- | --- | --- |
| <a id="m02-tl-166"></a>TL-166 只读角色不能维护 | R拥有读取/历史；八命令各用合法输入和目标 | 逐一尝试CREATE/ADD/CORRECT/REDATE/DELETE/DEACTIVATE/REACTIVATE/REVISE_TIMELINE | 写入全部拒绝，查询仍可用；无隐藏按钮绕过、无业务事实和成功审计。 | [AC009](011-m02-business-requirements.md#m02-ac-009)；主A |
| <a id="m02-tl-167"></a>TL-167 对象与实例隔离 | X及仅BU1授权者；使用BU2/实例I2的BU、record、revision、source | 逐种组合读写/查回 | 安全拒绝且不泄露名称、计数或依赖；不按全局UUID碰巧存在就读取/写入。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC011](011-m02-business-requirements.md#m02-ac-011)、[AC016](011-m02-business-requirements.md#m02-ac-016)；主A |
| <a id="m02-tl-168"></a>TL-168 日期专用权矩阵 | 分别只授普通、首条、S、R改日权，字段effectiveDate齐备 | 对四类合法目标交叉办理16格 | 仅相应类别权可办理；首条不要求普通改日权，S/R也不能由普通权代替；无字段权另拒绝。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC014](011-m02-business-requirements.md#m02-ac-014)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G；补A |
| <a id="m02-tl-169"></a>TL-169 删除专用权矩阵 | 分别只授普通、S、R删除权，各取无依赖尾条 | 对三类目标交叉办理9格，再试首条 | 仅匹配类别成功；首条无任何权可删；不额外要求名称写权或错误地强制普通删除权。 | [AC005](011-m02-business-requirements.md#m02-ac-005)、[AC009](011-m02-business-requirements.md#m02-ac-009)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G；补A |
| <a id="m02-tl-170"></a>TL-170 有效类型决定授权 | F2 c2原来源为CHANGE，当前经分类纠错已成为S | 普通删除权主体与仅S删除权L分别删c2 | 普通权拒绝，S专用权可办理合法删除；按所选来源修订类型判定，不用最初登记类型替代。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC018](011-m02-business-requirements.md#m02-ac-018)；主A |
| <a id="m02-tl-171"></a>TL-171 固定操作须两方实际权限 | 缺停用/恢复任一动作及其字段权；删除缺任一类别删除权；普通更正单独持权 | 对固定两条及普通更正分别提交 | 固定两条缺一方即整笔拒绝；普通更正按实际面办理，无通用入口权。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC016](011-m02-business-requirements.md#m02-ac-016)；主A |
| <a id="m02-tl-172"></a>TL-172 显式同值字段也需写权 | EW可读英文但不能写；普通更正中文真变、英文同值或null | 分别提交 | 两种英文显式声明均拒绝；只省略英文时可按权限改中文并保留原英文。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC019](011-m02-business-requirements.md#m02-ac-019)、[AC023](011-m02-business-requirements.md#m02-ac-023)；主A |
| <a id="m02-tl-173"></a>TL-173 默认继承不增加字段写权 | L有R动作/日期原因及读取，缺英文写权；F4 | 恢复省略changes，另例显式回传英文Alpha | 省略可W并独立保持Alpha，回传即使同值也拒绝；不可泄露无读权默认。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC015](011-m02-business-requirements.md#m02-ac-015)、[AC023](011-m02-business-requirements.md#m02-ac-023)；主A；补B |
| <a id="m02-tl-174"></a>TL-174 普通更正及固定结果读取与新执行分权 | 普通更正accepted/no_change/rejected含英文同值声明，固定两条accepted/rejected另验；仅撤英文写权，再撤实际动作写权，读权保持 | 各终态resolve及原内容重试；独立夹具以尚无终态身份重试。补当前CREATE有BU的accepted和无目标rejected对照；补B只用固定两条accepted丢响应后撤写权并刷新 | 已有终态仍可读、原引用解除且零业务新增；无终态执行因缺写权拒绝。原声明/判定不改写，撤读权的裁剪和拒绝见TL-218。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC023](011-m02-business-requirements.md#m02-ac-023)、[AC027](011-m02-business-requirements.md#m02-ac-027)；主A；补B；条件F |
| <a id="m02-tl-175"></a>TL-175 原结果与当前办理类别分离 | 原S改日后纠为CHANGE；另例S删除后原日新建CHANGE B；撤原S写权但保留读权 | 查回/重放原结果，再按当前目标尝试新维护 | 原结果仍原S身份/分类/日期且可读，不要求旧类别写权；新维护按当前类型核验写权。新身份不代替旧结果，失去对象/历史读权仍拒绝。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC010](011-m02-business-requirements.md#m02-ac-010)、[AC018](011-m02-business-requirements.md#m02-ac-018)、[AC023](011-m02-business-requirements.md#m02-ac-023)；主A；条件F |
| <a id="m02-tl-176"></a>TL-176 授权完整单元 | 一份Grant有BU1动作，另一份仅BU2字段；另用有效重叠读授权 | 提交BU1字段写入并查询列表 | 不能拼动作/范围越权；同BU合法读授权按合同合并，不因窄授权删除已有可读字段。 | [AC009](011-m02-business-requirements.md#m02-ac-009)；主A |
| <a id="m02-tl-177"></a>TL-177 认证和安全守卫不可用 | 会话失效、主体停用、缺policy_guard、授权存储故障；写缺CSRF或异源 | 分别预览、保存、历史读取及resolve | 拒绝且不返回受限正文；不临时补守卫、不用缓存allow放行；提交未知按原身份保护。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC010](011-m02-business-requirements.md#m02-ac-010)、[AC012](011-m02-business-requirements.md#m02-ac-012)；主A；条件F |
| <a id="m02-tl-178"></a>TL-178 撤权后的业务历史与原结果 | 一次成功后删除/收窄Grant或停用主体；新模型不存在授权过程档案 | 获权主体核对原业务来源；原主体重新操作/查回；以允许读取原结果的权限组合复验恢复 | 原业务来源/终态保持，当前访问重新判断；恢复仅依赖本域请求事实和结果，不复用旧许可、不需要原input/basis。新参数未运行。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC012](011-m02-business-requirements.md#m02-ac-012)、[AC023](011-m02-business-requirements.md#m02-ac-023)；主D；条件F；V128-03/04 |
| <a id="m02-tl-179"></a>TL-179 未接消费者不当故障 | 只有当前企业与M02内部引用，F1/F4/F5 | 分别做合法新增、停用/恢复、删除/改日 | 本域合法动作可W；不调用恒为空替身冒充全域保护，也不要求不存在消费者响应。 | [AC011](011-m02-business-requirements.md#m02-ac-011)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主A |
| <a id="m02-tl-180"></a>TL-180 历史引用不会一概阻断 | F5有T0来源、审计及受控不可变采用引用 | 删尾R或改合法日期 | 仅历史证据不阻断W，旧引用仍指T0；不自动重新采用、改员工或重算工资。 | [AC008](011-m02-business-requirements.md#m02-ac-008)、[AC011](011-m02-business-requirements.md#m02-ac-011)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G |
| <a id="m02-tl-181"></a>TL-181 真实业务点引用 | F5在05-15需ACTIVE；未来样本08-15需ACTIVE | 删r1或安排覆盖引用日的停用 | 均R REFERENCE_CONFLICT，检查历史/未来实际日期，不只看D；不暴露无权引用正文。 | [AC011](011-m02-business-requirements.md#m02-ac-011)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主G；条件F |
| <a id="m02-tl-182"></a>TL-182 段引用半开边界 | F1拟04-01停用；引用要求[03-01,04-01)或[03-01,04-02)启用 | 分别预览/提交 | 前者可W，后者R；上界不含04-01与跨入该日区别明确。 | [AC006](011-m02-business-requirements.md#m02-ac-006)、[AC011](011-m02-business-requirements.md#m02-ac-011)、[AC021](011-m02-business-requirements.md#m02-ac-021)；主G；条件F |
| <a id="m02-tl-183"></a>TL-183 首条后移消失期间 | F1在05-15需存在/启用；f改07-01 | 提交并读旧来源 | R，不允许把被引用日变NO_FACT；原事实及引用均保持。 | [AC011](011-m02-business-requirements.md#m02-ac-011)、[AC014](011-m02-business-requirements.md#m02-ac-014)；主G；条件F |
| <a id="m02-tl-184"></a>TL-184 分离区间不扩大守卫 | F9更正f和c2；引用分别位于02-15、04-15、06-15 | 记录传给端口的精确差异及各类冻结判据 | 影响是两段，04-15未变不被外包范围误阻断；实际受影响字段/资格不满足才拒绝。 | [AC011](011-m02-business-requirements.md#m02-ac-011)、[AC021](011-m02-business-requirements.md#m02-ac-021)；主G；条件F |
| <a id="m02-tl-185"></a>TL-185 必要依赖失效或覆盖未知 | 已声明必要引用端口超时、版本失配、覆盖未知 | 对受影响写提交 | 不当空引用放行；依赖不可核验时503且无部分事实，原提交结果按协议确认；替身仅验此分支。 | [AC011](011-m02-business-requirements.md#m02-ac-011)、[AC025](011-m02-business-requirements.md#m02-ac-025)、[AC028](011-m02-business-requirements.md#m02-ac-028)；主A；条件F |
| <a id="m02-tl-186"></a>TL-186 受限依赖解释 | 有办理权但无阻断引用详情读取权 | 触发引用冲突及查回 | 只给安全的受限依赖说明，不泄露人员名称/字段/不可见ID，仍能保留输入重新核对。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC011](011-m02-business-requirements.md#m02-ac-011)、[AC027](011-m02-business-requirements.md#m02-ac-027)；主B；条件F |
| <a id="m02-tl-187"></a>TL-187 引用与写入共同守卫 | 建立ACTIVE点引用与停用同一日范围并发 | 两种取锁顺序分别安排 | 只能引用先成立而停用拒绝，或停用先成立而新引用拒绝；不能双方成功后异步修补。无真实入口仅计端口合同。 | [AC010](011-m02-business-requirements.md#m02-ac-010)、[AC011](011-m02-business-requirements.md#m02-ac-011)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主D；条件F |

<a id="tl-concurrency"></a>
## 14 提交幂等、版本竞争与终结顺序

使用实际多个连接/会话和确定屏障，禁止用固定sleep猜测先后。所有步骤记录submissionId、来源、终态及实际响应。 本章默认P0。

| 案例 | 前置与输入 | 操作步骤 | 明确预期（并执行第2.3节共同断言） | 验收依据 · 主层/补层 |
| --- | --- | --- | --- | --- |
| <a id="m02-tl-188"></a>TL-188 同身份同内容并发 | F2合法固定两条；同主体同submissionId | 同时提交两次完整相同内容 | 仅一份W，另一回原结果，来源/审计/事件各增1；不逐项重放。 | [AC010](011-m02-business-requirements.md#m02-ac-010)、[AC016](011-m02-business-requirements.md#m02-ac-016)；主D；条件F |
| <a id="m02-tl-189"></a>TL-189 固定请求JSON键序重放 | 同一已接受固定请求；只交换JSON键顺序 | 以原身份精确重试 | 原摘要及结果不变，两条结果定位稳定；不因传输排列新增来源。 | [AC010](011-m02-business-requirements.md#m02-ac-010)、[AC016](011-m02-business-requirements.md#m02-ac-016)、[AC027](011-m02-business-requirements.md#m02-ac-027)；主G；补A |
| <a id="m02-tl-190"></a>TL-190 同身份异内容 | 已接受请求后分别改itemId、日期、原因、目标、expectedSource或字段值 | 每次只改一项并用原身份 | 均M02_REQUEST_MISMATCH；缺席与null、不同输入字符串即使归一后同值也不能改原请求含义。 | [AC010](011-m02-business-requirements.md#m02-ac-010)、[AC013](011-m02-business-requirements.md#m02-ac-013)、[AC016](011-m02-business-requirements.md#m02-ac-016)、[AC023](011-m02-business-requirements.md#m02-ac-023)；主A |
| <a id="m02-tl-191"></a>TL-191 不同主体及实例相同UUID | A写BU1、A2写BU2；实例I2另例，用同一submissionId | 并发合法提交并各自查回 | 按实例/主体/UUID分别判重；不能串结果或暴露另一主体内容，两个无关BU可各W。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC010](011-m02-business-requirements.md#m02-ac-010)；主D；条件F |
| <a id="m02-tl-192"></a>TL-192 同BU不同身份旧基准竞争 | F2，A将c1改07-01；B独立取08-01 ADD Z、c2更正Z、c1改09-01、删除c2、12-01 S五种合法请求 | 每组合分别安排A/B先成功 | 仅首个W，后者VERSION_CONFLICT；目标不同也比较完整来源，改日与其他维护不能只比目标名称。 | [AC010](011-m02-business-requirements.md#m02-ac-010)、[AC014](011-m02-business-requirements.md#m02-ac-014)、[AC016](011-m02-business-requirements.md#m02-ac-016)；主D；条件F |
| <a id="m02-tl-193"></a>TL-193 不同BU无需全局版本冲突 | BU1/BU2各F2，A/B同时合法维护 | 并发保存并回读 | 两者W；共享企业/权限读取不变成全企业业务版本，不用全局摘要拒绝。 | [AC010](011-m02-business-requirements.md#m02-ac-010)；主D；条件F |
| <a id="m02-tl-194"></a>TL-194 同编码及同日竞争 | 两个新建编码大小写相同；F1同基准两个03-01 ADD分别为B/Z | 两组并发 | 前组至多一BU，另一CODE_CONFLICT；后组至多一个新日记录，另一版本/日期拒绝，无半份身份。 | [AC002](011-m02-business-requirements.md#m02-ac-002)、[AC003](011-m02-business-requirements.md#m02-ac-003)、[AC010](011-m02-business-requirements.md#m02-ac-010)、[AC025](011-m02-business-requirements.md#m02-ac-025)；主D；条件F |
| <a id="m02-tl-195"></a>TL-195 未接受旧请求跨午夜 | F1新增06-16 Z，预览D=06-15；正式校验D=06-16或06-17 | 保持其他守卫不变后提交 | 可W，分别当天/过去说明，不因日期从未来变过去拒绝；若来源已变则仍版本冲突。 | [AC003](011-m02-business-requirements.md#m02-ac-003)、[AC010](011-m02-business-requirements.md#m02-ac-010)、[AC024](011-m02-business-requirements.md#m02-ac-024)；主A；条件F |
| <a id="m02-tl-196"></a>TL-196 已接受后对象继续演进 | 一次更正/改日/删除已成功，随后当前来源前进或目录变更 | 原内容重试和无正文resolve | 返回原来源/原目标/原日期与原分类，不拿后来D/当前目录重跑；说明当前事实已变化。 | [AC008](011-m02-business-requirements.md#m02-ac-008)、[AC010](011-m02-business-requirements.md#m02-ac-010)、[AC013](011-m02-business-requirements.md#m02-ac-013)、[AC014](011-m02-business-requirements.md#m02-ac-014)、[AC024](011-m02-business-requirements.md#m02-ac-024)；主A |
| <a id="m02-tl-197"></a>TL-197 拒绝和无变化不可重新解释 | 保存过rejected/no_change后移除拒绝条件或改当前头 | 原身份重试/resolve | 仍原终态与原观察/安全诊断；新业务必须明确新身份，不把原N/R转成功。 | [AC010](011-m02-business-requirements.md#m02-ac-010)、[AC023](011-m02-business-requirements.md#m02-ac-023)、[AC027](011-m02-business-requirements.md#m02-ac-027)；主A；条件F |
| <a id="m02-tl-198"></a>TL-198 响应丢失与进程重启 | 固定两条COMMIT成功后丢响应并重启API | 刷新后resolve；另原正文精确重试 | 返回原accepted，一份来源；重启不依赖后台补日志、提交截点或持久表单。 | [AC010](011-m02-business-requirements.md#m02-ac-010)、[AC012](011-m02-business-requirements.md#m02-ac-012)、[AC016](011-m02-business-requirements.md#m02-ac-016)；主A；条件F |
| <a id="m02-tl-199"></a>TL-199 resolve先到封闭 | 原POST在到达前延迟，先按本人身份resolve | 确认not_applied后释放原POST | 迟到POST REQUEST_CLOSED且无业务写；封闭行无伪造action/digest/请求定位。 | [AC010](011-m02-business-requirements.md#m02-ac-010)、[AC026](011-m02-business-requirements.md#m02-ac-026)；主D；补A；条件F |
| <a id="m02-tl-200"></a>TL-200 原写持锁时resolve | 合法写持提交锁，resolve在其间到达 | 分别让写提交或回滚 | 查回等待/pending；提交后回accepted，回滚后可封闭；不能先报not_applied后写又成功。 | [AC010](011-m02-business-requirements.md#m02-ac-010)、[AC026](011-m02-business-requirements.md#m02-ac-026)、[AC028](011-m02-business-requirements.md#m02-ac-028)；主D；条件F |
| <a id="m02-tl-201"></a>TL-201 整体回滚后精确重试先成功 | A技术失败，整笔事务确定回滚，无终态；延迟A的未知响应，B原身份原内容重试 | B重新授权并accepted，再放行A响应及resolve；补B验证迟到未知反馈不覆盖成功 | 只有B一份来源及accepted；A不另开事务补rejected，不能返回确定未保存。resolve读同一终态，页面保持已确认成功。 | [AC026](011-m02-business-requirements.md#m02-ac-026)；主D；补B；条件F |
| <a id="m02-tl-202"></a>TL-202 整体回滚后resolve先封闭 | A整笔回滚，无终态，未知响应延迟；resolve先取得同一提交锁 | 保存not_applied后放行A响应及B重试；补B验证封闭结果不被迟到未知反馈覆盖 | 只有not_applied；B不可执行业务，A没有二次补拒绝。封闭不造摘要，无业务来源，页面以已确认结果结束待确认。 | [AC026](011-m02-business-requirements.md#m02-ac-026)；主D；补B；条件F |
| <a id="m02-tl-203"></a>TL-203 原锁内拒绝先终结 | A遇可恢复业务拒绝，只回滚业务savepoint，外层提交锁仍持有；B及resolve同时到达 | A在原事务保存rejected并COMMIT，再放行等待者；补异内容请求 | 等待者只见原rejected，同内容不执行，异内容MISMATCH；业务零写入。不存在先释放原锁再补拒绝的窗口。 | [AC026](011-m02-business-requirements.md#m02-ac-026)、[AC027](011-m02-business-requirements.md#m02-ac-027)；主D；条件F |
| <a id="m02-tl-204"></a>TL-204 原拒绝提交或查回出错 | A在原外层事务写rejected，分别在INSERT成功后、COMMIT前取消/耗尽截止时间，以及COMMIT失败或通信未知；显式resolve另设取锁/存储超时、撤读取权分支 | 按真实已存终态恢复；观察原事务结束后无自动补写 | 原COMMIT未确认只报U；resolve读到终态才确认，否则pending或安全读取拒绝。没有新失败终结者，不覆盖已存accepted/rejected/not_applied。 | [AC025](011-m02-business-requirements.md#m02-ac-025)、[AC026](011-m02-business-requirements.md#m02-ac-026)、[AC028](011-m02-business-requirements.md#m02-ac-028)；主D；条件F |
| <a id="m02-tl-205"></a>TL-205 写与撤权线性顺序 | A写持共享授权守卫，撤权取排他；再反向安排 | 各顺序独立执行 | 写先获守卫可先W再撤权；撤权先完成则写R。不能事前缓存许可后越过撤权保存。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC010](011-m02-business-requirements.md#m02-ac-010)、[AC012](011-m02-business-requirements.md#m02-ac-012)；主D；条件F |
| <a id="m02-tl-206"></a>TL-206 登录与停用竞争 | 有效账号，登录/续访与停用或凭据轮换并发；首次同源登录无尚未签发的CSRF | 两种锁顺序及缺失/异源Origin分别验证 | 首次同源登录可行，缺失/异源Origin拒绝；停用后无存活越权会话，访问不延长绝对到期，旧响应不改新会话。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC010](011-m02-business-requirements.md#m02-ac-010)；主A；条件F |
| <a id="m02-tl-207"></a>TL-207 死锁、锁超时与碰撞隔离 | 故障注入死锁/锁超时；另用受控同advisory键模拟两个完整提交身份碰撞 | 等待解除后按原身份恢复 | 不得无限重试或提交部分事实；碰撞仅串行，按完整键分别读结果，不合并身份。模拟键只计守卫合同。 | [AC010](011-m02-business-requirements.md#m02-ac-010)、[AC025](011-m02-business-requirements.md#m02-ac-025)、[AC028](011-m02-business-requirements.md#m02-ac-028)；主D；条件F |

<a id="tl-failure"></a>
## 15 失败原子性、持久诊断与执行预算

对同一合法联合混合输入逐阶段注入；每次重建T0并对照身份集合、头、来源/修订/项、审计、事件和结果。 本章默认P0。

| 案例 | 前置与输入 | 操作步骤 | 明确预期（并执行第2.3节共同断言） | 验收依据 · 主层/补层 |
| --- | --- | --- | --- | --- |
| <a id="m02-tl-208"></a>TL-208 业务409不得泄漏身份 | 既有BU已插新effective_record而其他业务尚未完成 | 注入业务409并结束事务 | 整笔业务savepoint撤销，无孤立身份；仅确定提交的安全判定/拒绝可留，不能因Fault<500提交业务。 | [AC012](011-m02-business-requirements.md#m02-ac-012)、[AC025](011-m02-business-requirements.md#m02-ac-025)；主D；条件F |
| <a id="m02-tl-209"></a>TL-209 写阶段与异常类型覆盖 | 记录身份、来源头、修订、成员、source_change_item、审计、Outbox、accepted结果、BU头九阶段 | 各阶段各注入一次技术失败；另在写修订后panic、写Outbox后取消，覆盖共同异常清理入口 | 各子例均无部分事实；以9个阶段反例加2个异常类型代表例覆盖共享事务边界，不强制九阶段×三类型。COMMIT前可确认回滚，清理不明则U；实现出现不同清理分支时补相应例。 | [AC012](011-m02-business-requirements.md#m02-ac-012)、[AC016](011-m02-business-requirements.md#m02-ac-016)、[AC020](011-m02-business-requirements.md#m02-ac-020)、[AC025](011-m02-business-requirements.md#m02-ac-025)；主D；条件F |
| <a id="m02-tl-210"></a>TL-210 savepoint清理失败 | 业务写后错误，ROLLBACK TO本身失败；外层连接不可用另例 | 执行有界清理及resolve | 不继续同事务写rejected或COMMIT残留；外层回滚/丢弃连接，结果按真实终态确认。 | [AC025](011-m02-business-requirements.md#m02-ac-025)、[AC028](011-m02-business-requirements.md#m02-ac-028)；主D；条件F |
| <a id="m02-tl-211"></a>TL-211 技术故障不伪装业务拒绝 | 候选完成前/写入中发生依赖故障或取消 | 检查返回及command_result；显式resolve前后分开观察 | 整笔失败不自动补rejected；没有确定终态保持U，只有用户发起的resolve可封闭。可恢复业务拒绝的原锁内保存另由TL-208/203验证。 | [AC012](011-m02-business-requirements.md#m02-ac-012)、[AC025](011-m02-business-requirements.md#m02-ac-025)、[AC026](011-m02-business-requirements.md#m02-ac-026)；主A；条件F |
| <a id="m02-tl-212"></a>TL-212 判权与必要事实原子性 | 注入当前授权依赖失败、业务/必要事件写入失败；普通读取重复执行 | 提交/查回并核对事实与终态；检查新模型目录和运行SQL | 授权失败不能越权，实际写入失败回滚或待确认；无需授权档案仍能恢复原结果，普通判权无过程记录增长。新参数未运行。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC012](011-m02-business-requirements.md#m02-ac-012)、[AC025](011-m02-business-requirements.md#m02-ac-025)；主D；条件F；V128-01—05 |
| <a id="m02-tl-213"></a>TL-213 提交结果三分支 | COMMIT前回滚、COMMIT成功丢响应、COMMIT通信未知 | 三例按原身份查回 | 分别按封闭/成功/待确认推进；不能由503或断连猜未保存，成功只有一份来源。 | [AC010](011-m02-business-requirements.md#m02-ac-010)、[AC012](011-m02-business-requirements.md#m02-ac-012)、[AC025](011-m02-business-requirements.md#m02-ac-025)、[AC028](011-m02-business-requirements.md#m02-ac-028)；主A；条件F |
| <a id="m02-tl-214"></a>TL-214 同字段多项拒绝准确定位（109退役） | c1/c2两UPDATE均声明name/nameEn，仅c1中文201码点 | 丢拒绝响应、刷新查回、重排精确重试 | failure_details始终指c1项的changes.name；不按数组下标，不凭摘要或字段集合猜错误。 | [AC027](011-m02-business-requirements.md#m02-ac-027)；主A；条件F；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-215"></a>TL-215 原拒绝不受当前头改变 | 保存上述rejected后其他请求推进来源或目录变化 | 无正文resolve | 回原安全诊断及原expectedSource，不重验后改为版本冲突；不返回后来输入或原名称值。 | [AC027](011-m02-business-requirements.md#m02-ac-027)；主A |
| <a id="m02-tl-216"></a>TL-216 更正字段错误的稳定顺序 | 同一普通更正的中文/英文均有合法解析后的字段错误，交换JSON键顺序 | 同一基准提交并比较安全诊断，原身份重放 | 按固定规则优先级定位原错误；固定启停请求另测字段/整体拒绝，不接受项数组或重排。 | [AC027](011-m02-business-requirements.md#m02-ac-027)；主G |
| <a id="m02-tl-217"></a>TL-217 诊断安全和版本组合 | 当前CREATE及维护rejected；缺失/未知failure格式或非法itemId位置 | 正常保存及受控损坏读取 | 各动作安全issues完整；CREATE无多记录项不造itemId、无BU不造对象引用。损坏M02_RESULT_UNAVAILABLE保留引用，不panic或改成not_applied。 | [AC013](011-m02-business-requirements.md#m02-ac-013)、[AC027](011-m02-business-requirements.md#m02-ac-027)、[AC029](011-m02-business-requirements.md#m02-ac-029)；主D；条件F |
| <a id="m02-tl-218"></a>TL-218 当前读取权限裁剪诊断和结果 | 原accepted/no_change/rejected保持；分别只撤英文读权、对象读权、历史读权；原写权有无不影响该读取测试 | resolve及已有终态重放，再恢复读权；英文诊断使用TL-214的对应非法英文变体 | 仅字段读权撤销则裁剪该字段/位置/关联安全明细，仍可确认安全终态；对象/历史失权安全拒绝并保留引用。恢复读权回原结果，不要求恢复写权或旁路X03。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC023](011-m02-business-requirements.md#m02-ac-023)、[AC027](011-m02-business-requirements.md#m02-ac-027)；主A；条件F |
| <a id="m02-tl-219"></a>TL-219 正文大小和阻塞读取 | 合法JSON以末尾空白补齐恰64KiB、64KiB+1；另Body.Read停滞 | 分别发送，使用真实读取截止时间 | 64KiB仍受业务校验；超限拒绝，阻塞读取最迟5秒且不超整体剩余时间终止；context自身不足不能计通过。 | [AC028](011-m02-business-requirements.md#m02-ac-028)；主A；条件F |
| <a id="m02-tl-220"></a>TL-220 单项长当前线的取消延迟 | 代表性长当前线，只更正一个字段；排序使用标准库，前后检查取消 | 在候选/排序/继承/影响阶段发起取消，测量请求取消至本地计算停止的最大延迟 | 按013初始1秒响应预算核对，不开始新的业务步骤、不截断候选。未达标记录瓶颈再优化，不预定可取消排序；真实DB清理见TL-224，承载边界见TL-256。 | [AC025](011-m02-business-requirements.md#m02-ac-025)、[AC028](011-m02-business-requirements.md#m02-ac-028)；主G；条件F |
| <a id="m02-tl-221"></a>TL-221 累计SQL超过整体预算 | 数条SQL各<5秒，累积超过10秒 | 保持单条语句合法并提交 | 整体截止停止后续工作，不给每条/每项重开10秒；无分段提交，结果按原身份查回。 | [AC025](011-m02-business-requirements.md#m02-ac-025)、[AC028](011-m02-business-requirements.md#m02-ac-028)；主D；条件F |
| <a id="m02-tl-222"></a>TL-222 连接池和锁计入同一预算 | 先耗连接池等待，再等待授权/提交/BU锁 | 记录handler进入、剩余截止与取消 | 正文/认证/连接/锁均计时，锁至多2秒且不超剩余预算；不在BEGIN后重新计时。 | [AC028](011-m02-business-requirements.md#m02-ac-028)；主D；条件F |
| <a id="m02-tl-223"></a>TL-223 预览与resolve各有上限 | 预览计算或resolve取锁/读取超过5秒 | 等待返回并检查页面/DB | 预览503 EXECUTION_LIMIT且失效，无提交身份；resolve pending且引用保留，不当确定未提交。 | [AC022](011-m02-business-requirements.md#m02-ac-022)、[AC028](011-m02-business-requirements.md#m02-ac-028)；主A；条件F |
| <a id="m02-tl-224"></a>TL-224 有界清理和DB实际释放 | 原context取消、清理连接失联或回滚阻塞 | 观察2秒清理、连接丢弃及12秒空闲事务兜底 | 本地不无限WithoutCancel；实测DB锁释放，网络不可观测就记未知，不把本地返回当回滚证明。 | [AC025](011-m02-business-requirements.md#m02-ac-025)、[AC028](011-m02-business-requirements.md#m02-ac-028)；主D；条件F |
| <a id="m02-tl-225"></a>TL-225 清理不重开业务预算或补写终态 | A整体10秒将耗尽或已取消，业务事务回滚 | 观察仅进行有界清理；随后用户另发显式resolve | 无自动另开事务补rejected/not_applied，不续期执行业务；清理最多2秒，用户resolve独立5秒且遵守唯一终态。 | [AC026](011-m02-business-requirements.md#m02-ac-026)、[AC028](011-m02-business-requirements.md#m02-ac-028)；主D；条件F |
| <a id="m02-tl-226"></a>TL-226 写入跨午夜保留原办理日 | 写前采样D=06-15，新增06-16；插入来源跨到06-16；单条ADD及固定插入停用/恢复分别测 | 查原结果并在06-17重放/resolve | 原说明仍未来安排，evaluatedAt/业务日为原采样，recordedAt为实际16日；当前状态另读，不重标为过去。 | [AC024](011-m02-business-requirements.md#m02-ac-024)；主D；补B；条件F |
| <a id="m02-tl-227"></a>TL-227 原采样空值与同值结果 | 当前CREATE/维护accepted、维护no_change、提前rejected、not_applied | 逐种读取原办理说明 | accepted和N都有原采样，N无业务登记时间；提前拒绝/封闭不补造采样。服务器字段不能由请求传入，不保留旧CREATE空采样例外。 | [AC023](011-m02-business-requirements.md#m02-ac-023)、[AC024](011-m02-business-requirements.md#m02-ac-024)、[AC027](011-m02-business-requirements.md#m02-ac-027)；主D；条件F |
| <a id="m02-tl-228"></a>TL-228 正式提交协议完整性 | 八命令的固定DTO及公共解析器；缺/坏提交ID、UUID、版本、重复JSON键、额外字段及Content-Type错误 | 公共解析错误全集在代表维护命令执行；其余入口验证一组公共拒绝及各自特有必填/白名单边界 | 所有入口接同一严格解析合同；公共非法输入无业务写入，各DTO专有边界全部覆盖，不将同一解析器全集重复八遍。 | [AC002](011-m02-business-requirements.md#m02-ac-002)、[AC010](011-m02-business-requirements.md#m02-ac-010)、[AC013](011-m02-business-requirements.md#m02-ac-013)、[AC016](011-m02-business-requirements.md#m02-ac-016)、[AC027](011-m02-business-requirements.md#m02-ac-027)；主A |

<a id="tl-storage-upgrade"></a>
## 16 存储约束、源事件与当前初始化

由Repository maintainer在隔离空库使用真实运行角色/upgrader验证当前基线；SQL夹具不计正常菜单业务验收。本章现行案例默认P0。TL-241、242、246、247保留原输入和预期作为历史，按025退役，不执行、不计覆盖或通过；编号不复用。

| 案例 | 前置与输入 | 操作步骤 | 明确预期（并执行第2.3节共同断言） | 验收依据 · 主层/补层 |
| --- | --- | --- | --- | --- |
| <a id="m02-tl-229"></a>TL-229 来源成员完整和范围一致 | 实际合法创建/固定两条来源；另构造跨BU/跨record修订引用 | 核对完整成员，再分别尝试非法复合键 | 合法来源全部可解析；非法实例/BU/记录混用由约束拒绝，不依赖前端过滤。 | [AC008](011-m02-business-requirements.md#m02-ac-008)、[AC012](011-m02-business-requirements.md#m02-ac-012)、[AC016](011-m02-business-requirements.md#m02-ac-016)；主D |
| <a id="m02-tl-230"></a>TL-230 期间与日期约束 | 新来源成员中半空引用、复制日期/valid_range不一致、重叠区间、重复生效日 | 各在可写的新事务夹具中单独尝试 | 对应NOT NULL/CHECK/外键/排斥或唯一约束拒绝；合法相邻半开段通过。 | [AC003](011-m02-business-requirements.md#m02-ac-003)、[AC012](011-m02-business-requirements.md#m02-ac-012)、[AC014](011-m02-business-requirements.md#m02-ac-014)；主D |
| <a id="m02-tl-231"></a>TL-231 来源与修订前驱 | 跨BU头、跳号/循环来源前驱、跨记录修订前驱 | 分别以运行角色写入非法关系 | 全部拒绝；合法同BU连续来源、同record修订可保存，头只指真实完整来源。 | [AC008](011-m02-business-requirements.md#m02-ac-008)、[AC012](011-m02-business-requirements.md#m02-ac-012)、[AC025](011-m02-business-requirements.md#m02-ac-025)；主D |
| <a id="m02-tl-232"></a>TL-232 旧来源封口和运行角色 | 创建T0后在新事务向T0追加成员/修订/日志/事件，伪造xid8；尝试UPDATE/DELETE/TRUNCATE | 逐表执行并回读T0 | 全部拒绝，无晚追加；运行角色不能禁守卫或改首条指针/归属/编码；新来源复用旧修订合法。 | [AC008](011-m02-business-requirements.md#m02-ac-008)、[AC012](011-m02-business-requirements.md#m02-ac-012)、[AC025](011-m02-business-requirements.md#m02-ac-025)；主D |
| <a id="m02-tl-233"></a>TL-233 删除修订与成员分开 | F2删除c1；另尝试把删除修订列入新成员 | 查删除结果并执行非法插入 | 原删除结果引用本次删除修订，页面所选前段另定位；删除修订不得进入有效成员。 | [AC005](011-m02-business-requirements.md#m02-ac-005)、[AC008](011-m02-business-requirements.md#m02-ac-008)、[AC012](011-m02-business-requirements.md#m02-ac-012)；主D |
| <a id="m02-tl-234"></a>TL-234 单条及多记录终态组合 | accepted/rejected/no_change/not_applied逐种合法保存；另试半空/混合列 | 读取结果及复合外键 | 单条accepted有目标，固定两条accepted顶层目标空且有项；拒绝/封闭无成功来源，N仅观察既有来源；非法组合拒绝。 | [AC010](011-m02-business-requirements.md#m02-ac-010)、[AC012](011-m02-business-requirements.md#m02-ac-012)、[AC016](011-m02-business-requirements.md#m02-ac-016)、[AC027](011-m02-business-requirements.md#m02-ac-027)；主D |
| <a id="m02-tl-235"></a>TL-235 结果不能冒领来源 | 已有accepted及来源，另一主体/提交试关联同来源；原结果UPDATE/DELETE另试 | 直接SQL及API查回 | 来源唯一归原提交，终态只INSERT；跨主体同UUID不冒领，原结果不可改写。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC010](011-m02-business-requirements.md#m02-ac-010)、[AC012](011-m02-business-requirements.md#m02-ac-012)、[AC026](011-m02-business-requirements.md#m02-ac-026)；主D |
| <a id="m02-tl-236"></a>TL-236 固定两条直接项与派生项 | 固定ADD/ADD、DELETE/DELETE，必要期间与停用继承派生；另构造缺/重项、错误before及旧after修订 | 核对Go集合检查及DB外键/唯一性职责 | 合法直接项与本次修订一一对应；旧no_change结果只读核验，派生归整体；非法集合/旧after拒绝，每record仅一本次修订。 | [AC016](011-m02-business-requirements.md#m02-ac-016)、[AC021](011-m02-business-requirements.md#m02-ac-021)、[AC023](011-m02-business-requirements.md#m02-ac-023)；主D |
| <a id="m02-tl-237"></a>TL-237 影响与当前事件格式 | 当前CREATE、单条及固定两条维护均schemaVersion=2；分离区间和纯结构变化 | 读取Outbox/API/预览，尝试非当前版本、旧单范围字段或损坏组合 | 创建影响从首条起且含新增结构；各动作精确集合/结构引用一致，数据空结构非空合法，两者皆空不可发；非法格式拒绝，无v1读取/写入分支。 | [AC012](011-m02-business-requirements.md#m02-ac-012)、[AC016](011-m02-business-requirements.md#m02-ac-016)、[AC021](011-m02-business-requirements.md#m02-ac-021)；主D |
| <a id="m02-tl-238"></a>TL-238 源保存不冒充投递采用 | 实际新来源与Outbox已保存，无消费者 | 重启读取来源/事件及页面状态 | 可证明源原子落库，明确未接入/未投递，不报已采用，不新增通用投递平台作为本批前置。 | [AC008](011-m02-business-requirements.md#m02-ac-008)、[AC011](011-m02-business-requirements.md#m02-ac-011)、[AC012](011-m02-business-requirements.md#m02-ac-012)；主A；补B |
| <a id="m02-tl-239"></a>TL-239 当前模型空库安装 | 隔离空库、当前迁移基线和匹配程序；合成创建及历史样本 | 安装、初始化后执行创建/查询；维护存储夹具另验日期、类型和多目标终态 | 空库完整安装；当前格式的身份/修订/来源及诊断可存取，后续业务不改原来源和结果。存储夹具不计维护流程通过；无00004升级或旧CREATE元数据例外。 | [AC014](011-m02-business-requirements.md#m02-ac-014)、[AC015](011-m02-business-requirements.md#m02-ac-015)、[AC016](011-m02-business-requirements.md#m02-ac-016)、[AC027](011-m02-business-requirements.md#m02-ac-027)、[AC029](011-m02-business-requirements.md#m02-ac-029)；主M |
| <a id="m02-tl-240"></a>TL-240 初始化重复与失败关闭 | 当前空库安装中途失败；成功后准备当前模型资料再重复同配置初始化 | 检查实际迁移状态、业务路由及已有事实摘要 | 失败不开放业务，成功后相同初始化幂等且不改已有身份/来源/结果；实际回滚范围按迁移事务留证，不验证旧版本升级或回退。 | [AC025](011-m02-business-requirements.md#m02-ac-025)、[AC029](011-m02-business-requirements.md#m02-ac-029)；主M；条件F |
| <a id="m02-tl-241"></a>TL-241 一次密码轮换隔离旧配置（退役） | 实际旧程序及升级前配置；本次维护迁移采用既有最小权限角色换密 | 排空后轮换一次，旧程序用旧配置启动及旧密码直连均尝试；重复升级及失败恢复 | 旧配置不可连接；匹配新程序用新密码完成准入。重复/恢复复用已成功轮换配置、不自动再换密；不新增权限，不要求制品摘要/清单启动器。 | 历史关联：[AC029](011-m02-business-requirements.md#m02-ac-029)；按[025](../../adr/025-greenfield-implementation-policy.md)退役，不执行、不计通过 |
| <a id="m02-tl-242"></a>TL-242 存量连接排空（退役） | 旧实例仍有活动及空闲DB连接，旧登录已NOLOGIN | 尝试升级，随后停止/排空全部所属连接 | 存量连接存在时不得升级；NOLOGIN不会自动断旧连接；确认隔离后方可备份/迁移。 | 历史关联：[AC029](011-m02-business-requirements.md#m02-ac-029)；按[025](../../adr/025-greenfield-implementation-policy.md)退役，不执行、不计通过 |
| <a id="m02-tl-243"></a>TL-243 缺失和未知迁移不接入 | 当前程序连接缺失/不完整/不匹配当前基线的数据库 | 请求登录、列表、写入、历史、resolve以及readyz | 所有业务入口拒绝，不只是探针失败；不创建提交结果，不在结构不匹配时尝试业务读写，无多版本兼容路径。 | [AC029](011-m02-business-requirements.md#m02-ac-029)；主M |
| <a id="m02-tl-244"></a>TL-244 结果解码不假定单目标 | 当前格式的真实单条accepted及固定两条accepted；未知/损坏版本或必需元数据缺失 | 重启当前程序后按各原身份查回；当前固定两条提供真实保存证据，旧通用来源只读 | 当前合法结果安全解释；多记录顶层revision空不panic；坏格式RESULT_UNAVAILABLE保留ID，不伪造not_applied/最新结果。SQL夹具不替代T08真实流程。 | [AC016](011-m02-business-requirements.md#m02-ac-016)、[AC027](011-m02-business-requirements.md#m02-ac-027)、[AC029](011-m02-business-requirements.md#m02-ac-029)；主A；条件F |
| <a id="m02-tl-245"></a>TL-245 初始化与新增权限不自动扩张 | 当前模型初始化只读/普通维护主体，生命周期两方权限尚未配置 | 检查能力并尝试未授予动作；显式授权后再验证 | 初始化或重跑不把普通Grant变全权，专用动作明确授予后才开放；最小运行角色不扩为迁移所有者。 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC015](011-m02-business-requirements.md#m02-ac-015)、[AC016](011-m02-business-requirements.md#m02-ac-016)、[AC029](011-m02-business-requirements.md#m02-ac-029)；主M |
| <a id="m02-tl-246"></a>TL-246 受控恢复后真实读回（退役） | 保存升级前备份，升级失败或需要回退；使用匹配备份/旧程序 | 还原并从菜单创建/查询；升级成功另一分支验真实联合查回 | 匹配组合可核对旧事实，禁止旧程序直接读新库；SQL结果夹具只计解码，T08另供真实联合流程；不称生产恢复已完成。 | 历史关联：[AC008](011-m02-business-requirements.md#m02-ac-008)、[AC012](011-m02-business-requirements.md#m02-ac-012)、[AC016](011-m02-business-requirements.md#m02-ac-016)、[AC029](011-m02-business-requirements.md#m02-ac-029)；按[025](../../adr/025-greenfield-implementation-policy.md)退役，不执行、不计通过 |
| <a id="m02-tl-247"></a>TL-247 旧时间证据退役回归（退役） | 保留00003业务资料及缺提交证据/CAPTURE_CLOCK_REGRESSED受控旧夹具 | 先按既有00004退役路径升级，再执行维护迁移 | 旧身份/来源/摘要保持；不依赖采集worker、逻辑槽或knownAt读取历史；旧迁移通过不能代替新维护验收。 | 历史关联：[AC008](011-m02-business-requirements.md#m02-ac-008)、[AC012](011-m02-business-requirements.md#m02-ac-012)、[AC029](011-m02-business-requirements.md#m02-ac-029)；按[025](../../adr/025-greenfield-implementation-policy.md)退役，不执行、不计通过 |

<a id="tl-sequences"></a>
## 17 组合序列、模型性质与容量

以独立手工预期或受限小状态枚举核对Go，不复用受测候选函数生成答案；记录种子与最小失败序列。 本章默认P1。

| 案例 | 前置与输入 | 操作步骤 | 明确预期（并执行第2.3节共同断言） | 验收依据 · 主层/补层 |
| --- | --- | --- | --- | --- |
| <a id="m02-tl-248"></a>TL-248 状态矩阵与固定两条守卫 | 第2.5节12格及日期/首条边界；CREATE只走新建 | 主G完整矩阵；补A固定插入及删除全链拒绝、旧任意ADD CREATE请求拒绝 | 完整转换规则覆盖；固定两条不能绕过首条、同日或完整后继守卫；不再比较单项联合等价入口。 | [AC002](011-m02-business-requirements.md#m02-ac-002)、[AC015](011-m02-business-requirements.md#m02-ac-015)、[AC016](011-m02-business-requirements.md#m02-ac-016)；主G；补A |
| <a id="m02-tl-249"></a>TL-249 新建到历史修复完整旅程 | F0→1月A→4月S→5月R B→6月CHANGE C；更正R中文乙2；改R至05-15；删6月C | 每步从正常菜单保存，最后分别查所有来源 | 最终1—3月A、4月—05-14停用A、05-15起乙2/Beta；旧来源逐步保留B/C及原日期，各成功只一来源。 | [AC002](011-m02-business-requirements.md#m02-ac-002)、[AC003](011-m02-business-requirements.md#m02-ac-003)、[AC004](011-m02-business-requirements.md#m02-ac-004)、[AC005](011-m02-business-requirements.md#m02-ac-005)、[AC006](011-m02-business-requirements.md#m02-ac-006)、[AC007](011-m02-business-requirements.md#m02-ac-007)、[AC008](011-m02-business-requirements.md#m02-ac-008)、[AC014](011-m02-business-requirements.md#m02-ac-014)、[AC015](011-m02-business-requirements.md#m02-ac-015)；主B |
| <a id="m02-tl-250"></a>TL-250 单条拒绝后固定两条办理 | F2单加3月S失败；固定插入3月S/4月R成功；单删S失败；成对删成功 | 在每阶段重试、查回和读取T0—T2 | 两次失败不推进来源；两次成功各一来源，最终回连续启用但旧启停证据保留，不能重用旧提交改变内容。 | [AC005](011-m02-business-requirements.md#m02-ac-005)、[AC006](011-m02-business-requirements.md#m02-ac-006)、[AC010](011-m02-business-requirements.md#m02-ac-010)、[AC012](011-m02-business-requirements.md#m02-ac-012)、[AC016](011-m02-business-requirements.md#m02-ac-016)、[AC026](011-m02-business-requirements.md#m02-ac-026)；主B；条件F |
| <a id="m02-tl-251"></a>TL-251 首条更正到分类更正旅程（109退役） | F7先普通更正首条到1月A，不新增6月；后明确纠正首条原因、把7月S/8月R同时改CHANGE | 读每版起点、7月名称和10月后继 | 首条更正版7月停用A、8月C；分类版全程启用且7月独立A、8月C、10月Z；首条及后继身份保持。 | [AC017](011-m02-business-requirements.md#m02-ac-017)、[AC018](011-m02-business-requirements.md#m02-ac-018)、[AC019](011-m02-business-requirements.md#m02-ac-019)；主B；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-252"></a>TL-252 排列不变与同值性质（109退役） | 最多4个明确项的合法集合，含日期交换/删除同日新增/无变化项 | 穷举项排列，以独立BU副本同基准提交 | 每种排列的日期/资料/状态/结构结果一致，itemId映射不变；比较事实不比较不同副本新分配UUID；原身份重排只重放。 | [AC016](011-m02-business-requirements.md#m02-ac-016)、[AC020](011-m02-business-requirements.md#m02-ac-020)、[AC023](011-m02-business-requirements.md#m02-ac-023)、[AC027](011-m02-business-requirements.md#m02-ac-027)；主G；109退出通用编排，本例退役，不执行或计当前通过 |
| <a id="m02-tl-253"></a>TL-253 小状态空间非法后继 | 最多4条记录，用固定有限日期和4种类型独立枚举 | 用手写转换表筛选，再调用候选；保存发现的最小反例 | 所有接受集合满足唯一日期、固定首条及交替链；非法后继均拒绝，不能只验证新增点。领域枚举不代替DB/页面。 | [AC003](011-m02-business-requirements.md#m02-ac-003)、[AC015](011-m02-business-requirements.md#m02-ac-015)、[AC016](011-m02-business-requirements.md#m02-ac-016)、[AC018](011-m02-business-requirements.md#m02-ac-018)；主G |
| <a id="m02-tl-254"></a>TL-254 相同业务值不等于同结构 | F8删除/改日前序、首条原因纠错；旧交换来源只读 | 分别比较数据影响和完整结构 | 数据可以相同但结构变化必须W；真正全等才N，继承引用始终准确指向所选来源前序。 | [AC018](011-m02-business-requirements.md#m02-ac-018)、[AC021](011-m02-business-requirements.md#m02-ac-021)、[AC023](011-m02-business-requirements.md#m02-ac-023)；主G |
| <a id="m02-tl-255"></a>TL-255 既定规模与并发测量 | 2000BU×最多50来源×20成员；20并发；普通单条保存/无变化及1000次历史提交（109新工作负载未重跑） | 分别记录冷/热、p95、错误率、锁、WAL/存储和执行计划 | 对照列表/详情300ms、无竞争保存/确切来源500ms目标如实判定；达不到则列性能待办，不缩范围或写已通过。 | [AC001](011-m02-business-requirements.md#m02-ac-001)、[AC010](011-m02-business-requirements.md#m02-ac-010)、[AC012](011-m02-business-requirements.md#m02-ac-012)、[AC028](011-m02-business-requirements.md#m02-ac-028)；主P |
| <a id="m02-tl-256"></a>TL-256 长当前线与超预算压力 | 独立增加当前成员而非只增加历史版本；仅普通更正1条，任意全期间编排退役 | 测容量边界、最大取消延迟、清理/锁恢复、内存及输出；使用当前实际算法 | 按013初始10秒及取消响应/清理预算如实判定，超限安全失败/查回，不截断或分笔。不达标才优化实际瓶颈；不以自定义排序为验收条件，不发明业务条数上限。 | [AC016](011-m02-business-requirements.md#m02-ac-016)、[AC025](011-m02-business-requirements.md#m02-ac-025)、[AC028](011-m02-business-requirements.md#m02-ac-028)；主P；条件F |
| <a id="m02-tl-257"></a>TL-257 不在范围的办理明确拒绝 | 整BU作废、恢复已删record、跨BU联合、同日多变更、停用段普通CHANGE、隐式脚本/导入 | 检查菜单缺入口并直接构造请求 | 按协议/能力/业务守卫拒绝，无旁路物理删除、自动级联、同日序号或第二状态引擎；旧来源仍可授权读取。 | [AC005](011-m02-business-requirements.md#m02-ac-005)、[AC008](011-m02-business-requirements.md#m02-ac-008)、[AC013](011-m02-business-requirements.md#m02-ac-013)、[AC015](011-m02-business-requirements.md#m02-ac-015)、[AC016](011-m02-business-requirements.md#m02-ac-016)、[AC017](011-m02-business-requirements.md#m02-ac-017)；主A；补B |

<a id="tl-coverage"></a>
## 18 合同覆盖与实施追踪

### 18.1 场景及规则

每个案例的AC列给出验收依据；下表按011完整场景/规则反向定位案例组。范围链接包含区间内所有编号，不表示只运行端点。范围内退役编号仅保留历史定位，不纳入现行覆盖；下节AC反向映射只列现行案例。映射存在只证明可追踪，不能代替逐例预期或真实运行。

| 011场景 | 关联规则 | 案例入口 |
| --- | --- | --- |
| [SCN001](011-m02-business-requirements.md#m02-scn-001) | [BR009](011-m02-business-requirements.md#m02-br-009)、[BR010](011-m02-business-requirements.md#m02-br-010)、[BR015](011-m02-business-requirements.md#m02-br-015) | [按日查询、来源历史与只读定位](#tl-query)（[TL-017](#m02-tl-017)—[TL-032](#m02-tl-032)）、[联合预览、页面状态与返回保护](#tl-ui)（[TL-150](#m02-tl-150)—[TL-165](#m02-tl-165)） |
| [SCN002](011-m02-business-requirements.md#m02-scn-002) | [BR001](011-m02-business-requirements.md#m02-br-001)、[BR002](011-m02-business-requirements.md#m02-br-002)、[BR004](011-m02-business-requirements.md#m02-br-004)、[BR016](011-m02-business-requirements.md#m02-br-016) | [日期、身份与创建首条](#tl-create)（[TL-001](#m02-tl-001)—[TL-016](#m02-tl-016)） |
| [SCN003](011-m02-business-requirements.md#m02-scn-003) | [BR003](011-m02-business-requirements.md#m02-br-003)、[BR005](011-m02-business-requirements.md#m02-br-005)、[BR011](011-m02-business-requirements.md#m02-br-011)、[BR018](011-m02-business-requirements.md#m02-br-018) | [资料新增、追加与追溯插入](#tl-add)（[TL-033](#m02-tl-033)—[TL-045](#m02-tl-045)） |
| [SCN004](011-m02-business-requirements.md#m02-scn-004) | [BR006](011-m02-business-requirements.md#m02-br-006)、[BR009](011-m02-business-requirements.md#m02-br-009)、[BR016](011-m02-business-requirements.md#m02-br-016) | [普通更正、停用继承与后继快照](#tl-correct)（[TL-046](#m02-tl-046)—[TL-058](#m02-tl-058)） |
| [SCN005](011-m02-business-requirements.md#m02-scn-005) | [BR007](011-m02-business-requirements.md#m02-br-007)、[BR011](011-m02-business-requirements.md#m02-br-011)、[BR018](011-m02-business-requirements.md#m02-br-018) | [删除各类记录与完整依赖](#tl-delete)（[TL-075](#m02-tl-075)—[TL-089](#m02-tl-089)） |
| [SCN006](011-m02-business-requirements.md#m02-scn-006) | [BR008](011-m02-business-requirements.md#m02-br-008)、[BR011](011-m02-business-requirements.md#m02-br-011)、[BR018](011-m02-business-requirements.md#m02-br-018) | [新增停用、重新启用与多轮启停](#tl-lifecycle)（[TL-059](#m02-tl-059)—[TL-074](#m02-tl-074)） |
| [SCN007](011-m02-business-requirements.md#m02-scn-007) | [BR015](011-m02-business-requirements.md#m02-br-015) | [联合预览、页面状态与返回保护](#tl-ui)（[TL-150](#m02-tl-150)—[TL-165](#m02-tl-165)） |
| [SCN008](011-m02-business-requirements.md#m02-scn-008) | [BR009](011-m02-business-requirements.md#m02-br-009)、[BR010](011-m02-business-requirements.md#m02-br-010)、[BR014](011-m02-business-requirements.md#m02-br-014) | [按日查询、来源历史与只读定位](#tl-query)（[TL-017](#m02-tl-017)—[TL-032](#m02-tl-032)）、[存储约束、源事件与升级准入](#tl-storage-upgrade)（[TL-229](#m02-tl-229)—[TL-247](#m02-tl-247)） |
| [SCN009](011-m02-business-requirements.md#m02-scn-009) | [BR010](011-m02-business-requirements.md#m02-br-010) | [当前权限、真实引用与隔离](#tl-access-reference)（[TL-166](#m02-tl-166)—[TL-187](#m02-tl-187)） |
| [SCN010](011-m02-business-requirements.md#m02-scn-010) | [BR012](011-m02-business-requirements.md#m02-br-012)、[BR013](011-m02-business-requirements.md#m02-br-013) | [提交幂等、版本竞争与终结顺序](#tl-concurrency)（[TL-188](#m02-tl-188)—[TL-207](#m02-tl-207)） |
| [SCN011](011-m02-business-requirements.md#m02-scn-011) | [BR011](011-m02-business-requirements.md#m02-br-011) | [当前权限、真实引用与隔离](#tl-access-reference)（[TL-166](#m02-tl-166)—[TL-187](#m02-tl-187)） |
| [SCN012](011-m02-business-requirements.md#m02-scn-012) | [BR014](011-m02-business-requirements.md#m02-br-014) | [失败原子性、持久诊断与执行预算](#tl-failure)（[TL-208](#m02-tl-208)—[TL-228](#m02-tl-228)）、[存储约束、源事件与升级准入](#tl-storage-upgrade)（[TL-229](#m02-tl-229)—[TL-247](#m02-tl-247)） |
| [SCN013](011-m02-business-requirements.md#m02-scn-013) | [BR003](011-m02-business-requirements.md#m02-br-003)、[BR017](011-m02-business-requirements.md#m02-br-017)、[BR018](011-m02-business-requirements.md#m02-br-018) | [全类别改日与日期重排](#tl-redate)（[TL-090](#m02-tl-090)—[TL-107](#m02-tl-107)） |
| [SCN014](011-m02-business-requirements.md#m02-scn-014) | [BR018](011-m02-business-requirements.md#m02-br-018) | [新增停用、重新启用与多轮启停](#tl-lifecycle)（[TL-059](#m02-tl-059)—[TL-074](#m02-tl-074)） |
| [SCN015](011-m02-business-requirements.md#m02-scn-015) | [BR019](011-m02-business-requirements.md#m02-br-019) | [有限联合修订与最终候选](#tl-joint)（[TL-108](#m02-tl-108)—[TL-128](#m02-tl-128)） |
| [SCN016](011-m02-business-requirements.md#m02-scn-016) | [BR020](011-m02-business-requirements.md#m02-br-020) | [首条资料维护与分类录错处置](#tl-backfill-class)（[TL-129](#m02-tl-129)—[TL-149](#m02-tl-149)） |
| [SCN017](011-m02-business-requirements.md#m02-scn-017) | [BR016](011-m02-business-requirements.md#m02-br-016)、[BR021](011-m02-business-requirements.md#m02-br-021) | [首条资料维护与分类录错处置](#tl-backfill-class)（[TL-129](#m02-tl-129)—[TL-149](#m02-tl-149)） |

### 18.2 当前验收条件追踪

109按当前范围重新汇总；退役案例只保留原行，不计现行覆盖。下表为设计追踪，实际执行层次、输入和限制见各批验证。

| 011验收条件 | 现行案例 |
| --- | --- |
| [AC001](011-m02-business-requirements.md#m02-ac-001) | [TL-004](#m02-tl-004)、[TL-008](#m02-tl-008)、[TL-017](#m02-tl-017)、[TL-018](#m02-tl-018)、[TL-019](#m02-tl-019)、[TL-020](#m02-tl-020)、[TL-027](#m02-tl-027)、[TL-030](#m02-tl-030)、[TL-031](#m02-tl-031)、[TL-095](#m02-tl-095)、[TL-255](#m02-tl-255)、[TL-323](#m02-tl-323)、[TL-324](#m02-tl-324)、[TL-325](#m02-tl-325)、[TL-326](#m02-tl-326)、[TL-329](#m02-tl-329)、[TL-330](#m02-tl-330)、[TL-333](#m02-tl-333)、[TL-334](#m02-tl-334)、[TL-335](#m02-tl-335)、[TL-337](#m02-tl-337) |
| [AC002](011-m02-business-requirements.md#m02-ac-002) | [TL-001](#m02-tl-001)、[TL-002](#m02-tl-002)、[TL-003](#m02-tl-003)、[TL-004](#m02-tl-004)、[TL-005](#m02-tl-005)、[TL-006](#m02-tl-006)、[TL-007](#m02-tl-007)、[TL-009](#m02-tl-009)、[TL-010](#m02-tl-010)、[TL-011](#m02-tl-011)、[TL-012](#m02-tl-012)、[TL-013](#m02-tl-013)、[TL-014](#m02-tl-014)、[TL-015](#m02-tl-015)、[TL-016](#m02-tl-016)、[TL-070](#m02-tl-070)、[TL-075](#m02-tl-075)、[TL-194](#m02-tl-194)、[TL-228](#m02-tl-228)、[TL-248](#m02-tl-248)、[TL-249](#m02-tl-249) |
| [AC003](011-m02-business-requirements.md#m02-ac-003) | [TL-002](#m02-tl-002)、[TL-005](#m02-tl-005)、[TL-006](#m02-tl-006)、[TL-007](#m02-tl-007)、[TL-008](#m02-tl-008)、[TL-020](#m02-tl-020)、[TL-033](#m02-tl-033)、[TL-034](#m02-tl-034)、[TL-035](#m02-tl-035)、[TL-036](#m02-tl-036)、[TL-037](#m02-tl-037)、[TL-038](#m02-tl-038)、[TL-039](#m02-tl-039)、[TL-040](#m02-tl-040)、[TL-041](#m02-tl-041)、[TL-042](#m02-tl-042)、[TL-043](#m02-tl-043)、[TL-044](#m02-tl-044)、[TL-045](#m02-tl-045)、[TL-069](#m02-tl-069)、[TL-070](#m02-tl-070)、[TL-073](#m02-tl-073)、[TL-085](#m02-tl-085)、[TL-194](#m02-tl-194)、[TL-195](#m02-tl-195)、[TL-230](#m02-tl-230)、[TL-249](#m02-tl-249)、[TL-253](#m02-tl-253) |
| [AC004](011-m02-business-requirements.md#m02-ac-004) | [TL-011](#m02-tl-011)、[TL-012](#m02-tl-012)、[TL-022](#m02-tl-022)、[TL-046](#m02-tl-046)、[TL-047](#m02-tl-047)、[TL-048](#m02-tl-048)、[TL-049](#m02-tl-049)、[TL-050](#m02-tl-050)、[TL-052](#m02-tl-052)、[TL-053](#m02-tl-053)、[TL-054](#m02-tl-054)、[TL-055](#m02-tl-055)、[TL-056](#m02-tl-056)、[TL-057](#m02-tl-057)、[TL-058](#m02-tl-058)、[TL-249](#m02-tl-249) |
| [AC005](011-m02-business-requirements.md#m02-ac-005) | [TL-022](#m02-tl-022)、[TL-029](#m02-tl-029)、[TL-075](#m02-tl-075)、[TL-076](#m02-tl-076)、[TL-077](#m02-tl-077)、[TL-078](#m02-tl-078)、[TL-079](#m02-tl-079)、[TL-080](#m02-tl-080)、[TL-081](#m02-tl-081)、[TL-082](#m02-tl-082)、[TL-083](#m02-tl-083)、[TL-084](#m02-tl-084)、[TL-085](#m02-tl-085)、[TL-086](#m02-tl-086)、[TL-087](#m02-tl-087)、[TL-088](#m02-tl-088)、[TL-089](#m02-tl-089)、[TL-169](#m02-tl-169)、[TL-233](#m02-tl-233)、[TL-249](#m02-tl-249)、[TL-250](#m02-tl-250)、[TL-257](#m02-tl-257) |
| [AC006](011-m02-business-requirements.md#m02-ac-006) | [TL-036](#m02-tl-036)、[TL-038](#m02-tl-038)、[TL-049](#m02-tl-049)、[TL-055](#m02-tl-055)、[TL-059](#m02-tl-059)、[TL-060](#m02-tl-060)、[TL-063](#m02-tl-063)、[TL-064](#m02-tl-064)、[TL-065](#m02-tl-065)、[TL-067](#m02-tl-067)、[TL-069](#m02-tl-069)、[TL-070](#m02-tl-070)、[TL-071](#m02-tl-071)、[TL-073](#m02-tl-073)、[TL-078](#m02-tl-078)、[TL-079](#m02-tl-079)、[TL-080](#m02-tl-080)、[TL-182](#m02-tl-182)、[TL-249](#m02-tl-249)、[TL-250](#m02-tl-250) |
| [AC007](011-m02-business-requirements.md#m02-ac-007) | [TL-004](#m02-tl-004)、[TL-020](#m02-tl-020)、[TL-029](#m02-tl-029)、[TL-159](#m02-tl-159)、[TL-160](#m02-tl-160)、[TL-161](#m02-tl-161)、[TL-165](#m02-tl-165)、[TL-249](#m02-tl-249)、[TL-310](#m02-tl-310)、[TL-320](#m02-tl-320)、[TL-325](#m02-tl-325)、[TL-327](#m02-tl-327)、[TL-328](#m02-tl-328)、[TL-329](#m02-tl-329)、[TL-330](#m02-tl-330)、[TL-332](#m02-tl-332)、[TL-334](#m02-tl-334)、[TL-336](#m02-tl-336)、[TL-338](#m02-tl-338)、[TL-339](#m02-tl-339)、[TL-342](#m02-tl-342) |
| [AC008](011-m02-business-requirements.md#m02-ac-008) | [TL-017](#m02-tl-017)、[TL-018](#m02-tl-018)、[TL-019](#m02-tl-019)、[TL-021](#m02-tl-021)、[TL-022](#m02-tl-022)、[TL-023](#m02-tl-023)、[TL-024](#m02-tl-024)、[TL-025](#m02-tl-025)、[TL-026](#m02-tl-026)、[TL-027](#m02-tl-027)、[TL-028](#m02-tl-028)、[TL-029](#m02-tl-029)、[TL-031](#m02-tl-031)、[TL-032](#m02-tl-032)、[TL-044](#m02-tl-044)、[TL-057](#m02-tl-057)、[TL-074](#m02-tl-074)、[TL-076](#m02-tl-076)、[TL-077](#m02-tl-077)、[TL-085](#m02-tl-085)、[TL-106](#m02-tl-106)、[TL-107](#m02-tl-107)、[TL-180](#m02-tl-180)、[TL-196](#m02-tl-196)、[TL-229](#m02-tl-229)、[TL-231](#m02-tl-231)、[TL-232](#m02-tl-232)、[TL-233](#m02-tl-233)、[TL-238](#m02-tl-238)、[TL-249](#m02-tl-249)、[TL-257](#m02-tl-257)、[TL-311](#m02-tl-311)、[TL-322](#m02-tl-322)、[TL-334](#m02-tl-334) |
| [AC009](011-m02-business-requirements.md#m02-ac-009) | [TL-019](#m02-tl-019)、[TL-023](#m02-tl-023)、[TL-024](#m02-tl-024)、[TL-030](#m02-tl-030)、[TL-032](#m02-tl-032)、[TL-041](#m02-tl-041)、[TL-043](#m02-tl-043)、[TL-052](#m02-tl-052)、[TL-062](#m02-tl-062)、[TL-087](#m02-tl-087)、[TL-104](#m02-tl-104)、[TL-147](#m02-tl-147)、[TL-149](#m02-tl-149)、[TL-162](#m02-tl-162)、[TL-166](#m02-tl-166)、[TL-167](#m02-tl-167)、[TL-168](#m02-tl-168)、[TL-169](#m02-tl-169)、[TL-170](#m02-tl-170)、[TL-171](#m02-tl-171)、[TL-172](#m02-tl-172)、[TL-173](#m02-tl-173)、[TL-174](#m02-tl-174)、[TL-175](#m02-tl-175)、[TL-176](#m02-tl-176)、[TL-177](#m02-tl-177)、[TL-178](#m02-tl-178)、[TL-186](#m02-tl-186)、[TL-191](#m02-tl-191)、[TL-205](#m02-tl-205)、[TL-206](#m02-tl-206)、[TL-212](#m02-tl-212)、[TL-218](#m02-tl-218)、[TL-235](#m02-tl-235)、[TL-245](#m02-tl-245)、[TL-315](#m02-tl-315)、[TL-323](#m02-tl-323)、[TL-326](#m02-tl-326)、[TL-328](#m02-tl-328)、[TL-331](#m02-tl-331)、[TL-333](#m02-tl-333)、[TL-340](#m02-tl-340)、[TL-341](#m02-tl-341) |
| [AC010](011-m02-business-requirements.md#m02-ac-010) | [TL-015](#m02-tl-015)、[TL-027](#m02-tl-027)、[TL-032](#m02-tl-032)、[TL-041](#m02-tl-041)、[TL-057](#m02-tl-057)、[TL-086](#m02-tl-086)、[TL-087](#m02-tl-087)、[TL-104](#m02-tl-104)、[TL-107](#m02-tl-107)、[TL-161](#m02-tl-161)、[TL-162](#m02-tl-162)、[TL-175](#m02-tl-175)、[TL-177](#m02-tl-177)、[TL-187](#m02-tl-187)、[TL-188](#m02-tl-188)、[TL-189](#m02-tl-189)、[TL-190](#m02-tl-190)、[TL-191](#m02-tl-191)、[TL-192](#m02-tl-192)、[TL-193](#m02-tl-193)、[TL-194](#m02-tl-194)、[TL-195](#m02-tl-195)、[TL-196](#m02-tl-196)、[TL-197](#m02-tl-197)、[TL-198](#m02-tl-198)、[TL-199](#m02-tl-199)、[TL-200](#m02-tl-200)、[TL-205](#m02-tl-205)、[TL-206](#m02-tl-206)、[TL-207](#m02-tl-207)、[TL-213](#m02-tl-213)、[TL-228](#m02-tl-228)、[TL-234](#m02-tl-234)、[TL-235](#m02-tl-235)、[TL-250](#m02-tl-250)、[TL-255](#m02-tl-255)、[TL-317](#m02-tl-317)、[TL-318](#m02-tl-318)、[TL-321](#m02-tl-321)、[TL-331](#m02-tl-331) |
| [AC011](011-m02-business-requirements.md#m02-ac-011) | [TL-003](#m02-tl-003)、[TL-016](#m02-tl-016)、[TL-026](#m02-tl-026)、[TL-045](#m02-tl-045)、[TL-074](#m02-tl-074)、[TL-089](#m02-tl-089)、[TL-135](#m02-tl-135)、[TL-167](#m02-tl-167)、[TL-179](#m02-tl-179)、[TL-180](#m02-tl-180)、[TL-181](#m02-tl-181)、[TL-182](#m02-tl-182)、[TL-183](#m02-tl-183)、[TL-184](#m02-tl-184)、[TL-185](#m02-tl-185)、[TL-186](#m02-tl-186)、[TL-187](#m02-tl-187)、[TL-238](#m02-tl-238) |
| [AC012](011-m02-business-requirements.md#m02-ac-012) | [TL-015](#m02-tl-015)、[TL-028](#m02-tl-028)、[TL-177](#m02-tl-177)、[TL-178](#m02-tl-178)、[TL-198](#m02-tl-198)、[TL-205](#m02-tl-205)、[TL-208](#m02-tl-208)、[TL-209](#m02-tl-209)、[TL-211](#m02-tl-211)、[TL-212](#m02-tl-212)、[TL-213](#m02-tl-213)、[TL-229](#m02-tl-229)、[TL-230](#m02-tl-230)、[TL-231](#m02-tl-231)、[TL-232](#m02-tl-232)、[TL-233](#m02-tl-233)、[TL-234](#m02-tl-234)、[TL-235](#m02-tl-235)、[TL-237](#m02-tl-237)、[TL-238](#m02-tl-238)、[TL-250](#m02-tl-250)、[TL-255](#m02-tl-255)、[TL-311](#m02-tl-311)、[TL-321](#m02-tl-321) |
| [AC013](011-m02-business-requirements.md#m02-ac-013) | [TL-001](#m02-tl-001)、[TL-006](#m02-tl-006)、[TL-012](#m02-tl-012)、[TL-013](#m02-tl-013)、[TL-014](#m02-tl-014)、[TL-022](#m02-tl-022)、[TL-033](#m02-tl-033)、[TL-043](#m02-tl-043)、[TL-046](#m02-tl-046)、[TL-054](#m02-tl-054)、[TL-056](#m02-tl-056)、[TL-058](#m02-tl-058)、[TL-059](#m02-tl-059)、[TL-060](#m02-tl-060)、[TL-071](#m02-tl-071)、[TL-072](#m02-tl-072)、[TL-076](#m02-tl-076)、[TL-087](#m02-tl-087)、[TL-106](#m02-tl-106)、[TL-134](#m02-tl-134)、[TL-136](#m02-tl-136)、[TL-148](#m02-tl-148)、[TL-159](#m02-tl-159)、[TL-190](#m02-tl-190)、[TL-196](#m02-tl-196)、[TL-217](#m02-tl-217)、[TL-228](#m02-tl-228)、[TL-257](#m02-tl-257)、[TL-310](#m02-tl-310) |
| [AC014](011-m02-business-requirements.md#m02-ac-014) | [TL-007](#m02-tl-007)、[TL-018](#m02-tl-018)、[TL-056](#m02-tl-056)、[TL-075](#m02-tl-075)、[TL-090](#m02-tl-090)、[TL-091](#m02-tl-091)、[TL-092](#m02-tl-092)、[TL-093](#m02-tl-093)、[TL-094](#m02-tl-094)、[TL-095](#m02-tl-095)、[TL-096](#m02-tl-096)、[TL-099](#m02-tl-099)、[TL-100](#m02-tl-100)、[TL-101](#m02-tl-101)、[TL-102](#m02-tl-102)、[TL-103](#m02-tl-103)、[TL-104](#m02-tl-104)、[TL-105](#m02-tl-105)、[TL-106](#m02-tl-106)、[TL-107](#m02-tl-107)、[TL-133](#m02-tl-133)、[TL-160](#m02-tl-160)、[TL-168](#m02-tl-168)、[TL-183](#m02-tl-183)、[TL-192](#m02-tl-192)、[TL-196](#m02-tl-196)、[TL-230](#m02-tl-230)、[TL-239](#m02-tl-239)、[TL-249](#m02-tl-249)、[TL-319](#m02-tl-319) |
| [AC015](011-m02-business-requirements.md#m02-ac-015) | [TL-017](#m02-tl-017)、[TL-031](#m02-tl-031)、[TL-036](#m02-tl-036)、[TL-037](#m02-tl-037)、[TL-038](#m02-tl-038)、[TL-039](#m02-tl-039)、[TL-048](#m02-tl-048)、[TL-055](#m02-tl-055)、[TL-059](#m02-tl-059)、[TL-060](#m02-tl-060)、[TL-061](#m02-tl-061)、[TL-062](#m02-tl-062)、[TL-063](#m02-tl-063)、[TL-064](#m02-tl-064)、[TL-065](#m02-tl-065)、[TL-066](#m02-tl-066)、[TL-067](#m02-tl-067)、[TL-068](#m02-tl-068)、[TL-069](#m02-tl-069)、[TL-070](#m02-tl-070)、[TL-071](#m02-tl-071)、[TL-072](#m02-tl-072)、[TL-073](#m02-tl-073)、[TL-074](#m02-tl-074)、[TL-078](#m02-tl-078)、[TL-079](#m02-tl-079)、[TL-080](#m02-tl-080)、[TL-081](#m02-tl-081)、[TL-082](#m02-tl-082)、[TL-083](#m02-tl-083)、[TL-084](#m02-tl-084)、[TL-089](#m02-tl-089)、[TL-092](#m02-tl-092)、[TL-093](#m02-tl-093)、[TL-099](#m02-tl-099)、[TL-101](#m02-tl-101)、[TL-102](#m02-tl-102)、[TL-103](#m02-tl-103)、[TL-109](#m02-tl-109)、[TL-145](#m02-tl-145)、[TL-168](#m02-tl-168)、[TL-169](#m02-tl-169)、[TL-173](#m02-tl-173)、[TL-179](#m02-tl-179)、[TL-180](#m02-tl-180)、[TL-181](#m02-tl-181)、[TL-187](#m02-tl-187)、[TL-239](#m02-tl-239)、[TL-245](#m02-tl-245)、[TL-248](#m02-tl-248)、[TL-249](#m02-tl-249)、[TL-253](#m02-tl-253)、[TL-257](#m02-tl-257)、[TL-343](#m02-tl-343)、[TL-344](#m02-tl-344)、[TL-345](#m02-tl-345)、[TL-346](#m02-tl-346)、[TL-347](#m02-tl-347)、[TL-348](#m02-tl-348) |
| [AC016](011-m02-business-requirements.md#m02-ac-016) | [TL-040](#m02-tl-040)、[TL-064](#m02-tl-064)、[TL-070](#m02-tl-070)、[TL-074](#m02-tl-074)、[TL-080](#m02-tl-080)、[TL-084](#m02-tl-084)、[TL-108](#m02-tl-108)、[TL-109](#m02-tl-109)、[TL-159](#m02-tl-159)、[TL-160](#m02-tl-160)、[TL-165](#m02-tl-165)、[TL-167](#m02-tl-167)、[TL-171](#m02-tl-171)、[TL-188](#m02-tl-188)、[TL-189](#m02-tl-189)、[TL-190](#m02-tl-190)、[TL-192](#m02-tl-192)、[TL-198](#m02-tl-198)、[TL-209](#m02-tl-209)、[TL-228](#m02-tl-228)、[TL-229](#m02-tl-229)、[TL-234](#m02-tl-234)、[TL-236](#m02-tl-236)、[TL-237](#m02-tl-237)、[TL-239](#m02-tl-239)、[TL-244](#m02-tl-244)、[TL-245](#m02-tl-245)、[TL-248](#m02-tl-248)、[TL-250](#m02-tl-250)、[TL-253](#m02-tl-253)、[TL-256](#m02-tl-256)、[TL-257](#m02-tl-257)、[TL-307](#m02-tl-307)、[TL-308](#m02-tl-308)、[TL-309](#m02-tl-309)、[TL-310](#m02-tl-310)、[TL-311](#m02-tl-311)、[TL-312](#m02-tl-312)、[TL-313](#m02-tl-313)、[TL-314](#m02-tl-314)、[TL-315](#m02-tl-315)、[TL-316](#m02-tl-316)、[TL-317](#m02-tl-317)、[TL-318](#m02-tl-318)、[TL-320](#m02-tl-320) |
| [AC017](011-m02-business-requirements.md#m02-ac-017) | [TL-075](#m02-tl-075)、[TL-094](#m02-tl-094)、[TL-096](#m02-tl-096)、[TL-129](#m02-tl-129)、[TL-130](#m02-tl-130)、[TL-131](#m02-tl-131)、[TL-133](#m02-tl-133)、[TL-134](#m02-tl-134)、[TL-135](#m02-tl-135)、[TL-146](#m02-tl-146)、[TL-257](#m02-tl-257)、[TL-302](#m02-tl-302)、[TL-303](#m02-tl-303)、[TL-304](#m02-tl-304)、[TL-305](#m02-tl-305)、[TL-306](#m02-tl-306) |
| [AC018](011-m02-business-requirements.md#m02-ac-018) | [TL-056](#m02-tl-056)、[TL-136](#m02-tl-136)、[TL-137](#m02-tl-137)、[TL-138](#m02-tl-138)、[TL-145](#m02-tl-145)、[TL-146](#m02-tl-146)、[TL-147](#m02-tl-147)、[TL-148](#m02-tl-148)、[TL-149](#m02-tl-149)、[TL-170](#m02-tl-170)、[TL-175](#m02-tl-175)、[TL-253](#m02-tl-253)、[TL-254](#m02-tl-254) |
| [AC019](011-m02-business-requirements.md#m02-ac-019) | [TL-034](#m02-tl-034)、[TL-042](#m02-tl-042)、[TL-046](#m02-tl-046)、[TL-047](#m02-tl-047)、[TL-048](#m02-tl-048)、[TL-049](#m02-tl-049)、[TL-050](#m02-tl-050)、[TL-052](#m02-tl-052)、[TL-053](#m02-tl-053)、[TL-061](#m02-tl-061)、[TL-062](#m02-tl-062)、[TL-093](#m02-tl-093)、[TL-108](#m02-tl-108)、[TL-130](#m02-tl-130)、[TL-131](#m02-tl-131)、[TL-138](#m02-tl-138)、[TL-172](#m02-tl-172)、[TL-312](#m02-tl-312)、[TL-314](#m02-tl-314) |
| [AC020](011-m02-business-requirements.md#m02-ac-020) | 退役，不计现行验收；原规则/案例保留历史定位。 |
| [AC021](011-m02-business-requirements.md#m02-ac-021) | [TL-045](#m02-tl-045)、[TL-049](#m02-tl-049)、[TL-078](#m02-tl-078)、[TL-088](#m02-tl-088)、[TL-105](#m02-tl-105)、[TL-130](#m02-tl-130)、[TL-131](#m02-tl-131)、[TL-136](#m02-tl-136)、[TL-137](#m02-tl-137)、[TL-182](#m02-tl-182)、[TL-184](#m02-tl-184)、[TL-236](#m02-tl-236)、[TL-237](#m02-tl-237)、[TL-254](#m02-tl-254)、[TL-314](#m02-tl-314) |
| [AC022](011-m02-business-requirements.md#m02-ac-022) | 退役，不计现行验收；原规则/案例保留历史定位。 |
| [AC023](011-m02-business-requirements.md#m02-ac-023) | [TL-040](#m02-tl-040)、[TL-041](#m02-tl-041)、[TL-054](#m02-tl-054)、[TL-104](#m02-tl-104)、[TL-148](#m02-tl-148)、[TL-149](#m02-tl-149)、[TL-172](#m02-tl-172)、[TL-173](#m02-tl-173)、[TL-174](#m02-tl-174)、[TL-175](#m02-tl-175)、[TL-178](#m02-tl-178)、[TL-190](#m02-tl-190)、[TL-197](#m02-tl-197)、[TL-218](#m02-tl-218)、[TL-227](#m02-tl-227)、[TL-236](#m02-tl-236)、[TL-254](#m02-tl-254)、[TL-315](#m02-tl-315)、[TL-322](#m02-tl-322) |
| [AC024](011-m02-business-requirements.md#m02-ac-024) | [TL-008](#m02-tl-008)、[TL-028](#m02-tl-028)、[TL-035](#m02-tl-035)、[TL-044](#m02-tl-044)、[TL-090](#m02-tl-090)、[TL-195](#m02-tl-195)、[TL-196](#m02-tl-196)、[TL-226](#m02-tl-226)、[TL-227](#m02-tl-227) |
| [AC025](011-m02-business-requirements.md#m02-ac-025) | [TL-015](#m02-tl-015)、[TL-185](#m02-tl-185)、[TL-194](#m02-tl-194)、[TL-204](#m02-tl-204)、[TL-207](#m02-tl-207)、[TL-208](#m02-tl-208)、[TL-209](#m02-tl-209)、[TL-210](#m02-tl-210)、[TL-211](#m02-tl-211)、[TL-212](#m02-tl-212)、[TL-213](#m02-tl-213)、[TL-220](#m02-tl-220)、[TL-221](#m02-tl-221)、[TL-224](#m02-tl-224)、[TL-231](#m02-tl-231)、[TL-232](#m02-tl-232)、[TL-240](#m02-tl-240)、[TL-256](#m02-tl-256) |
| [AC026](011-m02-business-requirements.md#m02-ac-026) | [TL-161](#m02-tl-161)、[TL-162](#m02-tl-162)、[TL-199](#m02-tl-199)、[TL-200](#m02-tl-200)、[TL-201](#m02-tl-201)、[TL-202](#m02-tl-202)、[TL-203](#m02-tl-203)、[TL-204](#m02-tl-204)、[TL-211](#m02-tl-211)、[TL-225](#m02-tl-225)、[TL-235](#m02-tl-235)、[TL-250](#m02-tl-250)、[TL-321](#m02-tl-321) |
| [AC027](011-m02-business-requirements.md#m02-ac-027) | [TL-163](#m02-tl-163)、[TL-174](#m02-tl-174)、[TL-186](#m02-tl-186)、[TL-189](#m02-tl-189)、[TL-197](#m02-tl-197)、[TL-203](#m02-tl-203)、[TL-215](#m02-tl-215)、[TL-216](#m02-tl-216)、[TL-217](#m02-tl-217)、[TL-218](#m02-tl-218)、[TL-227](#m02-tl-227)、[TL-228](#m02-tl-228)、[TL-234](#m02-tl-234)、[TL-239](#m02-tl-239)、[TL-244](#m02-tl-244)、[TL-317](#m02-tl-317) |
| [AC028](011-m02-business-requirements.md#m02-ac-028) | [TL-164](#m02-tl-164)、[TL-185](#m02-tl-185)、[TL-200](#m02-tl-200)、[TL-204](#m02-tl-204)、[TL-207](#m02-tl-207)、[TL-210](#m02-tl-210)、[TL-213](#m02-tl-213)、[TL-219](#m02-tl-219)、[TL-220](#m02-tl-220)、[TL-221](#m02-tl-221)、[TL-222](#m02-tl-222)、[TL-223](#m02-tl-223)、[TL-224](#m02-tl-224)、[TL-225](#m02-tl-225)、[TL-255](#m02-tl-255)、[TL-256](#m02-tl-256) |
| [AC029](011-m02-business-requirements.md#m02-ac-029) | [TL-164](#m02-tl-164)、[TL-217](#m02-tl-217)、[TL-239](#m02-tl-239)、[TL-240](#m02-tl-240)、[TL-243](#m02-tl-243)、[TL-244](#m02-tl-244)、[TL-245](#m02-tl-245)、[TL-322](#m02-tl-322) |
| [AC030](011-m02-business-requirements.md#m02-ac-030) | [TL-258](#m02-tl-258)、[TL-259](#m02-tl-259)、[TL-260](#m02-tl-260)、[TL-261](#m02-tl-261)、[TL-262](#m02-tl-262)、[TL-263](#m02-tl-263)、[TL-264](#m02-tl-264)、[TL-265](#m02-tl-265)、[TL-266](#m02-tl-266)、[TL-267](#m02-tl-267)、[TL-268](#m02-tl-268)、[TL-269](#m02-tl-269)、[TL-270](#m02-tl-270)、[TL-271](#m02-tl-271)、[TL-272](#m02-tl-272)、[TL-273](#m02-tl-273)、[TL-274](#m02-tl-274)、[TL-275](#m02-tl-275)、[TL-276](#m02-tl-276)、[TL-277](#m02-tl-277)、[TL-278](#m02-tl-278)、[TL-279](#m02-tl-279)、[TL-280](#m02-tl-280)、[TL-281](#m02-tl-281) |
| [AC031](011-m02-business-requirements.md#m02-ac-031) | [TL-282](#m02-tl-282)、[TL-283](#m02-tl-283)、[TL-284](#m02-tl-284)、[TL-285](#m02-tl-285)、[TL-286](#m02-tl-286)、[TL-287](#m02-tl-287)、[TL-288](#m02-tl-288)、[TL-289](#m02-tl-289)、[TL-290](#m02-tl-290)、[TL-291](#m02-tl-291)、[TL-292](#m02-tl-292)、[TL-294](#m02-tl-294)、[TL-295](#m02-tl-295)、[TL-296](#m02-tl-296)、[TL-297](#m02-tl-297)、[TL-298](#m02-tl-298)、[TL-299](#m02-tl-299)、[TL-300](#m02-tl-300)、[TL-301](#m02-tl-301)、[TL-319](#m02-tl-319) |

### 18.3 技术反例追踪

按[013第8.2节](013-m02-technical-design.md)的现行33项逐项对照；编号是上游阅读定位，业务验收仍链接011稳定AC。上游调整顺序时维护本表，不改变TL标识。

| 013反例序号 | 核对主题 | 具体案例 |
| --- | --- | --- |
| 1 | 原子可见、提交后读取 | [TL-027](#m02-tl-027)、[TL-209](#m02-tl-209) |
| 2 | 时刻同值/回退与来源顺序 | [TL-028](#m02-tl-028) |
| 3（退役） | 旧00003退役升级，不纳入现行验收 | [TL-247](#m02-tl-247)，只保留历史定位 |
| 4 | 成功后事实再变仍回原结果 | [TL-107](#m02-tl-107)、[TL-196](#m02-tl-196) |
| 5 | 查回先封闭与迟到POST | [TL-199](#m02-tl-199)—[TL-200](#m02-tl-200) |
| 6 | 字段部分授权保持及不泄露 | [TL-052](#m02-tl-052)、[TL-172](#m02-tl-172) |
| 7 | 名称长度、Unicode和清空 | [TL-011](#m02-tl-011)—[TL-012](#m02-tl-012)、[TL-053](#m02-tl-053) |
| 8 | 退役knownAt及旧页面 | [TL-025](#m02-tl-025)—[TL-026](#m02-tl-026) |
| 9 | 旧来源封口与运行角色 | [TL-232](#m02-tl-232) |
| 10 | 复合引用、前驱及成员约束 | [TL-229](#m02-tl-229)—[TL-231](#m02-tl-231) |
| 11 | 四终态合法组合和来源唯一归属 | [TL-234](#m02-tl-234)—[TL-235](#m02-tl-235) |
| 12 | X03共享守卫、认证与CSRF竞争 | [TL-177](#m02-tl-177)、[TL-205](#m02-tl-205)—[TL-206](#m02-tl-206) |
| 13 | 旧来源未来首条不混当前 | [TL-021](#m02-tl-021) |
| 14 | 当前权限、原结果独立恢复与必要事实原子性（128修订，未运行） | [TL-178](#m02-tl-178)、[TL-212](#m02-tl-212) |
| 15 | 当前模型安装及日期引用 | [TL-230](#m02-tl-230)、[TL-233](#m02-tl-233)、[TL-239](#m02-tl-239) |
| 16 | 改日与各动作竞争及原结果 | [TL-107](#m02-tl-107)、[TL-192](#m02-tl-192)、[TL-209](#m02-tl-209) |
| 17 | 多轮启停、删除/改日依赖和快照 | [TL-073](#m02-tl-073)、[TL-083](#m02-tl-083)、[TL-103](#m02-tl-103) |
| 18 | 目标类别权限与同名恢复 | [TL-061](#m02-tl-061)、[TL-168](#m02-tl-168)—[TL-169](#m02-tl-169) |
| 19 | 引用类别、范围、故障及竞争 | [TL-181](#m02-tl-181)—[TL-182](#m02-tl-182)、[TL-185](#m02-tl-185)、[TL-187](#m02-tl-187) |
| 20 | 固定DTO和摘要 | [TL-307](#m02-tl-307)、[TL-308](#m02-tl-308)、[TL-316](#m02-tl-316)、[TL-317](#m02-tl-317) |
| 21 | 分阶段失败、联合结果与封口 | [TL-209](#m02-tl-209)、[TL-232](#m02-tl-232)、[TL-234](#m02-tl-234)、[TL-236](#m02-tl-236) |
| 22 | 普通组合更正及固定两条直接/派生 | [TL-282](#m02-tl-282)、[TL-283](#m02-tl-283)、[TL-311](#m02-tl-311)、[TL-314](#m02-tl-314) |
| 23 | 两方实际权限与当前读权 | [TL-315](#m02-tl-315)、[TL-322](#m02-tl-322) |
| 24 | 准确停用继承引用 | [TL-088](#m02-tl-088)、[TL-105](#m02-tl-105) |
| 25 | 当前影响与事件；任意混合退出 | [TL-237](#m02-tl-237)、[TL-311](#m02-tl-311)、[TL-314](#m02-tl-314) |
| 26 | 固定弹窗及直接保存；通用部分编辑退出 | [TL-310](#m02-tl-310)、[TL-316](#m02-tl-316)、[TL-320](#m02-tl-320) |
| 27 | 执行与结果读权分离 | [TL-174](#m02-tl-174)、[TL-315](#m02-tl-315)、[TL-322](#m02-tl-322) |
| 28 | 原办理日与跨午夜 | [TL-226](#m02-tl-226)—[TL-227](#m02-tl-227) |
| 29 | 409、技术错误与取消的失败边界 | [TL-208](#m02-tl-208)—[TL-210](#m02-tl-210) |
| 30 | 原锁内拒绝及整体回滚后的显式恢复 | [TL-201](#m02-tl-201)—[TL-204](#m02-tl-204) |
| 31 | 拒绝诊断及原依据 | [TL-163](#m02-tl-163)、[TL-217](#m02-tl-217)、[TL-317](#m02-tl-317) |
| 32 | 整体预算、实测取消延迟及锁释放 | [TL-219](#m02-tl-219)—[TL-225](#m02-tl-225)、[TL-256](#m02-tl-256) |
| 33 | 当前存储匹配及结果解码 | [TL-239](#m02-tl-239)—[TL-240](#m02-tl-240)、[TL-243](#m02-tl-243)—[TL-244](#m02-tl-244) |
### 18.4 批次、责任与关闭证据

| 010待办 | 必须先落实的案例范围 | 责任及关闭证据 |
| --- | --- | --- |
| T01 公共模型与失败基础 | G主层先验证统一有限项候选，单条共用；事务/恢复D与读写分权A、取消测量随T02/T03接线 | Organization及Access维护者；独立预期、原子写入、当前权限和预算；与T02/T03组成第一批，不先等待无存储的DB验收 |
| T02 当前存储与初始化 | 存储约束D；当前空库安装/初始化M；单一格式解码A；退役项不执行 | Repository maintainer；当前构建与真实DB、初始化失败及匹配检查，夹具不计联合业务；不设旧库升级或换密门槛 |
| T03 新增/更正 | 第5/6章的主要层与明确补层，相关API/历史/故障；菜单代表由TL-249及本章补B承接 | Organization及体验实现者、验证维护者；规则完整，事务/协议及正常菜单实际连接，证据按实际断言关联 |
| T04 启停 | 第7章三时/状态规则在G，API接线及相应D风险；TL-063/173补B及TL-249菜单旅程 | 同上；多轮启停、快照与依赖拒绝各有主层结果，不把日期矩阵跨层重跑 |
| T05 删除 | 第8章G规则及指定D/A；TL-160/249/250正常菜单、原件读回 | 同上；类别、首条保护、依赖及历史真实验证，完整日期参数只在主层执行 |
| T06 改日 | TL-090—093九格完整G，其他例按表；TL-160覆盖四类菜单代表、TL-095未来首条反馈 | 同上；Go每格结果可定位，DB/API验证各自持久化/协议风险，浏览器不重跑36格 |
| 原T08及109 | T08原结果保留；109仅两个固定启停操作，TL307—322及仍适用的共用保护 | 实际新验证见109，旧退役参数不计通过；多目标分类、日期交换及任意混合编排退出 |
| 既有T07综合验收 | 原253个现行TL主要层和指定补充断言闭合；4个退役编号排除，103新增24例不计该旧批次通过 | 业务验收角色/验证维护者；分别记录主层参数和补充结果，允许可核验同基线证据复用；未接消费者不计已通过 |

110/111增量由111承接[第21节](#m02-list-pro-cases)的TL-323—334；本轮只编写12例并关联既有AC，全部未执行，旧T07/019/100结果不提升。

103增量由105承接TL-258—281的产品接线及分层验证；具体参数与限制见105报告，不把旧T07结果提升为全301例通过。

## 19 执行记录、维护与剩余事项

以下原编写说明和各日期执行记录保留当时范围；103当前增量及24项实际参数的执行状态以105报告、本稿开头、第5.1节及末条进度为准，不将旧“未实施”或旧T07结论误用于新规则。

本文持有测试设计，实际执行按[验证模板](../../templates/verification.md)新建报告，引用TL及参数后缀，记录主层完整参数与明确补充断言。共同预期和第2.4节共同界定执行范围；规定层未运行则不关闭，未规定层不列为缺失也不宣称通过。旧报告保留原基线和当时层次，不能把本次缩减回填为旧案例已完成。

单例记录至少包含：`TL/参数 → 基线与夹具 → 本层输入/操作 → 独立预期 → 实际结果 → 主层/补充断言 → 证据/缺口`。角色、菜单、时钟、UUID等按实际风险补齐，可引用共享夹具。矩阵保留逐参数结果，重用证据注明同基线、同输入及已观测断言；新增反例追加编号，不改号掩盖失败。

本次没有改变业务范围、Go实现、SQL、API或页面，没有启动这些业务/故障/容量测试。当前创建/查询能力仍按原报告管理，记录维护按010实施；人员与实际测试环境由对应角色安排。真实消费者未接入的引用样本只计合同测试；跨BU、同日多记录、误建对象处置、恢复已删除记录、通用投递与生产能力继续后置。

编写范围、追踪检查和原证据保护见[023编写验证](../../verification/023-timeline-test-cases.md)；文档检查通过不构成上述案例通过。

2026-11-01：按024修订读写分权、原锁内拒绝/显式查回、一次换密升级及取消测量案例，并以主要/补充层收敛重复执行；257个编号及业务范围保留。修订前版本由e58f8df及023原报告追溯，[本次检查](../../verification/024-timeline-simplification.md)不构成业务测试通过。

2026-11-01：按025退役TL-241/242/246/247四项纯旧版本专项，保留编号、原意和旧报告；其余混合案例移除兼容参数并保留当前业务断言，AC反向映射排除退役项。当前253例均未运行，不用退役掩盖未验范围。

第一批执行入口：[资料维护报告](../../verification/010-first-record-maintenance.md)及其证据清单记录TL、参数、实际层次与未覆盖部分；代码Go/DB/API/浏览器分别承担对应断言，不将253例全部记通过。该第一批不含T04；当前T04结果另见启停及本轮整合报告。T05删除及T06改日已分别接线验证；T06旧日结果/重读撤权补验与独立结论见[独立评审](../../verification/020-redate-independent-review.md)，TL-107及TL-218相关补充参数以该记录为最新证据。T01/T02分类及联合接线已随T08完成；当前执行映射见[T08报告](../../verification/022-joint-implementation.md)及其TL参数清单，T01/T02跨动作全矩阵和T07综合容量仍待后续，四个退役编号保持不执行。

T04实施追踪见[停用/重新启用实施验证](../../verification/021-lifecycle-implementation.md)。TL-059—073按各主层及实际参数登记；TL-064仅单条拒绝，联合成功留T08；TL-074只验证本域旧来源及事件保持，未接入消费者不计采用验证。名称沿用当前准入，T04不将删除/改日/联合参数记为通过。

T05实际覆盖与未完成参数见[分类删除验证](../../verification/020-delete-implementation.md)：TL-075—083及TL-085—088承接单条删除，TL-084联合成对删除留T08；TL-089只计本域历史不阻断及准确引用，真实消费者未接入不计通过。日期/规则主要在Go，DB/API和浏览器补各自风险；新增普通/启停三类删除的回滚/查回、取消和跨午夜参数，不将本批扩为全253例验收。


2026-11-02 T08实施：第10—12章及相关权限、失败、历史与恢复按实际主/补层记录于[T08报告](../../verification/022-joint-implementation.md)。TL-250/251连续正常菜单旅程、TL-143/144原停用默认和两种首条补全已实跑；TL-252/253独立手写排列与有限转换表已实跑。TL-255未运行，TL-256只做代表长线取消和20成员累计SQL超时，不记全容量通过；真实跨域引用未接入参数明确保留。原测试设计和旧定稿报告保持。

2026-11-02 T07校正：TL-003保持缺企业、歧义和有限终点全部参数及原阻断预期，按011第2.1节、013第7.2节及025当前唯一开放企业模型，将主要验证层从API调整为初始化/准入M。未初始化Open和异常企业DDL在业务API开放前拒绝；不伪造该层为HTTP执行，也不禁用约束重建不可达运行态。其他253现行/4退役编号及业务预期保持，本轮实际结果见[T07报告](../../verification/010-t07-acceptance.md)。

2026-11-02 T07实际执行：253现行案例逐例主/补层、参数及执行绑定见[验收案例表](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/010-t07-acceptance/cases.json)和[T07报告](../../verification/010-t07-acceptance.md)：241整例通过，TL045/089/128仅本域参数通过，TL135及180—187真实消费者未接，不计整例通过；4退役编号保持不执行。TL008的G是Go应用层独立四时刻预期，经真实API/DB观测，不称纯领域无DB测试；TL132未选保留选项的禁存断言发生于填写早期资料之前；TL164受控503信封只验证页面，真实执行预算另由D覆盖。

2026-11-02：按103新增24项TL-258—281及AC030映射，合计281编号/277现行/4退役；更新TL-020及旧普通ADD共同锚点输入，保持所有旧ID和历史报告。本次只编写文档与独立预期，新24项及受影响旧参数的产品实施/业务复验待开展。

2026-11-02按104追加TL-282—293十二项统一更正案例，合计293编号/289现行/4退役；103的TL-258—281仍未实施。旧T06跨节点参数现由显式联合承接；普通日期与分类菜单归入更正记录，历史报告不改写。

2026-11-02按105追加8项统一更正补例，合计301编号/297现行/4退役；103位置新增及当前补例的实际分层证据见105，历史报告不改写。

2026-11-02：按107取消首条专用引导，迁移129—131/251普通办理，132原保留选择退役；新增302—306，合计306编号/301现行/5退役。其他联合参数保持，实际执行范围见107报告。

2026-11-02：108多轮复核同步TL-164辅助故障仍可直接保存、TL-038/067/068当前错误码，澄清TL-097/098显式联合路径；补验TL-272两种撤读权、TL-277两种并发改日及TL-164新增/停用/重新启用/删除辅助503共8个浏览器参数，不新增或重用案例ID。执行证据见[108报告](../../verification/108-maintenance-consistency-review.md)。

108最终范围核对另按AC009补两项启停新日期输入/历史日期读取分权的B层断言：有启停及日期写权但无历史日期读权时仍可明确输入新日期；普通ADD仍须当前位置上下文。沿用既有Go授权，不新增业务权限或TL编号。

## 20 固定启停期间专用维护

主要范围AC016/AC009—012/AC019/AC021/AC023—028；来源协议及独立原子断言保持。

| 案例 | 输入与操作 | 独立预期 | 主要层 |
| --- | --- | --- | --- |
| <a id="m02-tl-307"></a>TL-307 位置与状态 | 选择当前首条/中段/末条，含未来首条及停用段 | Go按当前来源所选记录确定空档，只有其后启用可插入，不按今日状态选位置 | G/D/A/B按实际参数分别登记，不自动全层通过 |
| <a id="m02-tl-308"></a>TL-308 两日最小空档 | L=06-01，R=06-04/06-03/06-02；另例无R | 前例仅06-02停用、06-03恢复；不足两合法日只禁用入口，无额外原因或全灰日历 | G/D/A/B按实际参数分别登记，不自动全层通过 |
| <a id="m02-tl-309"></a>TL-309 历法与日期上界 | 闰年02-27至03-01，年末及9999-12-29/30 | 按日历取最早两日，9999-12-29可30/31，12-30不足两日禁用；非法日拒绝 | G/D/A/B按实际参数分别登记，不自动全层通过 |
| <a id="m02-tl-310"></a>TL-310 固定四字段及初始值 | 打开专用插入弹窗，随后改停用日/已编辑恢复日 | 仅两日期两原因；类型固定；停用多项待选，重新启用原因按Go唯一默认；默认最早两日，初始取消无修改保护，人工恢复日不被覆盖 | G/D/A/B按实际参数分别登记，不自动全层通过 |
| <a id="m02-tl-311"></a>TL-311 一次插入与来源 | F2所选f，03-01停用/04-01恢复，名称都继承A | 一个请求、两正式记录、一来源/事件/结果；后继B/C不变，任何一条失败无部分事实，旧来源不变 | G/D/A/B按实际参数分别登记，不自动全层通过 |
| <a id="m02-tl-312"></a>TL-312 恢复独立快照 | 311成功后更正前序名称，再查S/R及旧来源 | S按前序继承新名称，R保持当次保存的独立资料；名称相同的S/R不能按普通CHANGE同值消除 | G/D/A/B按实际参数分别登记，不自动全层通过 |
| <a id="m02-tl-313"></a>TL-313 当前配对与资格 | 从S或R打开删除，分别由两次单条创建；多组及末尾孤立S | 按当前相邻S/R配对，与原同次创建无关，无完整组不提供入口；不跨段配对、不能伪造目标/修订 | G/D/A/B按实际参数分别登记，不自动全层通过 |
| <a id="m02-tl-314"></a>TL-314 删除资料及派生影响 | 1月A→3月S→4月R B→7月S→8月R C，删除3/4月组 | 一次删除两条；3月恢复连续启用，4—7月含后续停用资料B→A，8月C保持；不保留/转型/补建 | G/D/A/B按实际参数分别登记，不自动全层通过 |
| <a id="m02-tl-315"></a>TL-315 字段与双方权限 | 分别缺停用/恢复权或一方日期原因写权；删除缺一类权，另例不可读英文/日期 | Go拒绝缺权；删除不要求名称写权，值/区间和诊断按读权裁剪，撤权清缓存与受限输入 | G/D/A/B按实际参数分别登记，不自动全层通过 |
| <a id="m02-tl-316"></a>TL-316 公开通用请求退出 | 调用旧timeline-revisions、通用preview、公开Maintain旧动作；固定请求夹带items/changes/type/classification | 通用通道拒绝，固定DTO严格拒绝额外字段；无通用表单、项目按钮或添加关联入口 | G/D/A/B按实际参数分别登记，不自动全层通过 |
| <a id="m02-tl-317"></a>TL-317 非法日期及原因 | 同日、逆序、占位、跨空档、早于L、缺任一原因；固定请求原样重试及resolve | 整笔拒绝且无业务残留，拒绝结果可按原身份读取；不是首次报错正确而后续结果不可读 | G/D/A/B按实际参数分别登记，不自动全层通过 |
| <a id="m02-tl-318"></a>TL-318 并发与不自动换组 | 他人改来源/日期/删除一方后原请求提交，显式重读 | 过期依据整笔拒绝，保留可读输入，固定原锚点/原组不自动重配；重新核对后用户另次提交 | G/D/A/B按实际参数分别登记，不自动全层通过 |
| <a id="m02-tl-319"></a>TL-319 非法单条不自动补偿 | 普通分类更正造成孤立恢复或跨节点改日 | 正常拒绝；不提供通用关联入口，不自动修改、转换或补建其他记录；旧边缘能力退出 | G/D/A/B按实际参数分别登记，不自动全层通过 |
| <a id="m02-tl-320"></a>TL-320 取消焦点与视口 | 两弹窗默认取消、真实编辑后继续/放弃，桌面/窄屏/短屏/真实缩放 | 初始值不脏，实际输入受保护；四字段、原因、保存/取消可达，关闭焦点回触发处，返回上下文保持 | G/D/A/B按实际参数分别登记，不自动全层通过 |
| <a id="m02-tl-321"></a>TL-321 未知及精确结果恢复 | 真实保存丢回复、失败回滚、重试与resolve竞争；随后撤写/读权 | 同一原身份恢复完整两条或明确未保存，无重复/半条；读取不重验原写权，受限字段不泄露，未知不伪报失败 | G/D/A/B按实际参数分别登记，不自动全层通过 |
| <a id="m02-tl-322"></a>TL-322 旧通用历史保持 | 有旧混合/分类/日期交换来源与终态的合成历史，当前读源/原件/日志/resolve | 旧来源、记录ID和原结果保持可读；只关闭旧编排执行，不删或改历史，不把旧通过扩成新操作验收 | G/D/A/B按实际参数分别登记，不自动全层通过 |

2026-11-02：109追加16项，通用编排39项退役，原5退役保持；322编号、278现行、44退役。执行参数及多轮发现见109报告，不以旧146项或文本设计当作新通过。

<a id="m02-list-pro-cases"></a>
## 21 Community列表目标与交互回归

上游为[113 Community决定](../../adr/113-bu-list-community-adoption.md)、[110剩余交互](../../adr/110-bu-list-and-datagrid.md)、[012列表目标](012-m02-functional-and-interaction-design.md#m02-ui-list-pro)及[013接线](013-m02-technical-design.md#m02-tech-list-pro)，执行归[111](../../archive/plans/111-bu-list-experience-upgrade.md)。TL-323—334共12例沿用稳定ID，本次更新Community目标，不新增编号或通过数；当前Community T01/T02代码及定向组件/合成HTTP检查见[本轮报告](../../verification/111-community-t01-t02.md)；旧Pro T02报告仅说明原基线，当前12例真实业务参数及七组条件另见[111 T03](../../verification/111-community-t03.md)，不跨批累加旧通过数。旧T07/019/100/109的通过不能迁入，复用既有BR/SCN/AC。历史带pro锚点保留追踪身份，不代表组件选择。

夹具使用任务自有合成BU，至少73个可分页对象，包含未生效首条、长名称/编码、三种状态及读字段受限身份；验证者独立给出排序预期，编码/名称顺序按013而非拼音。页面从正常菜单进入。安装/类型/构建及依赖图另按111 T01验收；实际页面须无Pro许可水印或许可console提示，不过滤或遮挡。主B负责界面及真实请求/响应，权限/一致计数的实际API断言复用同基线已有检查；只在本批新增业务查询时补A，不为第三方通用算法重建单元测试。以下范围和期望是合同，不是执行通过记录。

| 案例 | 前置与操作 | 独立预期及保护 | 关联与层次 |
| --- | --- | --- | --- |
| <a id="m02-tl-323"></a>TL-323 紧凑控制区 | 管理员/查询者正常菜单打开；新建/标题/更多操作，打开状态及列筛选入口 | 按部门样板三段布局及白色表格面板，常驻筛选行退出，#066b68保持，搜索框+查询保持；默认无状态摘要，非默认显示A摘要；无创建权不可办，列表更多操作不混入对象写动作 | [AC001](011-m02-business-requirements.md#m02-ac-001)、[AC009](011-m02-business-requirements.md#m02-ac-009)；主B |
| <a id="m02-tl-324"></a>TL-324 表头及窄屏摘要排序 | 多页数据；桌面名称升/降/清除、编码升序；窄屏同一排序入口 | 请求与URL分别映射name/name_desc/code；清除回code；独立预期验证跨页顺序与稳定次序，非当前页局部重排；状态/日期/编码降序及多列排序不可操作 | [AC001](011-m02-business-requirements.md#m02-ac-001)；主B |
| <a id="m02-tl-325"></a>TL-325 原生紧凑分页 | 0/1/20/21/73条结果，20/50/100大小；键盘上/下一页、改大小，含加载和失败 | 0–0 / 0、1–20 / 73、21–40 / 73等范围由实际响应给出；改大小回1，边界不可翻；加载/失败不显示旧count为最新，触发控件/反馈焦点稳定 | [AC001](011-m02-business-requirements.md#m02-ac-001)、[AC007](011-m02-business-requirements.md#m02-ac-007)；主B |
| <a id="m02-tl-326"></a>TL-326 窄屏摘要列 | 390px及600—899px，含长名称/编码、未生效首条及受限字段；来回切换桌面 | Community摘要列名称主行+元数据次行，同一A/D/分页/选中；无重复表头/标签，长文可读，字段/tooltip/可访问文字不泄露，未来首条明确未生效 | [AC001](011-m02-business-requirements.md#m02-ac-001)、[AC009](011-m02-business-requirements.md#m02-ac-009)；主B |
| <a id="m02-tl-327"></a>TL-327 仅名称真实链接 | 标准及窄屏摘要分别Enter、Ctrl/Cmd、中键、新标签、复制地址；点击普通单元格与行 | 链接有受控href且仅携带A；修饰键遵循浏览器，原标签D/滚动保持；其他单元格/行只聚焦或选中，不导航或编辑 | [AC007](011-m02-business-requirements.md#m02-ac-007)；主B |
| <a id="m02-tl-328"></a>TL-328 虚拟行返回与对象导航 | 第二页滚动至视口外对象后打开；链接返回/浏览器后退；详情跨页前后对象；另目标被移出筛选或撤权 | 重查当前授权范围后恢复页/定位/可恢复滚动，虚拟行出现后名称链接获焦；用户已操作时不抢焦点；目标未在本页直接显示当前结果并聚焦列表标题，不追加提示或详情链接，不推断删除或失权；真实读取拒绝保持错误反馈，直达不伪造总数 | [AC007](011-m02-business-requirements.md#m02-ac-007)、[AC009](011-m02-business-requirements.md#m02-ac-009)；主B |
| <a id="m02-tl-329"></a>TL-329 搜索草稿事件 | BU及部门详情两种返回后，输入新D或清空；翻页、改大小、失败重试、排序、原条件及新条件查询/回车、应用筛选，刷新/前进后退 | 分页/改大小/重试仍用A且保D；排序及查询应用动作时D并回1；主动查询/翻页/改大小/排序/应用筛选清旧定位及其提示，未应用草稿不清；跨页勾选独立保持；迟到响应不回写输入；刷新/历史按URL重建，不持久保存D | [AC001](011-m02-business-requirements.md#m02-ac-001)、[AC007](011-m02-business-requirements.md#m02-ac-007)；主B |
| <a id="m02-tl-330"></a>TL-330 状态面板草稿 | 预先有D；打开状态面板编辑/取消/Escape，再应用全部/启用/停用/未生效 | 打开编辑取消不请求/不改A或D；应用才合并D并回1；摘要只反映A，关闭回可用触发点；无查看日期入口 | [AC001](011-m02-business-requirements.md#m02-ac-001)、[AC007](011-m02-business-requirements.md#m02-ac-007)；主B |
| <a id="m02-tl-331"></a>TL-331 失败与授权交错 | 当前查询失败、撤读权、会话切换，延迟旧列表/筛选/分页响应 | 普通错误保合法输入及重试，权限拒绝清正文/计数/受限草稿；旧请求/会话不覆盖新A/D及选中，隐藏列不成为授权手段 | [AC009](011-m02-business-requirements.md#m02-ac-009)、[AC010](011-m02-business-requirements.md#m02-ac-010)；主B；条件F |
| <a id="m02-tl-332"></a>TL-332 实际滚动与返回 | 按 012 第 6.2 节适用条件从正常菜单搜索、分页、由名称进入详情并返回；真实滚轮/键盘，切换900px边界 | 控制/摘要/数据可达，无横向裁切或零高正文；实际缩放及innerWidth留证，不用CSS zoom冒充；焦点可见，短屏可进入文档流，搜索输入不丢 | [AC007](011-m02-business-requirements.md#m02-ac-007)；主B |
| <a id="m02-tl-333"></a>TL-333 未承接功能与默认行为 | 检查Community列菜单、显隐/列宽及全部隐藏后的恢复、headerFilters、行编辑/行导航、多排序及导出入口；操作后查请求 | 原生列设置真实可用，全部隐藏可恢复，A/D变化保持显隐和宽度；列拖动/固定不再纳入，不建自研入口，断点切换不隐藏摘要；纯展示操作只影响已授权数据的本地布局，不触发新查询/写入；查询仍仅现行白名单；单列增量由第22节另验，未承接常驻筛选行、编辑/导出等关闭，无客户端替代；Community选择不扩大产品功能范围 | [AC001](011-m02-business-requirements.md#m02-ac-001)、[AC009](011-m02-business-requirements.md#m02-ac-009)；主B |
| <a id="m02-tl-334"></a>TL-334 查询只读及维护上下文 | 对现有BU状态/列日期条件/排序/分页查询，随后新建/维护/历史及提交查回走受影响旧代表例 | 列表操作不调用业务写入口，不增身份/修订/来源；详情继续当前Go办理能力和原结果定位，历史只读；复用同基线对应代表例并留证，不把本例解释成所有维护全量复验 | [AC001](011-m02-business-requirements.md#m02-ac-001)、[AC007](011-m02-business-requirements.md#m02-ac-007)、[AC008](011-m02-business-requirements.md#m02-ac-008)；主B |

2026-11-03：只完成新增12例设计与关联，334编号/290现行/44退役；执行由111承接。本轮多轮文档检查不计上述任一业务案例通过，列条件评估样例也不计本表基础列表运行验收。

2026-11-03 T03执行补充：TL-323—334在真实Go/API/DB及Linux Chromium下按本节参数验证，含首次浏览器后退、动态行高、真实字段撤权、网络故障、原生复制地址与200%/400%缩放；实际12例映射、10组列表测试及4组维护代表回归见[111 T03报告](../../verification/111-community-t03.md)。334编号/290现行/44退役不变，不据此宣称其余现行案例全部通过。

2026-11-03 T04评估补充：比较及7项独立输入组件实验见[评估记录](../../verification/111-column-filter-evaluation.md)，本批仍关闭列过滤、headerFilters及高级查询。实验不新增TL编号，不计TL-323—334或新增列条件业务通过；未来如采纳新查询须另补案例和真实API/DB验收。

## 22 单列条件与统一应用/取消增量

用户采纳T04优先路线，由110第2.6节、011 BR009、012第3.1.5节及013第4.1.2节持有合同。当前342编号/298现行/44退役；旧评估和T03结果保持。实际执行见[本次报告](../../verification/111-single-column-filter.md)，未运行不计通过。

| 案例 | 前置与操作 | 独立预期 | 验收与层次 |
| --- | --- | --- | --- |
| <a id="m02-tl-335"></a>TL-335 桌面单列及明确应用 | 桌面列菜单打开；编码等于第4页对象、名称同值不命中，再名称等于真实名称；输入后立即应用 | 独立列谓词先筛全量，当前页外对象能命中；条件草稿无请求，应用一次，主网格图标只反映A | [AC001](011-m02-business-requirements.md#m02-ac-001)；B＋D |
| <a id="m02-tl-336"></a>TL-336 取消及清除草稿 | 保留搜索D；分别取消/Escape/外部关闭；清除后取消，再清除后应用 | A/URL/结果/图标不变且D保留；只有应用清除才移除三个参数，合并动作时D并回第一页；焦点恢复 | [AC007](011-m02-business-requirements.md#m02-ac-007)；B＋D |
| <a id="m02-tl-337"></a>TL-337 状态集合及展示记录日期 | 单条件状态active/inactive；换为日期等于及晚于停用日；Go补五种比较边界与未来首条 | 同列OR排除未生效；更换列只保留一个条件；日期比较展示记录且未来首条标明未生效；不是任意历史记录或businessAt替代 | [AC001](011-m02-business-requirements.md#m02-ac-001)；B＋D |
| <a id="m02-tl-338"></a>TL-338 A/D及分页历史 | 列条件包含获权73对象；新D翻页/改大小，排序合并D，刷新/前进后退 | 分页/大小用A保留D，动作时应用D回第一页；筛选总数/稳定排序/clamp一致，刷新及历史按URL重建 | [AC007](011-m02-business-requirements.md#m02-ac-007)；B＋D |
| <a id="m02-tl-339"></a>TL-339 详情跨页与返回 | 名称包含21对象；第20对象进入详情，下一个跨页，最后一个无下一项；列表链接返回及浏览器后退 | 详情对象请求及返回携带完整三参数，只在相同筛选内导航，选中/链接焦点和页码恢复；链接不含D | [AC007](011-m02-business-requirements.md#m02-ac-007)；B＋D |
| <a id="m02-tl-340"></a>TL-340 合同拒绝及授权 | API缺项/重复/错配/超长/非法枚举和纯日期；查询者去名称/日期权，缩对象范围，再撤读权；无效直达 | 400不静默删条件后宽查；不可读字段不参与命中或数量；无读权403且页面清正文/草稿；无nameEn入口 | [AC009](011-m02-business-requirements.md#m02-ac-009)；B＋D |
| <a id="m02-tl-341"></a>TL-341 失败重试及迟到结果 | 列条件查询受控503；较新D重试；两个真实请求先后，旧响应后到 | 失败不留旧正文/数量，重试用A保留D；旧响应不覆盖新条件/结果/输入，不本地过滤冒充成功 | [AC009](011-m02-business-requirements.md#m02-ac-009)；B＋D |
| <a id="m02-tl-342"></a>TL-342 响应式及键盘收尾 | 桌面、短屏及条件面板的原生 200% 键盘流程，按 012 触发 400% 专项。Tab/Enter/Escape；900px断点切换 | 列/操作符/值及动作可达，无横溢，焦点不出模态；断点关闭草稿回更多操作；任务自有浏览器/profile停止，数据默认保留 | [AC007](011-m02-business-requirements.md#m02-ac-007)；B＋D |

2026-11-03，111 T05限定交付已复核，见[交付记录](../../verification/111-community-t05-delivery.md)。单列增量运行输入未变，复用原真实API/DB及浏览器分层证据；补测日常4382/8382管理员与查询者名称包含丙及取消。旧报告保持原基线和结果，不将计划完成提升为全M02、全342例或后置Excel/跨列能力通过。

## 23 单条启停末尾追加补验

依据BR008、BR018和AC015；旧用例中允许任意所选位置的单条新增启停按本节收敛，原执行证据不回写。G/D/A/B分别按实际执行参数取证。

| 稳定案例 | 前置与操作 | 预期结果 | 证据边界 |
| --- | --- | --- | --- |
| <a id="m02-tl-343"></a>TL-343 所选末条可办性 | 选择首条/中间/末条，末条分别启用/停用；查询者与旧来源 | 仅选中当前末条且有对应权时开放该状态适合的一项；非末条不转去别处追加，成对操作保持 | 分层记录 |
| <a id="m02-tl-344"></a>TL-344 初始值及取消 | 过去/今日/未来末条打开单条表单，不修改即取消，修改后继续/放弃 | 默认max(D,末条次日)，恢复名称带出、恢复原因按Go唯一默认，停用多项待选；自动默认不脏，真实编辑受保护 | 分层记录 |
| <a id="m02-tl-345"></a>TL-345 日期范围与历法 | 末条同日/之前/次日，闰日、月末、年末、9999上界及缺锚点 | Go返回上下界且保存/辅助重验；无次日无入口，不能用旧所选锚点在末尾新增 | 分层记录 |
| <a id="m02-tl-346"></a>TL-346 未来末条状态 | 今日启用但未来末条停用；今日未生效但未来首条启用；末条未来恢复 | 按末条生效后状态决定单条入口，未来停用之后仍可安排恢复；今日事实不提前变化 | 分层记录 |
| <a id="m02-tl-347"></a>TL-347 自动业务反馈 | 填早于末条的单条日及已知业务错误；改合法日，另测辅助网络失败 | 业务错误就地显示、保留输入、禁止确认，修改后重验；网络失败仍可由保存事务校验，无手动预览 | 分层记录 |
| <a id="m02-tl-348"></a>TL-348 版本权限及历史保护 | 并发新增末条后原请求；重读；撤字段权；未知恢复及旧来源 | 来源冲突不自动换锚点，仍可读输入保持；历史/终态不变，隐藏日期边界不泄露、恢复名称仍独立 | 分层记录 |

合并列表20项及本节6项后实际定义348项、304现行、44退役；本轮实际范围见[单条补验报告](../../verification/021-single-lifecycle-append.md)，数量不是通过数。


<a id="m02-tests-query-selection"></a>
## 24 完整查询结果选择与导出所选

依据[SCN018](011-m02-business-requirements.md#m02-scn-018)、[BR022](011-m02-business-requirements.md#m02-br-022)、[BR023](011-m02-business-requirements.md#m02-br-023)、[AC032](011-m02-business-requirements.md#m02-ac-032)、[AC033](011-m02-business-requirements.md#m02-ac-033)及[013目标协议](013-m02-technical-design.md#m02-tech-query-selection)，由115实施。新增18例已取得[115实施报告](../../verification/115-query-selection-implementation.md)中的实际分层证据；本轮只按该报告明确参数验收，不从原文档评审、111或T07结果推导通过。下表A为真实HTTP、D为真实PostgreSQL/应用事务、B为正常菜单Chromium，主要层在前；每例另记录实际参数、请求、独立期望和受测基线。

独立夹具QS：实例I/企业E，在D=2026-11-03下有137条获权且命中的BU（ID及编码QS001—QS137独立登记），另有未命中BU、同实例失权BU和实例I2同编码BU，不能计入137。每页20条，默认编码升序。基础A具bu.read及全部所选bu.export；R仅有bu.read；字段/范围及导出分权反例分别配置，不按角色名称推导。手写137个ID、四字段及来源映射，预期集合不得调用被测解析器生成。状态/日期/同名及公式样本按各例单独准备，并明确137是否保持。

| 案例 | 前置与操作 | 独立预期及保护 | 主要/补充层；本轮结果 |
| --- | --- | --- | --- |
| <a id="m02-tl-349"></a>TL-349 全结果全选 | QS137/20；从正常菜单点击表头；再看第2页及最后页 | selectedCount=137，不是20/100；所有页行勾选；未命中/失权/I2均排除，queryKey及版本与同次列表一致 | B / A,D；本轮分层结果见115 |
| <a id="m02-tl-350"></a>TL-350 跨页显式选择 | 第1页选2条、第2页选1条，再取消第1页1条 | 依次2/3/2；回调不丢未加载页ID；表头按全局2/137半选 | B / A；本轮分层结果见115 |
| <a id="m02-tl-351"></a>TL-351 全选与排除 | 全选137，第2页取消1条，再回原页重选；最后取消全选 | 137/136/137/0；excludedIds只反映所取消对象；当前页全勾不掩盖另一页排除 | B / A；本轮分层结果见115 |
| <a id="m02-tl-352"></a>TL-352 空集与表头状态 | 0/1/137结果，另显式选满137；部分选择点击表头 | 0时禁用；未选/半选/全选按全集判定；显式选满也显示全选；半选点击选全集，无二级入口 | B / A；本轮分层结果见115 |
| <a id="m02-tl-353"></a>TL-353 页大小及纯排序 | 已选跨页对象；20→50→100；编码/名称合法排序、同名破同值 | ID集合及版本保持；结果序号/焦点可达；选中总量不受100限制；分页不应用D，纯排序以前后D与A关键词一致为前提 | B / A,D；本轮分层结果见115；排序同时应用D另见下一例 |
| <a id="m02-tl-354"></a>TL-354 A/D及实际范围 | 有选择时编辑/取消搜索和筛选草稿；分别应用q/status/列条件；跨日另见TL-391；排序应用不同D；另规范化等价q | 草稿不丢、不暗中查询；实际谓词改变清空；等价queryKey保持；仅排序保持，排序合并不同D清空；导出不用D | B / A；本轮分层结果见115 |
| <a id="m02-tl-355"></a>TL-355 返回与生命周期 | 有返回高亮时全选；名称新标签；同标签详情返回；刷新/POP/退出换账号 | 高亮不勾选；新标签不影响原页；其余重新进入清空，按现有合同恢复名称焦点，不能恢复旧选择权限 | B / A；本轮分层结果见115 |
| <a id="m02-tl-356"></a>TL-356 成员变化与同数替换 | 全选后并发新增命中、移出命中，以及保持137数量但替换一个ID | 所有变化触发不同版本/409并清空重选，不仅比较count；不自动纳入新对象或静默删掉原对象 | D,A / B；本轮分层结果见115 |
| <a id="m02-tl-357"></a>TL-357 来源和字段读权变化 | 成员数不变但资料/头来源变化；名称/编码/日期读权独立撤销；无关写权变化另测 | 来源/有效读权变化冲突，旧值不能导出；重新选后受限字段为空且筛选计数同源；纯写权变化不改变有效读取集合版本 | D,A / B；本轮分层结果见115 |
| <a id="m02-tl-358"></a>TL-358 恶意选择与主体隔离 | 混用模式、重复/非法ID、U外/失权/I2 ID、旧主体/会话版本、未知/重复JSON键及超正文预算 | 严格400/401/403/409/413按013顺序安全拒绝，无对象存在性/未获权计数泄露；伪造摘要不获得权限，不部分处理 | A / D；本轮分层结果见115 |
| <a id="m02-tl-359"></a>TL-359 迟到及失败恢复 | 暂停旧resolve后取消/新选择/换查询/会话；分页加载失败再重试 | 只有当前代际更新；核对中不显示确认数量且禁导出；取消可用，失败清空，不被旧成功或重试复活 | B / A；本轮分层结果见115 |
| <a id="m02-tl-360"></a>TL-360 完整CSV及顺序 | QS137全选排1条，按code及获权name排序导出；另显式跨页3条；空选择直调 | 恰136或3个数据行，稳定顺序/固定四列/UTF-8 BOM/日期状态正确；空选择400且无文件；无第二导出入口 | B / A,D；本轮分层结果见115 |
| <a id="m02-tl-361"></a>TL-361 读与导出分权 | R可读全137但无bu.export；另仅136条具export、其他对象具写权；随后完整授予 | 均可按read选择137；缺一条export仍整次拒绝，不变成136条可导出子集；完整授予后重验成功；按对象合并字段读权，隐藏列不是授权 | A,D / B；本轮分层结果见115 |
| <a id="m02-tl-362"></a>TL-362 CSV编码与公式 | 合法投影含逗号、双引号、换行、中文；编码/名称以=,+,-,@或前导空白/控制字符组合开始；无权字段 | RFC4180可解析；危险文本先中和再转义，原数据不回写；无权字段为空，无内部ID/版本/英文/日志字段 | A / D,B；本轮分层结果见115 |
| <a id="m02-tl-363"></a>TL-363 导出并发及零文件 | 版本核对前并发变更；撤bu.export或bu.read分别先赢；导出先获守卫；生成故障/超时、发送部分响应后断网；前端取消/离开后响应 | 版本变化409，撤权先赢拒绝；成功快照不混入其后新数据；校验/生成失败无CSV头/字节；传输中断可已有网络字节但前端不创建下载；旧代际不能创建文件；成功已交付不承诺追回 | D,A / B；本轮分层结果见115 |
| <a id="m02-tl-364"></a>TL-364 组织历史零变化 | 导出成功、拒绝、重试和取消前后对比组织来源、修订、Outbox、提交结果及原采用引用 | 全部组织事实计数/内容不变；允许Access按职责记录安全判定；不创建命令终态或伪造采用 | D / A；本轮分层结果见115 |
| <a id="m02-tl-365"></a>TL-365 键盘与响应式 | Space切换、Shift当前页连续选择、网格Ctrl/Cmd+A及输入框文本全选；取消后焦点；按 012 核对适用视口与原生缩放，从桌面切窄屏再返回 | mixed可访问状态、可见焦点；网格Ctrl/Cmd+A选全U且输入框仍只选文本；Shift不猜跨未加载页；窄屏全选/清空/导出可达，选择及D保持；不恢复四列表头或横向页面溢出 | B / 组件；本轮分层结果见115 |
| <a id="m02-tl-366"></a>TL-366 2000BU与资源取消 | 2000BU全选/排除/解析/CSV，另用2021BU及20条每页核对超过100页的结果；记录现有复杂来源规模；执行中取消和受控资源失败 | 不逐页前端拉取、不分页截断、不默加业务硬上限；单快照/权限守卫/版本一致；记录耗时、内存、锁与取消释放，临时环境回收，不宣称生产SLA | D,A / B；本轮分层结果见115 |

新增18例后总366个定义、322现行、44退役；这是案例定义数量，不是全产品通过数；本轮选择增量的实际通过范围由115报告与参数映射持有。接口错误、真实权限竞争、CSV零部分结果与正常菜单下载需各自证据，不用mock或静态检查代替。资源按003/017及115收尾，旧报告与失败原件保持。

<a id="m02-tests-operation-summary"></a>
## 25 基本资料操作摘要与变更日志表格化

本章承接[122计划草案](../../archive/plans/122-bu-operation-summary-and-change-log.md)、[SCN027](011-m02-business-requirements.md#m02-scn-027)、[BR034/035](011-m02-business-requirements.md#m02-br-034)及[AC041—044](011-m02-business-requirements.md#m02-ac-041)，页面和技术合同分别见[012](012-m02-functional-and-interaction-design.md#m02-ui-operation-summary)、[013](013-m02-technical-design.md#m02-tech-operation-summary)。20例全部**未运行**，以下是独立预期，不是复用旧产品通过。既有366编号及退役状态保持。

### 25.1 角色、样本与独立预期

正常入口：设置 → 组织配置 → 业务单元 → 详情 → 基本资料/变更日志。新建隔离合成样本，不改共享持久库。A为李明/liming、B为王芳/wangfang，具对象读、历史和所用写权；R只有对象及业务字段读取权，无历史权/写权；CN有历史权但无nameEn读权；X无对象读权。权限以真实授予为准，不根据角色名称硬编码。下列S号是测试别名，执行记录真实sourceVersion/recordId/revision和认证主体。

| 来源及操作者 | 已知输入（全部2026年） | 必须核对的修订和来源，不从被测摘要倒推 |
| --- | --- | --- |
| S1，A，11-03 09:00:00北京时间 | 创建R1，01-01生效，名称“华东一”，英文名空 | R1.1形成于S1，初始区间开放 |
| S2，B，11-03 09:01:00 | 新增R2，06-01生效，名称“华东二” | R2.1形成于S2；R1.2因截止到06-01派生，形成于S2 |
| S3，A，11-03 09:02:00 | 新增R3，10-01生效，名称“华东三” | R3.1形成于S3；R2.2因截止到10-01派生，形成于S3；R1仍为.2/S2 |
| S4，B，11-03 09:03:00 | 更正R3为“华东三新”及英文“East Three” | R3.2/S4，两个字段同一操作；R1仍.2/S2、R2仍.2/S3 |
| S5，A，11-03 09:04:00 | 更正R2为“华东二新” | R2.3/S5，R1/R3未变 |
| S6，B，11-03 09:05:00 | 删除R2 | 当前成员仅R1/R3；R1.3/S6派生截止到10-01；R2原件是S5中.3/A，删除操作B；R3仍.2/S4 |
| S7，A，11-03 09:06:00 | 在S6选R1作锚点，2026-06-01插入，名称改为“华东四”，CHANGE/NAME_ADJUSTMENT；英文名继承空值 | accepted，生成R4.1/S7，R4与R1/R2/R3身份均不同；R1因截止到06-01形成.4/S7，S5原件链接仍为R2.3 |

S1使用CREATE/NEW_ORGANIZATION，S2/S3使用CHANGE/NAME_ADJUSTMENT；S4/S5只更正名称而继承原分类。保存时中文标签分别取本次固定目录的“新建／新设组织”“数据更改／名称调整”，后续读取不能借用变更后的目录。S5仅中文名称变化，beforeContext/afterContext仍各含effectiveDate=2026-06-01、endExclusive=2026-10-01、endInclusive=2026-09-30、CHANGE/NAME_ADJUSTMENT及其原名称快照，diffs仅有name，不能因为上下文包含日期/分类就把它们计为变化。

S7按以下请求输入执行；S6/R1是上表独立样本别名，发送时替换为记录下来的真实来源/记录UUID，R4只能从accepted结果取得，不能由客户端指定。NAME_ADJUSTMENT是当前有限目录的合法原因，新增实际名称“华东四”与S6锚点R1的“华东一”不同；仅调整日期或原因不足以形成S7。

```json
{
  "expectedSourceVersion": "S6",
  "anchorRecordId": "R1",
  "effectiveDate": "2026-06-01",
  "changes": {"name": "华东四"},
  "operationType": "CHANGE",
  "operationReason": "NAME_ADJUSTMENT"
}
```

先以独立提交身份把同一请求的changes改为`{}`，预期no_change、BU头仍为S6、无R4；再用新提交身份发送上述有变化请求，预期accepted且新ID不同于R2。两次不得复用相同Idempotency-Key，因为正文不同；同日重新登记成功后，S5原件仍显示R2.3的“华东二新”，S7显示R4.1的“华东四”。

123后补充两类删除原件入口的独立预期：S5＋R2返回业务修订R2.3；S6＋R2从删除结果入口或复制URL打开，selectedRecord.deleted=true、revision=R2.4、resolvedSourceVersion=S6，operationSummary则为R2.3/S5/A及11-03 09:04:00。两入口均显示“华东二新”和原期间，摘要不能变成S6/B；S6日志的删除者仍为B。S7后再次直达上述两个URL，原修订、删除证据及摘要保持；不把新的R4当作原件。

固定两条删除使用独立P样本，操作者同A/B，来源时刻由各次真实提交记录：P1由A创建01-01生效的P-R1；P2由A插入06-01停用/09-01恢复的固定两条，形成P-R2.1/P-R3.1；P3由B只更正恢复记录名称为“恢复更正”，得到P-R3.2，P-R2.1不变；P4由A删除这对记录。P4＋P-R2/P-R3分别返回删除证据P-R2.2/P-R3.3，摘要分别是P-R2.1/P2/A、P-R3.2/P3/B；前来源P3链接也分别显示这两个摘要，不能把两条都显示为P3/B或P4/A。P4日志只有一次删除操作、操作者A；两条原期间和原分类按对应修订保持。P1按CREATE/NEW_ORGANIZATION，P2按DEACTIVATE/ORG_RESTRUCTURING与REACTIVATE/BUSINESS_RESUMPTION，P3继承原分类，全部通过真实当前入口执行。

未来首条使用独立F样本，固定D=2026-11-04：F1由A创建2026-11-10生效的F-R1“未来甲”；F2由A新增2026-12-01生效的F-R2“未来乙”；F3由B删除F-R2。F1/F2分类分别为CREATE/NEW_ORGANIZATION、CHANGE/NAME_ADJUSTMENT，写入的操作者/时刻来自认证和受控时钟。F3中F-R1因期间重新开放成为.3/F3/B；删除前F-R2是.1/F2/A，删除证据为.2/F3/B。明确核对以下投影，不能从被测摘要反推预期：

| 读取方式与角色 | 独立预期 |
| --- | --- |
| R从正常菜单打开当前F3，不传来源或记录 | result=M02_NO_FACT、viewFact=null，但selectedRecord=F-R1.3、摘要F3/B；展示未来甲及获权的尚未生效，无日志/原件入口 |
| A打开“删除后版本”：仅sourceVersion=F3 | 与当前F3选择相同，历史只读，时间线仅F-R1；无businessAt及被删ID，不显示无资料、不读取当前头 |
| A直达sourceVersion=F3、selectedRecordId=F-R2；另开前来源F2＋F-R2 | 前者保留删除证据F-R2.2，后者为F-R2.1；两个摘要均为F-R2.1/F2/A，原资料是未来乙，不能用F3/B |
| 当前F3或历史F3撤销effectiveDate读权 | 记录与合法摘要仍有；日期、期间及肯定的未来提示均不泄露，不能把裁剪后的记录当无资料 |
| A在F3显式指定未存在的记录ID | selectionMissing=true、selectedRecord=null，无摘要；不自动改选F-R1 |
| R或已撤历史权的A使用复制的前来源/删除原件/删除后版本URL | 历史读取拒绝，清空受限原件和摘要；R仍可重新读取当前F3的合法摘要 |

F样本的未来默认与空态在G/D/A层使用受控D验证；浏览器须通过隔离测试时钟或等价未来日期样本实际验证，并记录真实日期与来源。不能把内部按日调用或前端拼出的摘要当作浏览器通过；业务字段授权、失败清理仍用真实API核验。

业务操作通过真实Go入口，记录时刻由受控测试时钟提供，API不得允许客户端伪造记录时刻或操作者。若实际部署没有可注入时钟，DB/服务层使用隔离fixture与真实事务验证，浏览器另记录实际时刻，不能宣称受控时钟参数由浏览器覆盖。姓名/账号测试使用X03合成主体和公开端口。

分页样本H45为独立BU：H1由A创建01-01生效记录，H2—H45每次更正中文名称为`样本n`；n为3倍数时同时改英文名为`ENn`。奇数来源由A、偶数由B；共45次操作，A 23次/B 22次，英文名实际变化15次（H3—H45）。因此无条件三页为20/20/5，nameEn筛选15次，不能用字段子行数量作total。容量样本H1000按同样规则扩展，保留真实源计数、各字段预期和环境/时限记录。前后值期望直接由明确输入填写，不调用产品差异算法生成oracle。

### 25.2 目标案例与分层验证

案例表中context/through分别简写请求参数contextSourceVersion/throughSourceVersion；来源别名按25.1替换为真实UUID。

每例仅设一个主要层，沿用本文件G（Go规则）、D（真实DB）、A（真实HTTP/API）、B（真实浏览器）和P（真实容量测量）定义；斜线后为有独有风险的补充层，不全矩阵重复运行。身份原子性、历史保护和跨页查询不能只做组件mock；日期/归因预期与真实页面分别留证。每个参数都报告，部分参数通过不能提升整例。

| 案例 | 输入与步骤 | 独立预期与失败保护 | 主要／补充层及当前结果 |
| --- | --- | --- | --- |
| <a id="m02-tl-367"></a>TL-367 资料摘要与历史分权 | R从正常菜单读S4当前R1/R3及F3未来首条，尝试日志和直接旧源API；A重复；X直达 | R见R1的B/S2时刻、R3的B/S4时刻；F3无今日事实仍展示F-R1.3/F3/B摘要和获权未来提示。无日志/原件入口，历史API403；A可读；X安全拒绝，不从摘要获得历史权 | A / B,D；见122实施报告 |
| <a id="m02-tl-368"></a>TL-368 对象头与记录归因 | 依次完成S1—S5，在S3/S4/S5按recordId选R1，再选R2/R3 | S3—S5的R1始终R1.2、B/S2；S4的R2为A/S3；S5的R2为A/S5；不能跟随最新头B/S4或A/S5 | D / A,B；见122实施报告 |
| <a id="m02-tl-369"></a>TL-369 更正和旧修订 | S4后读R3当前及S3旧来源；另对R3合法改日至09-01，保存后返回旧源 | 当前R3显示S4/B，S3仍R3.1/A；改日形成新摘要，旧源日期/身份保持；准确版本读取无写入入口 | B / D,A；见122实施报告 |
| <a id="m02-tl-370"></a>TL-370 派生修订归因 | S2/S3/S6逐次读取前驱；查对应操作日志 | R1.2→B/S2、R2.2→A/S3、R1.3→B/S6；日志标明期间派生，与直接新增/删除项区分；不声称前驱名称被手工改过 | G / D,A,B；见122实施报告 |
| <a id="m02-tl-371"></a>TL-371 无变化与失败 | S5后同值保存、取消编辑、重复日期拒绝、网络失败查回no_change或rejected、刷新/读日志 | 明确终态下不新增来源/成功日志、不改R2.3摘要；结果未明走既有查回，不把网络失败臆断为未保存 | A / D,B；见122实施报告 |
| <a id="m02-tl-372"></a>TL-372 操作分组与记录上下文 | S4双字段、S5仅更正中文名；独立BU固定启停对插入/删除；另只读旧联合来源；读取时目录标签另改 | 每次成功一个操作组、total增1，各记录准确前后值及直接/派生行；S5前后上下文均有06-01、10-01及原分类名称，diffs只有name，UI无逐行原件请求；不借当前目录或当前详情补值；旧联合只读 | A / D,B；见122实施报告 |
| <a id="m02-tl-373"></a>TL-373 删除前后及同日新记录 | S6打开前来源原件、删除结果的墓碑原件、复制URL直达及删除后版本；先空changes再完整S7并重复；按25.1分别执行P固定两条与F未来首条样本 | 空changes后仍S6且无R4；accepted后R4身份不同。S5/R2.3与S6/R2.4两类原件的摘要均R2.3/S5/A；删除日志S6/B。S6后链接无R2 ID、时间线仅R1/R3；内部补验07-01为R1；S7不串原件。P4两原件摘要各为P2/A、P3/B，不用P4/A或统一P3/B。F3后链接无今日事实仍选F-R1.3/F3/B、显示未来甲及获权未来提示；两类F-R2原件摘要均F2/A；显式缺失不换条 | B / D,A；见122实施报告 |
| <a id="m02-tl-374"></a>TL-374 身份证据和改名 | 新来源姓名/账号齐全、姓名空、服务主体各一例；操作后改名/改账号/停用；旧source仅subjectId | 新快照各自保留原身份；无姓名显示账号；服务身份明确；旧源仅标识不补造姓名/账号，不显示当前登录者 | A / D,B；见122实施报告 |
| <a id="m02-tl-375"></a>TL-375 上下文裁剪先于筛选 | CN读H45及S4，筛nameEn；分别撤日期/分类读权，查看前后上下文及直接旧源；有日期权但期间开放另测；仅隐藏字段变化的操作；F3当前/历史撤日期权 | 受限条件403且无匹配数；上下文与diffs都省略失权字段，分类代码/标签一起裁剪，日期失权省略日期和两种期间终点，不将缺值写成开放null；有权空英文名和开放终点各为null。F3保留记录及F-R1.3/F3/B摘要，无日期/期间或肯定未来提示；无字段名/值/隐藏条数泄漏 | A / D,B；见122实施报告 |
| <a id="m02-tl-376"></a>TL-376 全集合筛选与页码 | H45按20分页，页3应用nameEn条件，页大小20/50/100，清除后再翻页 | nameEn恰15操作且回page1；无条件total45、三页20/20/5，50或100为一页；子行不计数、不截断匹配；全部来自同一上界 | A / D,B；见122实施报告 |
| <a id="m02-tl-377"></a>TL-377 差异筛选不误用上下文 | H45筛changedField=nameEn，再isAnyOf=name,nameEn；在包含S5操作的日志分别筛name/effectiveDate/operationReason；重复值、两个条件、未知/缺项参数 | H45为15/45组；S5操作只在name结果中，虽有日期/分类上下文也不出现在其他两项结果中；匹配后保留全部获权子项；非法/重复条件400，不静默扩大结果；同列OR | A / G,B；见122实施报告 |
| <a id="m02-tl-378"></a>TL-378 身份与动作过滤 | H45 actor equals liming/wangfang；A改名后搜索新名；合法身份文本含百分号/下划线/反斜杠；action CREATE/CORRECT及非法值 | 23/22；新名不命中过去快照；特殊字符按字面；H1创建1项、其他44项更正，动作不是业务分类，未知值400 | A / D,B；见122实施报告 |
| <a id="m02-tl-379"></a>TL-379 日界与时钟回拨 | 独立源在2026-11-02T15:59:59.999Z及16:00:00Z；查11-03的is/before/after/onOrBefore/onOrAfter；保存顺序后一来源时间更早；改日06-01→07-01 | 北京时间分别11-02/11-03；is只后者，before只前者，after均不匹配，onOrBefore两者，onOrAfter只后者；按source_no排序；生效日筛06-01或07-01均命中改日操作；非法日期拒绝 | G / D,A,B；见122实施报告 |
| <a id="m02-tl-380"></a>TL-380 历史上下文与分页上界 | H45后新增H46再翻页/筛选/刷新；S5已存在时分别请求context=S4/through=S5、无context/through=S5、context=S4无through、context=S4/through=S3；历史刷新/返回当前；在途撤权/换主体及非法参数 | 旧上界45、当前刷新46；四种请求分别400/成功至S5/成功至S4/成功至S3，响应回显准确上下文和上界；历史刷新仍S4，返回当前才取头；空值/重复/非法UUID400；每次重验，撤历史权可重读摘要，撤对象权清空，迟到响应不跨上下文覆盖 | A / D,B；见122实施报告 |
| <a id="m02-tl-381"></a>TL-381 伪造与失效引用 | 当次/前原件/后版本/上界换成另一BU、另一实例、随机UUID；撤权后用复制URL直达；S/P/F两类原件入口分别撤历史权；F3显式不存在记录ID | 非法来源按401/403/404现有安全规则拒绝，无对方事实；合法F3中的缺失记录为selectionMissing且无摘要，不静默选未来首条。两类原件及删除后版本失权均拒绝并清空，仍有当前读权可重读当前摘要；未知旧源不回退当前，旧URL不是授权 | A / D,B；见122实施报告 |
| <a id="m02-tl-382"></a>TL-382 返回上下文与新标签 | 列表保留查询，日志筛选/页2滚动后开原件再返回；原日志期间出现新操作；切换历史来源、新标签打开、返回当前及读取失败 | 原查询、contextSourceVersion/throughSourceVersion、筛选/页码/页大小/滚动/焦点恢复，原上界不混入新操作；切换来源或返回当前重置页及上界；新标签不改原页，历史标识可见，重读成功前无写入口 | B / A；见122实施报告 |
| <a id="m02-tl-383"></a>TL-383 直观显示与可访问性 | 桌面长文本及真实 200% 缩放，长姓名/账号/多行前后值；键盘展开/收起、筛选应用/取消 | 摘要默认可见、自然换行，无正文大块来源说明；主要差异不靠展开；链接可新标签、来源按钮有展开状态/焦点；清除应用回页1，取消不请求；窄屏内容和操作完整可达 | B / 组件；见122实施报告 |
| <a id="m02-tl-384"></a>TL-384 身份原子性及伪造 | 保存前使必要X03身份端口失败；另合法缺姓名；请求加operator/recordedAt/subjectId只读字段 | 身份失败无accepted、无新来源/修订/Outbox，遵守既有失败终态协议；缺可选姓名正常保存账号；伪造字段400，不采用客户端身份；数据库写失败快照和事实一起回滚 | D / A,G；见122实施报告 |
| <a id="m02-tl-385"></a>TL-385 封口历史零改写 | 122实施前保留旧单条/删除/联合来源、提交终态及采用引用摘要；运行新读模型/身份改名/成功与拒绝保存 | 旧证据原字节/语义保持，不补写旧actor姓名/差异；读取不改组织事实，成功只新增合法来源，原采用不改；旧协议不重新开放 | D / A；见122实施报告 |
| <a id="m02-tl-386"></a>TL-386 长日志预算与取消 | H1000真实DB，中文名1000次/英文名333次，20/100分页与筛选；20并发、冷热缓存、取消/超时 | 确切总数、完整操作组、无前端全量下载；按013预算记录分位/锁/内存/SQL及取消释放，未达标记失败；临时资源回收，不宣称生产SLA | P / B；见122实施报告 |

补充参数明确如下：TL-370的D补S6删除引起的R1派生及旧源保护；TL-372的B补双字段直接可见、S5上下文完整且无逐行补查请求，以及固定两条分组/逐项链接；TL-376的B补页3应用nameEn及清除回页1；TL-379的D/A补北京时间日界与回拨顺序，B仅核对两时刻显示；TL-380的D补新增/撤权屏障，B补历史刷新不越上下文、返回当前重置上界及在途撤权清缓存；TL-375的B补无日期权时不展示开放期间、无分类权时不残留标签；TL-386的P持有真实DB/API性能参数，B仅核对按页请求与取消界面。其余补充层按第2.4节代表输入范围；不把主要层执行一次推为所有补充层通过。

### 25.3 验收映射与执行交接

| 验收 | 主要案例 | 额外证据边界 |
| --- | --- | --- |
| [AC041](011-m02-business-requirements.md#m02-ac-041) | TL-368—371、383 | 必须同时证明归因和真实页面；单个截图不证明修订关联 |
| [AC042](011-m02-business-requirements.md#m02-ac-042) | TL-367、374、375、380、384、385 | 当前真实鉴权和DB原子性；不靠隐藏按钮证明拒绝 |
| [AC043](011-m02-business-requirements.md#m02-ac-043) | TL-372、376—380、386 | 跨页、并发及真实预算；不复用旧列表容量成绩 |
| [AC044](011-m02-business-requirements.md#m02-ac-044) | TL-373、381—383 | 链接实际目标、上下文及可访问性；文档图不是页面验收 |

本章编写及范围评审见[122合同记录](../../verification/122-operation-summary-contract-review.md)。合同初稿时总386个定义、342现行、44退役，新增20例当时未运行；当前结果见122实施报告。122 P1/P2/P3另记真实输入、环境、分层结果、失败和复验，旧报告不覆写；116 R3接入共用流程时验证BU，R5部门真实接入后另行验证；不能据本章宣称部门已实现。

2026-11-04：按[122评审修订](../../verification/122-operation-summary-contract-review-fixes.md)补齐记录上下文、历史上下文/分页上界的独立预期及S7实际资料变化，保持20个新增编号和未运行状态；原366行、44退役项及旧报告不改写。

2026-11-04，123后复核：补充S6删除证据入口、P固定两条不同修订归因、F未来首条/空态/权限/缺失记录样本，扩展TL-367/373/375/381参数；新增参数及122的20例仍全部未运行。全表现有394个编号、350现行、44退役，本次不新增或退役案例，123案例和旧报告不变；范围及检查见[本次记录](../../verification/122-post123-contract-review.md)。

## 26 普通维护取消查看日期

按123及011现有AC001/007/008/032/033执行；未运行参数不沿用旧通过。字段/权限/已保存来源原件仍按原案例核验。

| 编号 | 独立输入与步骤 | 预期及层级 |
| --- | --- | --- |
| <a id="m02-tl-387"></a> TL-387 | D当天有甲，D之后更名乙；普通列表及直接详情 | 列表仅一行甲、今日状态；详情默认今日记录，时间轴可选乙，无查看日期入口 |
| <a id="m02-tl-388"></a> TL-388 | 今日停用、未来恢复；另建未来首条丙；搜索/排序/全部及启用/停用/未生效筛选/CSV | 前者仍停用，丙显示未生效/首日；丙可找到且不被启用筛选命中；CSV同投影 |
| <a id="m02-tl-389"></a> TL-389 | 历史停用记录、未来启用记录、未来保存、删除结果、显式缺失ID | 名称/状态/期间随记录；未来提示独立，保存定位实际记录；缺失不换条 |
| <a id="m02-tl-390"></a> TL-390 | 普通GET与选择/导出body注入businessAt | 全部400，客户端无输入；正常响应日期由服务端给出 |
| <a id="m02-tl-391"></a> TL-391 | 受控D取列表和选择，推进次日后核对/导出旧version，再取新列表 | 409 M02_SELECTION_CHANGED、零CSV；新日列表和版本更新，新选择正常 |
| <a id="m02-tl-392"></a> TL-392 | 旧来源、被删原件、无日期读权及当前撤权 | 原件不读最新；未来提示不泄露不可读日期；拒绝清理受限正文 |
| <a id="m02-tl-393"></a> TL-393 | 搜索/排序/分页/返回/前后对象导航、窄屏/键盘/放大 | 上下文保持且不传播任意日期；选择与返回高亮独立；无残留日期菜单和样式 |
| <a id="m02-tl-394"></a> TL-394 | 部门记录期间内祖先更名、停用与未来首条；新旧来源 | 按119/120分段/停用前/首日口径；本轮仅合同，产品留116 R4/R5 |

本节新增8例中BU范围需本轮实际执行，TL-394部门参数未运行；原122的20例产品仍未运行。当前350现行/44退役，原章节历史计数保留其时点。

2026-11-04：用户批准122后，TL-367—386的独立S/P/F/H样本及主要/补充层按[实施报告](../../verification/122-operation-summary-implementation.md)与[逐例映射](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-implementation/case-matrix.json)执行；不改变394/350/44编号统计，不以单层通过代替未受测参数。此前“未运行”记录保留其合同阶段时点。

2026-11-04实施复核补验：原122汇总不能代替TL-373全部浏览器主层参数；本轮补齐S/P/F两类原件、双主体归因、同日新ID、未来默认/缺失及日期/历史撤权。TL-375/380补不reload的成功200收窄权限，TL-376/382/383补翻页、页大小、应用/清除、空结果和错误焦点。实际范围及失败复验见[122评审修复报告](../../verification/122-operation-summary-review-fixes.md)，原编号、定义和报告保持。

2026-11-05：116 R1当前夹具切换117目录。TL-283及TL-298—301中需要真实分类差异的组合/权限/撤权/冲突案例改用CHANGE的NAME_ADJUSTMENT→OTHER_DATA_ADJUSTMENT；首条创建单一原因不伪造第二选项。旧CREATE/INITIAL_SETUP、NEW_UNIT及CHANGE/NAME_CHANGE分别作为历史快照和accepted/rejected/no_change精确重放的受控隔离夹具，不开放旧目录新写。关系合成类型验证Clone、稳定身份资料差异、引用版本REFERENCE结构差异及组合差异，只计R1纯Go扩展点。旧122报告保持原字节，当前122样本使用新目录重跑；见[116 R1报告](../../verification/116-r1-kernel-and-directory.md)。

## 27 R2事务与恢复提炼执行交接

116 R2沿用第13—16章的当前权限、幂等/版本竞争、原子性/预算、存储/源事件案例及第18章追踪，不重新编号或改写原结果。R2-A覆盖业务保存点后拒绝/技术错误/取消、外层提交未明和安全查询审计；R2-B覆盖精确重试与resolve先后、四类持久终态、当前读权交集、原摘要/历史来源及字段诊断裁剪。输入、屏障和写前后快照保留在本轮证据中；新增技术参数复用原稳定案例，有新增业务预期时才追加编号。

BU实际通过必须来自当前接入后的真实DB/API；共用端口使用其他合成结果类型和不同锁命名空间的测试仅证明机制，不计部门产品。回归同时覆盖第25/26章122/123已实现参数。分批范围及执行结果分别由[121矩阵](121-dated-object-admission-matrix.md)和[R2实施记录](../../verification/116-r2-transaction-and-recovery.md)持有，当前案例总数保持394/350/44。

## 28 R3前端流程提炼执行交接

沿用原创建/维护/启停/删除/改日/固定两条、未保存保护、未知提交和会话竞争参数，不增加同义业务案例。R3-A独立补验原正文不可变、同类型阻塞/跨类型集合及事件隔离、查回与在途重试竞争、存储异常；强类型合成对象只计机制验证。R3-B从正常BU菜单执行全套现行Chromium用例，覆盖第25/26章122/123以及列表选择/CSV/返回上下文，保持日期/分类错误定位和字段撤权清理。

共享表单和列表受影响视口含桌面、短屏、390/320窄屏及真实200%/400%缩放；实际执行参数和失败复验分别留证，不把类型检查或部分复验写为整轮通过。当前394/350/44编号统计不变，部门TL-394及39个产品样本仍not_run。分批结果见[121](121-dated-object-admission-matrix.md)与[R3报告](../../verification/116-r3-frontend-workflow.md)。

R3-09补充未保存保护参数：管理员从正常菜单更正日期后确认退出，分别注入退出503及网络中断；原编辑器仍挂载时重新输入名称，取消、路由离开及真实浏览器关闭均重新保护，继续编辑保留新草稿，原来源不变。补验新建失败退出后的新草稿、明确放弃仅完成本次路由，以及固定两条既有取消/原引用恢复；实现见dated-workflow.spec.ts，结果见[评审修复](../../verification/116-r3-review-fixes.md)。不增加同义TL编号，不将修复前187项结果称为修复后整轮通过。

## 29 R4部门后端实施交接

R4按[121第9节](121-dated-object-admission-matrix.md#r4-backend-acceptance)将原D01—D39映射到真实DB/API，初版增加16组技术参数，关系简化后追加R4-V17—V20，分类查询/安全诊断补齐R4-V21—V24；设计完成后以5fa3275执行R4；39例和24组技术参数的真实DB/API、生成类型及定向故障结果见[本次报告](../../verification/116-r4-department-backend.md)和[实际映射](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-backend/acceptance-map.json)。原设计样本和旧通过记录均保持，不计R5菜单通过。沿本文件现行标准动作及TL-367—386的适用后端语义、TL-394路径分段提供独立预期，不新增同义TL编号、不修改旧报告和原案例结果。BU参数继续使用原BU字段；部门把H45的英文名差异参数明确映射为shortName，保留45组/15次变化/20+20+5分页及23/22操作者预期，不新增部门英文名。

特别核验122的基本摘要独立于日志读权、所选修订/派生修订归因、S/P/F两类删除原件、X03身份失败原子性、日志同列筛选和固定上界。部门关系始终通过真实合法BU/父安排建立，不能为了复用样本跳过关系保护。TL-386的容量分位/20并发留R6，R4仅覆盖相关预算、取消和锁释放；键盘/焦点/视口/未保存保护及正常菜单留R5。共用Prepare/Finalize和BU反向守卫发生产品变更时，执行受影响BU真实回归，不沿用R3旧通过代替。

2026-11-05 R4补验：department_summary_test.go以S1—S7、P1—P4、F1—F3和H45手工顺序核对摘要、两类原件、同日新身份及固定窗口；英文差异映射shortName，日期和操作者分配保持。缺失及空ADD、缺失选择和日志末页裁剪已修复，受限日期上下文省略字段，有权空简称及开放终点返回null。TL-394分段及损坏证据失败关闭由department_integrity_test.go和D08/D09/D33—D36承担；本批只计后端参数。

R4回归补充TL-380：历史日志切回基本资料后立即返回当前，React Router尚未完成渲染时也必须保留最后URL中的页签。确定性连续DOM操作在修复前重现；同时保持从历史日志直接返回当前日志的既有行为。失败、修复与定向复验见116 R4报告，不新增同义TL编号。

## 30 R5双对象页面执行交接

R5沿用现行标准维护案例，department-journeys.spec.ts与department-acceptance.spec.ts以BU、部门两种真实类型分别执行正常菜单创建、更正/改日、位置新增、删除、启停、固定两条、未知提交及R3-09退出失败后的新草稿保护。对象差异仅由资料字段、关系候选、路径和各自公开API承担；不使用BU DTO或合成适配器冒充部门。

TL-394菜单参数由部门父子关系/祖先分段/同头显式来源、停用后原上级转BU再恢复、未来首条读取查回承担；TL-367—386适用页面参数检验所选修订摘要、独立日志读权、行内准确来源及撤权清理。跨日授权另用TestDepartmentR5BusinessDayBrowser真实服务和两个浏览器时区；137项全结果选择/排除/八列CSV及路径依赖失效由department-selection.spec.ts承担。后台全量S/P/F、H45和关系竞争仍以R4资产在本轮完整integration的结果为准，不将代表性页面旅程说成每个后端参数都从菜单重跑。

部门新增七类页面的桌面、390/320、短屏、键盘及真实200%/400%由department-layout.spec.ts留证，共用BU既有Chromium集合另做完整回归。394个稳定编号、350现行、44退役不变，实际输入、首次失败、修复复验及分层边界见[R5报告](../../verification/116-r5-department-pages-and-acceptance.md)与[121第10节](121-dated-object-admission-matrix.md)。R6容量及真实外域消费者不计入本轮。

2026-11-06 R5多轮补充：121追加R5-13—18，覆盖当前所选引用投影与候选资格分离、200撤权及失败重读保护、三种维护动作省略空关系补丁、异步辅助与主动保存的焦点区别、搜索词保持、名称错误定位及真实加载后的布局截图。沿原授权、错误定位及草稿保护场景补参数，不增加同义TL编号；实际逐轮结果见[多轮评审](../../verification/116-r5-multiround-review.md)。
