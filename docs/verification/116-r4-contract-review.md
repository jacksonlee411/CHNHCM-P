# CHNHCM-116 R4实施设计与合同评审

登记信息：[registry.json](../governance/registry.json) `governance.verification_records`。
适用范围：用户采纳的R4四包设计、精确存储/类型端口/授权输入/依赖锁/候选/路径证据，以及真实DB/API验收映射；不含产品实现。
上游依据：[116](../archive/plans/116-m02-shared-dated-maintenance-roadmap.md)、[117](../adr/117-m02-dated-maintenance-and-department-boundary.md)、[118](../specs/m02/118-department-business-requirements.md)、[119](../specs/m02/119-department-functional-design.md)、[120](../specs/m02/120-dated-maintenance-technical-contract.md)、[121](../specs/m02/121-dated-object-admission-matrix.md)、[122](../archive/plans/122-bu-operation-summary-and-change-log.md)、[123](../adr/123-current-maintenance-and-dated-query-boundary.md)。

## 1 验证对象与受测基线

2026-11-05，工作树CHNHCM-wt-dev-b，固定分支wt-dev-b，HEAD为03078144666924896077471c290c25dff6c44c5d；起始工作区和stash为空。用户本轮授权完成R4设计，未要求产品实施或提交。修改为116/119/120/121、013/023的设计及交接增量、登记/导航和本记录/非敏感设计证据。

[基线](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-design/baseline.json)保存4281项产品输入及原验证文件摘要；当前运行代码、DDL、OpenAPI和生成客户端保持，原R0—R3及122/123报告不回写。文档输入清单与实际检查结果分别保存；4281项产品和旧证据静态摘要核对通过。

## 2 环境与前置条件

主代理对照当前源码、迁移、API、合同与手工样本进行设计和复核，非独立评审或人工业务验收。未启动服务、数据库、容器、浏览器或持久profile，003运行资源收尾不适用；没有为文档检查创建运行环境，也未操作已有环境。

当前代码事实：dated.Calculate已做无变化消除/影响计算，尚无关系定稿阶段；Access只匹配enterprise/bu及BU字段；BU引用范围尚未接入部门；OpenAPI无部门端点。因此120明确列出待实现扩展，不能把设计端口或合成命名空间检查称为真实部门实现。

## 3 验证过程与结果

| 检查 | 过程与预期 | 实际结果 | 证据 |
| --- | --- | --- | --- |
| 四包和退出范围 | A/B/C/D责任、基础保护前置、R4后端与R5菜单/R6容量区分 | 语义复核通过（设计范围） | 116第4.5节 |
| 存储与历史 | 全部独立表列/键/外键、准确成员、封口、路径段/节点/边、终态组合 | 语义复核通过（设计范围） | 120第3节 |
| 类型端口与122 | 上下文/原因/辅助/日志/摘要、严格请求、基本读取与历史读取分离 | 语义复核通过（设计范围） | 120第4.4节、119及013 |
| 候选与授权/锁 | Prepare/Finalize、关系重算、精确引用、完整Grant、单次D、缺失根和依赖重规划 | 语义复核通过（设计范围） | 120第2.4/5.4/6.4节 |
| 验收映射 | 原39例不改，新增16组后端技术参数，独立预期及全部not_run | 62项静态核对通过，产品仍not_run | [映射](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-design/acceptance-map.json) |
| 文档/历史保护/格式 | make docs、输入摘要与git diff --check | 通过：73项脚本检查、238份Markdown；diff格式检查通过 | [文档检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-design/docs-validation.txt)、[保护核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-design/mapping-and-protection.json) |

## 4 发现、修订与复核

设计阶段识别并处理以下内容：现有Calculate提早定稿，需统一内核两阶段扩展；部门权限需要Access目标/范围/字段同步；共同守卫不能等到第四包才保护首个写入；路径需要准确分段/节点/边证据。正文的这些合同仍等待产品实现，不能登记为已修复运行缺陷。

编写中对照源码移除多余expectedAnchorRevision，明确Idempotency-Key在请求头而非正文；校正122技术参数引用，将容量/浏览器参数保留在原阶段；摘要手工样本改为023既有S4预期，避免新增后前驱期间派生被误记为未变修订。另补全过去受影响BU依赖及修订到管理BU根的结构外键。14项语义复核及4项设计编写发现已关闭，见[逐项记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-design/semantic-review.json)。这些是设计修订，不声明运行缺陷已修复。

收尾包装首轮将git diff --no-index --check的新文件正常差异退出码1误判为失败；其输出无格式诊断，make docs和已跟踪文件diff检查均通过。已保留[首次结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-design/closing-check-first.json)，用临时干净/尾空格样本核实退出码分别为1/3及诊断差异，见[核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-design/format-check-probe.json)；修正为只接受0/1且无诊断后重验。临时样本目录已回收，合同和产品未因此改动。

## 5 未覆盖范围与限制

未运行Go/DB/API/浏览器或容量产品测试，未修改或执行迁移、OpenAPI生成和客户端接线。D01—D39、R4-V01—V16、TL-394及部门122参数全部not_run。R4产品四包、R5菜单和R6容量尚未开始；报告中的结构检查不代表部门可用、独立评审或人工签收。

## 6 结论与交接

R4共同设计和验收映射语义复核、文档门禁通过，未发现本次范围内尚未关闭的设计问题。实施依据为116四包、120当前设计与121映射；产品结果只能在隔离真实环境逐包执行后登记，不能以本轮设计结论替代。

## 7 证据保存

[基线及历史保护输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-design/baseline.json)、[验收映射和固定根身份向量](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-design/acceptance-map.json)。[静态映射及保护核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-design/mapping-and-protection.json)、[语义复核](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-design/semantic-review.json)。[文档检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-design/docs-validation.txt)、[最终输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-design/final-inputs.json)、[收尾核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-design/closing-check.json)。源样本原件仍为116-r0及116-r0-review-fixes两份contract-fixtures.json，本轮仅引用和校验摘要。
