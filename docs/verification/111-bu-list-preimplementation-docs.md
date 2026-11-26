# CHNHCM-111 BU列表实施前文档修订与多轮检查

登记信息：[registry.json](../governance/registry.json)的`governance.verification_records`。适用范围：[110决策](../adr/110-bu-list-and-datagrid.md)、[111计划](../archive/plans/111-bu-list-experience-upgrade.md)及关联列表合同的文档准备；本报告不记录Pro实现或产品验收通过。

## 1 验证对象与受测基线

2026-11-03，wt-dev-b，基线`94dd06466ffc44ee233e600d99f3bb82218a143b`，开始工作区干净、无stash。本批受测对象为该基线加文档、登记和计划采用差异；仅HEAD不能标识本批输入。新增110/111、本文及官方资料登记，修订002、010、011—013、017、019、023、100、inventory、登记表和导航；[输入清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-bu-list-preimplementation-docs/input-manifest.json)及[文档快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-bu-list-preimplementation-docs/document-inputs.tar.gz)标识实际受测输入，不把报告自己的最终提交号当作基线。

## 2 环境与前置条件

本仓Python文档检查环境；源码/接口/依赖和历史原件只读。官方MUI文档及候选8.29.3包元数据见[来源登记](../reference/111-datagrid-pro-sources.json)，只核对公开能力和peer范围，不执行依赖安装、构建或许可初始化。

用户已采纳Pro及减少自研的方向；本次授权为实施前文档和多轮检查。未创建、启动、停止或复用任何profile、容器、Web/API；无任务自有运行资源需要清理，003资源收尾的运行操作不适用。

## 3 验证过程与结果

多轮语义复核区分采用范围、服务端与导航保护、案例/登记和历史保护；均为主代理自查，不冒充独立评审。各轮结果独立保存：

| 检查 | 预期与操作 | 实际结果与结论 | 证据 |
| --- | --- | --- | --- |
| R1文档治理 | make docs，核对编号/登记/关联/来源及治理工具回归 | 通过，65项工具测试及185份Markdown检查；本轮语义发现另列 | [R1日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-bu-list-preimplementation-docs/round-1-docs.log) |
| R1语义与保护 | 逐项核对裁决、追踪、交接及保护摘要 | 发现2项待修订，未计语义通过；原322例正文及2,321份受保护文件保持 | [R1发现](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-bu-list-preimplementation-docs/round-1-semantic.json) |
| R2修复及复验 | 补AC追踪及012交接，重新检查合同/状态 | 通过，65项工具测试；8项辅助断言和语义自查通过，无新增发现 | [R2日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-bu-list-preimplementation-docs/round-2-docs.log)、[R2语义](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-bu-list-preimplementation-docs/round-2-semantic.json) |
| R3交付语义与历史保护 | 重读采用/待决/后置、版本证据、登记/旧状态及原件 | 通过，无新增发现；旧案例正文和业务规则身份保留，111执行为not_started | [R3语义](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-bu-list-preimplementation-docs/round-3-semantic.json)、[保护基线](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-bu-list-preimplementation-docs/protected-baseline.json) |
| 差异格式 | git diff --check | 通过，最终冻结后再核对，实际输出单列 | [差异检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-bu-list-preimplementation-docs/diff-check.json) |
| Pro产品实现与新增案例执行 | 安装/构建/许可、真实DB/API及浏览器 | 未运行，不计通过；TL-323—334全部待验 | [111计划](../archive/plans/111-bu-list-experience-upgrade.md) |

收尾结构检查及冻结输入核验另见[最终检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-bu-list-preimplementation-docs/final-checks.json)；该记录只承接本批文档，不扩大上述业务边界。

## 4 失败、修复与复验

R1发现：023第18.2节尚未关联新增12例到既有AC，012第7节缺少本次目标与旧100批次的交接区分。两项均补齐，R2重新检查对应AC链接、旧完成/验证元数据及A/D接线边界，R3再次检查交付状态和历史保护，两轮无新增主要发现。

起草时另补齐List view没有表头时的排序入口，以及官网9.x受控面板与8.x候选版本的适用边界。一次补丁因旧详情标题上下文不一致未匹配，工具未写入；重读标题后重新应用完成。三者不是产品运行缺陷或业务通过记录。

本批已执行的make docs轮次没有失败；R1语义发现单独保留，不以结构通过掩盖。R2/R3是同一主代理的多轮自查，未宣称单独代理或具名业务人员评审。

## 5 未覆盖范围与限制

本次未安装Pro、锁定最终patch、配置或验证商业许可，未修改产品/API/生成客户端/依赖/测试程序，也未运行Go、DB/API、浏览器、容量、真实缩放或生产验收。原生筛选面板、Excel候选及新API仅评估方向；当前页面仍是Table。023新增案例全部待实现后执行，旧报告不提升为新通过。

## 6 结论与后续事项

用户已采纳的Pro列表目标、规格、计划和待验案例完成文档同步；R1发现已闭合，R2/R3语义复核无新增发现，结构/关联/来源及差异检查通过。023现为334个稳定编号、290现行、44退役，新增12例未运行；六项资产仅新增111计划采用，原state、旧adoptions及new_evidence保持，不抬升实现/验证。

111的产品实施任务维持未开始；T01核对依赖/许可，T02—T03实现验证，T04完成列筛选评估，T05保存产品证据与环境收尾。未提交、推送、PR、合并或部署。本轮无任务自有运行资源，未触碰既有持久库、卷和环境，不以未启动资源冒充清理成果。

## 7 证据保存

长期证据目录为本报告data/111-bu-list-preimplementation-docs，保存各轮文档日志、语义发现/修复复核、输入摘要/快照、保护基线及最终核验。保护清单覆盖源码、接口、依赖、测试程序、历史报告和来源原件2,321项；本轮原始输出位于.local/111-bu-list-docs，必要证据已经保存到仓内，不依赖相邻仓库或临时运行服务。

输入清单列出报告及登记等受测资料的字节摘要，归档不包含运行凭据或业务库。最终检查自身和后生成的摘要/归档不递归计入输入摘要；最终结果另核对输入未变。后续产品实施必须新建报告，不能覆盖本次文档证据。
