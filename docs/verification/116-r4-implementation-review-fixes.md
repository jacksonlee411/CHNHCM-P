# CHNHCM-116 R4实施评审修复与补验

登记信息：[registry.json](../governance/registry.json)。关联[116路线图](../archive/plans/116-m02-shared-dated-maintenance-roadmap.md)、[119页面合同](../specs/m02/119-department-functional-design.md)、[120技术合同](../specs/m02/120-dated-maintenance-technical-contract.md)及[121验收映射](../specs/m02/121-dated-object-admission-matrix.md)。原[R4实施报告](116-r4-department-backend.md)与[本地样本](116-r4-local-department-samples.md)保留原时点证据。

## 1 评审采纳与基线

采纳用户提供的两项独立真库/API评审：P1启停对删除摘要在effectiveDate失权后仍返回变化区间；P2管理BU筛选把稳定身份与编码加名称比较，导致列表、选择及导出漏选。两者分别违反字段读权和119稳定引用合同，必须修复，不能用旧回归通过替代。起始基线wt-dev-b、5fa32750ebaf154fd476ed26dd4a983c11a39ef5加已有未提交R4增量；起始4710文件摘要见本目录证据baseline.json。

## 2 修改范围

DepartmentDataChange的start及endExclusive改为获权可缺省字段；无effectiveDate读权时省略两个键，有权的开放终点明确null。before/after、停用及恢复记录沿原字段投影；仅不可读业务字段存在差异时，不输出变化条目。保留删除动作权并不授予日期或名称读权。

managementBuId仅接受equals和完整稳定BU UUID，按获权引用ID比较；普通文字字段继续字面匹配。列表、选择及导出复用同一规范化和谓词；同名BU互不混选，BU更名不改变匹配对象，实际展示变更仍使旧选择失效。生成合同新增DepartmentSelectionQuery，避免沿用BU字段枚举漏掉managementBuId、shortName、departmentPath；BU公开合同保持。

## 3 反例与补验

真实管理员创建部门、停用、带名称变化恢复，分别形成有限和开放区间。查询者取得明确删除动作权；保持history读权，撤销普通effectiveDate读权后检查当前详情及整个删除摘要无日期键，名称差异仍可读；进一步撤销名称后摘要为空，恢复读权后区间正确；缺任一删除权时能力消失。读取/授权调整不改业务来源。

同名BU与同名部门按实际BU ID筛选，列表、全查询选择和CSV结果一致；BU更名后原选择拒绝，重选导出新名称；未知ID为空、文本及contains拒绝；BU标签失权仍可按已知获权ID筛选，管理BU字段失权后三入口拒绝。所有新增预期在修复前失败，修复后定向通过。

## 4 验证结果与边界

完整DB/API本次运行顶层151通过、1失败、4跳过；含子测试473通过、1失败、4跳过。bootstrap包通过，integration包因TestDepartmentV07V20GuardOrderBudgetAndReplan的4秒耗时断言失败而失败；两项新增回归在完整运行中均通过。首次日志没有记录精确耗时，但响应为503、无outcome，不是错误码断言失败。仅增加耗时打印、保留原断言的诊断复测3次通过，耗时2.0216/2.0416/2.0217秒；未改动原用例在完整运行结束后独立复验3次也均通过，日志另存。本次不把分批复测合并记为单轮全绿，首次超时原因不能仅据日志定性。make check、前后端构建及生成客户端检查已通过。首次check仅因新增报告缺少README导航而失败，补链接后重跑通过，首份日志保留。没有修改前端业务流程；R5菜单、页面、视口及R6容量不在本次验收范围。主代理补验不冒充独立评审重新批准。

## 5 证据与资源

证据目录data/116-r4-implementation-review-fixes：保留用户评审probes、原checks摘要、本次修复前日志及定向复验。隔离测试资源由Testcontainers清理；持久样本库不参与授权撤销或测试写入。最终1393文件检查快照保持一致（含1370个非Markdown运行/配置/测试输入）；3065份此前验证文件未变。本次日志观察到的218个Testcontainers容器均已回收，无残留，未新建smoke profile；详见[最终输入与资源](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-implementation-review-fixes/final-resources-and-inputs.json)。

本地m02-department-b已在相同存储模型上重新编译并重启API，Web4382/API8382/DB54373保持。管理员和查询者分别从正常BU菜单登录，经浏览器会话按稳定BU ID查询原5个部门并resolve选择；管理员CSV导出5行、查询者无导出能力，当前及历史名称保持。28张organization业务表重启前后行数及内容摘要完全相同。两种浏览器会话已退出且Chromium关闭；环境继续供用户验证，责任为wt-dev-b维护者，下次版本交接或用户结束使用时复查。旧m02-current-b保持停止，其程序、卷及7个BU不变。

- [修复前反例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-implementation-review-fixes/before-fix.log)、[修复后定向回归](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-implementation-review-fixes/targeted-fix.log)。
- [make check](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-implementation-review-fixes/check-final.log)、[首次文档导航失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-implementation-review-fixes/check-first-missing-navigation.log)、[构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-implementation-review-fixes/build-final.log)。
- [当前4382浏览器验证](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-implementation-review-fixes/runtime-browser.json)、[重启前摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-implementation-review-fixes/persistent-before.json)、[重启后摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-implementation-review-fixes/persistent-after.json)。
- [原BU合同保护](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-implementation-review-fixes/bu-contract-preservation.json)、[本次受测输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-implementation-review-fixes/tested-runtime-inputs.json)。

## 6 限定结论与最终记录

两项评审意见均采纳；P1字段投影、P2稳定引用及其生成类型已修复，反例与实际API复验完成，可据本报告复审这两项。未据此声称独立评审人已经批准，也不把完整回归中那次耗时失败抹去。R5菜单及R6容量仍待后续验收。

[原评审证据](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-implementation-review-fixes/reviewer-probes.log)、[采纳映射](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-implementation-review-fixes/review-acceptance.json)、[完整回归汇总](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-implementation-review-fixes/integration-summary.json)、[耗时诊断复测](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-implementation-review-fixes/guard-diagnostic.log)、[原用例复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-implementation-review-fixes/guard-original-retest.log)分别保留，不相互替换。本轮未提交、推送、PR、合并或部署。
