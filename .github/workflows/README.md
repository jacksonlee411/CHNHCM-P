# .github/workflows

[CHNHCM CI](ci.yml)在面向main的PR、main推送和手动触发时执行。三个检查都成功后才可合并；工作流定义或本地检查通过不代替GitHub实际运行结果。

长期规则见[137规范](../../docs/governance/137-actions-and-validation.md)，本轮实现和验证见[136按影响选择记录](../../docs/verification/136-impact-selection.md)。本节描述当前实现，远程生效及首轮失败接续见[发布记录](../../docs/verification/136-actions-publication.md)；下文按日期保留历史运行事实。普通开发分支push不触发，正式PR覆盖草稿/转ready事件，标签变化不触发；同PR新候选取消旧运行，手动诊断有独立并发组和diagnostic检查名称。

| 检查 | 执行内容 | 边界 |
| --- | --- | --- |
| quality | 产品及测试变更先执行 `make generate` 核对新增、修改、删除的文件，再执行 `make check`、`make build` 并核对所选测试清单 | 文档、草稿、同树main复用执行文档检查；草稿阻止合并 |
| preflight | 全量候选在一个独占环境中执行 2 个旧库升级测试根。明确要求远程浏览器时另执行 6 个浏览器回归用例及其企业前置依赖 | 本轮通过或已核实可沿用后启动其余分片，最终两个汇总都核对该 job |
| integration | 独立选择：不适用 0 片，按耗时选择 1–4 片，完整 4 片 | 共用/未分类测试根保留；缺任务、失败/取消/未知跳过不得通过 |
| browser | 公共仓库按入口耗时分片，首次入职按声明分为两组，完整 16 片 | 核对实际清单与企业前置依赖；独立运行、workers=1、retries=0 |

