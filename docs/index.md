# CHNHCM

此索引保留公开项目前身的技术文档和验收导航。历史运行附件由原仓保管，对应链接需要私仓权限。

X03 首批拟交付本地账号管理、部门授权和指定岗位核编授权。管理员授予、经办人办理、按份撤销及重启后持续失权是贯穿验收的业务路径。T01 已明确职责边界。公共 T02 用 00041 迁入存量授权，原办理人通过原会话和提交号查回业务结果，客户管理入口由 T04 接续。Casbin 和 OPA 均暂缓。[027 实施计划](plans/027-x03-local-access-rollout.md) · [028 客户管理员合同](specs/x03/028-local-account-and-authorization.md)

[M01 个人资料扩展](plans/504-personal-profile-extension.md)单列 T01 之后的九栏维护。字段、整页保存及七类弹框布局由[九栏目合同](specs/m01/505-personal-profile-contract.md)规定。[九栏方案评审](verification/504-personal-profile-review.md) · [弹框布局核对](verification/504-personal-profile-dialog-layout.md) · [第一批三栏实测与验收地址](verification/504-personal-profile-first-batch.md) · [第二批地址与家庭联系人实测及验收地址](verification/504-personal-profile-second-batch.md) · [第三批教育经历与专业凭证](verification/504-personal-profile-third-batch.md)

