# CHNHCM-116 R5关系候选与日志筛选授权复修

登记信息：[registry.json](../governance/registry.json)。适用范围：R5再次评审的P1关系候选拒绝后缓存残留，以及P2日志状态筛选未裁剪。
上游依据：[116路线图](../plans/116-m02-shared-dated-maintenance-roadmap.md)、[119交互合同](../specs/m02/119-department-functional-design.md)、[120技术合同](../specs/m02/120-dated-maintenance-technical-contract.md)、[121接入矩阵](../specs/m02/121-dated-object-admission-matrix.md)。

## 1 验证对象与受测基线

CHNHCM-wt-dev-b，wt-dev-b，HEAD为545860a40f9c2c2571258ea4a1a9ba768cea9e96，包含已有未提交R5及首轮修复。2026-11-05完成产品修复与复验，2026-11-06完成文档收尾。原[R5报告](116-r5-department-pages-and-acceptance.md)和[首轮修复](116-r5-review-fixes.md)保留；起始5042份文件摘要及资源身份见[基线](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/before.json)，两份产品文件原件见[修复前源码](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/before-source.json)。

## 2 环境与前置条件

本工作树独占临时DB/API与桌面Chromium，使用管理员合成数据，从设置→组织配置→部门/业务单元进入。权限真实撤销后恢复；仅指定的详情重读注入503、断连或延迟。既有m02-department-b持久环境不参与业务或授权写入。

## 3 验证过程与结果

| 检查 | 前置与动作 | 预期与实际 | 结论 | 证据 |
| --- | --- | --- | --- | --- |
| 评审反例 | 上级读权撤销后候选403/详情503；无状态历史字段权时应用状态筛选 | 前者取消编辑后显示旧上级；后者200可读日志被状态筛选403退出 | 失败，采纳修复 | [评审发现](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/reviewer-findings.json)、[候选复现](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/reviewer-relations.txt)、[筛选复现](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/reviewer-probes-first.txt) |
| 本轮修复前3项 | 候选403后暂缓详情返回；分别撤历史status或同时撤普通/历史status | 编辑仍保留旧关系输入；两种日志授权都仍提供状态选项 | 失败，符合反例 | [本轮首次结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/browser-before.txt) |
| 首批定向6项 | 上级503/断连、管理BU503；两种历史字段权及BU对照 | 部门5项通过；BU用例停留在API创建前列表，未刷新而无法定位新样本 | 5通过、1夹具失败 | [首批定向](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/browser-focused-first.txt) |
| 首批双对象及日志回归 | 修正列表查询后，6项新增、此前9项修复、双对象旅程和BU日志/历史权限/返回焦点 | 37/37通过，0重试，8.2分钟；此时尚未补修候选重读循环 | 通过，限定本批输入 | [首批37项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/browser-first-batch.txt)、[本批输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/runtime-inputs-first-batch.json) |
| 关系拒绝后的成功重读 | 真实撤上级读权，候选403但详情200，观察后续候选与详情请求 | 1.6秒内6次403及6次重读，发现自动循环 | 失败并补修 | [循环反例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/browser-reload-loop.txt)、[探针](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/reload-loop-probe.ts.txt) |
| 显式鼠标重试 | 限制同一被拒绝上下文自动重发，恢复真实授权后点击重试 | 循环已停止，但原重试按钮被候选弹层遮挡；移入弹层后真实点击得到200及合法候选 | 失败后1项复验通过 | [按钮遮挡](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/browser-retry-covered.txt)、[复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/browser-retry.txt) |
| 最终关系及双对象回归 | 7项新增授权参数、此前9项修复、双对象正常维护/恢复和137项部门选择导出 | 28/28通过，0重试，6.1分钟；失败读取不推进业务来源，恢复权限后关系及状态筛选恢复 | 通过 | [最终28项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/browser-final.txt)、[最终输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/runtime-inputs-final.json) |
| 最终静态及构建 | make check、make build | 73项Python、19项Node、Go单元/vet、导入边界、TypeScript与构建通过；Go缓存与Vite既有提示按日志保留 | 通过 | [check](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/check-final.txt)、[build](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/build-final.txt) |
| 资源及保护核验 | 比较原文件摘要、运行输入、Docker身份、临时端口及配置 | 3377份原验证文件不变，1412运行输入匹配；7个临时环境回收，原Docker身份集合无增减 | 通过 | [资源与范围核验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/scope-and-cleanup.json) |

