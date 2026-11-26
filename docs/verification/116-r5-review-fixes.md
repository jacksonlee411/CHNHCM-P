# CHNHCM-116 R5授权与根部门创建评审修复

登记信息：[registry.json](../governance/registry.json)。适用范围：R5评审的P1部门状态撤权缓存与P2根部门创建可选上级声明。
上游依据：[116路线图](../archive/plans/116-m02-shared-dated-maintenance-roadmap.md)、[119交互合同](../specs/m02/119-department-functional-design.md)、[120技术合同](../specs/m02/120-dated-maintenance-technical-contract.md)及[121验证矩阵](../specs/m02/121-dated-object-admission-matrix.md)。[原R5报告](116-r5-department-pages-and-acceptance.md)保留原时点结果。

## 1 验证对象与受测基线

2026-11-05，wt-dev-b，HEAD为545860a40f9c2c2571258ea4a1a9ba768cea9e96，已有R5未提交增量。用户授权修复本轮评审的两项问题；本轮不提交或推送。开始时4945份仓内文件摘要、工作区状态和Docker资源清单见[基线](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/before.json)，拟修改产品文件原件另存before-source.json。

## 2 环境与前置条件

使用本工作树独占临时DB/API与桌面Chromium，从“设置→组织配置→部门/业务单元”进入，管理员仅操作合成数据。权限撤销在自有隔离库中执行，测试结束恢复；503及断连只作用于指定详情重读。8个临时profile、实际端口及收尾见[资源核验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/scope-and-cleanup.json)。既有m02-department-b持久环境保持，不执行测试授权变更或样本写入。

## 3 验证过程与结果

已采纳[评审发现](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/reviewer-findings.json)：状态撤权后真实API省略state，详情重读503后页面仍显示旧状态；可选上级无写权时API根部门创建201、页面创建403。原[状态复现及夹具首次失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/reviewer-status-and-fixture.txt)与[根部门复现](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/reviewer-root-create.txt)分别保留。

| 检查 | 前置与动作 | 预期与实际 | 结论 | 证据 |
| --- | --- | --- | --- | --- |
| 修复前新增菜单反例 | 根部门受限创建；撤status后详情503/断连；固定期间保存403再重读失败 | 原代码重现创建403及旧“启用”残留；夹具问题另列第4节 | 失败，采纳并修复 | [首轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/browser-before.txt)、[第二轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/browser-before2.txt) |
| 首轮修复及双对象回归 | 新增7项、原R5菜单/关系/选择及全部BU固定期间参数 | 27项通过，0重试；随后补查不携字段列表的单条403，不将此轮充作最终输入 | 通过，限定本轮输入 | [首轮修复回归](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/browser-focused.txt) |
| 单条403遗漏补查 | 撤部门status读权及实际更正/首条改日动作写权；辅助或保存403未带readFields，详情503 | 首轮修复仍残留状态；补齐403空字段范围清理后纳入最终回归 | 失败后修复 | [保存反例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/status403-before-second.txt)、[辅助反例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/browser-status403-before3.txt) |
| 最终双对象定向浏览器 | 新增9项、两对象维护/恢复、关系/历史、137项选择CSV及8项BU固定期间 | 29/29通过，0重试，6.7分钟；根创建201，明确上级/更正null保持，状态失权后不显示旧值，失败不改变业务来源 | 通过 | [最终29项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/browser-final.txt) |
| 最终BU撤权回归 | 改日preview/reload、位置新增nameEn/effectiveDate、重新启用字段撤权 | 5/5通过，0重试，1.1分钟；保留可读输入，清理不可读内容，BU状态合同保持 | 通过，独立一批 | [BU5项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/browser-bu.txt) |
| 缓存投影单元 | 检查所选记录、时间轴、viewFact及新增/恢复默认；撤权后再恢复字段范围 | 状态缓存全部清除，不修改原输入；仅恢复字段范围不复活旧状态；BU仍显示自身状态 | 2项通过，已进入make check的19项Node检查 | [单元](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/projection-unit.txt)、[最终check](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/check-final.txt) |
| 最终静态、构建及收尾 | make check、make build、文档/差异检查、输入与资源核验 | 73项Python、19项Node、Go单元/vet、边界、TypeScript及构建通过；1411运行输入不变，3282份旧报告/证据不变 | 通过；Go缓存结果按日志读取 | [check](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/check-final.txt)、[build](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/build-final.txt)、[核验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/scope-and-cleanup.json) |

