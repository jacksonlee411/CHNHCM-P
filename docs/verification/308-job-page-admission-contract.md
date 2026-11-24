# CHNHCM-308 职务页面继承与独立验收合同复核

登记信息：[registry.json](../governance/registry.json)。本次按用户要求保护精简入口、恢复原实施，再把页面标准落实到对象合同及验收清单。基线为`1e830a4ccb0066085bef24239848b086971e4d73`加未提交工作，分支wt-dev-d；只做文档编制和源码只读对照，产品代码及测试／CI不在本批修改范围。

## 1 入口保护、原实施恢复与运行边界

先将入口整理的4个已跟踪文件和8个未跟踪附件完整保存至stash `6fb1d979d7757d24ef0e30e30b7b871841b44d50`，逐项核对内容及模式，再应用原实施stash `0cf171c5dd4325e61e56029167657b7897932bd8`。原318项改动逐项恢复核对通过，没有提交、切分支或删除stash。

合并时AGENTS采用精简版本；两个导航保留双方新增入口；登记表保留原307／308及其实施记录，再追加005整理记录；8个005附件原字节恢复。两个stash均保留，完整摘要见[保护与恢复记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/308-job-page-admission-contract/restore.json)。后续仅调整AGENTS中已过时的暂存阶段运行链接，继续保持8 KiB以内，未重新追加历史流水账。

