# CHNHCM-116 R4部门后端实施与分批复核

登记信息：[registry.json](../governance/registry.json)。对应116、118—121、023、122及123；本记录由wt-dev-b主代理实施和复核，不冒充独立评审或人工签收。

## 1 实施范围与基线

用户要求先提交设计再完成三批R4任务。设计已本地提交为5fa32750ebaf154fd476ed26dd4a983c11a39ef5；本次产品增量在固定wt-dev-b工作树实施，尚未另行提交；未推送、PR、合并或部署。起始4574个文件摘要见[baseline.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-backend/baseline.json)。原R0—R3、122及设计评审记录保持，本次另存执行结果。

四包责任为A独立存储/API、B最小行政关系、C路径证据、D共同守卫；按用户要求组成以下三批交付。第一次可用创建就包含关系资格、当前授权、完整来源证据、BU反向守卫和原子提交，没有以缺少保护的公开写入口作为阶段完成。Go承担日期、生命周期、权限编排和最终关系候选，数据库只承担结构、锁与原子性。

本批是部门后端实际接入。正常入口仍按119规划的“设置 → 组织配置 → 部门”；R4从对应真实API执行，R5才交付菜单/页面。没有组织调整方案、整树版本/发布/审批、外域消费者或生产交付。

## 2 第一批创建闭环

角色使用真实X03管理员、查询者及明确范围/字段Grant。前置为当前模型隔离库、有效企业和合法BU；创建根部门及子部门，检查编码身份、单值BU、完整启用段的上级同BU/覆盖/无环、根位于企业下、未来首条和当次来源路径。详情、当前列表、显式历史与本人查回均已连通。

实际实现为organization/domain/department、application/department、专属public DTO及postgres适配；迁移00005建立17张独立部门/关系/路径表，sqlc使用固定查询，bootstrap接入全部19条部门路径、20个操作。部门修订采用自己准确的关系成员，不增加第二套可写父边或共享BU业务表。当前模型为m02-department-1及迁移[1,2,3,4,5]。

主代理复核：first-chain-attempt3、api-ports-attempt1及fixed-d01-d18-attempt1通过；首次JOIN USING歧义、未来首条错误判定分别在前两次日志保留。首批已覆盖保存失败整笔回滚、无对象读权安全拒绝、最小引用不授完整详情及当前/旧源分离。存储封口、复合外键、跨对象与跨企业引用在第三批扩大验证。

## 3 第二批维护及投影

全部标准动作已接通：位置新增/插入、名称/简称/BU/父更正、日期/分类组合更正、分类删除、改日、单条启停和固定插入/删除停用期间。共用dated内核经Prepare保留候选，关系差异获权后Plan/Bind，再Finalize消除冗余并更新继承/影响；BU也使用同一流程。固定两条一次提交，后继独立资料、首条保护、删除墓碑和准确原件保持。

BU和上级部门的首条/普通改日、停用与恢复改日、删除恢复、固定插入/删除、追加停用、分类更正均按最终候选反查下级；18组BU/父动作分别执行。历史引用不会永久阻止父部门后来转BU，当前停用下级的未来恢复仍受保护。拒绝不自动改写另一对象。

路径覆盖当前启用、未来首条安排、停用日前一日及详情期间分段。祖先更名不推进子部门头；显式来源包括显式当前头都读封存证据，损坏证据失败关闭。可见路径变化使选择版本改变，受限路径返回固定投影，不泄露节点、段数、日期或隐藏祖先变动。137行全选/排除与八列CSV在真实API完成，字段和导出权限重新核验后一次输出。

基本资料摘要无需department.history.read。所选修订按真正形成的操作归因，派生修订归派生操作；S1—S7、P1—P4、F1—F3分别核对两类删除原件、配对独立归因、无今日事实的未来首条、失效选择、空变化及同日新身份。H45明确把英文差异映射shortName：45组、15组简称变化、操作者23/22、分页20/20/5及固定上界均核对。没有用对象头或祖先操作者替代记录摘要。

主代理复核通过：maintenance、API辅助、原39例对应测试、137行选择导出、reverse动作修复复验、S/P/F/H和完整Grant/路径隐私定向检查。日志筛选及全查询选择的纯机制由BU与Department实际共用；字段目录、授权、DTO、SQL和结果表独立。详细测试名称及未测层见[实际映射](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-backend/acceptance-map.json)。

## 4 第三批竞争故障与回归

真实多连接验证空集合新增部门与BU停用的两种顺序、A挂B/B挂A至少一方拒绝、跨BU固定锁序、依赖发现后新增/变化的重规划，以及撤权先行/写入先行。请求在BU守卫上跨午夜等待，锁后仅采一次执行日；当前授权归属不因所选旧记录或旧来源改变。

