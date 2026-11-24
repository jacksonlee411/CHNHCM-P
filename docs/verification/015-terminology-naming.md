# CHNHCM-015 中文命名原则登记验证

登记信息：[registry.json](../governance/registry.json) 的 `governance.verification_records`。验证对象为 [015 规范](../governance/015-terminology-naming.md)、[17 术语表](../architecture/target/17-terminology-glossary.md)及其源数据、登记与导航。

## 1 基线与变更

日期：2026-10-31；工作树 `/home/vivobook/projects/CHNHCM-wt-dev-a`，固定分支 `wt-dev-a`，HEAD 为 `f21c99ac42d1662bd7fc077399e27fd144d7a8d0`，开始时工作区已有未提交变更。本轮依据用户指令增加命名原则，不提交或推送。

修改前核对前轮 8 项输入及 269 项保护文件，保存[本轮前置快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/015-before-naming-principle.zip)的 8 项资料。原 011 报告、摘要、来源包和既有修改均保留；本轮另建[输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/015-terminology-naming-inputs.json)与修订链记录。

## 2 规则及语义复核

主代理 `/root` 对照用户原意复核以下内容，不将工具检查当作独立评审或额外人工批准：

| 用户要求 | 登记内容 |
| --- | --- |
| 优先双字或四字 | 保留为命名偏好，不机械截字或改变含义 |
| 一般不超过六字 | 作为默认长度原则；未将存量长名称自动豁免 |
| 超过四字一般为组合词汇 | 要求组合部分含义明确，避免解释性长句充当名称 |
| 例外有明确权威来源或惯例 | 记录具体依据、适用语境和保留理由；未核验不得记为已确认 |

015 持有规则及维护方式，17 的源数据增补带日期的原则及规范链接，不把本轮内容混入 2026-10-30 的外部建议评审范围。445 项词条及其 ID、中文名称、定义、别名、来源和对象/概念绑定均保持不变；本轮不宣称全部存量名称已合规。

## 3 检查结果

环境为本机 Linux、Python 3.12.3 标准库及 Git，使用已有生成器和文档检查。

| 检查 | 实际结果 | 结论 |
| --- | --- | --- |
| 规则语义复核 | 四项要求及有据例外、语义保留、历史范围已在 015 与 17 源数据中表达 | 通过 |
| 生成与一致性 | 既有生成器预检及生成成功，57 项产物一致性检查通过；目标设计内容仅 JSON 和 17 增加一条带日期的原则 | 通过 |
| 文档与治理 | `make docs`：44 项 Python 回归通过；105 个 Markdown、2480 个本地链接、53 张图稿与 197 项来源资产检查通过 | 通过 |
| 差异格式 | `git diff --check` 无格式错误 | 通过 |
| 输入与保护范围 | 7 项受测输入、8 项前置快照及 274 项保护文件摘要核对通过；445 项词条完整不变，原登记及修订链保留 | 通过 |

015 依据用户明确提出的原则登记为 active，主代理已复核规则表达及语义保留边界，审阅范围见登记表。本报告定稿后再次检查文档登记、引用、受测摘要和差异格式；不把该状态扩展为全词汇表或业务验收。

## 4 范围与证据

本轮只登记原则与维护入口；没有逐项审阅 445 项名称、认定具体例外、更改术语或业务规则、增加自动字数门禁、实施或验证业务。未重新查询外部权威资料，既有 010/011 生命周期及资产实现状态保持。

[本轮输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/015-terminology-naming-inputs.json)区分 Git 基线、修改前工作区及受测文件；报告与摘要自身不纳入以避免自引用。[设计修订链](../reference/local-revisions.json)以 CHNHCM-015 衔接本轮 JSON 与 17 摘要，不覆盖前次 CHNHCM-011 记录。临时输出位于 `.local/015-naming/`，长期证据以本报告、摘要和快照为准。
