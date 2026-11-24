# CHNHCM-506 职务信息业务与页面设计复核

## 1 范围与基线

2026-11-24 在固定树 wt-dev-f 的 `56189a0e59a1a8f2f35c34f5ae2d8b8e408ca37a` 上复核工作区文稿。初始工作区干净。本次受审材料是新增的 506、507，500、502、505 和 401–403 的衔接，以及入口、来源、资产与登记。Codex 主代理对照人事经办、受限读者和 Staffing 写入三种视角做文档复核。[业务规则](../specs/m04/506-job-information-records.md) · [页面设计](../specs/m04/507-job-information-tab.md) · [登记表](../governance/registry.json)

| 依据 | 核对范围及采用边界 |
| --- | --- |
| PeopleSoft HCM 9.2.055 | 官方 Job Data 资料以人员和任职记录号区分并行工作，在 Job Actions Summary 按权限处理所选行。CHNHCM 采用独立任职与生效记录。同日动作按本项目的业务前序解释，普通保存走直接办理。[任职记录编号](https://docs.oracle.com/en/applications/peoplesoft/human-capital-management/9.2.055/peoplesoft-human-resources-administer-workforce/employment-record-numbers.html) · [行操作](https://docs.oracle.com/en/applications/peoplesoft/human-capital-management/9.2.055/peoplesoft-human-resources-administer-workforce/fluid-updating-job-data.html) · [同日动作](https://docs.oracle.com/en/applications/peoplesoft/human-capital-management/9.2.055/peoplesoft-human-resources-administer-workforce/changing-job-data.html) |
| 本地 Bugs-And-Blossoms | 在提交 `a76db79227fdc4a0805fcd6e37fe22898d8b02ea` 核对 899 §10、894 §2.4 和 §21、1113 §4.42，留存三份文档的 SHA-256。采用记录操作与业务处理方式的绑定，借鉴一主两兼和受限读者样本。本仓原人独立登记，未来首单在生效前符合既有业务条件时可整笔撤销，沿用 401 的原办理与后继校验。[来源定位](../reference/506-job-information-sources.json) · [原办理与未来撤销](../specs/m04/401-direct-hire-requirements.md#m04-br-007) |
| Odoo 19 与 CHNHCM 参考页 | 核对 19.0 固定提交员工表单的身份区、Work 页签、工作与地点分组。以本仓员工详情、职务详情、BU 表单、横向日期轴和正常菜单返回作为实现时的直接参照。[Odoo 员工表单](https://github.com/odoo/odoo/blob/74d39b2fdaa8cd60885943fa58a08bff62e53207/addons/hr/views/hr_employee_views.xml#L149-L168) · [507 继承表](../specs/m04/507-job-information-tab.md#4-页面继承与适用差异) |
| 现行合同与实现 | 核对 106 第 8 节、401–403、500、502、505、012 与 121，以及人员详情、首单查询、00037 和名单采用的实际实现。e 的 404 在 `ec4572a27c791d6c5194c40f6c17ade2b2b3ab5e` 定稿，按岗位免占编产品验收留待其实施。接入页签时须承接已办首单及其链接。[按职务首单实测](400-first-hire-integration.md#decision-first-hire) · [数据库起点](../../db/migrations/00037_employee_hire.sql) · [批次衔接](../plans/500-personnel-roadmap.md#job-information-delivery) |

## 2 沿业务问题核对文稿

| 核对的问题 | 文档结论和可复查位置 |
| --- | --- |
| 业务变化与录错怎样区分 | 将真实异动、普通资料变化、迟报、原笔填错和整笔误建分别走查。对照未来地点由甲改乙、把乙纠正为丙的样本，今天与生效日的查询答案可分别断言，原笔只有一项业务发生。[记录操作矩阵](../specs/m04/506-job-information-records.md#m04-br-010) · [地点验收](../specs/m04/506-job-information-records.md#m04-ac-010) |
| 关系级转正怎样出现在三份任职上 | 按一主两兼逐行核对，关系持有一次转正，调动落在原主岗。转正时的研发部、当天主岗的产品部、两份兼岗的归属可以同时解释。结束一岗的样本写明最后任职日和次日计数。[同日与多任职](../specs/m04/506-job-information-records.md#m04-ac-012) |
| 谁看哪一个页签、保存落在哪里 | 复核现有详情按 M01 历史权决定是否出现页签的代码，修订 502 的双页签表述。将 M04 阅读／办理、M01 历史、个人九栏整页草稿和 M04 逐笔保存分别落实到同一身份下的页面。显式原件与当前业务日查阅各走其授权。[现有详情](../../apps/web/src/features/personnel/Detail.tsx) · [页面布局](../specs/m04/507-job-information-tab.md#job-tab-layout) · [页面场景](../specs/m04/507-job-information-tab.md#m04-ac-016) |
| 现有首单能否直接接第二次调动 | 检查 00037 的办理类型及每关系每类型唯一约束、现有读取按首单取值。分批表先接首单实物，再前向扩展同类多笔与逐段读取，并以旧原件、回执、名单防重和单笔撤回检验升级。[读写与升级](../specs/m04/403-hire-technical-contract.md#job-information-handoff) |

## 3 验证范围与交接

本次运行 `make docs`，180 项文档与检查脚本回归通过，仓库资料检查通过。`make boundaries` 核对 706 个 Go 文件和 114 份静态 SQL 来源，通过。`git diff --check`、新文件空白与末行检查通过。逐项核验两份规格的登记、三份本地 B&B 原件摘要以及体验、业务和验证三类设计采用状态。补充的逐链接检查首轮因重复读取文件清单主动中止，改为一次读取清单后，105 处新增或改写文句中的仓内链接及其锚点均通过。

以上为设计与实现基线的对读。旧首单、Bugs-And-Blossoms 的菜单实测各保留原受测范围。新页签、多动作与多任职的 Go、DB/API、正常菜单和视口验收均待对应批次实施。[500 交付表](../plans/500-personnel-roadmap.md#job-information-delivery)

<a id="job-record-number-review"></a>
## 4 主兼岗编号与任职行复核

2026-11-24 在同一 HEAD 接续已有文稿。按用户确认的规则重新走查编号、部门筛选和详情入口。Oracle 的 Bobby 示例在原 0 号记录上再聘，Job Indicator 另表组织关系内的主次。按人终身保留一个 0、已发正号不回收及本期暂缓转换是 CHNHCM 的业务决定。来源索引已分别登记两处原文。[再聘示例](https://docs.oracle.com/en/applications/peoplesoft/human-capital-management/9.2.055/peoplesoft-human-resources-administer-workforce/understanding-job-data.html) · [Job Indicator](https://docs.oracle.com/en/applications/peoplesoft/human-capital-management/9.2.055/peoplesoft-human-resources-administer-workforce/fluid-adding-organizational-instances.html) · [来源索引](../reference/506-job-information-sources.json)

| 原设计中的冲突 | 本轮核定的合同及预期 |
| --- | --- |
| T04 把切换主岗列为本期办理，跨关系容易把每段的主任职都读成 0 号 | 人层仅一个 0，关系内管理选择继续按 401 核资格。离职后再聘沿 0 建新任期。1、2 号结束后新增兼岗为 3。普通更正和改日均保留编号，API、菜单均不提供转换。[业务样本](../specs/m04/506-job-information-records.md#m04-ac-014) · [技术合同](../specs/m04/403-hire-technical-contract.md#job-information-handoff) |
| 原名录按人分页，先打开本人再选任职 | 员工列表以任职列行。产品部有周宁 0、1 两行，培训部有 2 号一行，各部门都是 1 人。返回恢复原部门与原行。M01 档案查询仍能找回尚未入职者。[页面示例](../specs/m04/507-job-information-tab.md#job-tab-layout) |
| 原共享资料样本从两岗都打开个人资料 | 人员档案与 0 号按权访问同一份 M01 资料。正号兼岗仅有职务信息，单份可见兼岗直接展开，历史和深链逐任期授权。[资料验收](../specs/m01/505-personal-profile-contract.md#m01-ac-011) · [页签验收](../specs/m04/507-job-information-tab.md#m04-ac-016) |

本轮按文档样本复算了部门行数和人数，核对两种详情页签、0 号前后任期及新增兼岗取 3 的预期。相对基线新增或改写文句中的 143 处仓内链接及锚点可定位，4 个未跟踪文件的行尾和空白检查通过。`make docs` 180 项回归及仓库资料检查通过，`make boundaries` 和 `git diff --check` 通过。编号落库、真实并发和菜单交互仍是 T03、T04 的产品验收项。