写锁顺序为X03共享守卫、对象类型提交锁、企业共享锁、排序BU头、排序BU行政关系守卫、排序部门/关系头及创建编码锁。R4写请求保守取得相关BU关系独占守卫；没有全企业独占锁或整树版本。普通读用一致MVCC快照。完整当前森林读取及该守卫粒度尚无R6容量结论。

故障检查包括14个配对持久化写点、创建/来源头写点、X03身份快照失败、回调后取消/截止时间、读池/查回池预算，以及真实TCP层COMMIT发送/应答丢失。accepted、配对、no_change、rejected、not_applied分别查回；无确认提交不交付假终态。跨类型同提交号、原内容重试、异内容拒绝、先resolve封闭、撤写权仍可获权读结果、失读权及损坏诊断安全拒绝均核对。

存储检查独立触发精确复合外键、区间排斥、继承/前驱、终态组合、重复facet、不可变更新、源封口后追加及Outbox载荷保护。测试中仅在回滚事务内暂时关闭不可变触发器以直接验证其他约束，不提交弱化结构；跨企业样本使用真实第二实例/企业。Outbox允许只更新delivered_at，不能改载荷。

最终完整DB两包通过：149个顶层测试、含子测试469项通过；4个显式负载/附加浏览器通道跳过，未计通过。另补1项空姓名人类身份真实API测试通过。完整浏览器190项中189通过、1项失败；快速导航独立反例、URL归属保护及原权限用例等待不足均已处理，最终16项定向复验全部通过；不记整轮全绿。当前已完成多轮定向复验、一次完整中间DB运行和check/build。完整中间DB是修订前输入，见integration-interim-inputs与integration-interim.jsonl，不能替代最终输入运行。首轮完整DB失败和修复证据独立保留。

| 最终检查 | 实际结果与输入边界 |
| --- | --- |
| make generate / check / build | 通过；生成后核对原BU 21条路径和54类型语义不变，完整YAML整理前后等价；最后前端修复后的check/build分别留证 |
| 完整integration | 两包退出0，149个顶层测试、含子测试469项通过；4个显式负载/附加浏览器通道跳过。另1项空姓名身份补验通过，不冒充在完整运行内 |
| 全量BU Chromium | 190项，189通过/1失败；[原日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-backend/browser-full.txt)与[失败截图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-backend/browser-first-failure.png)保留 |
| 浏览器修复复验 | 原用例3次重跑通过不能消除发现；确定性快速导航在修复前失败；中间两次16项分别15/1、14/2；诊断运行3+8项通过单独保留；最终[16项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-backend/browser-tab-fixed.txt)全部通过，包含原权限、快速切页签、在途日志、键盘及S/P/F原件，两次重复，无自动重试 |
| 文档 / diff | 最后登记完成后运行make docs及git diff --check，见最终日志和机器汇总 |

主代理三批及A/B/C/D责任包的本次限定范围复核通过，可交接R5。机器结果见[verification-summary.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-backend/verification-summary.json)。DB完成后113个生产Go/SQL输入未变，之后仅有等价OpenAPI格式整理、单独身份补验和上述BU前端修复/浏览器断言；完整浏览器后发生的部门专属变更已由最终DB覆盖，前端修复已由定向浏览器覆盖。完整运行、补验与最终文件之间的差异见[输入比较](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-backend/run-input-comparison.json)，没有把旧运行冒充最终全文件单轮通过。


## 5 发现与修复

| 发现 | 修复与复验边界 |
| --- | --- |
| 初始sqlc查询模板和JOIN归属歧义 | 改固定SQL后重新生成；存储/创建链路复验通过，first-failures及first-chain前两次日志保留 |
| 未来首条错误返回无记录 | 使用真实首条投影，仍返回M02_NO_FACT及未生效状态；创建链路、D20和F参数通过 |
| 部门Access证据扩展误改BU purpose/版本，旧结果查回503 | BU保留BU_BACKOFFICE及原算法/目录版本，部门独立；bu-recovery-fix及完整回归覆盖 |
| 引用查询要求多余写权、配对结果缺准确前后记录、路径证据不足校验 | 查询按read/上下文字段/最小引用裁剪；配对按准确源补结果；验证完整分段/节点/边；ports-review、V12及P参数通过 |
| 预算错误码与过期HTTP连接影响故障用例 | 变化日志超时统一M02_EXECUTION_LIMIT，查回仍pending；测试每次重新建连接；pool-budget-retest通过 |
| Outbox更新触发器在BEFORE阶段比较尚未重算生成列 | 改AFTER保护载荷，允许delivered_at更新；storage-summary-attempt2及storage-final-targeted通过 |
| 新增缺失/空changes、失效选择、日志页码超界偏离已有合同 | 空ADD完整校验后no_change；显式缺失selectionMissing；末页裁剪；H45/P/F及S7复验通过 |
| 受限日期上下文输出null，无法区分无权与开放终点 | 失权省略，获权开放终点和空简称明确null；H45及identity-empty-name通过 |
| 既有BU快速切页签后返回当前资料，旧渲染参数覆盖新页签（P2） | 全量190项中该项失败；原用例3次重跑通过但不据此忽略，新增确定性时序在修复前稳定失败。returnCurrent改读最新URL；首次16项复验仍有1项原失败，进一步把日志成功/失败响应绑定发起时的实际URL并在上下文变化时重读，防止旧响应写回旧页签/历史/筛选。原用例仅等待不变的日志条数，仍可能在刷新完成前操作；改为等待当次历史详情200和日志ready，再核页签及字段，业务断言保持。各次失败和诊断运行分别保留，最终定向结果另记 |
| 测试夹具与断言错误 | 默认Grant重复、主体状态值、SQL生成列插入、父资格错误码、配对items属性及类型投影编译错误分别修正；原失败日志保留，不当作产品失败或通过 |

