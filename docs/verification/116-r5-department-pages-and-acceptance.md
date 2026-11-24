# CHNHCM-116 R5部门页面与双对象验收

登记信息：[registry.json](../governance/registry.json)。依据[116路线图](../plans/116-m02-shared-dated-maintenance-roadmap.md)、[119功能设计](../specs/m02/119-department-functional-design.md)及[121验收映射](../specs/m02/121-dated-object-admission-matrix.md)。

## 1 范围与基线

wt-dev-b，545860a40f9c2c2571258ea4a1a9ba768cea9e96，起始工作区、暂存区干净。用户授权完成R5实施及评审验收；本轮未提交、推送或发布。R5-A接通部门正常菜单、列表详情、关系候选、路径、全部标准维护、日志和恢复；R5-B用真实DB/API与浏览器验证两种对象的共用流程及对象边界。R6容量、生产交付、远程CI及人工签收不在范围内。

## 2 实施与逐批复核

### 2.1 R5-A页面及端口

BU和部门分别拥有页面定义、字段目录、类型API序列化及提交命名空间，共用DatedPages、TimelineEditor、LifecyclePairDialog和日志、读取/导航/选择/离开/恢复机制。Department没有复制BU页面，也没有转成BU DTO。Go继续决定日期、授权、关系资格、候选、当前范围和路径；没有新增前端业务日历、存储迁移或整树版本。

正常入口为“设置→组织配置→部门”。列表保留筛选、返回、选择与CSV；详情按所选记录显示基本资料、组织关系、记录信息及操作者摘要。关系控件查询准确拟动作、日期、分类、来源和差量，使用稳定ID、分页候选与会话/查询归属检查；改BU不自动清空上级。路径支持当前分段、未来首条、停用前路径及显式来源封存，链接按当前详情权决定是否可打开。

后台增量限于Go提供路径最后适用日、祖先当前详情链接权、名称输入规则，以及仅用于列表筛选的BU引用读取模式。filter模式只返回获权标签及readable，不冒用维护资格；对象维护继续使用原create/maintain上下文。OpenAPI、生成类型、119/120/121和三类资产的限定采纳同步。

### 2.2 R5-B真实业务与边界

第一批两对象正常菜单旅程覆盖创建、普通更正及改日、位置新增、当前/准确来源、启停和删除。第二批覆盖固定两条、退出503后新草稿保护、未来首条丢响应查回、根/子部门及跨BU重选、原上级失格后的恢复、拟分类/日期候选刷新、迟到查询、122摘要独立日志权及撤字段权。两种对象使用同一个提交号的待确认记录分别恢复；切换主体不能读取原主体待确认引用。

选择批次使用137个真实部门，全结果选择、跨页排除一项、改页大小后136项CSV；校验八列、路径、稳定BU ID及BU更名、祖先更名和撤导出权。跨日专项通过隔离Go Clock与真实服务，在上海和洛杉矶浏览器时区检验未来归属不提前授权、跨日转出使旧选择/资料/来源失效。视口检查覆盖七类页面的桌面、390/320、短屏、键盘和原生Chromium200%/400%缩放。

## 3 评审发现与修复

本次为主代理实施后的范围复核，不冒充独立评审人或人工批准。按正常任务、字段可发现性、权限/来源边界、共用机制实际接线、证据与资源逐项检查。合同增量复核限定于116第4.6节、118执行口径、119第8节、120第10节、121第10节及023第30节；不重写历史合同评审结论，不将本轮结论扩展至R6。

| 编号 | 问题及影响 | 修复及复验 |
| --- | --- | --- |
| R5-F01，P2 | 部门页面忽略不支持的adoptedRef，可能误把当前资料当作指定采用结果 | 部门适配器明确拒绝、清除旧投影并提供返回当前资料；真实页面加入直达参数反例 |
| R5-F02，P2 | BU筛选复用详情标签布局，600px弹窗输入框被压缩，无法正常点击 | 筛选使用独立输入标签，目录筛选用有界网格；137项真实选择/导出复验通过 |
| R5-F03，P2 | 共用新建页在上下文403后隐藏名称框，偏离BU清空、禁用并可恢复的既有交互 | 恢复空白禁用字段；原3项BU回归在新完整轮次通过 |
| 验证资产修订 | 首轮编码定位、旧操作原因文字、路由/未保存弹窗等待及窄屏菜单定位不正确；截图助手拒绝覆盖已有文件 | 修正测试操作和等待，截图按临时profile单独保存；首次失败及中断均保留 |
| 编译中断 | 初始关系组件存在多余闭合标签，浏览器第二轮被主动中断 | 修复编译后typecheck及实际菜单复验通过；中断环境已回收 |

