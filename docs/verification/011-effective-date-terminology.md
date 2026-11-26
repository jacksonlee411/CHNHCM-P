# CHNHCM-011 生效日期用语修正验证

登记信息：[registry.json](../governance/registry.json) 的 `governance.verification_records`。适用范围：[010](../archive/plans/010-m02-scope-and-plan.md)、[011](../specs/m02/011-m02-business-requirements.md)、[17 术语表](../architecture/target/17-terminology-glossary.md)的生效日期用语，以及 [014](../governance/014-document-management.md)中的连续修订登记说明。

## 1 基线与范围

日期：2026-10-31；工作树 `/home/vivobook/projects/CHNHCM-wt-dev-a`，固定分支 `wt-dev-a`，HEAD 为 `f21c99ac42d1662bd7fc077399e27fd144d7a8d0`，开始时已有未提交变更。本轮按用户要求执行前轮用语建议，不提交或推送。

修改前核对前轮 7 项输入和 277 项保护文件；将本轮相关文件及前轮报告、摘要共 12 项保存到[修订前快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/011-before-effective-date.zip)。旧来源、报告和摘要保持原字节，本轮受测对象见[输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/011-effective-date-inputs.json)。

## 2 修订内容

新建、新增记录表单与记录详情统一使用“生效日期”；停用操作使用“停用日期”，表示停用事实的生效日期。“本次”只在说明文字中限定当前操作，不再单列日期名称。查看日期用于查询，原记录与新记录可有不同生效日期；普通更正保留所选记录的原日期。当前日期、时区、未来资格和停用期间规则不变。

17 的业务生效时间词条承接上述按日场景说明，当前日期词条同步相关名称；只修改两个词条的说明，445 个词条的名称、ID、别名及对象/概念绑定均保留。010、011 同步用语及操作定位，不新增日期维度或业务聚合。

现有摘要链将文档号与一次修订混为同一唯一键，不能追加同一文档的后续修订。本轮补充可选 `revision` 序号：历史缺省为 1，后续按同文档连续追加；保留全链文件摘要衔接、路径限制和摘要合法性检查。原修订不改写，本次以 CHNHCM-011 的 revision 2 追加，不为普通用语修订新占文档号。新增两项回归分别验证跨文档交错的连续修订和非法/重复/跳号拒绝，并检验旧摘要不能绕过新版本链。

## 3 检查结果

环境为本机 Linux、Python 3.12.3 标准库及 Git；使用既有生成器与文档检查，不新增依赖。

| 检查 | 实际结果 | 结论 |
| --- | --- | --- |
| 生成预检及生成 | 既有生成器预检、生成成功，覆盖 57 项产物 | 通过 |
| 独立只读复核 | `/root/m02_plan_contract_review` 静态复核日期用语、规则保留、连续修订校验及两项新增测试；未发现需修正问题，原五条修订记录保留 | 通过 |
| 文档、治理与修订链回归 | `make docs`：46 项 Python 回归通过；107 个 Markdown、2502 个本地链接、53 张图稿及 197 项来源资产检查通过 | 通过 |
| 差异格式 | `git diff --check` 无格式错误 | 通过 |
| 输入及历史保护 | 10 项受测输入、12 项快照及 277 项保护文件摘要匹配；445 个词条 ID、39 个业务条目 ID 保留，38 个条目正文原字节未改，仅 BR003 同步日期用语 | 通过 |

独立代理未运行业务测试，表内回归结果由主代理实际执行获得。014 本次审阅只覆盖新增连续修订说明及对应校验；其余条款的原评审保留在原验证记录和前置快照中。本记录定稿后再次核对资料登记、引用、摘要与差异格式。

## 4 记录与限制

本轮仅修改文档和维护用的修订链检查，不修改业务代码，也不运行 Go、数据库、API、浏览器或产品构建；未重验 Oracle 或法律资料。010/011 整体草案状态与资产实现/验证状态保持，文档检查不代表业务验收。

[修订链](../reference/local-revisions.json)、[输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/011-effective-date-inputs.json)和快照保存各次内容及摘要；报告和摘要自身排除以避免自引用。临时输出位于 `.local/effective-date/`，长期证据以本报告及持久摘要为准。
