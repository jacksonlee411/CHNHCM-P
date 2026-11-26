# CHNHCM-122 操作摘要与日志合同评审修订

登记信息：[registry.json](../governance/registry.json)的governance.verification_records。适用范围：用户采纳的三项122评审意见及其对应011—013/023合同和案例增量。上游为[122草案](../archive/plans/122-bu-operation-summary-and-change-log.md)、[BR035](../specs/m02/011-m02-business-requirements.md#m02-br-035)、[012日志表格](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-change-log-table)、[013读取合同](../specs/m02/013-m02-technical-design.md#m02-tech-operation-summary)及[023案例](../specs/m02/023-m02-timeline-test-cases.md#m02-tests-operation-summary)。本文从验证模板起草，另建记录，保留[初稿评审](122-operation-summary-contract-review.md)及其失败/复验证据。

## 1 验证对象与受测基线

2026-11-04，工作树CHNHCM-wt-dev-b、固定分支wt-dev-b，HEAD为`dbd40154087c0e1faa9a41f271ee2f78dc029cb9`。开始时已有上一轮未提交的122文档交付，暂存区和stash均为空；16份原合同/登记/来源文件与上轮final-inputs摘要一致。用户本轮授权完善三项意见，然后提交整个122文档交付到本地分支。

[修订前快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract-review-fixes/before-contracts.zip)保存这16份文件；[开工记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract-review-fixes/before-inputs.json)持有状态、快照摘要及2450份需保护的旧报告/来源摘要。本轮只修订计划、规格、导航及登记，新增本报告和证据，不修改产品代码、OpenAPI、生成客户端、原报告或其证据。受测输入以本轮最终摘要为准，不能用HEAD或原轮摘要替代本轮脏工作区。

## 2 环境与验证边界

仅使用本工作树Python标准库、Git及仓库文档检查器。三项修订由Codex主代理按业务规则、UI展示、请求/响应和独立预期逐项复核，不是独立评审或人工业务签收。未启动Web/API、PostgreSQL、浏览器、容器或数据卷；没有本轮运行资源需要清理。既有开发与PeopleSoft环境未操作，前轮实机/网页依据仅保留，不登记为本轮重新验证。

## 3 检查过程与结果

| 检查 | 实际结果 | 证据及结论边界 |
| --- | --- | --- |
| 三项合同语义复核 | 通过；均已落到唯一合同和独立预期 | [复核矩阵](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract-review-fixes/semantic-review.json)，产品结果仍未运行 |
| make docs | 通过；73项Python测试及仓库结构/关联检查通过 | [文档输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract-review-fixes/docs.log)；测试夹具的profile文本不是实际启动服务 |
| 固定分支及引用 | 通过 | [引用输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract-review-fixes/refs.log)，定稿输入另做最终核对 |
| 原件、范围、稳定ID及案例状态 | 通过；2450份原报告/来源原字节保持，旧报告登记不变；原366行不变，新增20例未运行，386/342/44计数不变 | [范围与保护](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract-review-fixes/scope-and-protection.json)；仅修改新增案例372/373/375/377/380/382 |
| 差异空白检查 | 通过 | [差异核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract-review-fixes/diff-check.json)，新增文本另查，最终提交还需暂存核对 |
| Go/API/DB、浏览器及20个新增案例 | 未运行 | 本轮仅修订合同，不运行make check/integration/smoke，不计产品通过 |

## 4 评审发现与修订

| 发现 | 合同修订及独立验收预期 |
| --- | --- |
| 122-R01：日志DTO只有引用/差异，缺少生效记录和未变化分类的展示数据 | 013定义ChangeRecordContext、beforeContext/afterContext及与引用的对应关系，012直接使用投影；字段按当前对象/历史读权交集裁剪，空值和失权分开。S5只改中文名仍带06-01/10-01及原分类，但changedField只命中name；TL-372/375/377覆盖字段裁剪及不逐行补查 |
| 122-R02：服务端无法识别历史S4页面用S5作为上界的越界请求 | 013显式定义contextSourceVersion和throughSourceVersion的默认、回显、同BU验证及source_no关系；历史S4/上界S5为400，当前模式/S5允许，历史刷新仍S4，返回当前才取头。012及TL-380/382补分页、筛选、刷新、切换/返回及迟到响应保护 |
| 122-R03：S7仅给日期，默认继承会成为no_change | S7明确expectedSourceVersion=S6、anchorRecordId=R1、2026-06-01、CHANGE/NAME_CHANGE及名称“华东四”。先用空changes和独立提交身份核对no_change，再以新提交身份实际更名插入并取得R4；TL-373验证R4与被删R2不同，S5仍为“华东二新”原件 |

上述修订只关闭本次文档发现。手写的请求结果/上下文例子是待执行预期，不能作为真实接口响应证据；旧报告的原结论与原始输入保持。未因文档完善而扩大领域动作、重开旧联合写入口或改变部门独立表边界。

## 5 未覆盖范围与限制

122维持draft/not_started，P1—P3产品实施未开始。此次没有新增稳定ID，023仍为386个定义、342现行、44退役；TL-367—386全部未运行，原366条案例行保持。未验证X03身份快照、Go/SQL读模型、真实字段授权竞争、分页性能、页面行为或生产环境。116 R1—R6及部门真实接入未启动。

本报告不重审全M02或上一轮外部产品提交，不把原行业观察、旧通过记录、文档检查或局部合同复核当作产品验收。提交仅保存文档交付，不授权远程推送、PR、合并或部署。

## 6 结论与后续事项

三项评审意见已在合同层闭合，文档结构/引用、范围、案例及原件保护和差异检查通过。用户已授权本地提交122文档与修订；提交前核对全部暂存路径、差异与旧证据，实际提交号由交付回复给出，不在报告中嵌入报告自身的最终提交号。

后续依122 P1/P2/P3实施并另建产品报告，保留本报告和未运行参数。本轮未创建或接管运行环境，资源收尾不适用；既有持久数据和他人资源保持。

## 7 证据保存

本轮证据存于[data/122-contract-review-fixes](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract-review-fixes)：before-contracts.zip为修订前的16份资料原件；before-inputs.json为状态、原登记和保护摘要；semantic-review.json为三项发现及独立预期。[本轮输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract-review-fixes/final-inputs.json)标识修订后的合同/登记/导航，报告与输出自身不做递归摘要；[检查汇总](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract-review-fixes/validation.json)、scope-and-protection.json及diff-check.json记录实际结果。原[data/122-contract](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract)及初稿报告保持原字节，其旧final-inputs只标识原轮合同，不能要求与本轮修订后内容相等。