首批37项与最终28项是不同输入、存在重叠的两批，不能相加声称65项全量单轮通过。首批BU日志权限/筛选/历史/焦点回归后，产品只追加部门关系控件补修；最终28项再次覆盖BU状态筛选及共用维护保护。

## 4 失败、修复与复验

R5-F06（P1）：关系候选拒绝只调用详情重读，未先清理编辑与父详情。现改为复用共用字段清理入口，以空字段范围清除受保护投影、输入及重试正文，再尝试详情重读；成功后按真实响应恢复。补查成功重读发现同一受限上下文在候选403/详情200间自动循环，控件现保留该上下文的拒绝，仅显式重试或拟查询上下文变化后再请求；成功清除拒绝。原重试按钮被下拉弹层遮挡，现移入候选弹层并通过真实鼠标点击验证，不使用force绕过遮挡。

R5-F07（P2）：日志变更字段选项曾无条件保留state；现依据对象statusReadField和日志响应readFields判断，部门使用status字段权，BU保留自身合同。两种撤权参数分别证明只有历史status被撤销、以及普通/历史status一起被撤销时均不提供状态筛选；名称筛选继续200，恢复授权后状态筛选200。

评审首次关系探针误以编码定位名称链接而超时，修正评审夹具后复现产品问题；[原夹具](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/reviewer-probes-first.ts.txt)和首次输出保留，不计为产品缺陷。

本轮首次BU对照用例通过API新建样本后仍在同一路由，列表未刷新；已改为正常菜单按编码查询后进入详情，[首次夹具](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/browser-tests-first.ts.txt)及失败输出保留。第一次make check因新报告缺少导航入口失败，补齐验证入口后首批check通过，见[首次检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/check-first.txt)及[复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/check-first-batch.txt)。这些失败不计产品缺陷；循环和按钮遮挡按真实产品反例处理。

## 5 未覆盖范围与限制

本轮不改变Go、SQL、公开API或业务规则；不重跑完整DB/API、全量浏览器、视口/原生缩放或受控跨日。原完整验证保持原基线，不能计为本轮执行。R6容量、生产、远程CI与人工签收不在范围内。

## 6 结论与后续事项

主代理实施与补验确认R5-F06/P1、R5-F07/P2已关闭，成功重读的循环与鼠标重试阻挡也已修复，121追加R5-11/12。结论限定于本报告的实际反例、恢复及双对象回归，不代表另一轮独立评审、人工签收、全量产品或生产验收。本轮未提交、推送或发布。

7个自有临时profile的容器、网络、数据卷、Web/API/DB端口与私密配置已回收。Docker恢复原45个容器、3个运行容器、36个网络、59个卷，身份集合无新增或缺失。m02-department-b仅作只读status核验，数据库健康、Web/API运行；未向其写入业务或授权样本，也未迁移或重建。该持久环境继续供用户日常观察，由wt-dev-b维护者负责，下次模型或环境维护时复查。原R5及首轮修复报告和3377份既有验证文件保持原字节，验证导航只追加本轮入口。

## 7 证据保存

本轮证据位于[data/116-r5-authorization-followup](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup)，正式日志使用.txt；保留评审首次失败、本轮反例和后续复验的独立输出，不覆盖既有验证文件。[产品增量](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/product.patch)、[最终运行输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/runtime-inputs-final.json)、[命令退出状态](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/command-results.json)和[脱敏失败截图索引](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/failure-screenshots.json)可复核。最终文档检查见[docs](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-authorization-followup/docs-final.txt)。
