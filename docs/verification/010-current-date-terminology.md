# CHNHCM-010 当前日期用语修正验证

登记信息：[registry.json](../governance/registry.json) 的 `governance.verification_records`。适用范围：[010](../archive/plans/010-m02-scope-and-plan.md)、[011](../specs/m02/011-m02-business-requirements.md)与 [17 术语表](../architecture/target/17-terminology-glossary.md)的当前日期用语修正，遵循 [015 命名规范](../governance/015-terminology-naming.md)。

## 1 基线及范围

日期：2026-10-31；工作树 `/home/vivobook/projects/CHNHCM-wt-dev-a`，固定分支 `wt-dev-a`，HEAD 为 `f21c99ac42d1662bd7fc077399e27fd144d7a8d0`；开始时已有未提交文档变更。本轮按用户确认将“今日业务日期”改为“当前日期”，不提交或推送。

修改前核对前轮 7 项输入与 274 项保护文件，将本轮直接输入及上轮报告、摘要共 9 项保存到[修订前快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/010-before-current-date.zip)。原报告、摘要、来源包及其他变更保留，新证据见[本轮摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/010-current-date-inputs.json)。

## 2 修订与语义复核

主代理 `/root` 按用户确认的定义复核：

- 首选中文为“当前日期”，英文说明名为 Current Date；原“今日业务日期”“业务今日”和 Business Today 作检索别名。稳定词条 ID `x01-business-today`、锚点及引用保持，不新建同义词条或接口字段。
- 定义保持服务端按指定业务时区计算当天日期，M02 首批采用 `Asia/Shanghai`，用于默认查询和日期资格判断；跨午夜重新校验，不采用浏览器日期。
- 用户选择查看日期或本次生效日期不改变当前日期。011 日期规则、39 个场景/规则/验收标识及业务预期保留，010 同步简要说明。
- 17 由 JSON 重新生成；另同步业务生效时间词条中对此名称的引用。445 项词条数量及全部稳定 ID 保持，不修改其他术语或历史资料。

本轮修改项目用语，未重新查阅外部资料，也不把 Current Date 说明名当作 Oracle 统一界面译名或特定数据库时间函数的证明。

## 3 检查结果

环境为本机 Linux、Python 3.12.3 标准库及 Git，使用仓内既有生成器和检查。

| 检查 | 实际结果 | 结论 |
| --- | --- | --- |
| 命名及语义复核 | 名称按四字原则修正，日期含义、时区、引用及旧名保留 | 通过 |
| 生成及一致性 | 既有生成器预检及生成成功，57 项产物一致性检查通过；生成器和图稿内容未变 | 通过 |
| 文档与治理 | `make docs`：44 项 Python 回归通过；106 个 Markdown、2491 个本地链接、53 张图稿及 197 项来源资产检查通过 | 通过 |
| 差异格式 | `git diff --check` 无格式错误 | 通过 |
| 输入及历史保护 | 7 项受测输入、9 项快照及 277 项保护文件符合摘要；445 个词条 ID、39 个业务条目 ID、原登记及旧修订链保留 | 通过 |

本报告定稿后再次核对资料登记、引用、摘要及差异格式。上述通过范围限于命名修正及文档一致性，没有独立评审或业务运行结果。

## 4 证据与限制

[设计修订链](../reference/local-revisions.json)以 CHNHCM-010 衔接本次 JSON 和 17 的前后摘要，保留先前 CHNHCM-011、015 等记录；没有新建编号文档或反转既有日期规则。原 010/011 仍为草案，资产实现/验证状态不变。

本轮仅文档修正，不实现业务或运行 Go、数据库、API、浏览器和产品构建。本报告、[输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/010-current-date-inputs.json)与快照保存持久证据；临时输出位于 `.local/current-date/`，不作为唯一长期依据。
