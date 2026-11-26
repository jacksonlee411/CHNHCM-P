# CHNHCM-011 记录分类与原因输入简化验证

登记信息：[registry.json](../governance/registry.json) 的 `governance.verification_records`。关联 [010 计划](../archive/plans/010-m02-scope-and-plan.md)、[011 业务需求](../specs/m02/011-m02-business-requirements.md)、[012 页面合同](../specs/m02/012-m02-functional-and-interaction-design.md)、[016 决策](../archive/adr/016-m02-effective-record-history.md)和 [17 术语表](../architecture/target/17-terminology-glossary.md)。

## 1 基线、前置与范围

日期：2026-10-31；工作树 `/home/vivobook/projects/CHNHCM-wt-dev-a`，分支 `wt-dev-a`，HEAD 为 `d822836cac3ffe5f90007277e85932cd2e7fc26c`。开始时有前轮删除用语的 12 个已跟踪文件修改及 3 个未跟踪证据文件，stash 为空；本轮在这些已授权修改上继续修订，不提交或推送。

开工前核对前轮 13 项受测输入和 286 项保护文件，299 项摘要一致。保存当前 301 个仓库文件的起点摘要，将直接相关的 12 份文件保存到[修订前快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/011-before-classification-simplification.zip)。环境为本机 Linux、Python 3.12.3 标准库及 Git，不启动业务运行。

## 2 来源核对与采纳边界

B&B 固定只读输入为 `a76db79227fdc4a0805fcd6e37fe22898d8b02ea`，来自当时与本地 `origin/main` 对齐的 `wt-dev-b`；没有 fetch、修改、运行测试或启动浏览器。本仓构建、验证及运行不依赖该工作树。

| B&B 来源路径及定位 | 本轮核对事实 | M02 采纳边界 |
| --- | --- | --- |
| `docs/dev-plans/repo-governance/899-core-business-object-lifecycle-semantics-contract.md` §10.2 | 删除禁止客户端提交分类，继承目标最新更正态；更正修复原事实，不另造更正/删除业务类型 | 业务分类与记录处理分开，更正/删除不另填处理原因 |
| `docs/dev-plans/orgunit/524-orgunit-field-metadata-and-ext-fields-contract.md` §7、§8、§12.2 | 更正默认带出原分类，可在同一对象操作方式内修正；删除不取分类候选，五类动作的 note 均可省略 | 首批继续只改名称、原分类及日期不变；不复制可选说明或配置平台 |
| `apps/web/src/pages/org/OrgUnitDetailsPage.tsx` 删除提交分支；`modules/orgunit/api/query_models.go` 删除请求结构 | 删除不发送类型或原因，只读展示原分类；note 可选 | 删除确认核对目标和影响，无分类/处理原因输入 |
| `modules/orgunit/services/mutation_query.go`、`orgunit_authorized_command.go`；`modules/orgunit/infrastructure/persistence/schema/functions/submit_org_event_rescind_command.sql` | 服务端继承目标分类、校验更正边界和准确删除目标，保留接受时证据 | Go 仍为本仓权威；准确目标、当前授权、并发与历史守卫保留，不迁入 B&B DB Kernel |

本轮是对上述输入的静态核对及文档采纳，不是 B&B 或 CHNHCM 的新运行验收。未采纳分类改写、历史改日、整对象删除、可选说明、审批、工作台或通用平台；首条与已生效记录删除限制保持。B&B 的更正请求可提交最终分类，本批更正不开放分类编辑，因而与删除一样由服务端继承并拒绝客户端分类输入，该差异在 011 明示。

## 3 修订场景与预期

新建、资料变更、停用仍按类型选择一个业务原因，服务端校验目录并固化名称快照；更正直接修改允许字段、核对差异后保存，删除只读核对目标及影响后确认。两者不出现原因选择器，不提交类型/原因，也不新增办理依据或可选说明。

服务端从所选记录最新已接受修订继承原分类，自动记录动作、操作者、记录时间、目标修订、前后结果和来源版本。当前目录改名或不再提供原选项，不改写旧事实，也不单独阻断更正/删除；缺失目标证据或版本过期仍拒绝，不回退当前目录补造。当前权限、日期、引用、幂等和同事务历史保存继续生效。

011 更新规则、场景和 AC004/AC005/AC013 的输入、成功及拒绝预期；012 更新字段表、日志、表单与验证路径；010/016 和入口同步。17 保留 448 项稳定术语身份，修改操作类型、操作原因、记录处理、数据更正、删除未来记录五个词条的直接说明。CHNHCM-011 第 5 次本仓修订接续原链；旧报告及前轮的不同原因设计作为历史证据保留。

## 4 检查过程与实际结果

| 检查 | 实际结果 | 结论 |
| --- | --- | --- |
| 生成预检及同步 | `build_modules.py --dry-run`、生成及最终同步检查通过；57 份文本产物中仅 17 随源数据更新，未改图稿 | 通过 |
| 独立只读语义复核 | `/root/m02_012_flow_review` 按前置快照复核输入禁止、准确目标继承、目录变化、历史/重试及范围差异，未发现实质缺陷 | 通过 |
| 文档与治理检查 | `make docs`：46 项 Python 回归通过；资料检查覆盖 113 个 Markdown、53 张图稿、197 项资产；未修改检查器 | 通过 |
| 固定分支引用与差异格式 | `check-repository.py --check-refs` 核对六个固定分支引用通过，`git diff --check` 无错误；报告和输入定稿后再次检查引用与摘要 | 通过 |
| 输入、历史与范围保护 | 448 项词条身份、41 个业务锚点、17 份既往验证登记保持；12 个快照成员、13 项受测输入和 289 项保护文件摘要一致，前轮八项修订条目保持 | 通过 |

## 5 问题处置与证据保存

初稿核对发现 010 的更正路径仍要求选择处理原因，已同步为修改允许字段、核对差异后保存。独立复核未发现实质缺陷，确认现行文本无更正/删除必填原因残留，B&B 与本批输入和范围差异已明示。历史进度和旧验证记录保留原词义及结果，由本轮明确接续；不删除旧证据或把它们改写为当前新合同通过。

前置快照 SHA256 为 `2018ebc958e94eba5e57719987b1db68b8ff1c548cf58622e00d92058a1579d5`。[受测输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/011-record-classification-simplification-inputs.json)保存基线、工作区修改前后摘要、快照成员、B&B 七份静态输入摘要和未改文件，覆盖最终 304 个仓库文件中的 302 个；报告与摘要自身排除以避免循环引用。工作用清单位于 `.local/m02-classification-simplification/`，长期证据以本报告和仓内摘要为准。

## 6 未覆盖事项与交接

本次不修改业务代码、检查器、资产实现状态或外部参考仓库；未运行 Go、DB、API、浏览器、产品构建或外部系统。010/011/012 保持草案，013 尚无正文；本轮部分合同的采纳不代表整篇规格或业务验收。后续 013 承接分动作请求字段、分类继承、禁止输入、目标并发、持久化与实际验证，不恢复独立处理原因。

报告定稿后补验另建记录。原来源包、初始采用摘要、前轮修订条目和既往验证保持，不以本轮通过扩展其原始范围。016 保持 active，登记本轮已实际完成的复核及部分采纳范围；其余生命周期和计划执行状态不提升。