storage-summary-attempt1因断言失败后的测试事务未关闭而挂起，已终止本任务进程和对应临时容器，增加defer Rollback后复跑；该次记中断，不计通过。生成客户端首次重复enum导致检查失败、首次check缺报告导航也已修复并重跑。没有删掉或覆盖首次失败日志。

本轮复核范围内已发现的问题均已按对应修复和定向复验闭合；分层结论以第4节为准，不据此声明最终单轮全量浏览器通过。代理自评不代表另一个评审人批准。

## 6 R5接口与受测边界交接

| 用户任务 | 已接真实端口/数据 | R5应完成的页面行为 |
| --- | --- | --- |
| 进入列表/创建 | GET context、operation-reasons、departments；POST departments | 正常菜单、字段直接可见、日期/分类目录、创建后定位真实首条 |
| 选记录/基本资料 | GET departments/{id}，selectedRecordId及可选sourceVersion；selectionMissing、capabilities、operationSummary、path | 跟随记录显示；摘要独立日志权；失效选择不自动换条；当前与旧源、单段/多段及停用前路径表达 |
| 普通/启停维护 | records、correct、delete、redate、deactivate、reactivate、lifecycle-pairs及其delete | 复用R3提交/恢复/离开保护；按能力直接编辑，展示Go字段错误，无强制人工预览 |
| 关系与自动辅助 | references/business-units、references/parents；完整命令preview | 原因尚未选可查候选；拟分类/日期/BU变化刷新；区分applicable与写权限，防止迟到响应覆盖 |
| 日志及准确链接 | GET changes，单列filter、contextSourceVersion、throughSourceVersion、detailReadFields | 表格筛选、固定分页窗口、按beforeRef/afterRef跳转当次资料或删除前原件；失权清正文 |
| 全查询选择/CSV | selection/resolve、export；当前D与授权投影版本、八列CSV | 跨页排除、排序/页大小保持、集合变化重选、完整下载、返回高亮独立 |
| 结果未明 | submissions/{id}/resolve；本类型/本人终态及准确items | 使用部门命名空间和类型适配器，失读权保留必要引用、清理正文；不猜测未保存 |

类型合同在api/openapi.yaml、public/department.go和生成schema.d.ts；apps/web/test-fixtures/department-types.ts验证部门独立类型和禁止BU/伪造授权输入。R5只需消费这些真实端口，不建设临时mock业务链路。接口索引见[api-handoff.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-backend/api-handoff.json)。

R5菜单、关系控件、键盘/焦点、撤权页面清理、迟到查询、视口及真实缩放仍not_run；本轮BU浏览器回归不计部门页面通过。R6的20000部门、深度100、2000扇出、部门H1000、20会话混合负载及p95/WAL/内存仍not_run。外域投递/采用、客户迁移、生产恢复及人工验收保持原后置边界。

## 7 证据与资源收尾

证据位于本报告data/116-r4-backend；原39例与V21—V24手工预期只引用原文件和摘要，未用产品算法生成expected。逐测试日志是断言经过真实API/DB的记录，随机UUID由测试运行分配，不将静态映射表当作执行成功。

最终1369个运行/构建/测试输入与当前文件一致，见[最终清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-backend/final-runtime-inputs.json)；2967份既有验证文件摘要未变，见[历史保护](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-backend/historical-preservation.json)。8个smoke临时项目及日志中观察到的651个Testcontainers容器均无残留，相关卷/网络、临时端口和仓外临时配置已回收，见[资源检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-backend/resources.json)。既有三个持久PostgreSQL及原服务不升级、不清卷；本轮Testcontainers和smoke profile均在创建时定义为独占临时。当前m02-current-b保留旧模型及原程序，不将本次新程序接到旧库；后续R5需创建匹配m02-department-1的当前模型环境。未提供虚构的持久部门页面入口。
