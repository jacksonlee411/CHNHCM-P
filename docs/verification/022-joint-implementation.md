# CHNHCM-022 T08有限联合修订实施与独立评审

登记信息：[registry.json](../governance/registry.json)。依据：[010](../plans/010-m02-scope-and-plan.md)、[011](../specs/m02/011-m02-business-requirements.md)、[012](../specs/m02/012-m02-functional-and-interaction-design.md)、[013](../specs/m02/013-m02-technical-design.md)、[023](../specs/m02/023-m02-timeline-test-cases.md)和022/024/025。

## 1 对象与基线

2026-11-02，固定工作树CHNHCM-wt-dev-a、分支wt-dev-a；开始HEAD为e03269ee8f6f98c888b2e1a8fa29624104bd63bb，工作区干净、stash为空。受测对象是此基线与本轮差异，报告自己的提交不冒充受测基线。当前工具与隔离环境摘要见[环境记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/environment.json)。T08交付同一BU有限联合修订、首条历史补全、业务分类纠错，以及逐项权限、来源、诊断和整笔结果恢复。T07综合验收未启动。

[受测输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/tested-inputs.json)记录实际文件SHA；三轮独立评审的冻结清单和原始报告另行封存，前三轮产品按冻结清单验证；最后增加两个编辑器的重读互斥修复，以round7冻结为准，Go/API产品与round3保持。测试补充确定重读/三请求屏障、COMMIT确认丢失和目标资格，以及既有外观测试的静态采样等待。

## 2 环境与正常入口

最终profile为m02-t08-final，Web4497/API8497/DB54577，Compose项目chnhcm-bu-15cedb53c7c0，隔离空库安装00001/00002和m02-timeline-7。早期m02-t08-current的4496及schema6、旧T06的4486/schema5与其他历史环境保留，未对旧持久库执行改写后的迁移，未清空旧卷。七组视口另用新profile m02-t08-viewport（4507/8507/54587，项目chnhcm-bu-7a381840fd58），同为schema7空库，与82条回归分库并行，避免共享授权或登录变更。凭据只在仓外受保护目录，操作见[017](../guides/017-local-bu-development.md)。

角色为获联合入口及全部项目权限的组织管理员；权限收窄用例使用合成主体或受控修改本任务授权，finally恢复。正常菜单为“设置 → 组织配置 → 业务单元 → 列表 → BU详情 → 更多操作”。其中提供“联合修订时间线”“补全首条历史”“更正业务分类”。单条新增、更正、启停、删除、改日保留原入口及专用权限。

可复制编码和URL见[业务单元清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/business-units.json)。原始结果见[正常菜单旅程](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/menu-journey.json)、[TL250](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/TL250-menu.json)、[TL251](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/TL251-menu.json)及[视口样本](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/matrix/results.json)，包含实际BU编码、身份、提交及原/新来源，可在4497搜索验证。均为合成资料。F7/F9尾段的部分实现样本使用“终 / Zulu”，相当于023的Z（“新 / New”）；TL映射按实际字符串留证，只复用独立快照和期间规则，不声称逐字同一输入。

## 3 实现与业务结果

### 3.1 领域与原子保存

有限项统一为ADD/UPDATE/DELETE；UPDATE可合并名称、日期和分类修改面。同一基准来源按itemId规范化，最终集合统一检查固定首条、按日唯一、企业范围及完整状态链，不提交中间结果。新增取最终准确前序；原停用转明确资料取基准目标完整快照。独立后继不随前段更正传播，停用按最终前序继承，变化只为每记录形成一个合并修订。

保存整笔使用原事务、原提交锁和一个业务savepoint；业务拒绝回滚业务写入后在原锁保存安全诊断，技术异常整体回滚并保持待确认。一笔accepted只有一个来源、汇总审计及Outbox。直接变化项与派生期间/继承分开；全同值没有新业务事实，声明字段、版本和当前权限仍须核验。Outbox保留一份canonical impact，DB生成datemultirange和结构投影，未变化的间隙不被合并成外包范围。

### 3.2 权限、历史与恢复

