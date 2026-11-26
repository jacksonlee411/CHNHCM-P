# CHNHCM-113 部门列表样板与列能力范围修订复核

登记信息：[registry.json](../governance/registry.json)的governance.verification_records。依据[113决定](../adr/113-bu-list-community-adoption.md)、[111计划](../archive/plans/111-bu-list-experience-upgrade.md)、[012功能合同](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-list-pro)及[014](../governance/014-document-management.md)，从[验证模板](../templates/verification.md)起草。

## 1 对象与基线

2026-11-03，wt-dev-b，HEAD为`5748df9a5882865002df563fc0615b19ca312901`。开始时已有上一轮Community文档未提交变更，前轮受测快照见[原摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-community-adoption-docs/final-inputs.json)；本次继续修订并按用户要求将前轮与本轮文档一并本地提交。前轮快照不覆盖本轮合同，本轮最终文档输入及摘要见[本次输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-department-list-template-docs/final-inputs.json)。运行源码、依赖和锁文件保持。

## 2 环境与前置

只读读取4180构建身份与来源清单，复用上一轮分析的1440×900/390×844截图及测量；九个关键组件与固定CubeBox提交一致。样板来自合成原型预设，不是CHNHCM业务验收，见[部门样板依据](../reference/113-department-list-template.json)。样板构建sourceDirty=true的身份与九个组件的固定来源分开登记，不把整个构建归为同一固定提交。

未启动或停止BU profile、容器、API/DB或Web服务；本轮未启动浏览器，既有4180服务保持。截图复用前已核对当前HTTP构建的htmlSHA256与上一分析一致。未访问、注入或保存密钥。

## 3 过程与结果

| 检查 | 操作及预期 | 实际结果 | 证据 |
| --- | --- | --- | --- |
| R1结构 | make docs检查登记/链接/格式/来源 | 65项工具回归及资料检查通过 | [R1日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-department-list-template-docs/round-1-docs.log) |
| R1语义 | 正向核对新裁决及现行概述/入口 | 2项遗漏：范围概述仍仅写撤销Pro、规格入口仍指前轮复核；本轮失败，后修复 | [R1发现](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-department-list-template-docs/round-1-semantic.json) |
| R2修复后复核 | 当前范围、T01/T02、样板边界、案例、来源与历史保护 | 32项辅助检查及语义自查通过，无未闭合发现 | [R2复核](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-department-list-template-docs/round-2-semantic.json) |
| R2工具及差异 | make docs、check-repository.py --check-refs、git diff --check | 全部通过，含65项工具回归及固定分支引用检查 | [命令结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-department-list-template-docs/round-2-checks.json) |
| R3反查 | 由合同/案例反查计划、当前状态及交付边界 | 7组追踪和32项辅助检查通过，无新增主要发现 | [追踪](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-department-list-template-docs/round-3-traceability.json)、[复核](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-department-list-template-docs/round-3-semantic.json) |
| 原件保护 | 对比本轮开始时文件、登记及资产采用批次 | 2326个既有运行/测试、来源及验证文件字节不变；旧验证登记、资产状态/采用批次保持 | R2/R3复核 |
| 定稿及提交检查 | 最终文本资料、引用、差异和输入摘要核对，再检查暂存范围 | 以实际命令结果登记，不预记提交或通过 | [最终检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-department-list-template-docs/final-checks.json) |
| 产品实施与验收 | 本轮只修订文档 | Community源码/依赖、类型/构建、真实Go/API/DB/浏览器和七组视口未运行 | 第5节 |

## 4 修订与复验

取消列拖动/固定、自研替代及额外列顺序入口，保留Community原生菜单/显隐/列宽/全部隐藏恢复。将部门页三段控制区、面板/密度、摘要列与采用差异纳入113/012/013，111细化T01协同接线和T02-A/B/C，TL-333不再要求已取消能力；源版本、未转译方式、A/D、链接、分页、Go权威和#066b68保持。

R1遗漏只修订当前概述和本次复核入口，历史进度/报告及其原结论不覆盖。修复后R2/R3无未闭合文档发现，辅助断言仅支持语义复核，不能代替人工批准或产品运行。本次是同一Community方向下的范围收敛，在尚未提交的113中持续修订，没有重选组件或业务模型。

## 5 限制

当前代码仍Pro，Community源码/依赖、类型/构建、真实Go/API/DB/浏览器及七组视口验收未运行。T01—T05产品任务及TL-323—334仍待实施验收，不因取消附加功能或文档检查通过计为完成。

部门树、层级、多选、负责人/路径/人数、导出及原型search JSON/useDemo/客户端全量切页不进入本批；原部门截图保留这些元素只作来源观察。手机原URL先打开部门抽屉，关闭后的摘要图另存；不是BU默认抽屉设计。列筛选T04继续评估，高级查询等保持后置。

本轮为主代理多轮自查，非独立评审或人工产品验收；未修改或重验第三方通用算法。前轮113及Pro报告、证据与采用条目原样保留，本次只追加范围修订和采用计划。

## 6 结论与后续

本次文档范围通过修复后的R2/R3复核，取消列拖动/固定并落实部门样板和T02拆分；定稿后工具/摘要检查以最终结果文件为准。文档可按用户授权本地提交，实际提交号由Git历史及交付回复持有；不推送、PR、合并或部署。产品实施继续111 T01—T05，不再等待密钥或列拖动/固定方案。

本任务无自有运行环境需要停止或回收，未操作既有服务、容器和卷。4180只读样板继续保留；未凭历史端口宣称CHNHCM业务服务在线。

## 7 证据

本轮长期证据位于[data/113-department-list-template-docs](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-department-list-template-docs)，包含失败、复验、样板截图/测量和最终输入摘要。前轮113报告、检查输出、来源归档及原采用批次保持；.local辅助编辑/检查脚本不是唯一长期结论依据。