人员登记已按主要证件号码判断重复，同名可直接建档。旧档案升级核对和人事试办记录见[本次实施与试办](verification/500-personnel-registration-trial.md)。[三种登记情况](specs/m01/501-person-registration-requirements.md#m01-registration-examples) · [人员详情与变更日志](verification/500-personnel-change-log.md) · [4380 手工验收环境](verification/500-personnel-change-log.md#personnel-log-4380)

M03 按岗位、按日核编。招聘可提前寻访，在发出正式录用时一并落实名额。人事从已接受且资料齐备的录用直接办理入职，从原调动办理两岗接替、改期或撤销。[编制管理合同](specs/m03/313-position-capacity-contract.md) · [调动与两岗交接](specs/m03/313-position-capacity-contract.md#m03-br-037) · [办理简化复核](verification/313-staffing-procedure-review.md)

对象详情中的字段分组、多条记录摘要、可见条数及明细返回遵循[012 详情正文布局与交互](specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-detail-sections)。普通页面使用五种 100% 视口，关键流程必测原生 200%，400% 按条件做专项验证。[012 视口与缩放](specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-viewport-matrix)

M03 岗位核编和部门调剂都可在 `wt-dev-d` 的 4486 验收。管理员从「设置 → 职务配置 → 岗位」打开已有岗位，在「岗位编制」办理核编。岗位默认显示当前值，历史记录按生效日或变更日志原件定位。「部门编制分配」从统计日进入部门详情。点选时间轴可查看总额或岗位分配变动后的整份分配，也可查询任意获权日期、办理同日多岗调剂。[打开验收环境](http://127.0.0.1:4486) · [部门时间轴与 4486 现场复核](verification/315-department-quota-allocation.md#71-4486-原库更新与双角色复核) · [统一环境交接](verification/315-department-quota-allocation.md#6-统一验收环境-4486) · [岗位演示与验收](verification/315-position-quota-implementation.md) · [313 编制管理合同](specs/m03/313-position-capacity-contract.md) · [315 后台实施计划](plans/315-staffing-quota-administration.md) · [架构复核](verification/315-staffing-quota-plan-architecture-review.md)
140 的行政区划授权、组织记录期间显示和切片登记已分别通过本地验收。T04 地点与法律实体共同历史流程待下次相关变更触发。[分项实施记录](verification/140-module-cohesion-implementation.md) · [实施路线图](plans/140-module-cohesion-roadmap.md) · [方案评审](verification/140-module-cohesion-roadmap-review.md)

岗位维护从「设置 → 职务配置 → 岗位」进入。按部门找岗位，从岗位详情调整管理上级。发布历史可查回当时的管理上级。看整条汇报线时，从列表的「更多操作 → 查看岗位上下级」进入。[岗位关系现行合同](specs/m03/312-position-contract.md#31-管理汇报的唯一入口) · [139 实施方案](plans/139-position-workspace.md) · [四轮方案评审](verification/139-position-workspace-review.md) · [本地实施与验收结果](verification/139-position-workspace-implementation.md)

从「设置 → 职务配置 → 编制管理」可按岗位和业务日期核定、调整名额，按直接所属部门查看核编小计。招聘预留、录用及人事任职占编由后续消费者接入。[313 首批实测](verification/313-quota-implementation.md)。各模块已有入口与实际签收范围见[上下文登记](architecture/contexts.json)。[编制管理合同](specs/m03/313-position-capacity-contract.md) · [调动与两岗交接](specs/m03/313-position-capacity-contract.md#m03-br-037) · [办理简化复核](verification/313-staffing-procedure-review.md)

[M03 职类实施批次路线图](plans/314-m03-job-function-roadmap.md)已完成[路线图评审](verification/314-job-function-roadmap-review.md)、[T01 升级基线](verification/314-job-function-t01-baseline.md)、[T02 职类目录](verification/314-job-function-t02-directory.md)、[T03 职务双维写入与升级](verification/314-job-function-t03-upgrade.md)和 [T04 职务候选、查找与导出](verification/314-job-function-t04-query-export.md)。[T05 分层验收](verification/314-job-function-t05-acceptance.md)已完成 308 独立签收，314 执行已关闭。AC090 的核编独验及旧职务、岗位、核编并存联验见 [313 首批实施记录](verification/313-quota-implementation.md)。职务双维分类、旧记录及与编制首批分别验收的规则见[308 合同第 14 节](specs/m03/308-job-foundation-contract.md#m03-job-function)及[职类合同独立评审](verification/308-job-function-contract-review.md)。

[138 行政区划目录交互实施方案](plans/138-geo-area-interactions.md) · [评审与验证](verification/138-geo-area-interactions.md)

[M03岗位定义与岗位层级实施方案](plans/311-m03-positions.md)及[对象合同和带日期验收样本](specs/m03/312-position-contract.md#position-acceptance)。两批已分别完成受托业务签收。[第一批岗位定义签收单](verification/311-position-definition-signoff.md) · [第二批岗位层级签收单](verification/311-position-hierarchy-signoff.md)

[三份stash已清理](verification/005-stash-disposition.md)：无待恢复实现，仓外归档已逐份实际恢复验证；当前stash为空，运行环境保持。

[M03第二批方案](plans/309-m03-grade-ladder.md)与[职级／阶梯接入合同](specs/m03/310-grade-ladder-contract.md)：先Grade完整维护，再Ladder及Job可用职级；[方案评审](verification/309-grade-ladder-plan-review.md)已完成，[G1职级实施与验收](verification/309-grade-g1-implementation.md)已完成，独立4381环境保留验收；[G2阶梯与职务职级实施](verification/309-grade-g2-implementation.md)已完成本地完整验收并关闭T03／T04，独立4382环境保留；正式CI及推送／合并未执行。

[职务页面标准整改](verification/308-job-page-standard-implementation.md)：7项偏差已修复，14项验收、28个尺寸角色单元及6个权限单元通过；24根DB/API与90项浏览器分轮核验，关键要求已接自动检查。[PR #22](https://github.com/jacksonlee411/CHNHCM/pull/22)已通过最终22任务并合入`a9e5daca`，main三项检查成功；补修日志直接维护和筛选后退后，[4380](http://127.0.0.1:4380)前端为`3b18f024`，API仍为`1d65ecea`，更新后8个现场单元通过，见[发布交接](verification/308-job-page-standard-implementation.md#m03-308-publication)。此前用户授权的实际写入及留存验收中，25个场景与8个原生缩放单元通过，新增164个对象／245份来源保留，原4个对象及组织、身份授权保持，样本入口见[实际体验验收](verification/308-job-page-standard-implementation.md#m03-308-live-retained)。精简AGENTS保持；三份旧stash已按[005处置](verification/005-stash-disposition.md)归档清理。

[代理入口精简与暂存交接](verification/005-agents-entry-simplification.md)：关键规则与页面标准集中到根AGENTS，原文完整保存；该记录保留首步暂存时点，后续恢复以上述308记录为准。4380验收环境继续保留。

[M03职务基础方案](plans/307-m03-job-foundation.md)与[首批合同](specs/m03/308-job-foundation-contract.md)：首批职务系列／职务，职级、岗位及真实消费者分批接续；文档评审及实施边界见[评审记录](verification/307-m03-job-foundation-plan-review.md)。

[125暖态目标与当前构建补验](verification/125-target-adoption-and-current-validation.md)：采纳大子树路径及末页暖态400/2000ms；当前查询、20条保存及法律实体共62组11600次零错误并阶段通过，普通保存500ms保持。219634份原来源及新增回执重启核对通过，自有资源已回收；原长期失败、其他未测边界及125整体P1保持，固定环境未更新。

[136 正常交付成本观察](verification/136-actions-observation.md)记录观察窗口、逐次交付分钟、主线复用、失败重做与可比历史。[按影响选择 CI](verification/136-impact-selection.md)记录 DB/API 与浏览器独立选测及限定、文档 PR 的等价复用。

减少 80% GitHub 托管分钟的试点以 5 次业务交付为基线，记录全量回归如何前置、分片如何迁至一次性 runner，以及实际用量的验收口径。[136 托管分钟试点](verification/136-hosted-minutes-pilot.md)

客户实库案例的组织明细由私有原仓保存。 [306部门路径与集团案例](plans/306-department-path-simplification.md) · [实施记录](verification/306-department-path-simplification.md)

[136 Actions发布与4390交接](verification/136-actions-publication.md)记录 PR #19 在 22 项全量检查通过后合并，main 实际复用，三项门禁成功。4390 已更新至 6ce8a3e9，双角色 16 条旅程和原数据复核通过。记录保留该次首失败及估算的 598 runner 分钟。

[136发布前本地收口](verification/136-actions-prepublication.md)：原CI/规范已提交，本地同步main并保留双方功能；检查与限定业务复验单列，远程仍待发布。

2026-11-16按用户要求，本轮162文件已本地提交650db772，原m02-current-vanke5-b已更新[4390](http://127.0.0.1:4390)（API8390/DB54392），共享内存512MiB，追加00013；原30表、132299组织来源及账号保持，双角色12条页面流程与4组返回定位通过。两批部门整改生效，单会话路径400ms仍为参考、7组时延及125容量P1保持；其他52容器/47原卷保持，未推送。见[固定更新记录](verification/125-4390-runtime-refresh.md)。

[125部门范围查询接续](verification/125-query-scope-continuation.md)：已采纳20会话大子树列表／路径1500ms，完成查询减负和两形状28800次零错误复验，137/144组阶段通过，万科7项时延失败保持。单会话路径400ms参考6组全通过，正式300ms未改；40个DB/API根、800浏览器、CSV、全原历史及回执恢复通过，自有资源已回收，固定4390未更新。

[125大范围写入与树容量整改](verification/125-tree-write-remediation.md)：已采用5秒部门准备/累计2秒显式取锁，15个相关DB/API根及新构建200次正式保存通过，P95为1520毫秒，满足该组2000毫秒阶段目标。合成19200/万科9600HTTP零错误，仍有44组时延失败；800浏览器、CSV及全部原历史/新增回执重启核验通过。自有资源已回收，125容量P1保持，固定环境未更新。

[125同步main与性能复查](verification/125-main-sync-performance.md)：已提交并同步main d810757b，修复整合导航503；正式3000次零错误。列表／路径无明显变化，导航API约快15—16%；双角色少量页面切换诊断约1.67→0.42秒，非P95。125容量P1保持。

地点所在地区交互整改见[本次实施记录](verification/204-location-region-picker.md)：连续选择、结果回填、上级路径、组合搜索及滚动追加，验证结果单独登记。


本批法律实体基础维护及实际验证见[法律实体第一阶段实施](verification/206-legal-entity-implementation.md)；用途/组织绑定和M04消费者后续接续。
法律实体首阶段：[206实施计划](plans/206-legal-entity-implementation.md)、[207对象合同](specs/m02/207-legal-entity-design.md)、[计划评审](verification/206-legal-entity-plan-review.md)。

M04 直接入职沿用人员原档，可引用任用决定或说明无编号通知。先交付按职务路径，再接入按岗位免占和占编。[400 实施方案](plans/400-hire-readiness.md) · [401 业务验收](specs/m04/401-direct-hire-requirements.md) · [402 办理页面](specs/m04/402-hire-interactions.md) · [403 事务与来源](specs/m04/403-hire-technical-contract.md) · [办理负担复核](verification/400-hire-simplification.md) · [T03 开工与实测](verification/400-first-hire-integration.md)

2026-11-15 G4/G5集成接续：[性能及集成记录](verification/205-g5-publication.md)登记20会话14组中的6组达标、8组超标，D08/D09及125其他容量P1保持；用户授权提交、同步main、push/PR及最终三项CI成功后合并。已同步e359f741并补齐地点302导航，3个DB/API根及26条浏览器同轮通过；PR #16最终三项CI已通过并合入d810757b；随后按用户要求更新4383，原38表与全部历史保持，147项API及双角色8组导航通过，见[运行更新](verification/205-g5-publication.md#7-合并构建的4383运行更新)。

- [304企业完整维护](plans/304-enterprise-maintenance.md)：唯一企业完整按日维护已实施，原实施批255根DB/API及98项浏览器分轮通过、6条件项跳过；后续已在[45300](http://127.0.0.1:45300)运行，原库保持，集成及发布见[本批记录](verification/304-enterprise-maintenance.md)。

- [303部门一级菜单](plans/303-department-primary-navigation.md)：部门与设置平级、部门在前，已在[45300](http://127.0.0.1:45300)生效；原数据保持，分轮验证及旧环境保留见[实施记录](verification/303-department-primary-navigation.md)。

2026-11-15 G5实施、主代理复评及首批限定验收完成：[205记录](verification/205-g5-implementation.md)持国家400次、区划3600次零错误与31条浏览器结果；G4完整证据按1392个不变产品文件复用。4383的m02-current-vanke-c已追加00013，原30表、132284份组织来源及目录/身份保持，147项API和双角色菜单/地点未保存草稿通过。200首批关闭；20会话地点D08、区划D09及125其他容量P1保持，自有临时资源已回收。

2026-11-15 G4地点实施及限定验收完成：方案已提交da164345，迁移00013、四地地址、完整共用维护、独立表组/API、固定选择及CSV已交付；主代理复评通过，251个DB/API根及354个浏览器用例分轮通过，8组1600次HTTP零错误、400份回执与全部原历史重启核验通过，详见[实施记录](verification/204-g4-implementation.md)。G4限定验收已完成，自有临时环境已回收；20会话四组P95仍超500ms，按非阻断口径由200-D08接续。该批G5及运行更新尚未执行，当前结果见上条；原失败和125其他容量P1保持。

2026-11-15按用户要求核对最新main并制定[G4地点实施方案](plans/204-g4-location-implementation.md)：完整共用维护、四地地址、逐动作采用、企业授权、固定选择与CSV及可靠提交；三轮主代理文档评审通过，3项P1、3项P2文档层关闭；状态同步见[记录](verification/204-g4-plan-review.md)。G0/G1完成、G2功能交付、G3限定验收及后续4383运行已交付；原G2三条缩放已有G3后续通过证据并关联，国家容量待补。该次方案编制时G4/G5尚未实施；当前结果见上述实施记录，125其他容量P1保持。

- [302详情导航整改](plans/302-detail-navigation.md)：位置／匹配总数及重复读取已修复，45300已更新；三次诊断约3.8→0.9秒，非P95，实际结果与原失败见[实施记录](verification/302-detail-navigation.md)。

[125路径阶段目标采纳](verification/125-path-stage-target-adoption.md)：用户采纳20会话路径阶段1000ms、长期300ms；单会话300ms，原完整样本另存双判据复评，原报告及失败保持。125继续整改，300阶段关闭保持。

[125统一收口与116/131/133关闭](verification/125-roadmap-consolidation.md)：2026-11-15按用户决定关闭三计划执行，剩余容量/成本/最终交接统一转125；原目标和失败保留。300随后按[关闭记录](verification/300-closeout.md)完成阶段收尾；本批同步main带入200首批完成后，当前进行中计划仅125。

[125当前版本查询容量接续](verification/125-query-capacity-continuation.md)：查询整改、限定验证及资源收尾完成，三轮7200操作/9600HTTP零错误；最终4组达原目标、5组小幅超标按用户决定接受、1组达既有阶段目标，本批阶段关闭。路径2组和其他未覆盖范围继续由125持有，整体容量P1保持。

[125统一容量清单与129/133状态核对](verification/133-capacity-closeout-preparation.md)：129限定六项交付完成；133已核对33项去向和原摘要勘误，该次核对时整体仍进行中；现已按上条用户决定转125并关闭执行。本批只整理证据，下一轮范围见[125第8节](plans/125-r6-capacity-remediation.md#r125-unified-closeout)。

[固定4390已更新至合并模型5版本](verification/135-4390-model5-refresh.md)：原库17张组织表和132299来源保持，双角色菜单/历史及原回执通过。

2026-11-14 G3已实施并完成主代理复评及限定验收：3620条固定区划、独立存储/API/页面、维护历史、候选与锁内引用通过；DB/API分轮238根通过、6专用条件项跳过，浏览器298用例分轮通过。18组3600次零错误；用户将20会话目标统一为500ms且性能非阻断，实测最差394.726ms，单会话最差57.942ms，原300ms失败保留。详见[实施验收](verification/203-g3-implementation.md)。自有资源已回收，4383/4388未更新，G4/G5未实施。

2026-11-14[G3方案](plans/203-g3-geo-area-admission.md)已按用户要求评审完善通过：2项P1、3项P2在文档层关闭，发布门槛、历史字段权、完整条件导航、窄批发布与维护日期已明确。见[评审记录](verification/203-g3-plan-review.md)；该次方案评审时G3实施未开始，当前接续见实施记录。

2026-11-14按用户要求同步最新main并制定[G3必要行政区划接入方案](plans/203-g3-geo-area-admission.md)：CN3209/HK21/TW390，澳门保留基础地址；结构固定、目录维护、候选与历史引用、六批任务及验收已细化。同步与方案检查见[本批记录](verification/203-g3-plan-and-sync.md)。该次编制时G3产品未实施，4383/4388固定环境未更新。

2026-11-14来源资料精简版已更新到[4383](http://127.0.0.1:4383)，profile m02-current-c、源码eb6c3d38；原23张业务表保持，详见[运行验证](verification/200-country-resource-runtime.md)。

2026-11-14国家资料版本与来源区精简见[本批报告](verification/200-country-resource-simplification.md)：251条资料随Git保存，删除国家HTTP资源/页面的提供方来源区与source字段权；业务历史保持。该精简实施批当时未更新4383/4388；后续4383更新见上条。

2026-11-14按用户要求，c分支手工环境m02-current-c已更新并保持运行：[4383](http://127.0.0.1:4383)（API8383/DB54374）。251国家目录及双角色菜单核验通过，原15张业务表内容保持；运行构建为d98715b6对应源码，追加00008/00009。4388原环境保持，详见[默认端口验证](verification/200-g2-default-port-verification.md)。容量及原生缩放仍后补，行政区划与地点未实施。

[200 G0/T00](verification/200-g0-source-admission.md)已完成固定资料与范围准入；251国家、3620区域离线输入已生成，澳门基础地址范围保留；G0当时产品入库及页面未实施，后续G2交付与4383更新见本页当前记录。

2026-11-13下一批方案：[200国家地区行政区划与地点路线图](plans/200-geography-and-location-roadmap.md)、[201建设方案](specs/m02/201-geography-and-location-design.md)已完成[本轮方案评审](verification/200-geography-design-review.md)，收敛首批能力和最小公共适配。国家/地区参考本机CubeBox及Odoo；行政区划和地点限定中国大陆、香港、澳门、台湾。用户确定配置与主数据首次新建默认1900-01-01，见[202决定](adr/202-master-data-default-effective-date.md)。来源核对及文档检查见[本轮报告](verification/200-geography-plan.md)，[G1默认日期适配](verification/200-g1-default-date.md)已完成，企业初始化及BU/部门复用Go默认；[G2国家目录](verification/200-g2-country-catalog.md)已完成251条目录及共用维护，229个DB/API根与285条浏览器用例分批通过；2026-11-14用户明确本轮先交付G2功能，3条X11缩放后补、正式容量另行安排；原结果与目标保持；G3—G5未开始；该批未更新固定环境，2026-11-14的4383更新见上条。

[模型5与20条场景集成前验证](verification/135-model5-publication.md)：完整本地检查、继承性能提交及资源边界；远端三项CI以实际PR为准。

2026-11-14用户明确纠正单对象10000条生效记录的不合理验收场景，正式采用单个BU／部门各20条记录，并保留模型5（m02-compact-5／read-segment-4／query-seal /3，迁移00001—00008）。10000条专用保存P95／分页及其驱动的模型6改造退出当前任务，不再列为待修复容量项；原测量和失败仍保留为历史证据。20条是验收场景，不是产品硬上限，不缩减企业对象总量、完整来源历史或并发规模；原500ms目标及125其他容量P1保持。 实施与验证见[模型5及20条采纳记录](verification/135-model5-and-20-record-adoption.md)。本轮未提交、推送或更新固定4390。

2026-11-14本批收尾：135模型6隔离候选已实施并验证；按用户最后20记录范围，2412保存及全部回执重启恢复通过。完整输入读取、48.646倍合成空间及维护成本表明净收益不成立，不建议采纳；固定4390保持原模型5，125容量P1保持。首错、完整范围及资源收尾见[135执行记录](verification/135-real-head-product-integration.md)。工作区候选未提交、推送或部署。


2026-11-13本批普通部门冷态510.246→500.240ms，仍保留0.240ms数值失败；完整3000请求零HTTP/语义错误，3500回执重启及1104206原历史通过。真实头最终首/中/尾单次720.737/503.372/508.375ms，P95未建立，格式未采用，固定4390和125容量P1保持；见[本批记录](verification/133-cold-ordinary-and-head-proof.md)。

客户实库案例的组织明细由私有原仓保存。 [修复与交接](verification/133-property-rename.md) · [前批记录](verification/133-cold-save-and-storage-design.md)

2026-11-13前批覆盖预览：按用户要求整改BU覆盖变化预览；正式1200次零503、逐组P95≤300ms，覆盖不变6000次通过，详见[整改记录](verification/133-bu-coverage-preview.md)。合法通过800次、全部原历史核对、10份恢复及资源收尾已完成；原M3/M4失败不改写，共享绑定未采用，其他冷态保存及长线500ms目标、125容量P1保持，固定环境未更新。

2026-11-12原阶段决定：用户接受当前M2及长线补验的P95阶段成果，并要求本地提交；本阶段交付收尾，原始失败、单次/P95口径及125长期容量P1保留。当时列出的后续任务为M3条件项净收益评估、129的BU职责及M4最终交接，现已由本批报告接续，原决定见[阶段接受与任务清单](verification/133-long-save-overhead.md#6-用户阶段接受与后续任务)。本次不自动追加容量微调或更新固定环境。

独立 China HCM 产品：全新Go领域内核，React/TypeScript/MUI页面，保留CubeBox的体验、业务与验证资产。

当前M02业务单元已接通创建/查询、资料维护、停用/重新启用、删除、改日及固定启停期间维护。维护交互已统一：直接保存；按所选位置新增/插入；名称、日期和分类共用更正记录；首条前移改名后，后续资料按需独立新增/插入。当前实现与执行范围分别见[104统一更正](verification/104-unified-record-correction.md)、[105位置新增及补测](verification/105-positioned-addition-and-coverage.md)、[107首条维护](verification/107-first-record-maintenance.md)。[108多轮复核](verification/108-maintenance-consistency-review.md)记录此前残留修复及实际检查，不代表全M02、真实消费者或生产验收。

当前成对启停维护范围已按[109](plans/109-lifecycle-pair-maintenance.md)收敛为两个专用操作，通用写入口退出；当前验证见[109报告](verification/109-lifecycle-pair-maintenance.md)。旧104—108分别保留原批次结果。单条停用/重新启用继续保留，现按所选末条生效后状态开放，进入即带出日期及获权资料；补验见[单条启停一致性](verification/021-single-lifecycle-append.md)。

## 当前规则

[141 本地预检计划](plans/141-local-preflight.md)覆盖候选与实际用例结果、逐次证据保护及生成物核对。

[137 Actions 使用与验证规范](governance/137-actions-and-validation.md)统一触发、范围、门禁、失败补验和用量口径。[136 节流计划](plans/136-actions-cost-reduction.md)的首个窗口已按 10 次有效交付收口，结果见[终期成本观察](verification/136-actions-observation.md#6-窗口收口)。此前的[本地实施](verification/136-actions-implementation.md)和[方案验证](verification/136-actions-policy-validation.md)保留 61 项离线反例及 3944 分钟历史快照。

[128授权过程与关联跳转整改](adr/128-authorization-evidence-removal.md)：已明确删除逐次判权永久档案、解除终态依赖，并撤除详情关联快捷链接及其专属机制，保留当前权限与必要业务/安全事实；局部替代013/120及126完整证据路线。调查、删除清单、验收条件和固定库保护见正文及[128报告](verification/128-authorization-evidence-investigation.md)。[清理完整性评审](verification/128-cleanup-completeness-review.md)补齐工具、生成源、恢复与防误删边界。初始文档已提交b33df10；删除实施及全量万科验证已完成，压实后整库430.54MB、普通读取授权记录零增长；部门列表与全量父候选容量仍未通过，结果、原库保护和清理见[128实施记录](verification/128-implementation.md)。

[123普通维护日期调整](adr/123-current-maintenance-and-dated-query-boundary.md)：今日列表、未来首条可发现、按记录详情、跨日选择/导出与部门路径合同；实际检查按123报告记录。

- [文档管理 CHNHCM-014](governance/014-document-management.md) · [模板](templates/README.md) · [唯一登记表](governance/registry.json)
- [固定分支与编号 CHNHCM-005](governance/005-branches-and-documents.md) · [目录与依赖 CHNHCM-004](architecture/004-repository-layout.md)
- [技术选型 CHNHCM-002](adr/002-technology-stack.md) · [实施阶段 CHNHCM-003](adr/003-implementation-stages.md) · [全新实施 CHNHCM-025](adr/025-greenfield-implementation-policy.md)
- [日期格式 CHNHCM-026](adr/026-date-display-and-input-format.md)：完整日期统一YYYY-MM-DD，记录时刻保留北京时间及秒精度。
- [中文命名 CHNHCM-015](governance/015-terminology-naming.md) · [大陆组织角色 CHNHCM-009](adr/009-mainland-organization-terms.md) · [术语表](architecture/target/17-terminology-glossary.md)
- [来源追溯 CHNHCM-018](adr/018-effective-records-and-source-history.md) · [删除/改日 CHNHCM-020](adr/020-effective-record-delete-and-redate.md) · [生命周期 CHNHCM-021](adr/021-bu-lifecycle-and-record-maintenance.md)
- [原联合决定 CHNHCM-022（部分替代）](adr/022-joint-timeline-revision.md) · [实现与验证收敛 CHNHCM-024](adr/024-timeline-implementation-simplification.md)
- [Community采用 CHNHCM-113](adr/113-bu-list-community-adoption.md) · [列表剩余规则 CHNHCM-110](adr/110-bu-list-and-datagrid.md) · [改造计划 CHNHCM-111](plans/111-bu-list-experience-upgrade.md)：已完成Community T01/T02接线及定向检查，采用部门样板并取消列拖动/固定；T03真实业务及七组条件已验，见[报告](verification/111-community-t03.md)；[112原Pro源码路线](adr/112-datagrid-pro-source-integration.md)已被替代。
- [直接保存 CHNHCM-101](adr/101-direct-save-and-form-assistance.md) · [位置新增/插入 CHNHCM-103](adr/103-positioned-record-addition.md) · [首条通用维护 CHNHCM-107](adr/107-first-record-maintenance.md)
- [China HCM目标设计基线](architecture/target/README.md) · [资产保留 CHNHCM-006](assets/006-preservation-contract.md)

113已接入Community DataGrid8.29.3原字节源码及同版声明，完成T01/T02-A/B/C；采用部门样板、摘要列、原生列设置，保留A/D、名称链接、20/50/100与#066b68，取消列拖动/固定。实际类型/构建及组件/合成HTTP检查见[111实施报告](verification/111-community-t01-t02.md)，T03真实Go/API/DB与七组条件另见[新报告](verification/111-community-t03.md)，原T04评估已完成并维持当批查询，列条件原为后置，见[评估记录](verification/111-column-filter-evaluation.md)；T05限定交付已完成，见[交付报告](verification/111-community-t05-delivery.md)。旧Pro及113文档报告保留原结果；用户随后采纳单列条件与应用/取消，见本次增量报告，Excel/跨列组合仍后置。

## 当前交付与规格

[116共用按日维护路线图](plans/116-m02-shared-dated-maintenance-roadmap.md)：R0—R5及R6标准/实测交接已交付；2026-11-15按用户决定完成执行关闭，R6-P01/P02及P03剩余容量/成本由125统一接收。

[131部门列表与父候选查询减负计划](plans/131-department-query-reduction.md)：T00—T03及T05阶段交接已完成；2026-11-15按用户决定完成执行关闭，T04剩余容量/消费者/成本及最终汇总转125。原[R153/T05交接](verification/131-t04-capacity-and-handoff.md#41-本次t05阶段交接结论)、首失败、R65仅记录与后置范围保持，当前移交见[登记记录](verification/125-roadmap-consolidation.md)。

[131 T02选择与导出](verification/131-t02-selection-export.md)已完成V07—V10/V15及主代理问题修复复核：完整smoke272项、DB/API分包503项/五默认专项另验、万科87343行八列逐值及132290原来源保护通过。首失败、resolve预算5→10秒调整和全量成本单列；T04/T05、125容量P1保持，固定环境未切换或重启。

[131 T03父候选期间收敛](verification/131-t03-parent-candidates.md)：V11—V14及主代理复核通过；516项分包DB/API、五专项、完整smoke274项、完整万科12项候选/冷暖正常菜单和132290原来源重启保护通过；实际三片浏览器并发、资源已回收。首失败与约5秒全量成本单列，T04/T05及125容量P1保持。

[130部门存储优化方案](plans/130-department-storage-optimization.md)：三轮主代理文档评审通过，先回收无引用字典、合并相邻等内容读取段、由主键承接两项覆盖索引；保留历史、完整身份和当前授权。原方案评审见[130记录](verification/130-department-storage-plan-review.md)；用户要求的模型4/read-segment-3实施、两形状空间/增长/完整历史及规定容量验证已执行，功能与空间通过、容量超标分项保留，实际结果见[实施记录](verification/130-implementation.md)，125容量P1及固定基准保持。

[301用户提示与异常恢复方案](plans/301-user-feedback-and-recovery.md)承接38组问题及2项相邻表达，形成8项处理方向和17项验收条件；[四轮主代理评审](verification/301-user-feedback-plan-review.md)关闭3项P1、4项P2，方案通过；共享保存、查询恢复、错误分类及中文表达已完成实施；17项限定验收、315项浏览器分批覆盖、45300更新及资源交接见[实施记录](verification/301-user-feedback-implementation.md)。保存结果、筛选交集、局部重试和权限按真实状态处理；原排查及容量问题保持。

[300部门列表左侧部门树实施计划](plans/300-department-tree-implementation.md)已完成限定阶段关闭：完整万科输入、6份CSV、800浏览器和全部132290来源保护已补齐；60组正式12000请求零错误，41组达原目标、10组阶段接受。9组明显超标及36共用待验、原合成容量和历史写入失败转125统一收口；详见[关闭记录](verification/300-closeout.md)，原失败与目标保留。

[133过度防御与高成本处理整改路线图](plans/133-overdefensive-processing-remediation.md#r133-roadmap)：M0—M3采用项、129六项、33项处置及M4阶段交接已交付；2026-11-15按用户决定完成执行关闭，G3剩余容量及G4/T07最终汇总转125统一收口。各批[历史核对](verification/133-capacity-closeout-preparation.md)和原实测保持，当前移交见[登记记录](verification/125-roadmap-consolidation.md)。

2026-11-14列表返回定位：主动查询、分页及筛选结束旧定位；同步最新main后沿用无对象缺失提示、焦点回标题的行为。[同步前记录](verification/133-list-return-notice.md)保留原范围，当前对齐与检查见[同步记录](verification/133-list-return-main-sync.md)。

[129业务单元去过度设计与部门对齐计划](plans/129-bu-simplification-and-department-alignment.md)：关系处理、预览、内部类型、固定启停包装、原因默认及未实现采用入口的限定六项已完成；16组历史验收与后续修复已[核对交接](verification/133-capacity-closeout-preparation.md)，执行状态completed。容量遗留由125统一持有，原[编制核对](verification/129-bu-simplification-plan.md)保留当时范围。

[127万科全量组织及历史验证](plans/127-vanke-full-history-verification.md)在126本地提交后独立执行；全部132302行已处理，87344个组织及132284个导入来源、全部冻结路径独立核对通过。32路完成历史续跑；普通页面通过，20会话并发部门列表失败，容量P1保持。含审计/来源/索引的完整库导入整理后1334.75MB，81次验证写入后最终1399.38MB；用户随后要求固定保留万科库，已以64路重建持久profile m02-vanke-validation-b并核对全部历史，普通入口及重启通过，使用后整库1328.61MB，页面[4388](http://127.0.0.1:4388)；原名、扩展字段、两轮环境及首失败边界见[127记录](verification/127-vanke-full-history.md)。

[125容量整改](plans/125-r6-capacity-remediation.md)已获用户批准实施；实际基线、分批检查和容量进展见[实施记录](verification/125-implementation.md)。原三项方案评审修订保留，容量P1尚未关闭。

部门已从“设置→组织配置→部门”接通真实维护、关系/路径、日志及选择导出，与BU共用办理流程。R5限定验收及修复复验见[R5报告](verification/116-r5-department-pages-and-acceptance.md)；R6标准和容量实测已交付，容量P1仍未关闭，整改方案见[125](plans/125-r6-capacity-remediation.md)。

[122实施计划](plans/122-bu-operation-summary-and-change-log.md)已获批准并接通记录摘要、身份快照及分组日志，实际检查与资源结果见[122实施报告](verification/122-operation-summary-implementation.md)。116后续阶段及部门不在本批范围。当前权限缓存、焦点与S/P/F删除补验见[122评审修复](verification/122-operation-summary-review-fixes.md)。

[122在123后的复核](verification/122-post123-contract-review.md)补齐删除原件操作者归因和未来首条空态，新增的是既有案例参数；合同阶段20例未运行；本轮执行另见122实施报告，原报告和123产品证据保持各自边界。

- [M02边界 CHNHCM-007](architecture/007-m02-entry.md) · [本版范围 CHNHCM-010](plans/010-m02-scope-and-plan.md)
- [业务需求 CHNHCM-011](specs/m02/011-m02-business-requirements.md) · [功能交互 CHNHCM-012](specs/m02/012-m02-functional-and-interaction-design.md) · [技术合同 CHNHCM-013](specs/m02/013-m02-technical-design.md)
- [规格入口](specs/README.md) · [时间轴测试设计 CHNHCM-023](specs/m02/023-m02-timeline-test-cases.md)：394个稳定编号、350现行、44退役；110/111新增12例的当前执行见[111 T03](verification/111-community-t03.md)；逐批结果与未运行参数分开记录，退役不计通过。
- [页面优化 CHNHCM-100](plans/100-ui-experience-improvement.md) · [日志与表单批次 CHNHCM-102](plans/102-bu-maintenance-usability.md)：保留原批次范围，后续入口由104/107收敛。
- [统一更正实施 CHNHCM-104](plans/104-unified-record-correction.md) · [位置新增及补测 CHNHCM-105](plans/105-positioned-addition-and-coverage.md)：均已完成各自记录的实施及验证范围。
- [维护改造一致性复核 CHNHCM-108](plans/108-maintenance-consistency-review.md)
- [本地运行 CHNHCM-017](guides/017-local-bu-development.md) · [并行开发/其他主机与万科来源共享 CHNHCM-134](guides/134-parallel-development-baseline.md) · [GitHub CI](../.github/workflows/README.md)
- [Docker与临时验收环境](verification/017-docker-runtime-lifecycle.md)：make smoke默认新建并回收独占临时环境；持久开发环境复用和停机保留规则见017。

完整查询结果选择及CSV导出所选已形成[114决定](adr/114-bu-query-selection-and-export.md)与[115计划](plans/115-bu-query-selection-and-export.md)：全选覆盖全部分页的当前获权结果，分页保持、范围变化重选。已实施并完成限定分层验证，见[115实施报告](verification/115-query-selection-implementation.md)；不计入111原交付。

后续目标与路线见[116 M02共用按日维护实现路线图](plans/116-m02-shared-dated-maintenance-roadmap.md)：先让BU接入提炼后的实现，再以真实部门检验共用页面、时间线、可靠提交和接入标准；BU与部门的身份、记录、来源、审计、Outbox及提交结果均使用独立物理表。R0合同准备已完成，见[117决定](adr/117-m02-dated-maintenance-and-department-boundary.md)、[118—121规格入口](specs/README.md)和[实际报告](verification/116-r0-contract-preparation.md)：单BU多部门、部门必填单值归属、最小行政上级及路径、新分类目标和分类型恢复均已明确；R1共用Go内核/BU适配及117目录已完成，见[R1报告](verification/116-r1-kernel-and-directory.md)；R2事务/恢复提炼已完成并复核通过，见[R2记录](verification/116-r2-transaction-and-recovery.md)；R3前端流程与BU接入已完成，R3-09 P2已修复并通过定向复核，R4部门后端已有本次实施证据，R5部门页面与双对象限定验收已完成并复核通过，状态读权及根部门创建评审修复见[首轮修复](verification/116-r5-review-fixes.md)，后续关系候选及日志筛选授权复修见[本轮记录](verification/116-r5-authorization-followup.md)，实际范围见[R5报告](verification/116-r5-department-pages-and-acceptance.md)；R6标准与容量实测已交付，容量未通过，见[R6报告](verification/116-r6-standards-and-capacity.md)，109范围保持。当前BU页面及CSV已将“业务状态”更名为“状态”，内涵不变。 后续[授权与层级合同修订](verification/116-r0-authorization-and-hierarchy-review-fixes.md)补齐未生效/停用授权、有效父边与历史引用及路径读取，追加20个独立预期；原设计阶段产品未运行记录保持，当前R4结果见本次实施报告。

## 验证与历史批次

- [验证记录入口](verification/README.md)：报告持有受测基线、实际结果与限制，旧通过不自动代表当前实现。
- [T07本版综合验收](verification/010-t07-acceptance.md)：原257编号基线为241整例通过、3例仅本域参数通过、9例消费者前置未接、4例退役；不覆盖后续新增或修订参数。
- [T04启停](verification/021-lifecycle-implementation.md)及[评审修复](verification/021-lifecycle-review-fixes.md)；[T05删除](verification/020-delete-implementation.md)及[评审修复](verification/020-delete-review-fixes.md)；[T06改日复审](verification/020-redate-independent-review.md)；[原T08联合修订](verification/022-joint-implementation.md)。这些报告保留原入口及原时点，当前步骤见012/017。
- [共享体验迁移](verification/019-cubebox-experience-migration.md)及[图面对照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/019-ui/index.html)；[100页面交付](verification/100-ui-experience-improvement.md)及[优化前后图面](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/index.html)。
- [018多轮复测](verification/018-multiround-review.md)和[018旧来源设施退役验证](verification/018-source-history-simplification.md)保留原范围，旧升级不是当前运行前置。

## 本地运行

2026-11-12的[4390模型5更新](verification/131-4390-model5-refresh.md)采用固定构建b82a102、profile `m02-current-vanke5-b`，API8390、DB54392，模型m02-compact-5/read-segment-4/query-seal /3。完整数据保护、菜单及重启结果由该报告持有，当前操作和旧环境保留边界统一见[017第12节](guides/017-local-bu-development.md#12-当前模型4390基准)；[原模型4记录](verification/130-4390-model4-refresh.md)保留，125容量P1仍开放。

环境为Node24.19.0、pnpm10.24.0、Go1.26.0、Python3、Docker/Compose。原4387紧凑案例和4388万科案例保留原程序、数据库及历史证据，作为旧批次对照，不再作为日常验证基准；不得用当前程序接入旧模型。在线状态以选定profile的verify为准，完整操作和版本边界见[017](guides/017-local-bu-development.md)。

```bash
cd /home/vivobook/projects/CHNHCM-wt-dev-b
export CHNHCM_LOCAL_PROFILE=m02-current-vanke5-b
python3 scripts/local-bu.py verify
```

本地公开开发账号为admin、reader，密码均为admin1234567。数据库维护凭据仍随机产生并只存仓外0600文件。旧批次 4416/4436/4466/4476/4496 和曾使用 4486 的历史专项保留原数据。当前 4486 由本树的综合库提供，实际连接信息可用 `python3 scripts/local-bu.py connection` 查看。

## 来源与历史资产

[建仓记录 CHNHCM-001](plans/001-repository-bootstrap.md) · [早期编写计划 CHNHCM-008](plans/008-next-steps.md) · [共享体验计划 CHNHCM-019](plans/019-cubebox-experience-migration.md) · [来源边界](reference/README.md) · [本仓设计修订链](reference/local-revisions.json) · [未采用引用](reference/not-imported.md)。来源原件和定稿报告保留原结果；当前规范由规格及已采纳决策持有。

远程配置见登记表。Git提交、CI及仓库发布不等于应用部署或外部业务接入。

## 待评审草案

- [132目标决定](adr/132-query-and-validation-boundary.md)：第7节已冻结M0主线具体合同；其余条件项继续按项冻结，路线图及后续交付由133持有，125容量P1保持。

- [126完整历史紧凑存储研究](plans/126-compact-history-storage-research.md)：覆盖现案例历史、审计和索引的隔离模型约191.5 MB；[实测与限制](verification/126-storage-feasibility.md)不等于产品迁移或容量P1关闭。

- [M04同日人事动作、日级状态与统计口径 CHNHCM-106](adr/106-same-day-personnel-actions-and-daily-state.md)：仅为提案，未批准或实施，不改变现行M02合同。

- [不必要限制清理记录](verification/111-defensive-design-cleanup.md)：构建与本地运行解耦、来源核对改显式、开放纯展示列设置及独立录入，保留必要业务保护；按实际检查范围读取。

单列条件与统一应用/取消增量见[本次实施验收](verification/111-single-column-filter.md)；T04原评估与T03原报告保留原证据，最终执行范围按新报告记录。

此前列表交付记录：[111 T05](verification/111-community-t05-delivery.md)。当时页面为[4382](http://127.0.0.1:4382)，profile为m02-current-b；2026-11-03已核对前后端并从正常菜单完成名称包含丙查询，在线状态以017的verify及实际业务请求为准。

R3及R3-09修复已完成；原两批实施与复核见[实施记录](verification/116-r3-frontend-workflow.md)；单轮187项Chromium及21组视口通过，部门真实接入仍后置。

2026-11-05 R3-09评审修复：已采纳退出失败后新草稿保护缺失的P2意见，移除跨动作持续放行。修复前反例重现，修复后12+7项定向Chromium通过，check及前端构建通过；原187项、DB及21组视口保留修复前证据，本轮未重跑全套。实际范围见[修复记录](verification/116-r3-review-fixes.md)。

2026-11-05 R4设计细化：已按用户采纳意见补齐四包退出条件、部门存储/端口、授权/锁序/候选及路径证据，并映射39例和16组技术参数；见[范围评审](verification/116-r4-contract-review.md)。本轮仅设计，R4产品、R5菜单和R6容量均未开始。

2026-11-05 R4关系版本简化见[限定范围复核](verification/116-r4-relation-simplification.md)；其后[评审修订](verification/116-r4-review-fixes.md)补齐拟分类查询和授权诊断顺序。该设计阶段验收映射为39例及24组参数，上轮三项问题完成设计修订；原设计产品not_run记录保持，本次实施结果另列如下。

2026-11-05 R4后端实施：设计基线已提交为5fa3275，部门独立存储、全维护API、路径及共同守卫已接线；分批验证与R5交接见[本轮实施记录](verification/116-r4-department-backend.md)。最终完整DB及浏览器修复复验通过；全量浏览器189通过/1失败和最终16项定向通过分别保留，R5菜单和R6容量另验。

2026-11-05当前4382环境切至m02-department-b，含1个验证BU和5个真实部门样本，管理员及查询者登录/API读取已核验；入口、历史链接及旧库保留见[本地部门样本](verification/116-r4-local-department-samples.md)。部门正常菜单仍待R5。

R4实施评审的P1删除摘要日期读权及P2管理BU稳定ID筛选已采纳修复，新增反例、完整回归一次守卫耗时失败及定向复验见[实施评审修复](verification/116-r4-implementation-review-fixes.md)；原实施及样本记录保留。

- [R4守卫预算诊断与收尾复核](verification/116-r4-guard-budget-closeout.md)：墙钟回拨复现、Go取锁预算修复；最终单轮DB/API 477项通过、4专项跳过，旧证据和4382样本保持，R4后端可交接R5。

2026-11-06 R5完成六轮评审与修复，关闭1项P1及5项P2（含修复回归和布局验证缺口）；最后两轮32项、18项分别通过，无新增主要问题。完整DB/API的外层异常和单项耗时失败、随后3次定向通过分别保留，不称单轮全绿。实际结果、当前输入和资源收尾见[多轮报告](verification/116-r5-multiround-review.md)，R6仍未开始。

R6接入标准、容量实测与交接已交付；容量P1整改未关闭：[124操作指南](guides/124-dated-object-admission-and-capacity.md)、[本批记录](verification/116-r6-standards-and-capacity.md)。

当前126实施及验证：[紧凑历史实施报告](verification/126-compact-history-implementation.md)。完整案例静态整理276.22MB，真实读写后普通维护311.02MB；200MB已按用户决定改为参考值，部分延迟未达，容量P1保持。

125恢复容量整改的当前执行见[恢复记录](verification/125-capacity-resumption.md)，125原交付及原4390基准采用m02-compact-3，130原实施另用模型4候选验证；用户随后要求更新4390，当前模型4入口见[更新记录](verification/130-4390-model4-refresh.md)，原库及原证据保留。

- [307 M03职务基础实施与验证](verification/307-m03-job-foundation-implementation.md)：首批限定实施已完成；页面整改、发布及4380接续见308记录。
