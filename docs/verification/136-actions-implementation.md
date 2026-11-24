# CHNHCM-136 Actions节流工作流本地实施

## 1 范围与结论

2026-11-16用户明确“同时实施CI工作流，本地验证后保留待发布”。本批实现[136计划](../plans/136-actions-cost-reduction.md)的T01—T04本地部分，并同步[137规范](../governance/137-actions-and-validation.md)。工作流、反例、真实测试清单和文档验证通过；待授权发布及T06远程验证。未提交、推送、创建PR、触发Actions、修改保护或更新固定运行环境。

当前固定工作树wt-dev-b，HEAD为84046c17dfb123751385e7aaba821ffc28d130be；方案批次已有文档改动保留。只读fetch取得远端main 371290140d9d585b9b8d37cc89c99cd427ae91ef，PR #18已合入，无开放PR。没有将main业务代码合入当前分支；另用任务临时目录导出该SHA、安装锁定依赖并应用本批CI补丁，验证其消费者和依赖项目。此隔离验证不表示b已同步main。

此前[方案报告](136-actions-policy-validation.md)及其61项离线模型、53轮636job／3944分钟估算保持历史事实。本批不重新扩大账单快照，不把运行估算当实付账单；浏览器2881分钟、main1293分钟仍是优先节流依据。原方案24份输入按原摘要保存在[快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-implementation/prior-plan-inputs.zip)，对应旧inputs-final可逐字核对；规范当前修订不覆盖旧报告。

## 2 实际改动

| 部分 | 行为与目的 |
| --- | --- |
| 触发与并发 | 普通开发分支push不触发；PR显式覆盖opened/synchronize/reopened/ready_for_review/converted_to_draft，标签不触发；手动诊断使用独立并发组及diagnostic检查名 |
| 质量先行 | 草稿make docs后明确拒绝合并；正式候选完整check/build/generate后才运行昂贵套件 |
| 选择 | 完整Git差异、重命名旧新路径和非普通文件检查；纯说明可免DB/浏览器；8个地理前端和7个专用测试文件使用窄消费者集合；未知/共享/删除/Go/CI/迁移/锁文件全量 |
| 实际清单 | Go编译发现根后筛选/分片，空集合失败；Playwright清单对账完整、选择和每片，缺文件、遗漏和普通重复失败；企业维护依赖重复单列 |
| 三项门禁 | 保留quality/integration/browser；汇总always执行并核对当前PR的head/base/ready、计划及全部实际job的最新attempt，失败/取消/缺失/非预期跳过拒绝 |
| main复用 | 只读核对合并PR、两个父提交、同树、同CI合同、7天内全量正式PR成功run和全部job；仅接受对应attempt的小型JSON证据；资料不足回全量 |
| 手动与重跑 | 原限定输入geography-legal-entity保留为诊断；不能满足PR检查；失败job补验允许沿用同run之前成功的job，最新失败覆盖旧成功，首错保持 |
| 分片 | 全量8路Go／11路浏览器保持，限定8／4；未做同覆盖计时比较，不声称减分片净收益 |
| 凭据延迟读取 | 9份旧测试／辅助模块不再在导入时读取凭据，实际首次访问仍查原profile并缓存；没有默认凭据或绕过登录的分支 |

入口：[工作流](../../.github/workflows/ci.yml)、[范围映射](../../.github/ci-scopes.json)、[计划与汇总](../../scripts/ci_policy.py)、[浏览器清单](../../scripts/ci_browser.py)、[Go分片](../../scripts/integration-shard.py)。完整PR只新增一个7天JSON证据，正常截图规则保持，不上传凭据、trace或运行状态目录。

## 3 本地验证与清单

