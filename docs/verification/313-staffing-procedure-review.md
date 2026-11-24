# CHNHCM-313 编制与招聘办理复核

## 1 办理与验收

2026-11-21，承接[上一轮简化](313-capacity-simplification.md)。本轮核对编制、招聘、人事、员工历程及共用影响处理合同。下表记录方案推演和对应的产品验收条件。

| 场景 | 应能从正常业务入口核对的结果 |
| --- | --- |
| 需求只改招聘说明 | 原预留有效。编制消费方保存来源、版本、政策和无影响依据。重复通知仍只有一份判定，影响案及结案回执均为零。缺依赖时待核，已封存工资按工资政策保留追溯依据。[共用合同 AC07](../architecture/target/10-ddd-and-state-contracts.md#ac07) |
| 人事已办入职后，来源又有变更 | 仅改说明而任职仍有效，历程步骤保持已满足。明确撤销后自动变为未满足，原完成依据可查。两种情况均无人工复核待办。依据相互矛盾时才交给人处理，模板要求人作决定的步骤仍由有权人办理。[员工历程](../architecture/target/modules/m20.md) |
| 需求预留、缩招或取消 | 预留引用补缺依据，随缩招或取消释放时引用该次决定。招聘继续却决定提前释放时，独立记录释放日和新原因。[正常入口](../specs/m03/313-position-capacity-contract.md#5-正常入口与应用合同) |
| 五份普通空缺缩到三份 | 单次保存结束两份可互换的空缺。样本中 01、02 已受保护，03 至 05 可互换，按份额顺序结束 03、04，留下 01、02、05。已绑定且去留有别时，由招聘选择。重试可查原两份。[AC068](../specs/m03/313-position-capacity-contract.md#m03-ac-068) |
| 两份录用争最后一编 | 两名招聘人员均从未预留的草稿发出录用，只有先取得名额的一方可作出承诺。缺额、缺新增预留权或保存失败时保留原状。已提前落实的预留可供有发出权的人沿用。[AC099](../specs/m03/313-position-capacity-contract.md#m03-ac-099) |
| 录用已接受，资料后补齐 | 人事可直接核实身份和到岗安排，保存一次。成功后招聘可查同一任职，预留减一、已占加一。失败时保留预留。换人后的旧交接失效。[AC061](../specs/m03/313-position-capacity-contract.md#m03-ac-061) · [候选撤回](../specs/m03/313-position-capacity-contract.md#m03-ac-080) |
| 调动改期 | 无冲突直接显示两岗结果。原例延期后，销售岗只在 2027-04-01 至 2027-04-03 每天缺 1 编，获权人可从提示进入原交接安排。取消草稿保留原调动，实际延长交接后两岗分别为 10 天、7 天。[AC111](../specs/m03/313-position-capacity-contract.md#m03-ac-111) |

## 2 基线与验证

固定工作树 wt-dev-d，HEAD 为 `362239f5fae11bcbd6d3249742d6f296f4fc7978`。在隔离目录依次应用[上一轮开工差异](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/313-simplification/before.patch.gz)和[本轮开工增量](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/313-simplification/before-procedure.patch.gz)，还原了全部 14 份开工时已有修改的文件。原 5 份未跟踪文件均已核对字节。313 正文 SHA-256 由 `fb63576f0ffee0f10d749663348764bb6cec8b68e0d143da8cb84c12929bd9a9` 变为 `854bdcd217a23e7ae34b49ab663e6c09e2ae7c00d621770e51b7a344e82e5b84`。

复算了两种办理顺序争最后一编、五减二后仍有三份、受保护份额及三天岗位缺额，并复跑[部门、份额与日期样本](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/313-staffing-establishment-review/recalculate.py)和[两岗调动算例](313-staff-transfer-review.md#3-独立日期与人数算例)。文字抽查曾因匹配“确认到岗”失配。改按正文“招聘与候选人确定一个到岗日”、同笔落编和改期回原录用三处核对，七项对照复验通过。58 个稳定锚点逐项一致，43 项现行编制验收沿用原编号。[架构修订链](../reference/local-revisions.json)第 6 次修订记录 22 份目标文件的前后摘要。

首次局部渲染写入了历史清单，已按开工摘要逐字节恢复。在隔离副本用 Mermaid CLI 12.0.0 和 Chrome 140.0.7339.186 重新渲染五图。逐图核对无影响分支、录用交接、两岗结果和案件结论，图面文字均在画布内。[本轮渲染清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/313-simplification/procedure-render.json)

修订前后均通过 `make docs`。修订后 128 项脚本测试、414 份 Markdown 的登记及链接检查通过，架构生成检查的 57 份文本一致，`git diff --check` 和新增报告空白检查通过。实际 Go 办理、数据库并发、浏览器操作及业务签收由相应产品批次验证。