X03明确联合入口、原/新分类、日期及显式字段的逐项守卫；缺省继承不伪造显式名称写入。M02通过X03公开合同读取原安全声明，核对终态request_items、facets、准确原后修订和诊断itemId；no_change日期绑定原请求或准确原修订。当前对象/历史/字段读权决定原结果是否可见；仅撤写权仍能读accepted/rejected/no_change，撤读权安全裁剪或拒绝。

联合终态顶层不伪造单个结果目标，按各itemId给原后结果。原身份重排重试、resolve及后续来源推进后仍返回原结果，不能把当前头冒充当时内容。客户端只持久保存待确认引用，刷新不保存业务正文；同页原内容重试与迟到响应保护沿用共有协议。办理日由Go原采样保持，只有ADD输出过去/当天/未来说明；今日状态由Go独立计算。

### 3.3 正常菜单代表旅程

| 旅程 | 前置与动作 | 实际结果与保护 |
| --- | --- | --- |
| F2整段启停 | 菜单创建1月甲/Alpha、6月乙/Beta、10月丙/Gamma；联合加3月停用和4月重新启用 | 空集合incomplete；仅S为invalid，无完整候选；补R后合法。3月停用、4月恢复甲，6/10月独立快照保持，旧来源连续启用；保存汇总、返回搜索/查看日期和刷新正确 |
| F7首条补全 | 6月首条、7月停用、8月恢复丙、10月终；前移到1月早期资料 | 保留/不保留无预选，未选禁存。保留才显式新增原日后继及原因；7月分别继承原日资料/早期资料，8/10月保持，旧源早期NO_FACT，首条身份固定 |
| 分类和日期 | 首条只改原因；普通改停用；TL143交换S/R类型及日期；TL144同时改日期/分类/部分名称 | 纯原因只有结构影响；停用继承准确；原S转资料的英文仍基准Alpha，不取修改后前序West。旧分类及原内容可按旧来源读回 |
| TL250连续旅程 | 单加S失败→联合S/R成功→单删S失败→成对删除成功 | 两拒绝不推进来源，两成功各一个来源；各阶段原身份重试/resolve、原来源均保持 |
| TL251连续旅程 | F7不保留补全，再联合改首条原因及S/R为CHANGE | 补全版7月停用甲，分类版全程启用，7月甲、8月丙、10月终独立；首条及后继身份保持 |
| 页面异常与恢复 | 201码点错误、撤日期/英文读权、迟到预览、来源冲突、丢保存响应 | 按原itemId聚焦；受限输入/候选/重试包清理，仍可读输入保留；重核全部项；刷新只按原引用读原多目标来源 |

## 4 独立评审与修复

用户明确授权的独立代理/root/t06_independent_review只读实现并运行自有反例，不参与产品实现。工具评审不冒充人工业务批准。

| 发现 | 原反例与影响 | 修复及原代理复验 |
| --- | --- | --- |
| R1 / P2 | 业务写后409未清预计算itemOutcomes，拒绝终态CHECK失败而变503。整体回滚无残留，但缺合同要求的原锁内rejected | 拒绝前清临时结果与项结果；原锁保存唯一安全拒绝，原反例及共有提交边界测试通过 |
| R2 / P2 | 损坏原声明/facets或把诊断改为外来itemId，resolve仍可能200 | 通过X03公开RecordedIntent绑定原声明，严格核验声明、facets、直接项及诊断位置；原损坏反例安全不可读，当前读权仍优先检查 |
| R3 / P2 | ADD itemId等于既有recordId，派生影响误归ADD | ADD候选内部命名空间与持久身份分开；Compare仅用真正目标/内部候选身份归属，原碰撞反例及永久测试通过 |
| R4 / P2 | no_change ADD/UPDATE的item_outcomes日期可被损坏为2099而resolve伪输出 | 绑定原ADD拟定日、UPDATE声明日或准确before修订；保存JSON不含响应派生说明；原代理原反例及永久测试通过 |

