# CHNHCM-125 R6容量整改方案编制与核对

登记信息：[registry.json](../governance/registry.json)的governance.verification_records。适用范围：[125方案](../plans/125-r6-capacity-remediation.md)的编制、语义自查和文档检查。本记录使用[验证报告模板](../templates/verification.md)，不证明整改已经实施或容量已经达标。

## 1 验证对象与受测基线

2026-11-06，工作树CHNHCM-wt-dev-b，固定分支wt-dev-b，HEAD为587667a1a044d1d99df9baf869d4b0401f07c1e9。开始时存在R6未提交的产品、测试、文档与证据增量；本轮以这些实际文件为分析依据，未将HEAD视为全部受测输入。输入及文件摘要见[inputs.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/125-r6-capacity-plan/inputs.json)。

本轮新增125方案、附属记录及非敏感检查证据，调整登记、导航和116交接链接。现行120/013、产品源码、SQL和容量工具不在本次修改范围；原116 R6报告与数据原件保留。

## 2 环境与前置条件

使用仓库源码、合同、已保存的R6原始数据和Python文档检查器；没有启动浏览器、数据库或容量环境，没有修改持久profile。资源启停和清理不适用于本次文档编制，不为填表重启已回收的R6环境。

编制前核对cwd、分支、HEAD、工作区、remote及固定分支登记；stash为空。对5509份原文件保存本地SHA-256基线，其中3816份位于docs/verification，包含导航README；后续保护核对将导航修改与原报告/证据分别计数。

## 3 核对过程与结果

### 3.1 方案语义自查

本次由主代理编制并自查，不称外部独立评审或人工批准。以下“通过”只表示方案已明确所列边界，不表示候选设计已经经产品测试证明。

| 检查 | 对照材料及操作 | 核对结论 |
| --- | --- | --- |
| 事实与推断分开 | 原冷/暖/混合JSON、R6报告、列表及守卫源码 | 通过：记录150次冷列表失败、74次混合BU失败及原身份not_applied；保留逐段归因缺口，不把请求时长当持锁时间 |
| 避免重复实施已有优化 | department_list SQL/缓存、X03 read_projection、department_write | 通过：指出窄查询、批量授权、部分批量写入已经存在；新工作以测量确认剩余重复为前提 |
| 集合完整性 | 013选择合同、118 BR033、120快照/缓存 | 通过：不提前截断获权集合；全量核验下界、祖先依赖、字段拒绝语义及无关变更反例明确 |
| 合同调整边界 | 120第6.4/6.5/7.1节、014生命周期 | 通过：首轮保留现行摘要/锁模式；轻量摘要、Go生成读取索引、证据格式或锁范围变化单列T04/合同节点，不静默采用 |
| 写入完整性 | 共同守卫、事务和日期合同 | 通过：锁后重读、共同2秒预算、最终D、空集合竞争、授权先于诊断及原子证据均进入方案 |
| 可执行验收 | 120目标、121 R6参数、原容量驱动 | 通过：规定冷暖完整分组/样本数、特殊形状、原阈值、失败计数、输入控制和资源归属；按工作包给出前置/责任/退出条件 |
| 完成状态 | 116、124、登记及资产边界 | 通过：125产品实施not_started，方案待评审；P01/P02仍未关闭，P03不虚设磁盘阈值，旧证据不转成新通过 |

### 3.2 实际检查

| 检查 | 实际结果 | 证据 |
| --- | --- | --- |
| make docs | 通过：73项Python检查；253份Markdown、7966条本地链接。后续仅收敛方案文字及检查记录，最终结构复核单独保存 | [首次输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/125-r6-capacity-plan/docs-first.txt) |
| git diff --check及新增文本空白检查 | 通过：现有差异无空白错误；新方案、报告及输入JSON共3份文本无行尾空白且有末尾换行 | [首次输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/125-r6-capacity-plan/diff-first.json) |
| 原文件、代码、证据及Git索引/stash保护 | 通过：5509份原文件中仅5份预定导航/登记/路线图文件变化；其余5504份一致，其中3815份原验证资料一致；HEAD、索引和stash保持 | [保护核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/125-r6-capacity-plan/protection.json) |
| 产品DB/API、浏览器、容量重测 | 未运行；本轮仅方案 | 125第4/6节安排后续执行 |

## 4 修订与复核

方案编制时收敛三项易混淆内容：当前列表读取的是全部对象的当前来源而非全部历史；分页优化仍需核验完整集合，不能承诺只有本页成本；现行缓存禁止存储业务日/状态投影，拟议Go分段读取索引必须作为实质合同调整审阅。将名称变化的资格重复计算与删除共同守卫分开，保留后者的路径/并发证明要求。最终文字复核进一步明确：新摘要按合同全部版本依据比较等价性，不仅比较页面文字；读取和写入分包保留各自证据，一条主线未达标不阻塞另一条获得修复。

文档入口中的当前阶段描述仍写R6标准/容量后置，已按现有R6报告修正为“标准及实测已交付、容量未通过”；带日期的历史记录保留原时点。

## 5 未覆盖范围与限制

没有新增性能结果、运行时阻塞链或SQL执行计划；方案中的瓶颈排序由T00实际测量决定。未修改或验收产品、数据库结构、安全证据格式、锁范围、缓存结构或读取索引。文档检查和本次语义自查均不表示300ms/500ms已达标，原失败结论保持。

## 6 结论与后续事项

125已将用户提出的解决方向落为可评审工作包、合同决策点和验收安排。具体性能方案须结合T00归因实施；最终关闭由新证据决定。登记保持方案待评审、产品实施未开始；116继续持有容量P1。收尾未新增运行资源。

## 7 证据保存

[inputs.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/125-r6-capacity-plan/inputs.json)保存分析输入、原工作区及保护范围摘要；检查原始输出和最终保护核对在同目录保存。最终编制内容的结构复核与差异输出分别保存为[docs-final.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/125-r6-capacity-plan/docs-final.txt)和[diff-final.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/125-r6-capacity-plan/diff-final.json)，按其实际退出值读取。原116 R6报告及data/116-r6文件不覆盖。本地完整文件基线仅用于本轮差异保护，长期结论以本目录提交范围内的摘要和输出为依据。
