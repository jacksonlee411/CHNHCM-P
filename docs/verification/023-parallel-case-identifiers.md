# 并行时间轴案例编号消歧记录

登记信息：[registry.json](../governance/registry.json)的同路径verification_records。关联[023](../specs/m02/023-m02-timeline-test-cases.md)与[014](../governance/014-document-management.md)，仅修正当前定义和引用，不增加业务范围。

## 1 对象与基线

a输入57cf1366b08842e6060661a4abc4c44b7e397bf9，b输入1009a93f6d5384b737d0516520b8ff01c8333c87；共同基线94dd06466ffc44ee233e600d99f3bb82218a143b。b于3aac1c2先定义列表TL323—334，a于a27caf8又定义单条启停TL323—328，b随后增加TL335—342。合并前发现同号不同义，保留先定义的b编号，给a六项分配343—348。

## 2 来源映射与历史保护

下列旧编号仅指a提交57cf136的单条启停定义，不能不带来源解释成b列表用例。

| 来源a旧编号 | 当前编号 | 内容 |
| --- | --- | --- |
| a@57cf136 / TL323 | [TL-343](../specs/m02/023-m02-timeline-test-cases.md#m02-tl-343) | 所选末条可办性 |
| a@57cf136 / TL324 | [TL-344](../specs/m02/023-m02-timeline-test-cases.md#m02-tl-344) | 初始值及取消 |
| a@57cf136 / TL325 | [TL-345](../specs/m02/023-m02-timeline-test-cases.md#m02-tl-345) | 日期范围与历法 |
| a@57cf136 / TL326 | [TL-346](../specs/m02/023-m02-timeline-test-cases.md#m02-tl-346) | 未来末条状态 |
| a@57cf136 / TL327 | [TL-347](../specs/m02/023-m02-timeline-test-cases.md#m02-tl-347) | 自动业务反馈 |
| a@57cf136 / TL328 | [TL-348](../specs/m02/023-m02-timeline-test-cases.md#m02-tl-348) | 版本权限及历史保护 |

原021报告逐字节保存在[原件](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/023-parallel-case-identifiers/original-report.md.txt)，[摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/023-parallel-case-identifiers/preservation.json)记录来源与SHA-256。当前阅读入口仅修正引用并增补勘误；原数据、截图、命令结果及执行结论未改写。b历史报告保持原件，合并时保留其323—342定义。

本支实际328项（284现行、44退役），最大编号348不等于已定义348项；合入b的20项后才能计348项（304现行、44退役）。上述数量均不是业务通过数。

## 3 检查与边界

初轮make docs因结果尚未登记失败，保留[初轮输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/023-parallel-case-identifiers/docs-initial.txt)；补齐后73项测试与登记检查通过，见[复验输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/023-parallel-case-identifiers/docs-final.txt)。原报告与57cf136逐字节一致，SHA-256已核对；git diff --check通过。此轮不重跑业务；PR三项CI必须按最新提交全部通过才允许合并。主代理复核，不冒充独立评审或人工批准。