第一轮报告SHA为e7b206ef8677b648509722681fafb53ad325bdc64bbe42ef5b4c3ef73cbe1281；第二轮为e39db98b855b00578ae48b447b57e6c2cc5a2eb6514f94b60d221c2779d6c797。原件及第三轮复验见[独立评审封存](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/independent-originals.tar.gz)。第三轮当时关闭R1—R4；完整回归和必要项逐项核对随后确认R5重读竞态及R6同值定位，均修复并独立复验。最终独立评审已通过，R1—R6全部关闭，无未关闭P1/P2/P3，本阶段必要项剩余为空；正式原件见[最终独立结论](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/independent-final-review.json)，SHA256为1fcf7308b85c9cdfb56341003cbef3a12f5b259dbf27d8b07a08eb55b2a14fa3。

独立代理另验证未来唯一首条时“候选记录active、今日ACTIVE→NO_FACT”的区别，以及原POST/retry/resolve在三个独立session中的确定顺序。初次同session夹具串行于TouchSession、未到advisory锁，失败原件保留；改三session后观测两个真实锁等待者，拒绝先提交、回滚后retry先、回滚后resolve先三种终态均唯一。

主线完整回归另外发现R5：单条和联合编辑的显式重读期间仍可点预览，预览增加epoch可能丢弃重读响应并沿用旧basis，表现为偶发再次来源冲突。新增独立reloading及请求序号，重读期间禁用输入、预览、保存及重复重读；finally只清理本次请求状态，取消/卸载仍有迟到保护。两个既有用例用真实响应屏障确认重读完成前预览禁用，释放后使用新来源成功，2/2通过；原代理还指出转停用警告按钮的禁用遗漏；现同时限制两枚警告操作及两页原内容重试，并扩展UPDATE名称/分类警告的确定屏障，2/2通过。该原反例已由原代理复验关闭。本项由主线回归发现，区别于四项独立代码发现。

R6 / P2由独立代理按TL157正常菜单确认：查看日11月、手选首条做同值联合后，父页因项含所选recordId就清掉选择，回到后继。现仅实际accepted删除所选项时清选择，整笔no_change保持原选择。永久用例包含同值UPDATE和同值ADD，核对无新来源/ADD无ID、查看日与首条身份均保持；主线通过且原代理原v2反例独立关闭。原v1未触发React修改属夹具失败、v2将no_change的sourceVersion误当观察来源的单项观测均保留，独立复验以observedSourceVersion及实际head核验。

## 5 检查与TL证据

### 5.1 已完成检查

| 检查 | 结果及实际范围 | 证据 |
| --- | --- | --- |
| make generate | 当前sqlc、OpenAPI客户端生成通过 | [日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/generate-final.log) |
| make check | docs、boundaries、65个文档脚本用例、运行脚本测试、Go单测/vet、前端类型、差异检查通过；最后定稿另跑docs/diff | [日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/check-complete.log) |
| make build | Go/API/前端构建通过；前端保留既有chunk体积提示 | [日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/build-closeout.log) |
| make integration | round6完整DB/API通过，含全部新增永久专项；tests/integration 335.864s、bootstrap 146.959s | [末轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/integration-closeout.log)、[前轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/integration-final.log) |
| 规则与授权 | 手写F1/F2/F4/F5/F7/F9/F10规则、所有项排列、最多四成员独立转换表、六向分类权限通过 | [日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/rules-final.log) |
| 目标资格 | 跨BU、已删、旧修订、同笔ADD被当UPDATE/DELETE目标均整笔拒绝；业务表全文保持，只有唯一技术拒绝新增 | [日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/eligibility-passed.log) |
| 故障及确定竞争 | 九写阶段回滚、写后409安全拒绝、三session三种确定顺序通过 | [原子性](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/safety.log)、[提交边界](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/commit-fixes.log)、[三请求](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/three-request-fixes.log) |
| COMMIT确认丢失 | PG真实提交accepted后，受控替换返回为503/pending；原身份retry/resolve读取唯一原结果，无新事实 | [日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/commit-ack.log)；这是应用边界注入，未运行物理断链全矩阵 |
| 午夜菜单 | Go受控原办理日6/14、实际来源登记6/15，菜单ADD6/15仍为未来安排，原结果查回一致；未改主机时钟 | [日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/midnight-browser-final.log) |
| 代表预算 | 120000当前记录、两UPDATE、五次取消，最大停止71.273004ms，小于1秒；20成员两ADD累计SQL约10秒中止，13秒内含清理结束，实际锁释放 | [取消](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/cancellation.log)、[累计SQL](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/safety.log)；不算TL255/256全容量 |
| 默认桌面Chromium | 最终round7完整86/86通过，12.4m，实际退出码0；早期82/82和86轮85/86分别保留原结果 | [最终完整日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/smoke-complete.log)、[最终增量](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/reload-controls.log) |
| 七组视口/原生缩放 | 7/7通过，包含真实200%/400%、320×256、320宽；物理滚轮/Tab可达，400%实际保存；桌面及400%截图已实际查看 | [矩阵](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/matrix/results.json)、[日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/viewport.log) |