本地反例测试直接调用生产选择器、实际临时Git合并、job汇总和证据读取，包含完整多提交差异、代码改名为文档、文档符号链接、草稿、候选变更、未知文件、消费者缺失、分页截断、空片、最新attempt失败、手动混入、同树不同范围、过期证据、异常zip及来源job缺失。CI专项23个Python测试通过；含原脚本后完整Python107项，Node31项。凭据专项使用任务自有假数据与独立子进程验证导入零读取、首次读取一次、字段保持及失败拒绝，不访问用户凭据。

actionlint v1.7.12通过；实际YAML十项结构检查见[结构结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-implementation/workflow-structure.json)。make check、make build、make generate、生成物前后摘要、引用/登记与差异检查的最终结果见[检查清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-implementation/checks.json)。生成物核对按工作区基线前后比较，不把本批已有修改误报成生成漂移。

| 源码／范围 | Go根及8片 | 浏览器去重项 | 分片执行总项 | 浏览器分片 |
| --- | --- | --- | --- | --- |
| b当前／全量 | 263；33/33/33/33/33/33/33/32 | 366 | 366 | 11 |
| b当前／地理消费者 | 55；7/7/7/7/7/7/7/6 | 57 | 57 | 4 |
| main37129014／全量 | 280；每片35 | 394 | 434 | 11 |
| main37129014／地理消费者 | 76；10/10/10/10/9/9/9/9 | 87 | 99 | 4 |

来源：[b Go清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-implementation/go-partitions.json)、[main Go清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-implementation/main-go-partitions.json)、[浏览器清单摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-implementation/browser-inventories.json)。main的enterprise-maintenance有4项，每个独立分片均先执行，分别多40／12次；这些是明确的依赖开销，未当成新增业务覆盖。无空分片，普通测试无重复、遗漏。清单检查只发现/编译测试，未执行DB业务或页面断言，不替代全量产品回归。

## 4 首失败与修复

1. 最新main干净导出首轮Playwright清单失败：旧测试导入即调用credentials-path，环境未初始化。原输出见[首失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-implementation/main-browser-geography.log.gz)。采用实际使用时读取后，同一main源码的全量和限定清单通过；b另以不存在的ci-inventory-unconfigured profile全量核对通过，不依赖既有profile。没有为列清单启动数据库或创建凭据。
2. 实际GitHub API显示合并后run的pull_requests可能为空，见[必要字段样本](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-implementation/merged-run-api-example.json)。复用判断不再强依赖该可空列表，但提交关联的已合并PR、head/base、树、run、范围及证据仍全须匹配；相关反例通过。样本本身失败，不构成任何复用或远程通过证据。
3. 对一个现存run只读验证分页jobs接口：15个job均有run_attempt字段。API读取和本地夹具不代表当前GITHUB_TOKEN权限、实时状态延迟及远程汇总已验收。

## 5 交接与限制

- 当前交付为本地实现及规范，136保持in_progress，T06未运行。发布前同步当时main、重新检查最终合并候选；首次工作流变更全量。远程需确认事件、草稿拒绝、三项检查、只读权限、证据上传/读取及下一次等价main复用。
- 不把限定套件57／87项与全量366／394项混算节省；没有实际费用下降或25%—33%达标声明。main1293分钟是历史去重候选上界，仍要扣除限定PR后的全量、证据不可用回退和质量检查。
- 保持原8/11分片、workers=1、retries=0、原业务断言与超时；没有用容量回归替代CI接线验证。原四次同树main失败及125容量P1仍按原证据承接。
- 完整main失败后暂停新集成是137中集成维护者的责任，本批没有修改保护或增加自动集成控制服务。普通开发继续。
- 未启动/更新数据库、浏览器服务、容器或固定端口。任务临时main导出在保存结果后移除；原方案证据、原验证记录和其他工作树保持。范围与输入摘要见[保护核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-implementation/scope-check.json)。

机制依据：[GitHub workflow语法](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax)、[jobs REST接口](https://docs.github.com/en/rest/actions/workflow-jobs#list-jobs-for-a-workflow-run)、[Playwright分片说明](https://playwright.dev/docs/test-sharding)。最终效果以授权发布后的实际运行记录为准。