提交前优先使用本地预检发现问题、修复并复验，再集中推送。远程集成可按需全量。公共仓库自动运行选中的浏览器分片并核对结果。私有仓库默认本地实测浏览器，显式要求远程时核对所选分片。后续候选自动核对原始成功 job、两次合并树的变化和实际分片清单。补文档时可仅运行文档检查与汇总。局部修复只派发受影响和仍欠验证的分片，quality 摘要列出本次派发数和来源。最终候选和 main 汇总复核原件。[逐片复用规则](../../docs/governance/137-actions-and-validation.md#14-同一-pr-的后续全量候选)

开发和修复可在草稿中进行，quality 摘要显示正式 PR 需要覆盖的分片数。实际派发数见正式候选的 quality 摘要。本地用 `make ci-preview` 查看完整差异，转为正式 PR 前用 `make ci-ready` 核对生成物并运行对应的前置回归。[候选准备与分组回放](../../docs/governance/137-actions-and-validation.md#13-减少正式候选的全量启动次数)

本地可用 `make ci-preflight` 复验前置流程。全量的其他分片执行各自余集，限定计划保留原范围。试点默认全部使用托管 Linux。独立一次性 runner 池就绪后，可分别设置仓库变量 `CI_BROWSER_SELF_HOSTED_SHARDS` 和 `CI_INTEGRATION_SELF_HOSTED_SHARDS`，按分片数逐步扩大。两个变量的边界、宿主和撤回步骤见 [137 资源约定](../../docs/governance/137-actions-and-validation.md#12-全量前置回归与一次性-runner-试点)。覆盖与成本数据见 [136 试点记录](../../docs/verification/136-hosted-minutes-pilot.md)。

按完整差异证明某执行层不适用时，由汇总明确记录未执行。full/selected/docs正式PR均保存当前attempt的schema2范围证据7天；main重算影响范围并核对实际树、父提交、合同、来源run及全部job，等价成功才复用。复用证据不充分时按推送差异选测，浏览器全量遵循同一授权规则。当前调整的远程生效以main实际版本为准。

前端及已映射业务按消费者选择，多业务取并集；被动附件不扩大范围，运行代码/测试输入仍受保护。共用内核、验证机制和未知运行输入保持全量；单层共享变化只扩大对应层。首次发布本次CI变更自身仍全量。


Go和Node版本分别读取go.mod及package.json，pnpm沿用packageManager锁定；Corepack固定0.35.0。Actions固定为官方v7对应提交，升级时重新核对[checkout](https://github.com/actions/checkout)、[setup-go](https://github.com/actions/setup-go)、[setup-node](https://github.com/actions/setup-node)及[upload-artifact](https://github.com/actions/upload-artifact)的合同。

数据库任务和浏览器任务仅在各自临时检出目录中，将锁定文件的镜像站前缀改为Docker Hub；保留原镜像路径、标签和SHA-256摘要。本地锁定文件继续使用原镜像站，quality的生成一致性检查使用原始检出内容。[首次CI浏览器任务](https://github.com/jacksonlee411/CHNHCM/actions/runs/41212651204/job/124571471786)因镜像站TLS超时未启动数据库，未进入业务测试；镜像下载路径调整后的结果由后续CI运行另行证明。

工作流只读代码，临时数据库凭据由现有脚本生成并保存在仓外且不打印；本地公开开发登录预设同017，不需要仓库业务Secret。除正式PR的无凭据JSON范围验证清单外，仅在失败或取消时上传已生成的遮罩PNG截图，保留7天；不上传仓外状态、Cookie、trace、错误文本快照或整个.local目录。browser在初始化后为独立测试管理员显式授予新增/更正及字段权；正常初始化和迁移不自动扩权。make smoke使用任务独占的临时profile ci-browser，成功、失败和可处理取消在finally回收容器、网络、卷和临时配置，always再执行幂等destroy；不部署应用。宿主机故障及强杀后的最终释放还依赖临时runner回收，不能把工作流接线当成全部取消路径已通过。

依据与入口：[005 远程协作边界](../../docs/governance/005-branches-and-documents.md) · [014 文档检查](../../docs/governance/014-document-management.md) · [本地检查入口](../../docs/index.md)。

2026-11-03的[PR #5首轮CI](https://github.com/jacksonlee411/CHNHCM/actions/runs/41538914877)中，quality、integration成功，browser被整项任务20分钟时限取消；已完成124项均通过，其余未完成不计通过。132项顺序浏览器回归连同依赖/运行环境准备超过原预算，因此只将browser任务时限调至30分钟，保留全部用例、workers=1、retries=0及原单例超时/断言；新提交重新执行三项检查，不把取消记成功。

2026-11-03的[PR #7首轮CI](https://github.com/jacksonlee411/CHNHCM/actions/runs/41633997124)又达到30分钟总时限：156项中135项明确通过、1项失败、20项未获完成结果，整轮取消。Community完整运行预算调至45分钟，保留用例、worker、重试和单例断言；补足取消截图及安全失败位置输出。本地同名失败经导航追踪定位为测试未等查询完成即点击旧链接，补齐真实查询及返回链接A的前置；原CI缺失堆栈的限制保持，范围与复现见[111补验](../../docs/verification/111-ci-browser-budget.md)，合并继续要求最新提交三项全成功。

2026-11-09多分支发布前，当前完整浏览器回归已增至237项；[130最终单轮记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/130-implementation/smoke-final-r2.json)实测47.1分钟，已超过原45分钟整项CI预算，尚未计入安装与启动。因此browser预算调整为60分钟，保留全部用例、workers=1、retries=0、单例超时和断言；quality/integration预算不变。该调整不表示远程通过，合并仍以最新PR提交三项实际成功为准，原失败及取消记录保持。

[PR #9首轮quality](https://github.com/jacksonlee411/CHNHCM/actions/runs/42553110999)在文档检查失败：R5/R6报告引用的13份原.log输出仅存在于本地，被通用忽略规则排除，远程干净检出缺失；integration/browser未运行。现将这13份原件纳入Git，按原字节核对摘要；仅三份原始命令输出豁免行尾空白检查，产品代码检查保持。恢复清单见[原件摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/131-publication/evidence-restoration.json)，原报告和首次失败不改写，恢复后必须从干净检出重新检查。

[PR #9后续browser](https://github.com/jacksonlee411/CHNHCM/actions/runs/42553460532)完成237项：234通过、3失败（37.6分钟）。三项原生缩放场景均在xdpyinfo探测显示环境时失败，未进入缩放断言；quality/integration成功。原生缩放还依赖xwininfo和libXtst，而Playwright 1.55.1的Ubuntu 24.04 Chromium默认依赖未声明x11-utils/libxtst6。CI现显式安装xvfb、x11-utils、libxtst6，并在创建业务临时环境前运行[独立显示检查](../../scripts/check-x11.py)，实际启动及回收自有Xvfb、核对命令与共享库。完整237项、真实200%/400%缩放、单例预算及零重试均保持；该修复须由最新提交的完整远程CI验证，首次失败和遮罩截图保持。

2026-11-09按用户要求加速浏览器CI：采用Playwright的8个分片，GitHub矩阵max-parallel=8、fail-fast=false；每个分片在独立runner创建并回收自己的运行环境，片内保留workers=1、retries=0，防止共享管理员授权及账号修改相互干扰。默认本地make smoke仍完整顺序执行。合并门禁保留名称browser，只有全部分片成功才通过；失败、取消、跳过均不放行，失败截图按分片单独命名。实际同时运行数受GitHub账户并发额度和runner调度限制。[本地划分核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/131-publication/browser-shard-validation.json)确认237项无遗漏、无重复，缺失xdpyinfo会提前失败，汇总判断仅接受success，三项原生缩放在独占临时环境全部通过并已回收；这些检查不能代替最新提交的全部分片及quality/integration远程验收。

[首轮8分片CI](https://github.com/jacksonlee411/CHNHCM/actions/runs/42558093312)实际同时启动8片：quality/integration及7片成功，第1片33通过、1失败，总计236通过、1失败，browser汇总按要求失败。原生缩放三场景均已通过；剩余滚轮失败在本地5次诊断中复现1次，滚轮发送时日志仍是加载占位，内容出现后的滚动恢复覆盖了输入。测试现等待真实日志表格，按实际刷新控件定位，并断言物理滚轮使scrollTop超过输入前值；不调整产品、超时、重试或以脚本滚动代替用户输入。定向连续10次通过，自有临时环境已回收。[复现与复测数据](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/131-publication/browser-scroll-reproduction.json)及[首次遮罩截图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/131-publication/scroll-ci-first.png)保留；仍须最新提交完整远程CI通过后才可合并。

[后续完整分片CI](https://github.com/jacksonlee411/CHNHCM/actions/runs/42559546655)中滚轮场景已通过，quality/integration及7片成功，第6片33通过、1失败，仍为236通过、1失败。权限用例切换页签时，全页Alpha正向读取同时匹配基本资料与日志单元格，触发定位歧义；现将6处正向断言限定在基本资料页签，并等待首次收窄日志的真实200响应完成，2处撤权后全页Alpha零残留断言保持。两条受影响业务路径各连续复测5次，合计10次通过；产品、超时及零重试保持，临时环境已回收。[修复核验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/131-publication/browser-panel-selector-validation.json)保留首次失败与边界，合并仍须最新提交三项检查及全部分片成功。

quality取得完整Git历史，用于按最终验证报告的基线或实际提交核验已删除源码的历史链接；文档检查不会自动联网补取。integration和browser分片仍使用各自默认检出范围。远程执行结果按各PR最新提交核对，本地结构检查不代替远程验收。

2026-11-12按用户要求提高CI并发：quality、integration和12个浏览器分片独立启动，主要任务最多14路并行；browser汇总仍只接受全部分片success，合并仍须最新提交quality/integration/browser三项成功。各分片使用独立runner和运行环境，片内workers=1、retries=0、全部用例及单例断言/预算保持。实际并发受账户额度与runner调度限制；上述8分片记录保留原结果。

本轮只读分片检查：完整274项，12片分别26/29/22/16/29/34/9/23/20/22/25/19项，无空片、遗漏或重复。16片方案会出现空片，因此未采用；串行测试组保持，未启用fullyParallel。

2026-11-14按用户要求将集成测试分为8路：脚本从两个原测试包的实际编译列表按包名与测试名排序、轮转分配，每个测试根恰好属于一片，空片及错误参数直接失败。片内保留顺序执行、原20分钟Go预算及全部断言；默认本地make integration保持完整执行。integration汇总只接受所有分片success，失败、跳过或取消均不放行；浏览器保持12片，主要任务最多21路，实际并发受账户额度限制。

本轮合并后336项浏览器用例按Playwright文件组原生分片，12路分配为28/29/33/37/43/0/31/27/30/26/27/25；清单无遗漏重复但存在空片。改为11路后为36/26/37/28/43/22/36/22/34/27/25，336项完整且无空片。保留文件内顺序、workers=1及retries=0，集成仍8路，主要任务最多20路；原12路运行结果保留。

以下保留PR #18合入main时的标签方案；136实现已改为完整差异选择，当前工作流不再由标签触发或压缩正式范围。

2026-11-16按用户“针对性CI”要求增加限定范围：PR显式标记 `ci:geography-legal-entity`，或手动选择 `geography-legal-entity`；quality保持完整；integration从实际编译测试根筛选国家、行政区划、地点、法律实体、企业及T07，仍分8片；browser选择工作流列明的18个相关文件并分4片，保留企业先行依赖、原生缩放、workers=1及retries=0。未匹配测试、空Go分片及任何分片失败、取消或跳过都不能作为门禁成功；测试内明确的容量等条件项仍按原合同登记，不计通过。未标记范围的PR、main推送及未指定范围的手动运行保持完整。添加/移除标签会重新运行对应范围，PR说明必须登记实际选择。受保护main按PR合并引用的quality/integration/browser放行，必须核对最终head及对应PR检查实际成功；手动补验不能抵消取消/失败的PR检查，不能声明全量产品验收。本批具体范围及首失败见[206集成记录](../../docs/verification/206-legal-entity-implementation.md#10-与最新main的发布集成)。
