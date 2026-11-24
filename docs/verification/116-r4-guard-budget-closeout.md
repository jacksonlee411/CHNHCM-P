# CHNHCM-116 R4守卫预算诊断与收尾复核

登记信息：[registry.json](../governance/registry.json)。关联[116路线图](../plans/116-m02-shared-dated-maintenance-roadmap.md)、[120技术合同](../specs/m02/120-dated-maintenance-technical-contract.md)及[121验收映射](../specs/m02/121-dated-object-admission-matrix.md)。原[实施评审修复报告](116-r4-implementation-review-fixes.md)保留原结果。

## 1 范围与基线

基线wt-dev-b、82087469605daebdf34e8501d15864d5f1566b41，起始工作区干净。本轮处理原完整DB回归唯一的守卫4秒耗时断言失败，核实原因、补充诊断和必要修复。R5页面、R6容量、人工签收及远程CI不在范围内。

## 2 因果诊断

对原完整回归test2json的run/end墙钟时间与Go Elapsed进行比对，50项测试出现大于1秒的差异；两个测试包同一时段出现约2.1秒的差异。原失败用例所在区间也有此现象，但原日志没有精确请求/SQL耗时，不能事后复原其全部阶段。独立80秒采样确认WSL墙钟两次回拨2.246/2.188秒，单调计时仍向前；timesync状态显示32秒轮询和约负2.2秒偏差。本轮未修改宿主时间、时区或时间同步服务，也未断言是哪一层服务触发了调整。

在同一产品基线的隔离DB上，仅用Go overlay增加查询阶段观测，保持原4秒断言；第14个真实请求重现超限：请求4.225791秒，其中关系advisory取锁4.211354秒，墙钟在该SQL内回拨2.199134秒，返回真实SQLSTATE 55P03；保存点回滚0.732毫秒、外层回滚1.091毫秒。该请求只有一段长锁等待，回滚不是主要耗时。复现直接确认了与原故障一致的因果机制，原始故障自身的未记录阶段仍不补造。

