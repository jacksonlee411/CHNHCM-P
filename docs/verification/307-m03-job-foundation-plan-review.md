# CHNHCM-307 M03职务基础方案与合同评审

登记信息：[registry.json](../governance/registry.json)。对象为[307实施方案](../archive/plans/307-m03-job-foundation.md)及[308首批合同](../specs/m03/308-job-foundation-contract.md)。本记录仅证明主代理文档评审及实际文档检查，不是独立代理评审、人工业务签收或产品验收。

## 1 任务、基线与保护范围

2026-11-16用户要求完成M03职务基础实施方案并多轮评审完善通过。当前分支wt-dev-d，HEAD为1e830a4ccb0066085bef24239848b086971e4d73，开始时工作区干净。本轮先登记307／308，再创建计划及首批合同；产品T01—T05未开始。

在本工作树读取M03目标、DDD、004、014、124、202、304及现有Organization日期、企业引用、提交恢复与边界检查代码。源码核对用来确认真实接入限制，没有运行产品。开始时1975份非文档文件摘要保存在[基线](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/307-m03-job-foundation-plan-review/baseline.json)，各轮全文压缩稿及摘要随[证据目录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/307-m03-job-foundation-plan-review)保留。

未fetch、提交、推送、触发Actions或更新固定环境。本轮没有创建或操作profile、容器、数据库、卷及浏览器进程，无自有运行资源需要回收；既有运行环境的可用性未重新测试。125、200-D08/D09及206-D01—03继续由原记录持有。

## 2 方案结果与范围

首批限定企业内单层JobFamily与单值归属Job，完整适用按日维护、查询导航、选择CSV、权限及可靠提交。企业仅提供身份与首条日期下界，停用不级联职务目录；Job启用期间必须完整受系列覆盖，原采用与当前名称展示分开。

Staffing拥有独立领域、表、API与授权空间，不直接导入Organization内部内核。企业正反向公开端口、同物理事务、M02／M03双消费者验证和具体生成源由T01交付；文档通过不代替技术可行性验证。职级、阶梯、岗位、容量和M04/M05属于后续批次，未提前创建产品占位。

## 3 多轮评审

| 轮次 | 输入／检查重点 | 实际结论 | 修订输出 |
| --- | --- | --- | --- |
| 第一轮 | draft-0；全期间引用、来源采用和并发锁 | 要求修改，1项P1、1项P2 | draft-1 |
| 第二轮 | draft-1；页面显示、查询权限、跨域实施出口和API身份 | 要求修改，1项P1、2项P2 | draft-2 |
| 第三轮 | draft-2；容量、场景日期、拒绝与成功口径 | 要求修改，2项P2 | draft-3 |
| 第四轮 | draft-3；普通记录读权、历史权、开放新建与有限系列 | 要求修改，2项P2 | draft-4 |
| 第五轮 | draft-4；反向检索各节权限汇总 | R307-02复开：第7节未同步纯继承例外 | draft-5，修订关闭 |
| 第六轮 | draft-5；26项反例、九项问题及全部范围交叉复核 | 通过，无新增或未关闭P1/P2 | 接受draft-5；随后仅回填进度及登记 |

前五轮指出的是方案缺口，不是本轮已复现的产品故障。九个不同问题合计2项P1、7项P2；R307-02第五轮复开修订不重复计数。评审者均为Codex主代理/root，按不同关注面顺序复核；没有声称独立评审。

| 问题 | 等级 | 修订及文档关闭依据 |
| --- | --- | --- |
| R307-01 | P1 | 308第6.3节：锁后READ COMMITTED最新依赖、系列共享／排他锁、空集合保护、同物理事务及回滚后重规划；AC012 |
| R307-02 | P2 | 308第3.4节：逐动作采用矩阵，固定期间仅改停用端不重取恢复来源，继承与新采用权限分开；AC005 |
| R307-03 | P2 | 308第3.4节：今日、未来首条、停用前日、显式记录及CSV展示日期固定；当前资料与旧采用分开 |
| R307-04 | P1 | 308第4.1节：受限系列标签不参与名称排序／搜索；稳定ID筛选检查字段及系列读权，撤权不静默扩大查询 |
| R307-05 | P2 | 307第4节：T01五项实际证据出口；308第6.2节区分创建无expectedSource、业务维护及选择草稿身份 |
| R307-06 | P2 | 307第6节：双角色读、20独立对象写、预热／正式／拒绝分开、普通目标和大影响观察的容量边界 |
| R307-07 | P2 | 308第5节：独立场景固定D=2026-02-01，仅隔离Clock；容量使用真实执行日及对应预期 |
| R307-08 | P2 | 308第4.1节：本对象当前头普通过去记录与显式旧源／日志分别按read／history.read判定；引用历史权独立，AC010 |
| R307-09 | P2 | 308第3.3节：新建开放尾段与有限系列候选一致，不自动生成停用；明确合法显式办理顺序和本批限制，AC004 |

逐轮原问题、决定和稿件摘要见[评审清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/307-m03-job-foundation-plan-review/review.json)。R307-02第五轮补齐第7节：所有隐式归属变化都核字段写权，只有动作矩阵触发新采用才追加新采用权限，纯继承不误增历史读权。每项关闭只在文档层成立，未来实现仍须分别验证。

## 4 检查与证据

| 检查 | 当前结果 | 证据／边界 |
| --- | --- | --- |
| 13条规则、8场景、16验收条件及逐项追踪 | 通过，26项文档反例逐项复核 | [范围检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/307-m03-job-foundation-plan-review/scope-check.json)、[反例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/307-m03-job-foundation-plan-review/counterexamples.json)；不代表运行测试 |
| make docs | 通过：126项脚本测试，380份Markdown、12064条本地链接 | [完整日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/307-m03-job-foundation-plan-review/make-docs-first.txt.gz)、[退出码](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/307-m03-job-foundation-plan-review/make-docs-first-result.json)；此时为draft-3和评审草稿，回填后另做结构检查 |
| git diff --check及新建文本空白 | 通过 | [检查结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/307-m03-job-foundation-plan-review/diff-check.json)，只检查改动格式 |
| 1975份非文档文件保护 | 通过，逐份与初始摘要一致 | 范围检查；无新增产品文件 |
| 产品Go／DB/API／浏览器／容量／部署 | 未运行 | 本轮纯文档；T01—T05和全部M03验收未执行 |

## 5 交接

307／308通过六轮主代理文档评审，九项问题在文档层关闭；登记为active，T00文档准备完成，产品执行状态保持not_started，T01—T05和全部产品验收未运行。下一产品任务从T01跨域接入验证开始；真实结果、首次失败、资源收尾及需要的合同变化另存实施记录。

文档通过不等于跨域内核已接通，也不关闭M02容量。无正式时延目标的大影响系列维护仍是待定观察，不能写成M03全部容量达标。本轮未提交、推送或更新固定环境。最终回填后的结构／差异检查见[收尾结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/307-m03-job-foundation-plan-review/final-check.json)，业务合同正文仍与第六轮接受稿一致。
