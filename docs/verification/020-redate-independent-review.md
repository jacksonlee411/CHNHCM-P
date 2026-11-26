# CHNHCM-020 T06独立评审与修复

登记信息：[registry.json](../governance/registry.json)。范围：冻结T06提交的独立代理评审、发现项修复与实际复验。依据[010](../archive/plans/010-m02-scope-and-plan.md)、[011](../specs/m02/011-m02-business-requirements.md)、[012](../specs/m02/012-m02-functional-and-interaction-design.md)、[013](../specs/m02/013-m02-technical-design.md)、[023](../specs/m02/023-m02-timeline-test-cases.md)及024/025。原[技术验证](020-redate-implementation.md)保留当时结果，不覆盖其中的待授权记录。

## 1 对象与基线

2026-11-02，用户明确授权单独独立评审代理。评审对象为93fcf32013e995282ae3808ac2443e3401a51470，父提交3e724797dc788cdaf4c97e4624949991b020f60a；固定工作树CHNHCM-wt-dev-a、分支wt-dev-a，开工工作区干净、stash为空。本轮基线外变化按发现项和受测差异记录。

## 2 评审身份与环境

独立代理任务为/root/t06_independent_review，与实施者分离；代理只读受审代码、文档和索引，允许任务独立的非敏感临时验证。代理评审不冒充独立人工批准。授权和冻结对象见[会话记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-independent-review/review-session.json)。

原T06环境m02-t06-current，Web4486/API8486/DB54566；凭据仅仓外受保护目录。额外验证采用隔离Go-overlay或Testcontainers；浏览器操作与实施者协调，不并跑会话或更改共享权限。

## 3 评审范围与结果

检查首条、日期排序、完整快照、准确停用继承、四类专用权与日期字段权、同值及并发、严格输入、历史与原结果、事务取消及COMMIT待确认、正常菜单与恢复、TL映射和证据边界。第一轮确认两项P2需修改；修复后第二轮独立复审通过，未发现新增问题。

## 4 发现、修复与复验

| 发现 | 独立触发与影响 | 修复及实际复验 |
| --- | --- | --- |
| P2 / T06-R1 | 6月1日改7月1日，再改8月后，首次接受、原内容重试和resolve均缺previousEffectiveDate，违反013第4.3节。原新日仍正确，不能据此认定原结果完整 | Result/OpenAPI/客户端补旧日；首次从准确基线目标取得，恢复从原source_version的准确target_record_id/target_revision_id读取，不新增冗余存储列。按effectiveDate读权同时裁剪原/新日。独立复验原三路径及目标后来删除、撤写权、撤日期权、撤历史权均通过 |
| P2 / T06-R2 | 正常菜单先制造来源409，再撤日期读权、显式重新核对。输入框隐藏但旧date留在状态，可继续把缓存7月1日预览为accepted。现有响应裁剪测试未覆盖显式reload | 两种重读路径共用clearUnreadInputs，清受限日期/原因/名称、触碰字段和旧重试正文，保留仍可读输入；清旧字段诊断。日期不可读时给可取消编辑反馈。永久浏览器补preview/reload两种路径，独立正常菜单重演后预览和确认都禁用，原/新日期正文清除，授权finally恢复 |

第一轮原结论及9项原始证据：[round1-review.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-independent-review/round1/round1-review.json)。原始字节与原SHA保存在同目录independent-evidence.tar.gz，可读日志副本仅移除行尾空白。第二轮原结论及7项新证据：[round2-review.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-independent-review/round2/round2-review.json)；其中8项受审实现/测试SHA在主代理交付前逐一核对一致，第一轮原件SHA全部保持。

| 检查 | 实际结果与证据 |
| --- | --- |
| 第一轮独立检查 | Go改日全部规则、真实DB/API四根及bootstrap跨午夜/提交边界通过；两条新反例仍失败，不将既有测试通过等同全功能通过，见第一轮日志与结论 |
| 第二轮独立复验 | 2条隔离Go-overlay/API测试与真实正常菜单撤权时序通过；R1/R2关闭、无新增发现，见第二轮结论。代理没有冒认主代理运行的检查 |
| make generate / make check / make build | 当前格式生成、静态/边界/Go/vet/前端类型、构建通过，见[生成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-independent-review/generate.log)、[检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-independent-review/check.log)、[构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-independent-review/build.log)。既有前端主块大小提示保持 |
| make integration | 当前修复下两个真实DB包完整通过，见[集成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-independent-review/integration.log) |
| 当前T06 Chromium | 定向9/9通过，含新增显式reload撤日期读权路径及丢响应/迟到/冲突/未保存回归，见[浏览器](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-independent-review/browser-fixes.log)。未重跑全套70；原69条和七组视口按未改变的布局及正常分支复用，原边界见技术报告，不能称本轮独立重跑全套 |
| 文档及差异检查 | 完成状态、当前运行及023补验入口同步后通过，见[封存检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-independent-review/docs-final.log) |

本次不改日期候选算法、迁移和存储合同；运行环境仍为m02-timeline-5，旧源码/报告保留原证据。

## 5 未覆盖范围

不启动T08联合、首条补全或分类纠错，不运行T07全容量。完整网络COMMIT未知、生产恢复及真实外部消费者仍按原范围后置。

## 6 结论与计划状态

2026-11-02，单独代理第二轮结论为passed，R1/R2均关闭、无新增发现。所评修复源码及测试与交付输入一致，主代理必要复验通过，T06本批完成。010仍为M02进行中，T01/T02分类/联合部分、T08及T07继续保留；本次在T06停止，不自动启动下一批。独立代理结论不冒充人工业务批准。

## 7 证据保存

必要原始独立证据、修复复验、受审源码和受测输入摘要保存到本报告data目录；原始证据两轮分别tar.gz封存并保持原SHA，可读日志副本仅整理行尾空白。本机原始临时输出位于.local/verification/t06-independent与.local/verification/t06-independent-closeout。旧定稿技术报告及登记条目不覆盖，旧报告/来源及资产保护另留核对结果。

交付前核对：[受审源码摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-independent-review/tested-inputs.json)、[运行程序匹配](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-independent-review/runtime-match.json)、[旧报告/来源保护](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-independent-review/history-protection.json)与[资产补验入口保护](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-independent-review/asset-protection.json)。原始tar内文件与代理原SHA一致，归档展示日志的整理不改变原始证据。