旧轮完整integration保留原日志；最终round6已纳入三请求/确认丢失/资格、负边界和死锁等全部永久补充，完整通过。R5重读及R6同值定位修复后已重新构建；round7与round6的产品字节完全相同，仅增加旧历史滚轮用例的稳定等待/正文落点修正。最后86条按round7的57项输入完整通过，最终摘要逐项匹配。

### 5.2 参数映射与过程失败

[TL参数清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/tl-map.json)逐项记录场景参数、实际层、测试函数及证据，含明确未覆盖部分。同一TL有多条补验时合并读取；listed parameters passed不等于整条案例或253例全通过。规则全矩阵主要在Go，真实DB/API检验事务、协议、引用和当前权限，浏览器验证真实入口与交互，不机械跨层重跑所有日期。

保留的过程失败包括：初始7条浏览器5通过，修正拦截器解除时机及提示定位后6/7，再改为实际历史只读提示；首次完整78条76通过，旧滚轮测试把指针放在水平页签，另有新增明细拼接恢复提示文本的问题；改在实际日志正文发物理滚轮，并将主恢复提示保留独立语义文本。第二次82条81通过，唯一外观断言失败；移动指针到空白处并等待字体及首帧后，在保留字号/高度/下划线/标签原断言下定向通过。第三次82条仍为81/82，失败在旧冲突重读用例，定向通过后确定为R5竞态，已修复并加确定屏障。最终完整复测结果另外记录，不把失败轮覆盖为通过。

[初始浏览器](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/browser.log)、[第二轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/browser-round2.log)、[78条失败轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/smoke.log)、[82条失败轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/smoke-final.log)、[重读竞态失败轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/smoke-reload-failed.log)、[确定屏障复测](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/reload-barriers.log)、[滚轮复测](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/scroll-fixes.log)、[外观复测](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/field-fixes.log)保持。新增资格测试初版把技术拒绝终态误当业务残留，修正为业务表全文相同且技术终态恰增一条后通过；中途编译失误及失败输出也保留。最终资料检查曾因先更新链接、后复制构建日志而失败；补齐真实日志后重跑，过程见[资料顺序失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/check-evidence-order-failed.log)，不算规则或业务通过。原文件、原来源及资产保护见[保护核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/history-protection.json)。

真实菜单COMMIT后丢失响应，随后只重启4507本任务API，再刷新从原引用resolve；原逐项结果完全一致，当前3月停用、旧来源3月启用，证据见[重启恢复](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/restart-recovery.json)。这与池重连断言分别记录；程序二进制与构建摘要见[运行匹配](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/runtime-match.json)。

TL207由独立代理以隔离Store适配器构造，真实PG观察40P01死锁、55P03锁超时，以及受控共用advisory键的实际等待，三项通过；随后同语义采用为永久测试并由主线复跑15.323s通过，见[共享守卫](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/lock-failure.log)。共用锁键为受控注入，不声称发现自然SHA64碰撞。失败整笔回滚，原身份安全resolve；不同完整身份不串终态/来源/重试结果。

