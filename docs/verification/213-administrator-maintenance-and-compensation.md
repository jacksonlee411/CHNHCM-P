# CHNHCM-213 管理员维护生效与薪酬首期范围核对

## 1 基线与决定

2026-11-27 在 `wt-dev-c` 的 `1284f50e32c16d7191d6ea8f375c2e4b67aa7941` 及未提交的 210 至 212 设计上核对。此前走查将津贴联动、薪档和复核责任列为当时待议事项。本次依据用户对全项目阶段的明确要求，将获权管理员直接办理、按生效日期管理写入实施规范。根据两套样例和行业对照，路线图选取计划内固定津贴、定额或连续薪带。213 记录阶段决定，210 至 212 继续按各自范围送审。[阶段决定](../adr/213-administrator-maintenance-and-effectivity.md) · [原走查](210-compensation-design-walkthrough.md) · [登记表](../governance/registry.json)

## 2 业务与来源核对

| 核对项 | 观察及结论 |
| --- | --- |
| 项目阶段与目标架构 | 原薪酬稿将发布复核留给客户选择，DDD 只列出编制直接办理。213 与 003 明确各模块本阶段实施规则，DDD 和 M09 区分本阶段与远期目标。共享页面及薪酬计划、页面和验收预期均能追溯到 213。 |
| 固定津贴 | Fusion 文档提供按金额及按规则确定津贴的例子，PeopleSoft 提供固定金额费率代码。首期将固定津贴写在具体计划。另以两个计划的同名津贴为独立产品验收样例，十月只把 PRO-M 从 300 调至 350 元/月，OPS-M 维持 200 元/月。这个场景与两计划共用 P5 薪带分别验证。 |
| 离散薪档 | Fusion 记录了无薪档时使用职级费率的场景，PeopleSoft 明列职级内的薪档是可选设置。当前 P5 的区间和年定额都可直接建标。集中调整固定津贴和逐档金额表留待有对应制度的后续专项。[官方来源与适用说明](../reference/README.md#compensation-configuration-benchmark) |

## 3 原型复验

从「设置 → 薪酬配置」用本地 Chromium 140.0.7339.186 复跑已有走查。1440×900、1440×540 的 100% 和原生 200% 均通过，检查到 76 个页面及来路组合，未捕获页面脚本异常。缩放复原，临时浏览器已关闭。原型及脚本的摘要与 210 原走查一致。[原型](../specs/m09/prototype/compensation-configuration.html) · [走查脚本](data/210-compensation-design/walkthrough.mjs)

```sh
node docs/verification/data/210-compensation-design/walkthrough.mjs --native-zoom
```

另以无头 Chromium 从正常菜单核对四项。专业计划的岗位津贴输入为 300.00，运营样例只有基本工资。计划保存样例直接回详情。七月月标的保存样例直接显示 16,000.00 和「尚未生效」，以上表单及结果均无送审或待审批提示。新建薪标的形式选择只有「薪带」「定额」，无薪档字段。

`make docs` 的 189 项回归和仓库资料检查通过，目标架构生成物与来源一致，`git diff --check` 通过。

## 4 实施验收边界

本轮核对设计资料与预置页面。后续 I1、I2 由同一位有权管理员实测新建、更正、启停和撤去，核对回执、生效日前后、历史按现权读取及失败零写入。数据库中零人工审批任务、两个同名津贴的独立保存结果也待产品实测。原验收环境未参与。[首期业务场景](../specs/m09/211-compensation-configuration.md#design-cases) · [实施里程碑](../archive/plans/210-compensation-configuration-roadmap.md)
