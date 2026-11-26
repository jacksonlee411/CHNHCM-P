# CHNHCM-400、CHNHCM-404 首次入职计划归档与主线复验

400 和 404 于 2026-11-26 完成执行。2026-11-27，wt-dev-e 合入主线 #29 后，按 014 将两份计划移入归档。登记表保留原审阅、执行关闭依据和文件原路径。4496 上的签收对应产品提交 `ccd0900a`。本轮在合并提交 `ffe2c82e` 上用独立 PostgreSQL 复验两分支交叉的行为。[归档规则](../governance/014-document-management.md#82-归档) · [400 原计划](../archive/plans/400-hire-readiness.md) · [404 原计划](../archive/plans/404-position-exempt-first-hire.md) · [4496 原卷签收](404-4496-acceptance.md)

## 1 已采纳条款的去向

| 原计划位置 | 适用条件、结果与现行合同 |
| --- | --- |
| 400 第 1–4、6 节，首单的准备与提交 | M01 核对本人的原档，M02 核对整个拟任期间的主体用途。M04 保存关系、首任职、适用的两层主选择及来源，资格收窄与入职在同一组守卫下重核。[身份](../specs/m04/401-direct-hire-requirements.md#m04-br-002) · [主体用途](../specs/m04/401-direct-hire-requirements.md#m04-br-004) · [主选择](../specs/m04/401-direct-hire-requirements.md#m04-br-006) · [公开端口](../specs/m04/403-hire-technical-contract.md#hire-ports) |
| 400 第 5–6 节、404 第 3、5 节，原决定、菜单与恢复 | 人事从人员档案带入原单及本人行。相同原单行重复提交定位同一次办理，私人回执按经办人查回。默认详情可见后续撤销，指定签发修订仍显示当时状态。业务日期与原件读权分别核对。[人员菜单](../specs/m04/402-hire-interactions.md#hire-form) · [决定及原行](../specs/m04/403-hire-technical-contract.md#appointment-original) · [撤销与查回](../specs/m04/403-hire-technical-contract.md#hire-recovery) · [同日事实](../adr/106-same-day-personnel-actions-and-daily-state.md#first-hire-adoption) |
| 400 第 4 节、404 第 3–5 节，三种模式与补员 | 仅按职务时岗位和占编为零。按岗免占保留实岗及 M04 使用来源，暂停补员时拒绝新增。占编段按 M03 定额、M04 实占及 M05 预留重核，跨段任期整笔提交。最后一份名额只供一笔使用。[任职模式](../specs/m04/401-direct-hire-requirements.md#m04-br-005) · [预留与占用](../specs/m03/313-position-capacity-contract.md#m03-br-027) · [暂停补员](../specs/m03/313-position-capacity-contract.md#m03-br-029) · [跨段验收](../specs/m04/401-direct-hire-requirements.md#m04-ac-008) · [事务](../specs/m04/403-hire-technical-contract.md#hire-transaction) |
| 400 第 5 节、404 第 6–7 节，验收与交付 | 五种常规视口、键盘和关键流程的原生缩放按 012 验收。固定环境由责任树按 003 更新，候选按 137 核对本地与远程门禁。计划采纳状态由原审阅支持，执行完成由登记的关闭证据支持。[页面验收](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-viewport-matrix) · [环境更新](../adr/003-implementation-stages.md#acceptance-environment-update) · [CI 规则](../governance/137-actions-and-validation.md) · [文档状态](../governance/014-document-management.md#6-生命周期与完成口径) |

## 2 合并候选的实测与接续

在独立 PostgreSQL 上运行 `internal/bootstrap` 的五项集成测试，全部通过，合计 142.535 秒。用例覆盖首次入职与主体用途收窄的两种提交顺序、默认原办理与签发修订的撤销状态、旧库撤销及替代首单升级、按岗免占与暂停补员，以及 M04 入职和 M05 招聘同时争最后一编。竞争中一笔成功、一笔返回 409，实占加预留为一。[首单与用途](../../internal/bootstrap/first_hire_integration_test.go) · [原件与修订](../../internal/bootstrap/record_modes_restore_integration_test.go) · [旧单升级](../../internal/bootstrap/employment_record_upgrade_integration_test.go) · [免占与补员](../../internal/bootstrap/position_hire_integration_test.go) · [最后一编竞争](../../internal/bootstrap/counted_position_integration_test.go)

客户开放主选择政策换版时，由 M04 政策维护者按 AC-009 补验政策激活与同期入职。4496 继续由 wt-dev-e 负责，在下次更新或用户验收结束时复查原单、私人回执和实岗免占样本。[政策激活条件](../specs/m04/401-direct-hire-requirements.md#m04-ac-009) · [4496 样本与实测结果](404-4496-acceptance.md)
