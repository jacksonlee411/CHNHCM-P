# 中国大陆HCM图集

[返回总览](../README.md)。共53张目标设计图：5张综合概念、5张整体、20张业务模块、6张横向能力、9张联合流程、8张关键设计。每图都有SVG和Mermaid；点击SVG可放大查看。

本页只列独立China HCM目标设计。对象存在于目标图不表示已实现。模块图颜色仅表示职责：蓝色对象、橙色处理、灰色其他域/外部系统、绿色横向合同。

阅读主线：**L1整体业务架构 → L2领域概念 → L3模块设计 → L4关键设计**。先从G02总图定位业务区域，再按C编号展开；[对应关系与补充入口](../02-target-architecture.md#1-整体模块架构)说明跨域引用、招聘/服务及C05的横向范围。

图集采用的来源提交与待整合范围见[来源说明](../../../reference/README.md)。[源阶段发现记录](../../../reference/not-imported.md#source-32e2669f2b3b)及[源验证范围](../../../reference/not-imported.md#source-5801fdf4bf92)保留历史路由；本仓 [009](../../../verification/009-m02-terminology.md)仅重新渲染并检查了 C01、E01、E02、M02、M04 五张图，其余图面没有因此重新验收。

## L1 整体业务架构

G02是总图；G03补充事实归属，G05补充DDD限界上下文。它们是整体层的不同视角。

G02分别展示服务/专业办理入口、业务区域、配置与权限支持、技术设施及授权分析。实线表示事实、建议或外部交换，虚线表示办理访问或支撑，具体含义以标签为准；业务核准及结果确认归所属域。

| 图 | SVG | Mermaid |
| --- | --- | --- |
| G02 整体业务架构与概念展开 | [查看](g02-target-domain-map.svg) | [源图](g02-target-domain-map.mmd) |
| G03 事实归属与依赖 | [查看](g03-fact-ownership.svg) | [源图](g03-fact-ownership.mmd) |
| G05 限界上下文地图 | [查看](g05-context-map.svg) | [源图](g05-context-map.mmd) |

## L2 领域概念架构（对标HCM颗粒度）

C01—C04按业务主题展开；C05贯穿各业务域。概念粒度和DDD关系说明见[概念架构](../12-concept-architecture.md)。

| 图 | SVG | Mermaid |
| --- | --- | --- |
| C01 核心人事与任职引用 | [查看](c01-core-hcm-concepts.svg) | [源图](c01-core-hcm-concepts.mmd) |
| C02 薪资领域概念 | [查看](c02-payroll-concepts.svg) | [源图](c02-payroll-concepts.mmd) |
| C03 时间缺勤与权益 | [查看](c03-time-absence-concepts.svg) | [源图](c03-time-absence-concepts.mmd) |
| C04 人才评价与发展 | [查看](c04-talent-concepts.svg) | [源图](c04-talent-concepts.mmd) |
| C05 引用适用权限与核准 | [查看](c05-reference-access-concepts.svg) | [源图](c05-reference-access-concepts.mmd) |

## L3 业务模块设计

本组编辑源为 [business-modules.json](../data/business-modules.json)；表中 Mermaid 是生成产物，修改 JSON 后再生成。

| 图 | SVG | Mermaid |
| --- | --- | --- |
| M01 人员身份与个人资料 | [查看](m01.svg) | [生成图源](m01.mmd) |
| M02 企业、组织与地点 | [查看](m02.svg) | [生成图源](m02.mmd) |
| M03 职务、职级、职位与编制 | [查看](m03.svg) | [生成图源](m03.mmd) |
| M04 用工、任职与劳动合同 | [查看](m04.svg) | [生成图源](m04.mmd) |
| M05 招聘、录用与入职协同 | [查看](m05.svg) | [生成图源](m05.mmd) |
| M06 排班与工时制度 | [查看](m06.svg) | [生成图源](m06.mmd) |
| M07 考勤与实际工时 | [查看](m07.svg) | [生成图源](m07.mmd) |
| M08 休假、缺勤与权益账户 | [查看](m08.svg) | [生成图源](m08.mmd) |
| M09 薪酬体系、定薪与激励 | [查看](m09.svg) | [生成图源](m09.mmd) |
| M10 工资核算、累计与追溯 | [查看](m10.svg) | [生成图源](m10.mmd) |
| M11 个人所得税扣缴与申报 | [查看](m11.svg) | [生成图源](m11.mmd) |
| M12 社会保险与住房公积金 | [查看](m12.svg) | [生成图源](m12.mmd) |
| M13 企业福利 | [查看](m13.svg) | [生成图源](m13.mmd) |
| M14 工资支付与银行用途 | [查看](m14.svg) | [生成图源](m14.mmd) |
| M15 人工成本与财务衔接 | [查看](m15.svg) | [生成图源](m15.mmd) |
| M16 人才档案、技能与资格 | [查看](m16.svg) | [生成图源](m16.mmd) |
| M17 目标与绩效 | [查看](m17.svg) | [生成图源](m17.mmd) |
| M18 学习与发展 | [查看](m18.svg) | [生成图源](m18.mmd) |
| M19 人才盘点、人才池与继任 | [查看](m19.svg) | [生成图源](m19.mmd) |
| M20 员工经理服务与HR任务 | [查看](m20.svg) | [生成图源](m20.mmd) |

## L3 横向能力设计

本组编辑源为 [crosscutting-modules.json](../data/crosscutting-modules.json)，Mermaid 与模块说明同源生成。

| 图 | SVG | Mermaid |
| --- | --- | --- |
| X01 身份、有效期、修订与采用历史 | [查看](x01.svg) | [生成图源](x01.mmd) |
| X02 共享配置、适用范围、默认与政策版本 | [查看](x02.svg) | [生成图源](x02.mmd) |
| X03 身份权限与隐私 | [查看](x03.svg) | [生成图源](x03.mmd) |
| X04 任务、作业与人工核对 | [查看](x04.svg) | [生成图源](x04.mmd) |
| X05 导入导出与外部集成 | [查看](x05.svg) | [生成图源](x05.mmd) |
| X06 报表、指标与事实追溯 | [查看](x06.svg) | [生成图源](x06.mmd) |

## L4 关键对象、聚合与状态设计

| 图 | SVG | Mermaid |
| --- | --- | --- |
| G06 聚合与事务边界 | [查看](g06-aggregate-transactions.svg) | [源图](g06-aggregate-transactions.mmd) |
| E01 人员、关系、任职、合同与计薪 | [查看](e01-person-employment.svg) | [编辑源文件](e01-person-employment.mmd) |
| E02 法律实体及不同用途 | [查看](e02-legal-contexts.svg) | [编辑源文件](e02-legal-contexts.mmd) |
| E03 参缴、申报、扣收与资金 | [查看](e03-contribution-ledgers.svg) | [编辑源文件](e03-contribution-ledgers.mmd) |
| E04 生效、修订与采用 | [查看](e04-effective-adoption.svg) | [编辑源文件](e04-effective-adoption.mmd) |
| E05 账户身份、用途与付款 | [查看](e05-bank-payment-identity.svg) | [编辑源文件](e05-bank-payment-identity.mmd) |
| E06 外部执行与未知结果 | [查看](e06-external-state.svg) | [编辑源文件](e06-external-state.mmd) |
| E07 税评估、承诺与累计确认 | [查看](e07-tax-commitment.svg) | [编辑源文件](e07-tax-commitment.mmd) |
| E08 工资单发布与撤回 | [查看](e08-payslip-publication.svg) | [编辑源文件](e08-payslip-publication.mmd) |

<a id="跨模块流程"></a>

## 贯穿各层的业务流程

| 图 | SVG | Mermaid |
| --- | --- | --- |
| F01 招聘到首次工资 | [查看](f01-hire-first-pay.svg) | [编辑源文件](f01-hire-first-pay.mmd) |
| F02 调动与晋升调薪 | [查看](f02-transfer.svg) | [编辑源文件](f02-transfer.mmd) |
| F03 假勤到工资 | [查看](f03-time-payroll.svg) | [编辑源文件](f03-time-payroll.mmd) |
| F04 绩效到支付及财务 | [查看](f04-performance-to-finance.svg) | [编辑源文件](f04-performance-to-finance.mmd) |
| F05 历史更正与差额 | [查看](f05-retro-correction.svg) | [编辑源文件](f05-retro-correction.mmd) |
| F06 独立参缴闭环 | [查看](f06-contribution.svg) | [编辑源文件](f06-contribution.mmd) |
| F07 离职退休与再聘 | [查看](f07-exit-rehire.svg) | [编辑源文件](f07-exit-rehire.mmd) |
| F08 人才发展与任命 | [查看](f08-talent-development.svg) | [编辑源文件](f08-talent-development.mmd) |
| F09 可靠通知与消费者采用 | [查看](f09-reliable-delivery.svg) | [编辑源文件](f09-reliable-delivery.mmd) |

## 技术与部署视图

G04说明实现与部署选择，与上述业务阅读层级分别表达。

| 图 | SVG | Mermaid |
| --- | --- | --- |
| G04 正式技术与部署 | [查看](g04-formal-deployment.svg) | [源图](g04-formal-deployment.mmd) |

## 编辑来源与本地检查

模块描述及 M/X 图由 [business-modules.json](../data/business-modules.json) 和 [crosscutting-modules.json](../data/crosscutting-modules.json) 生成；概念矩阵由 [concept-mapping.json](../data/concept-mapping.json) 生成，术语及双语引用由 [terminology-glossary.json](../data/terminology-glossary.json) 维护。[命令索引数据](../data/command-contract-index.json)只保存导航关系，聚合归属仍从模块数据读取。C/G/E/F 图直接编辑对应 Mermaid，SVG 由渲染器生成。工具原件及本仓路径见 [document-tooling.json](../../../reference/document-tooling.json)。

从本仓根目录执行以下只读检查；Python 使用标准库，渲染器的帮助和 dry-run 不需要 Mermaid CLI 或浏览器：

```bash
python3 -B docs/architecture/target/scripts/build_modules.py --help
python3 -B docs/architecture/target/scripts/build_modules.py --dry-run
python3 -B docs/architecture/target/scripts/build_modules.py --check
node docs/architecture/target/scripts/render_diagrams.mjs --help
node docs/architecture/target/scripts/render_diagrams.mjs --dry-run
make docs
python3 -B scripts/check-repository.py --check-refs
```

修改编辑源后，使用下列命令重新生成 57 份文本产物，再核对实际差异：

```bash
python3 -B docs/architecture/target/scripts/build_modules.py
python3 -B docs/architecture/target/scripts/build_modules.py --check
```

生成检查覆盖 26 份模块说明、26 份 M/X Mermaid、概念 Markdown/CSV、聚合目录、命令索引及术语表。它不渲染 SVG，也不验证图面、外链时效或全部领域语义。采用文件变更需按 [014](../../../governance/014-document-management.md) 保存影响与验证记录，并追加[本仓修订链](../../../reference/local-revisions.json)，不能覆盖初始采用摘要。

## 渲染与图面证据

实际渲染使用 [render_diagrams.mjs](../scripts/render_diagrams.mjs) 的 `--cli-root`、`--browser` 和可选 `--only` 参数。需要已有的 Mermaid CLI、Chrome 及中文字体；工具准备在本工作树的独立文档工具目录进行，不引用相邻项目的 node_modules 或服务。[009 渲染记录](../data/render-partial.json)记载当时使用 Mermaid CLI 12.0.0、Chrome 149.0.7827.102；这些是历史环境记录，不表示当前工具已经安装。

该脚本会写 SVG，并将全量或局部清单写入 `data/render-manifest.json` 或 `data/render-partial.json`。现有 `render-partial.json` 是 009 的定稿证据；后续渲染应在任务隔离副本中进行，将新清单以本次验证的唯一文件名保存，再同步受影响 SVG、修订链及新报告，不能覆盖旧清单冒充新证据。隔离副本只用于文档工具操作，不进入产品构建。

实际图面检查须逐张查看受影响图形，记录文本裁切/重叠、连线与标签的检查结果、环境和源/产物摘要。当前没有引入源仓的 `inspect_diagrams.mjs`；`--dry-run` 只列范围，XML 可解析也不能代替图面复核。源[渲染清单](../../../reference/not-imported.md#source-2553dccdb474)和[画布记录](../../../reference/not-imported.md#source-f9d8908714e6)仅提供历史路径，未声明原文均已打包。

## 本仓检查与来源 CI

本仓当前没有远程 CI，实际本地入口为 `make docs` 和上述仓库检查；托管接入状态见[工作流范围说明](../../../../.github/workflows/README.md)。[源 GitHub Actions 工作流](../../../reference/not-imported.md#source-613f49ffd8a8)和[源工作区引用清单](../../../reference/not-imported.md#source-2328b9491b94)只保留历史路由，不作为本仓已运行的工作流或可执行指令。

本地文档检查的覆盖与限制由 [014](../../../governance/014-document-management.md) 持有，实际执行记录见[验证入口](../../../verification/README.md)。它不重做图面或业务验收，源仓的 `validate_docs.py` 也不是本仓检查入口。

## 历史图纸

G01 不在当前 53 图和生成清单中；[历史资料路由](../../../reference/not-imported.md#source-dac46486614d)记录其源路径，不表示该图已经打包进本仓。需要查阅未采用材料时按固定来源提交定位，不作为当前构建或验证依赖。
