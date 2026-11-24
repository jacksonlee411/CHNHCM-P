# CHNHCM-506 职务信息直接办理负担复核

## 1 受审范围

2026-11-24，Codex 在 wt-dev-f 的 `87c4b07dd26acb9b4f21a0bb441a5a7cc3f1a519` 及上一轮工作区文稿上复核 401、403、500、506、507。首单表单在手工办理入职时要求填写任用通知说明。本次修订界定在职维护各自需要的输入，沿用 012 的直接保存、106 的日级状态和 313 的已履职实况口径。[已有首单表单](../../apps/web/src/features/hire/HirePage.tsx) · [上一轮同日办理复核](506-same-day-actions-design-review.md) · [登记表](../governance/registry.json)

## 2 五项修订走查

| 操作 | 核对的输入与业务结果 |
| --- | --- |
| 改地点后纠正错字 | 普通变更省略任用通知，纠错只改地点。旧值、办理人和记录时刻仍可查，两次保存只计一次业务变更。[普通更正样本](../specs/m04/506-job-information-records.md#m04-ac-010) |
| 撤销误录晋升 | 原通知或一条可追溯的误录原因均可采用。撤销后独立调动保留，资格依赖失效则整笔拒绝。[五步状态对照](../specs/m04/506-job-information-records.md#m04-scn-018) |
| 满编时拟调入或补录已履职 | 同一门店岗位拟调入提示缺 1 编。已有任职的实况补录按有效期间记核 3、实占 4，保留他人预留。校验已有履职事实、事实更正权及任职资格。[313 原样本](../specs/m03/313-position-capacity-contract.md#m03-ac-129) · [职务信息入口](../specs/m04/507-job-information-tab.md#actual-duty-entry) |
| 一份通知同时办理两项 | 从原任职直接进入，共用通知录入一次、两笔原件各可查。另一项确有独立来源时仅指定该项。单项资格失败另验，业务入口可直接办理合法组合。[专用办理验收](../specs/m04/506-job-information-records.md#m04-ac-019) |
| 补录顺序与转正归属 | 原记录能确定先后时自动带入，独立事项直接保存。仅统计归属不明时由系统记原因，可以从兼岗完成关系转正。若资格或授权取决于该部门，无法核实则零业务写入。[关系转正规则](../specs/m04/506-job-information-records.md#m04-br-014) · [受限经办人验收](../specs/m04/507-job-information-tab.md#m04-ac-018) |

## 3 实施与验证范围

逐项对照了业务样本、表单输入、提交合同和 T04 交付顺序。独立复算地点更正前后值、门店岗三天的实际用量和次日边界，并复算 106 样本的五步日状态、事件次数、登记顺序互换及保留或拒绝后继调动的条件。上述结果均为设计预期。

`make docs` 的 180 项文档治理回归及仓库检查、`make boundaries` 和 `git diff --check` 均通过。另核对本次改写文句的仓内链接、锚点及新增文稿空白。

附加的 `check-repository.py --check-refs` 返回缺少本地分支引用。按 005 对照登记表，本机有 main、wt-dev-e、wt-dev-f 三个本地分支，wt-dev-a 至 wt-dev-d 仅有 origin 跟踪引用。登记的七个 origin 跟踪引用均在。[005 文档交付检查](../governance/005-branches-and-documents.md)

T04 按职务先验地点变更及原笔更正，按岗批次从同一满编岗位走拟调入和实况补录。共同办理、原笔撤销、关系级授权、并发及回滚随真实 Go、数据库、API 和正常菜单验收。[T04 分批安排](../plans/500-personnel-roadmap.md#job-information-delivery)