R5-F01—F03已按第4节结果关闭，无未关闭产品发现。原始失败分别见[browser-first](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/browser-first.log)、[中断](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/browser-second.log)、[边界首轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/browser-boundaries-second.log)、[扩展首轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/browser-expanded.log)及[布局选择复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/browser-layout-selection-retest.log)。[第一次完整浏览器](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/browser-full.log)在确认F03后主动中止，25项通过、3项失败，后续未执行，不记完整通过。未正确引用的shell筛选参数导致一轮未启动测试，见[browser-boundaries-first](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/browser-boundaries-first.log)，残留的自有临时profile已销毁。后续原生缩放超时留下的自有Xvfb也已停止，测试收尾改为先终止显示服务。

## 4 实际验证

最终完整DB/API运行退出0：156个顶层、含子测试480项通过，4项专项跳过，见[统计](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/integration-final-summary.json)、[原始事件](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/integration-final.jsonl)及[退出码](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/integration-final-exit.json)。R5受控Clock浏览器包含在该完整运行中，真实服务分别验证上海/洛杉矶时区。静态/单元/边界/typecheck及构建已通过，见[check](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/check-final-inputs.log)与[build](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/build-final-inputs.log)；最终文档检查见[docs](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/docs-final.log)，差异检查结果见[命令状态](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/command-status.json)。构建保留既有大chunk提示，未计入容量验收。

完整Chromium实际执行205项：204通过、1项响应式布局等待失败，见[完整日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/browser-final.log)与[统计](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/browser-summary.json)。工具外壳返回143时，原测试子进程仍继续运行；已观察完整终态，之后核销该临时profile。不将外壳退出记为0，也不将后续复验合并为单轮全绿。

首轮DB同样有480项通过和两包最终pass，但工具会话返回143，保留[首轮统计](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/integration-summary.json)，未登记退出0。当前完整浏览器中的响应式布局因断点导航等待失败，产品代码未改。修正测试等待导航模式后，两项独立布局复验通过：28组响应式、14组原生缩放，共42组，见[日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/layout-final-retest.log)、[矩阵](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/input-and-layout-review.json)、[响应式](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/layout-smoke-2edd5498a725/responsive.json)与[真实缩放](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/layout-smoke-2edd5498a725/native-zoom.json)。主代理查看了320长路径、桌面BU筛选及400%关系输入等截图；控件可达，长路径换行。测试输入仅该文件变化，产品输入不变；不把分批结果合并称为单轮全绿。

## 5 持久环境与历史保护

本分支保留m02-department-b，实际入口http://127.0.0.1:4382，API8382、DB54373。API已按当前代码重启，复用原数据库与卷。管理员和查询者分别从正常菜单读取5个既有部门，并切换R4HR的旧记录；未来R4DIG、停用R4PROJ和操作者摘要均实际可读，见[浏览器结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/runtime-browser.json)。没有向持久库新增验收数据。

28张organization业务表在重启前、重启后及浏览器读取后行数与排序行摘要一致，见[实施前](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/persistent-before.json)、[重启后](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/persistent-after-restart.json)、[读取后](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/persistent-after-browser.json)及[最终](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/persistent-final.json)。3130份既有报告和证据保持，仅验证导航README新增R5入口；核验见[历史保护](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/prior-evidence-check.json)。既有R0—R4及其他报告按[原证据清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/prior-evidence.json)核验，不改写原结果。旧m02-current-b与其他工作树持久环境不清空、不重建。

本工作树开发者负责4382日常观察；复查节点为本轮交接及下次模型/运行环境切换。本轮记录的427个Testcontainers及11个smoke临时profile均已回收，关联容器、网络和卷无残留；Xvfb和测试浏览器已停止。1409个最终运行输入及4个固定夹具摘要一致，唯一的布局测试修订已补验。完整资源/输入审计见[核销记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/final-resources-and-inputs.json)，长任务外壳和检查结果见[命令状态](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/command-status.json)。

## 6 结论边界

R5-A页面接入、R5-B双对象限定验收及合同增量复核通过，三项发现均已关闭；逐项映射见[评审记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5/review-and-acceptance.json)。R6接入模板/标准、容量及混合负载、跨域推广评估继续后置。4项未运行专项为BU负载、旧Joint午夜浏览器、T07午夜浏览器及长时间线容量；本轮R5双时区Clock已运行，不用它改写旧专项状态。不声明整个M02、生产可用性、远程CI或人工签收完成。