## 4 失败、修复与复验

R5-F04（P1）：部门状态有独立status读权，共用缓存清理遗漏记录state。对象定义新增statusReadField映射，部门声明status；共用清理移除详情status及所选记录、时间轴、viewFact、默认投影的state，详情展示也执行同一映射。单条辅助/保存及固定期间403未返回readFields时按空字段范围清理，再请求详情；503/断连后旧状态不再恢复。BU无独立status字段权，保留原对象语义。最终两个403反例均使用真实拒绝响应，不伪造字段权限列表。

R5-F05（P2）：新建适配器把未选择的parentDepartmentId统一序列化为null，导致Go将其计为显式字段声明。新建根部门已改为省略该可选字段，已选择上级仍传稳定身份；更正时明确清除上级仍提交null，不改变后端授权或根部门默认。真实浏览器验证了受限创建、子部门创建、更正清除、旧来源保留及新建选后清除。

测试夹具错误分别保留：首轮未聚焦上级输入，清除按钮尚不可访问；根部门读取允许省略空上级，原断言误要求null；固定期间拒绝可不带readFields；单条403补查起初使用了不存在的授权动作，随后首条改日仍误用普通记录改日动作。修正控件操作及合同预期后，在正确department.record.correct/department.first.redate下复现产品问题，最后进入正式用例。夹具失败不计产品缺陷，不改变后端合同；[第一版](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/browser-tests-first.ts.txt)、[第二版](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/browser-tests-second.ts.txt)、[403首次夹具](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/status403-fixture-first.txt)和[第二次](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/status403-before-second.txt)均保留。一次status命令误传--profile被参数解析拒绝，改用CHNHCM_LOCAL_PROFILE后只读核验成功，未启停服务。

## 5 未覆盖范围与限制

Go、SQL、公开API和业务规则不变。本轮未重跑完整DB/API、原205项完整浏览器集合或视口/原生缩放；原R5和评审阶段的480项DB/API、布局及受控跨日结果均保留修复前证据，不能转记为本轮运行。当前前端按实际影响以真实DB/API菜单验证，最终29项与5项是两批，不合并声称全套单轮通过。R6、生产、远程CI及人工签收不在范围内。

## 6 结论与后续事项

主代理修复及补验确认R5-F04/P1、R5-F05/P2已关闭，121追加R5-09/10映射。R5结论结合原报告和本次定向修复证据读取，不以旧全轮结果替代当前反例，也不宣称人工签收或另一轮独立批准。当前用户要求的两项修复已完成；本轮未提交、推送或发布。

8个临时profile的容器/网络/卷、Web/API/DB端口及私密配置均已核验回收。Docker身份集合恢复评审前基线：45个容器、3个运行容器、36个网络、59个卷，无新增或缺失。既有m02-department-b仅执行status只读检查，数据库健康、Web/API继续运行；持久库未执行授权/业务写入、迁移、重建或清空。该环境继续供用户日常观察，由wt-dev-b维护者负责，下次模型或环境维护时复查。原R5报告及3282份既有验证文件保持不变，仅导航README新增入口。

## 7 证据保存

本轮证据保存于[data/116-r5-review-fixes](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes)。[实际产品补丁](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/product.patch)、[最终运行输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/runtime-inputs-final.json)、[命令退出状态](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/command-results.json)、[失败截图索引](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/failure-screenshots.json)及[资源/范围核验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-review-fixes/scope-and-cleanup.json)可复核。评审脚本、首次失败与本轮补验分别保存，不覆盖原R5目录或评审方临时文件。最终文档检查另存docs-final.txt。正式证据使用可纳入版本管理的.txt副本，原.log输出保持，不调整全仓忽略规则。