TL152未知分类单独为awaiting_input，与另一明确越权字段同笔立即拒绝，真实DB/API通过；TL163两项重排后的真实拒绝丢响应、刷新、后来头推进及原内容查回/重试均保持原itemId/相对字段，未恢复原正文，见[TL163](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/tl163-menu.json)。TL163以路由注入合成201码点名称产生真实业务拒绝，故障范围明确，未绕过领域或数据库。

必要项补验包括：G117/146/147/216负参数与多错误稳定顺序；API120缺项及214/215两个UPDATE同字段原拒绝；B150/151/154连续缺项/非法/合法与成对删除、155清前序必填/改日期/旧默认失效、157同值定位、158稀疏两段影响；以及独立162/164/174跨会话迟到、超限信封/暂不可读和撤写刷新。实际记录见[菜单专项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/menu-acceptance-passed.log)、[独立补验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/independent-additional-review.json)。这些属于T08必要闭合，未改弱验收条件转交T07。专项TL155初版因last()惰性定位误指新增项而失败，改按data-item-id选择后3/3通过，原失败日志保持。

round6完整86条为85通过/1失败，失败为旧历史短屏用例的scrollY持续0；该旧脚本未等历史资料/响应式布局就以固定坐标发一次滚轮，后续定位观测该坐标落在标题H1。新增等待历史只读资料、320×256实际视口及文档可滚，指针落在实际正文P，再用真实滚轮；原scrollY>0、历史短条/返回inViewport断言均保持。主线连续3/3和独立1/1通过，产品无变化，原失败轮与实际H1/P落点观测见[原轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/smoke-closeout.log)、[三次定向](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/history-scroll-proof.log)、[独立补验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/independent-history-scroll.json)。round7只增加此旧测试修正，57项冻结后的新完整86条已全部通过，退出码0，未重跑无变化的Go/API/构建。

最后状态定稿后的文档检查见[最终docs](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/docs-complete.log)；其结构检查不替代上述业务及独立验证。

最初完整DB/API轮的删除/改日预览与保存影响不一致，差异为单条保存泄露内部candidate项标识；已限制单条不输出内部项，随后完整integration多轮通过，原失败见[初轮DB/API](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/integration-first-failed.log)。本轮原始命令输出另封存[实施日志原件](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/implementation-original-logs.tar.gz)，不以新通过覆盖旧失败。

## 6 未覆盖与结论

T08实现、全部必要补验与最终完整门槛通过，独立复审通过；57项round7输入与当前源/测试文件逐项匹配，无本阶段未闭合必需项。010保持in_progress；T01/T02已接通分类/联合内核及多目标存储，跨动作完整主/补层矩阵留T07逐项核对，不笼统标全部完成。

未启动TL255既定2000BU×最多50来源×20成员、20并发和冷热p95/存储/计划测量；TL256仅代表取消/累计SQL超时，不宣称全容量通过。真实M03/M04引用、字段冻结、消费者采用、事件投递、生产恢复及完整物理网络COMMIT未知矩阵未接入/未运行。TL241/242/246/247已退役，不执行、不记通过。Firefox、WebKit及外部部署未运行。

## 7 证据保存与交付

持久证据在[data目录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation)，原始独立报告/反例封存保持原字节及SHA；可读日志副本仅移除行末空白，归档清单区分副本与原件。凭据、会话正文及用户未保存表单正文不进入证据；合成业务的已保存结果和测试输入按本报告留证。旧定稿报告及登记条目保持；资产只追加本批new_evidence，不改原来源、state或旧adoptions，不提高整项验证状态。

最终输入摘要、程序与运行实例一致性、真实重启原结果查回已留证；最终make check已通过，文档与差异检查随状态定稿补验后本地提交本任务差异；不push、不开PR、不合并、不部署，不继续启动T07。

独立评审时的原输入摘要和源归档分别保存在[reviewed-inputs](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/reviewed-inputs.json)及[reviewed-changes](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/reviewed-changes.tar.gz)，保持当时字节；评审后仅更新状态/导航和报告，当前摘要同步记录这些文档变化，代码及测试57项保持round7。逐文件归档摘要见[manifest](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/manifest.json)，凭据检查见[扫描](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/022-joint-implementation/credential-scan.json)。