005报告中“原代码仍在stash、当前源码不含M03”描述首步结束时状态，原报告保持；本轮结束时原M03源码已恢复到工作区。4380仍运行原独立固定构建，API8380／DB54370及迁移1—17保持，详见[307运行交接](307-m03-job-foundation-implementation.md#10-用户授权的独立页面验收环境)。恢复源码不等于运行更新；本批不启动／停止／迁移／重建或回收任何固定环境，不向用户验收库写样本。

## 2 标准来源与交付

现行判定来自012页面合同、026日期格式及已纳入012的301／302交互规则；BU列表、新建、详情、TimelineEditor、LifecyclePairDialog和日志仅作实现参照。输入及源码摘要见[输入清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/308-job-page-admission-contract/inputs.json)，本轮未重新执行BU或M03产品验收。

| 交付 | 当前内容及入口 |
| --- | --- |
| 通用接入表式 | [按日对象模板](../templates/dated-object-admission.md#page-inheritance)补充逐页继承、差异、现有偏差及独立验收表；[功能模板](../templates/functional-design.md)接入相同要求，非按日对象不继承按日语义 |
| 过程拥有者与接入路由 | [124第7节](../guides/124-dated-object-admission-and-capacity.md#page-standard-admission)维护过程，[121第24节](../specs/m02/121-dated-object-admission-matrix.md#24-页面标准继承与m03接入映射)只映射现行标准及M03定义，不复制稳定ID |
| 对象继承合同 | [308第10节](../specs/m03/308-job-foundation-contract.md#m03-page-inheritance)：11项逐页映射，区分现有公共原语、M02耦合参照及M03待接线；覆盖列表、详情、表单、日期、操作、历史与返回 |
| 业务差异及偏差 | [308第11节](../specs/m03/308-job-foundation-contract.md#m03-page-differences)：6项有依据的差异和7项待整改偏差；不存在独立领域整体豁免 |
| 独立验收 | [308第12—13节](../specs/m03/308-job-foundation-contract.md#m03-page-acceptance)：6组手工样本、10个新增场景、14项新增验收；两对象×两角色×七组条件共28单元，另有权限专项 |
| 原交付边界 | [307接续](../plans/307-m03-job-foundation.md#7-进度与后续交接)保留原首批completed及原报告；新增页面清单未运行，文档完成不冒充产品修复 |

308现行合同合计14条规则／18场景／30项验收，其中本轮新增1条页面规则、10场景和14项验收。原13规则／8场景／16验收的报告与输入不改写。本轮验收覆盖登记为被动清单：[acceptance-status.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/308-job-page-admission-contract/acceptance-status.json)，不是自动测试，也没有执行成功记录。

## 3 文档语义复核及当前缺口

复核由Codex主代理顺序执行，非独立代理或人工签收。先按标准推导表格，再反向核对每个页面、差异、场景和验收能否定位；源码用于证明当前落点或缺口，不用于生成期望。当前产品偏差P01—P07均开放，来源及出口逐项位于308第11.2节。

| 复核问题 | 处理与范围 |
| --- | --- |
| 仅复用ObjectForm等基础组件是否足够 | 明确还需外框／滚动、两区表单、控制区动作、日期轴、更多菜单、日志／来源和返回；AC017／020／021／028／029逐项观察 |
| M03差异能否直接套用BU四字段固定期间 | 保留既有独立恢复资料及Job恢复换系列的业务能力；D05在同一专用弹窗明确字段、原子提交和恢复端定位，取消及原采用另验；不复制BU整个业务内核 |
| 固定选择能否照抄旧BU“集合变化即清空” | D06以308原固定身份合同为准，选择工具条复用，身份成员与授权判定仍由本域持有；AC019用41项／后增42号／排除1项独立计数 |
| 日志是否只换排版即可 | 当前Staffing Changes返回来源和快照，缺操作差异投影及分页／筛选上下文；10.3明确后续Staffing Go和OpenAPI同步补齐，前端不得反算历史 |
| 采用M02日志谓词会否带入其处理目录 | 明确Staffing自己的处理、字段及错误码，固定两条不借REVISE_TIMELINE；业务分类CHANGE与记录处理ADD分开，保持当前跨域边界 |
| “保存可见”或脚本滚动是否证明可达 | 原staffing307用例使用偏离后的下拉框定位及scrollIntoViewIfNeeded；P07保留旧结果边界，新清单要求正常入口、滚轮／键盘、实际尺寸及原生倍率 |
| 新验收是否误继承原通过状态 | 14项新增验收与28单元全部not_run；R0／RF／W0另作权限专项，原307完成不填入新清单 |

上表只说明文档如何处理问题，不能把P01—P07改为关闭。后续仍需实际页面与必要日志读取整改、独立预期测试接入及真实验收，才能声明标准一致。

## 4 检查结果

| 检查 | 本批结果与证据 |
| --- | --- |
| 两批改动保护／合并 | 通过，原318项与入口12文件完整；[restore.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/308-job-page-admission-contract/restore.json) |
| 稳定ID、继承／差异／场景／验收覆盖与链接 | 修正后通过；[专项核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/308-job-page-admission-contract/contract-check.json)，新增结果均为not_run |
| 产品／旧报告与精简入口保护 | 2013个非文档文件集合及内容与恢复基线一致；005的8个附件和任务外311项原变更保持，AGENTS为7454字节；同上核对结果 |
| 文档登记、格式和仓内引用 | make docs通过，含127项脚本测试；[初轮输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/308-job-page-admission-contract/make-docs.txt)。最终登记和链接补充由[收尾文档检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/308-job-page-admission-contract/final-docs-check.txt)核对 |
| 差异格式 | 通过；[git diff --check](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/308-job-page-admission-contract/diff-check.json) |
| 4380／8380只读可达性 | 两次GET均为200，固定构建目录存在；[记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/308-job-page-admission-contract/runtime-reachability.json)，不作为业务或数据库完整性验收 |
| 页面整改、DB/API、浏览器、自动门禁 | not_run；本轮未实施 |

专项核对首次把合法目录链接误判为文件，修正临时核对器后又发现121第21节沿用301旧标题“待执行验收”；已将该链接改为实际的“实施验收”，301正文及原报告不变。首失败与处置保留在[记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/308-job-page-admission-contract/check-first-failure.json)，不是产品测试失败；随后完整专项复验通过。仓库检查脚本未改动。

## 5 接续与限制

本轮交付仅为合同、模板和验收依据；不提交、推送、触发Actions或部署。新验收和28单元没有实际通过值，旧报告与旧失败保持；不凭文档检查推导当前页面合格。当前产品代码、API定义、迁移和测试保持恢复后的原内容；固定环境保持运行供用户验收。

后续实施以308的U01—U11、D01—D06及P01—P07为范围，包含必要的Staffing日志读合同接线；完成后再把新增验收接入自动检查，并验证真实浏览器滚动／原生缩放及实际业务结果。不得用新的全页截图基线接受当前偏差，也不因页面工作扩大到职级、岗位、容量或真实M04／M05消费者。
