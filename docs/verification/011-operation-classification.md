# CHNHCM-011 操作类型、原因与日期修订验证

登记信息：[registry.json](../governance/registry.json) 的 `governance.verification_records`。适用范围：[010](../archive/plans/010-m02-scope-and-plan.md)、[011](../specs/m02/011-m02-business-requirements.md)、[17 术语表](../architecture/target/17-terminology-glossary.md)的操作分类、原因、日期及对应验收说明。

## 1 基线与范围

日期：2026-10-31；工作树 `/home/vivobook/projects/CHNHCM-wt-dev-a`，固定分支 `wt-dev-a`，HEAD 为 `f21c99ac42d1662bd7fc077399e27fd144d7a8d0`。开始时已有未提交文档及检查器变更，本轮按用户确认的建议修订文档，不提交或推送。

修改前已核对上一轮 10 项受测输入、277 项保护文件及前置快照摘要；本轮开始前共登记 290 个仓库文件。相关正文、术语源及生成物、登记和前轮证据共 10 项保存到[修订前快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/011-before-operation-terms.zip)，本次受测对象及未改文件见[输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/011-operation-classification-inputs.json)。来源包和原验证报告保持原字节，原日期用语报告作为历史结论保留，现行用语由本轮修订接续。

## 2 修订与依据

按用户决定取消独立“办理依据”字段，不以备注、补充说明或自由文本原因替代。新建、资料变更和停用使用操作类型、受控操作原因及统一的生效日期；停用日期只解释停用类型的生效日期，不再单列输入、存储字段或术语。M02 保留既有菜单和办理入口，采用固定目录，不增加配置页面、审批、工作台或通用操作引擎。

记录处理与业务类型分开：更正保留原生效日期及原业务分类，撤销只移除允许撤销的未来效力；本次处理原因单独保留，不覆盖原业务原因。011 的原 39 个稳定 ID 保留，新增 M02-BR-016、M02-AC-013，形成类型/原因/记录处理的合法组合、服务端校验、代码和名称快照、幂等及原/此次原因的验收预期。原日期、首条、普通字段、未来停用冲突和历史保护边界保留。

17 从源 JSON 生成，新增操作类型、操作原因、记录处理三个四字中文词条，445 项增至 448 项；既有 ID、别名及对象/概念绑定保留。业务生效时间、当前日期的说明同步；M04 人事业务原因定义改为直接解释人事业务为何发生，避免误读为独立依据字段，原英文映射、适用范围和本域目录归属不变。X01 仅共享用语和小型值协议，不接管 M02/M04 目录或状态规则。

B&B 仅作本地静态设计比较。核对的固定分支为 `wt-dev-main`，提交为 `055db2e0b7f4e31cbcdb906783d9c16f3ccd1049`；参考范围与内容摘要如下，不形成运行或构建依赖。

| B&B 资料路径 | SHA256 | 本轮采用边界 |
| --- | --- | --- |
| `docs/dev-plans/repo-governance/899-core-business-object-lifecycle-semantics-contract.md` | `1205ef366a3d108853aa13b9476f311b27b63099ad71b1771497829e47a8f8d8` | 采用业务类型/原因与记录处理分开的概念；各域独立拥有状态守卫 |
| `docs/dev-plans/orgunit/524-orgunit-field-metadata-and-ext-fields-contract.md` | `4866a5d2bdbc9388f8c01861a21b8f42a13dd5b9dbca184188c5924faea3016c` | 参考分类关系与历史快照；M02 自定更正/撤销必选原因，未照搬可选 note、删除或改日能力 |
| `modules/orgunit/services/orgunit_write_service.go` | `1df12454341209fdd8f321c67165b991f2f2e14313c655cb99aa0875f424b8d9` | 确认统一有效日期及独立分类字段的代码表达；不复制 DB Kernel 或业务实现 |

## 3 检查结果

环境为本机 Linux、Python 3.12.3 标准库及 Git；使用仓内既有生成器和资料检查，不新增依赖。本表只记录实际执行结果。

| 检查 | 实际结果 | 结论 |
| --- | --- | --- |
| 生成预检及生成 | `build_modules.py --dry-run` 与生成成功，覆盖 57 项产物 | 通过 |
| 独立只读复核 | `/root/m02_plan_contract_review` 对照前置快照检查分类、日期、历史及 DDD 边界，未发现需修正问题；未修改文件或运行测试 | 通过 |
| 文档与治理回归 | `make docs`：46 项 Python 回归通过；108 个 Markdown、2534 个本地链接、53 张图稿、197 项资产及生成一致性通过；`--check-refs` 核对六个固定分支通过 | 通过 |
| 差异格式 | `git diff --check` 无格式错误 | 通过 |
| 输入与历史保护 | 9 项受测输入、10 项快照及 282 项保护文件摘要匹配；旧 445 个词条 ID、39 个业务条目 ID 和六条修订记录保留，文件变更均在本轮白名单内 | 通过 |

独立复核只针对文档语义及静态差异；回归和资料检查由主代理实际运行。报告和登记定稿后再次核对最终输入摘要、登记、引用及差异格式，文档检查器自身未被修改。

## 4 限制与交接

本轮仅修改文档、术语数据及生成文本；没有修改业务代码或文档检查器，没有运行 Go、数据库、API、浏览器或产品构建，也未重新核对外部 Oracle/法律资料。011 中新增的验收条件仍是待实现的预期，不是本轮运行通过结果。

010/011 继续为草案，010 执行未开始；012/013 仍为预留且无正文。197 项来源资产及其实现/验证状态保持，文档检查不代替规格采纳或产品验收。[修订链](../reference/local-revisions.json)按 CHNHCM-011 的 revision 3 接续，旧六条修订不改写。报告及输入摘要自身排除出受测摘要，避免循环引用；临时输出位于 `.local/m02-operation-terms/`，持久证据以本报告、输入摘要及修订前快照为准。
