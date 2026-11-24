# CHNHCM-116 R3退出失败后草稿保护评审修复

登记信息：[登记表](../governance/registry.json)的verification_records。
适用范围：R3-09 P2，共用useDraftExit及新建、单条维护、固定两条的离开保护。
上游依据：[116](../plans/116-m02-shared-dated-maintenance-roadmap.md)、[120](../specs/m02/120-dated-maintenance-technical-contract.md)、[121](../specs/m02/121-dated-object-admission-matrix.md)及[023](../specs/m02/023-m02-timeline-test-cases.md)。[原R3报告](116-r3-frontend-workflow.md)和证据保留原结果。

## 1 验证对象与受测基线

2026-11-05，wt-dev-b，HEAD为2c8b9c898938cafcbedb9f6250b1c308673a3280，原R3增量尚未提交。开始时保存全部已有文件摘要与工作区状态，见[基线](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3-review-fixes/before.json)。本轮按用户意见分析并修复，不提交或推送。

## 2 环境与前置条件

主代理检查用户提供复现与当前源码，确认持续放行状态在dirty未变为false时不复位。原工作区修改和日常环境保留；三个独占临时BU环境分别为r3-fix-before-20261105、r3-fix-focused-20261105、r3-fix-regression-20261105。使用真实API/DB、管理员及合成数据，从正常菜单进入；只对退出注入503或连接中断，验证后恢复真实退出。实际端口见各次日志开头，资源归属及收尾见[核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3-review-fixes/cleanup.json)。

## 3 验证过程与结果

| 场景或检查 | 前置与操作 | 预期 | 实际结果 | 结论 | 证据 |
| --- | --- | --- | --- | --- | --- |
| 用户提供P2复现及源码核对 | 更正日期→退出→放弃并离开→退出503→重新输入名称→取消编辑 | 新草稿再次确认，页面关闭保护有效，已保存来源不变 | 取消直接丢弃，beforeunload未拦截；已保存来源不变，源码allowed持续为true与之吻合 | 失败，采纳发现 | [用户复现](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3-review-fixes/provided-reproduction.json) |
| 修复前新增反例 | 原hook，正常菜单同路径；同时断言beforeunload和取消确认 | 两种离开重新保护，来源不变 | 1项失败，两处断言均重现评审意见 | 失败，确认回归 | [首次结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3-review-fixes/browser-before.txt) |
| 静态与单元 | 修复后make check | 文档、边界、Go单元/vet、17项Node及TypeScript通过 | 退出0；Go缓存结果按输出读取 | 通过 | [检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3-review-fixes/check.txt) |
| 前端构建 | pnpm --filter @chnhcm/web build | 当前前端可构建 | 退出0，保留既有包体积提示 | 通过 | [构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3-review-fixes/build-web.txt) |
| 新反例及固定两条 | 更正退出503/断连后的新草稿；新建失败退出；同页迟到查回；固定两条全部8项现行参数 | 新草稿取消/路由/原生关闭重新保护；继续保留输入，明确放弃正常离开；未知引用不丢失 | 12/12，零重试，2.4分钟；含3项新增反例 | 通过 | [第一批](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3-review-fixes/browser-focused.txt) |
| 既有退出及维护路径 | 新建未保存/窄屏退出、失败退出、迟到保存、旧退出断连/204、维护取消与分类初值 | 原退出/会话/未知引用及取消行为保持 | 7/7，零重试，56.8秒 | 通过 | [第二批](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3-review-fixes/browser-regression.txt) |
| 输入与历史保护 | 比较本轮两批及检查输入、全部开始时文件摘要 | 产品及用例受测期间不漂移，旧报告不改写，其他R3修改保留 | 1339项运行输入不变；2885份旧报告/证据不变 | 通过 | [运行核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3-review-fixes/runtime-verification.json)、[范围核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3-review-fixes/scope-check.json) |

## 4 失败、修复与复验

R3-09：allowed=true仅在dirty=false后复位，放弃后的表单清理不保证dirty为false。产品修复仅改useDraftExit：移除allowed引用及依赖dirty的复位effect，始终根据当前dirty与saved判定保护。路由通过当前blocker.proceed完成，局部取消/退出通过当前回调完成；不另建持续授权，不修改字段清理或业务规则。

先用新增503反例在原hook上得到预期失败，再修复源码；首次失败、脱敏截图与修复前hook原件保留。新增两种更正退出故障（503、网络中断）及新建控制用例均实际触发Chromium原生beforeunload对话框并选择取消关闭，不只用合成事件代替页面关闭。继续编辑保留新值；更正来源在重复取消、路由离开及关闭尝试后保持不变，最后恢复正常退出并成功登录页收束。

## 5 未覆盖范围与限制

原187项完整浏览器、DB及21组视口为修复前证据，不转记为本轮通过。产品只改共用离开保护，本轮19项为两批定向回归，未重跑全部浏览器、完整integration或七组视口；Go、SQL、API和页面布局均未修改。部门/R4—R6仍在范围外，无独立评审、人工签收、远程CI或生产验收结论。

## 6 结论与后续事项

采纳该P2意见，已修复并关闭R3-09。主代理复核确认：放弃后没有跨动作放行标记，路由只通过当前proceed；退出失败或中断后的新草稿重新触发取消、路由及真实关闭保护，继续编辑保留输入，已保存来源不变。19项定向回归及check/前端构建通过，结论结合原R3批次和本次修复证据读取；原“无未关闭发现”仅保留为原时点记录，不用原全轮通过否定后来发现。

三个本轮临时profile的容器/网络/卷、私密配置及独占构建产物均已核验移除，脱敏证据保留。未启停或写入既有持久环境，未动评审方临时目录；日常m02-current-b继续由wt-dev-b负责开发及手工复核，下次模型或运行维护时复查。本轮及原R3增量均未提交、未推送。

## 7 证据保存

本轮证据位于[data/116-r3-review-fixes](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3-review-fixes)，用户提供复现及修复前hook原件另存；原R3报告、数据、输入清单和补丁不覆盖。[源码增量](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3-review-fixes/source.patch.gz)、[受测源码](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3-review-fixes/source.json)和[执行结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3-review-fixes/execution.json)保存本次范围；首次失败与两批复验分别留证。