PostgreSQL 18.6的[timeout实现](https://github.com/postgres/postgres/blob/REL_18_6/src/backend/utils/misc/timeout.c)使用GetCurrentTimestamp计算及复核超时终点，回拨时可重新安排尚未到期的闹钟；仅设置lock_timeout不能保证Go单调计量下的2秒阶段上限。独立[诊断摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-guard-budget-closeout/diagnosis.json)、[原运行时差分析](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-guard-budget-closeout/original-run-timing-analysis.json)、[系统采样](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-guard-budget-closeout/clock-observation.json)、[修复前真实回拨复现](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-guard-budget-closeout/clock-diagnostic.jsonl)和精确overlay源文件保留。

## 3 修复与反例

仅在BU/部门共用的lockDepartmentDependencies入口增加Go单调计时2秒子context，继承原请求更早的截止时间，覆盖整个对象锁阶段、依赖读取和保存点重规划；循环和后续锁不重新起算。成功离开阶段后业务计算继续使用原请求context；数据库锁/语句超时和既有2秒清理机制保留。保存点回滚无法确认时继续返回M02_RESULT_PENDING，不伪造rejected或accepted终态。没有放宽原4秒断言，没有新增客户端字段、存储结构或重试平台。

新真实DB反例在测试专用Store中将lock_timeout关闭，确定性模拟数据库侧计时不能及时终止等待，不改变宿主时钟。修复前部门及BU反向守卫分别等待5.067/5.070秒并失败；修复后约2秒结束。部门参数还先在真实BU行锁消耗1.2秒，再进入关系守卫等待，检查预算未重置。API返回503且无outcome；来源、关系、Outbox、command_result计数不变，DB锁实际释放，解除阻塞后原submissionId可成功重试。

修复后的另一轮真实系统回拨诊断中，墙钟在请求内回拨2.257242秒，请求仍在2.041232秒结束。该轮经过16个请求并以实际观察到回拨为通过前提，不以几次未撞到回拨的成功重试代替根因补验。见[修复前反例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-guard-budget-closeout/before-fix.jsonl)、[修复后回拨验证](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-guard-budget-closeout/clock-diagnostic-after.jsonl)、[最终位置定向测试](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-guard-budget-closeout/relocated-targeted.jsonl)。

## 4 检查与范围

故障注入测试最初放在tests/integration，引入组织内部适配器后被make boundaries正确拒绝；已迁至组合根internal/bootstrap/department_guard_budget_integration_test.go，未放宽导入规则。原失败[检查日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-guard-budget-closeout/check-first-boundary.txt)和主动中断的首次完整DB运行均保留，中断不计通过；两只遗留自有Testcontainers已定向回收，见[中断与清理](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-guard-budget-closeout/interruption.json)。

最终位置定向测试、make check及构建通过；受影响的原守卫、连接池剩余预算、取消、保存点回滚故障及共享清理期限定向通过。最终单轮完整DB/API运行完成：顶层153通过、4跳过，含子测试477通过、4跳过、0失败；bootstrap包398.715秒、integration包1006.760秒，命令与make integration同一集合、额外保留-json日志。跳过项为BUSliceLoad、JointMidnightBrowser、T07BusinessDayBoundaryBrowser及T07LongCurrentTimelineCapacity，不计已执行。该结论只指本次最终单轮，不覆盖原失败和中断记录。见[完整结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-guard-budget-closeout/integration-summary.json)及[原始日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-guard-budget-closeout/integration-final.jsonl)。本轮无前端产品改动，部门菜单/页面、视口、R6容量、外域消费者及生产交付仍不在验收范围。

## 5 R5交接与资源

已复核当前OpenAPI部门端点、DepartmentContext/DepartmentDetail/DepartmentSelectionQuery/DepartmentDataChange及生成客户端输入，见[接口核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-guard-budget-closeout/r5-handoff-review.json)。R5继续按原[R4交接清单](116-r4-department-backend.md#6-r5接口与受测边界交接)消费页面上下文、原因、能力/自动辅助、基本资料摘要、变更日志、引用、选择/导出和结果查回；摘要独立日志读权、稳定BU ID及日期裁剪均为当前合同。

4382持久环境API已按当前输入重新编译并重启，Web/API/DB端口保持4382/8382/54373。管理员和查询者分别从正常BU菜单登录，通过同一浏览器会话验证5个部门当前/未来/停用状态、稳定BU ID筛选及所选历史记录读取，随后退出并关闭Chromium。28张organization业务表在更新前后行数/摘要一致；本轮临时DB与持久样本隔离。m02-current-b保持停止并保护旧程序/卷，m02-department-b继续供用户验证，责任为wt-dev-b维护者，下次版本交接或用户结束验证时复查。持久库前后摘要、环境/旧程序摘要与两角色浏览器结果分别保留在persistent-before/after.json、persistent-environments-before/after.json和runtime-browser.json。本轮日志标识的318个自有Testcontainers最终均已回收，其中主动中断遗留的2个由本任务定向停止并删除；未清理其他任务或持久环境。3091份既有验证文件与1386个本轮运行/配置/测试输入摘要未变；tsconfig.base.json及.node-version另核对与基线一致。见[资源和输入核验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-guard-budget-closeout/final-resources-and-inputs.json)。


## 6 限定范围复核结论

主代理复核覆盖Go截止时间在循环外建立、继承更早请求期限、阶段成功后不缩短后续业务预算、失败无伪造终态、真实锁释放和原身份重试，以及BU反向调用使用同一守卫。相应真实DB/API、受影响取消/清理回归、当前持久环境和文档合同已核对，本轮范围内无未关闭发现，逐项见[范围复核清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-guard-budget-closeout/scope-review.json)。R4后端收尾通过，可进入R5正常部门菜单和页面实施；116整体仍进行中，R5/R6、人工业务签收及远程CI不在此结论内。本轮为主代理实施后复核，不冒充原独立评审人重新批准。

原始失败保留，本次没有扩大等待阈值或改变主机时间服务；既有两项P1/P2修复随最终完整集合复验通过。检查、构建和最终文档检查分别保留在[check-final.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-guard-budget-closeout/check-final.txt)、[build.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-guard-budget-closeout/build.txt)及[docs-final.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-guard-budget-closeout/docs-final.txt)。本轮未提交或推送。
