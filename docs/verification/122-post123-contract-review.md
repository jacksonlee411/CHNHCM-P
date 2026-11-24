# CHNHCM-122 123实施后的合同复核与完善

登记信息：[registry.json](../governance/registry.json)的governance.verification_records。适用范围：用户采纳的122删除原件归因、未来首条空态及关联基线/案例/计数修订。上游为[122计划](../plans/122-bu-operation-summary-and-change-log.md)、[BR034/035](../specs/m02/011-m02-business-requirements.md#m02-br-034)、[012摘要](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-operation-summary)、[013技术合同](../specs/m02/013-m02-technical-design.md#m02-tech-operation-summary)、[023案例](../specs/m02/023-m02-timeline-test-cases.md#m02-tests-operation-summary)及[123决定](../adr/123-current-maintenance-and-dated-query-boundary.md)。本文按验证模板另建记录，保留初稿、三项评审修订及123报告原件。

## 1 验证对象与受测基线

2026-11-04，CHNHCM-wt-dev-b固定wt-dev-b，HEAD为833e71e852a188bea5a49c3a22664a817621c8f6。开始时工作区、暂存区和stash为空；用户授权按本次复核建议完善文档。受测对象是该提交加本次计划、011—013/023、登记和导航修订，不包含产品调整。开工文件摘要及2526份原报告/来源保护清单见[before-inputs.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-post123-review/before-inputs.json)。

基线代码已取消普通查看日期，但仍展示来源级actor/recordedAt及条目式日志，未提供122的operationSummary或OperatorSnapshot。当前删除来源＋被删ID读取墓碑修订；普通详情无今日事实时会默认选未来首条。这两处现行行为是本次合同衔接的输入，不计作122功能已经实现。

## 2 环境与验证边界

仅使用本工作树Git、Python标准库及文档检查器。由Codex主代理按用户采纳建议复核规则、页面、DTO和独立样本，不是独立评审或人工业务签收。未启动或接管Web/API、数据库、浏览器、容器、数据卷或PeopleSoft环境；本次资源收尾不适用，不为文档验收另启服务。

## 3 合同发现与处理

| 发现 | 本次合同处理 | 独立预期与后续验证 |
| --- | --- | --- |
| 122-R04：删除证据修订的形成操作者可能冒充删除前资料操作者 | 013明确沿墓碑previous_revision_id取删除前业务修订，并校核前来源成员；摘要的revision/operationSourceVersion指该业务修订。selectedRecord的删除证据revision、resolvedSourceVersion及旧终态保留，不增加第二套DTO或客户端推断 | S6＋R2仍有删除证据R2.4，摘要为R2.3/S5/A；日志S6/B。P4两条分别归因P2/A和P3/B。TL-373覆盖前来源、结果入口、复制链接和固定两条；TL-381覆盖失权 |
| 122-R05：无今日事实被误解为无可展示记录 | 011—013按123区分viewFact与selectedRecord；未来首条即使M02_NO_FACT仍展示资料和摘要，日期失权不泄露未来提示；显式缺失不自动换条 | F3默认选F-R1.3/F3/B；原件摘要F-R2.1/F2/A；日期失权仍有合法摘要，缺失ID无摘要，失去历史权拒绝原件。TL-367/373/375/381补充参数 |
| 实施基线与案例计数滞后 | 122更新到123完成后的833e71e基线，保持D0→P1→P2→P3及BU先交付；登记表023改为394编号/350现行/44退役 | 不新增稳定ID，不提升20例或新增参数状态；123及116产品边界分别保持 |

摘要的业务修订与删除证据修订有意分开，非删除原件仍按所选业务修订取值。固定两条样本让两条原修订来自不同来源，防止把删除前来源的操作者统一套用；F样本同时核对派生前驱与被删原件，不能将删除者从所有摘要中一概排除。

## 4 检查过程与结果

| 检查 | 实际结果 | 证据及边界 |
| --- | --- | --- |
| 两项合同语义自审 | 通过，122-R04/R05在合同层闭合 | [复核矩阵](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-post123-review/semantic-review.json)，S/P/F均为独立预期，产品未运行 |
| make docs | 通过，73项Python测试及结构/链接检查 | [首次输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-post123-review/docs-first.txt.gz)，脚本夹具输出的profile名称不表示启动了真实产品环境 |
| 固定分支及引用 | 通过 | [引用输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-post123-review/refs-first.txt.gz) |
| 原件、案例及范围保护 | 通过，2526份原报告/来源原字节保持，旧报告登记不变；仅TL-367/373/375/381四行扩充参数，其他390行保持 | [保护与范围](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-post123-review/scope-and-protection.json)；全表394/350/44，122的20例仍未运行 |
| 123受测输入核对 | 61个产品/测试文件摘要全部与123报告一致 | 同上；只证明文件未变，不是本轮重跑123 |
| 差异及新增文本检查 | 通过 | [差异检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-post123-review/diff-check.json) |
| Go/API/DB、浏览器及122业务案例 | 未运行 | 本轮为合同修订，真实验收留P1—P3 |

报告及登记定稿后，另运行最终文档与引用检查，实际命令及退出码统一记入[检查汇总](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-post123-review/validation.json)；不将前一次输出冒充修改后的受测输入。

准备保护清单时，首次git文件列表因默认转义中文路径导致采集失败，未生成快照或改动原件；改用ls-files -z后重新采集成功，重试事实保存在开工记录。该准备错误不属于产品测试失败。

## 5 未覆盖范围与限制

122维持draft/not_started，P1—P3及116后续阶段均未开始。TL-367—386的20例及本次新增参数全部未运行。S/P/F预期是合同样本，不是真实API响应或浏览器结果；本轮不运行make check、integration、smoke、性能或生产验收。123已有产品证据用于核对当前基线，不能借此宣称122通过；外部行业来源和PeopleSoft不重新观察。

没有修改产品代码、数据库、OpenAPI、生成客户端、资产来源或旧验证报告。原不可变资料、终态与前轮失败/复验证据继续保护；新合同不回填旧身份，不恢复旧通用联合写入口，也不承诺部门已接入。

## 6 结论与后续事项

两个边界已在本次合同范围内闭合，文档结构/引用、原件保护、案例状态、计数及差异检查通过。011—013/023登记本次限定审阅范围，122仍为草案且产品未开始；下一步按P1服务端、P2页面、P3验收执行，后续另建产品报告。本轮仅完善文档，未执行本地提交或远程操作。

本次没有创建运行环境，无需清理容器或数据卷；既有持久开发与PeopleSoft环境未操作。

## 7 证据保存

本轮证据存于[data/122-post123-review](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-post123-review)。开工记录保留基线、原登记与文件摘要，[定稿输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-post123-review/final-inputs.json)标识本次9份合同/登记/导航文件，[检查汇总](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-post123-review/validation.json)记录实际命令结果；报告与检查输出不递归计算自身摘要。所有原报告及其证据保持原字节，旧final-inputs仍只标识旧轮输入，不要求与本次修订合同一致。
