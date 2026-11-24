# CHNHCM-315 部门与岗位编制后台方案复核

## 1 结论与文档范围

2026-11-22 按编制管理员和只读 HR 两个角色复核。首批在同一后台按生效日给部门或岗位定额，允许一次调剂多岗。减编依据是调整后各段的岗位分配。部门待分配数只能用于分配额度。招聘预留、员工实际占编及相应可用数留待真实消费者接入。[313 额度口径](../specs/m03/313-position-capacity-contract.md#m03-br-025) · [部门分配与减编](../specs/m03/313-position-capacity-contract.md#m03-br-040)

| 文档 | 本次处理及后续用途 |
| --- | --- |
| 业务合同与开发计划 | 先修订 313 的额度、减编、入口及分批验收，再编写 315。第一步接通部门和岗位独立定额，第二步完成部门内调剂、反向校验与统一签收。[313](../specs/m03/313-position-capacity-contract.md) · [315](../plans/315-staffing-quota-administration.md) |
| 部门与岗位来源 | 118 补部门身份、有效期及反向检查，312 补岗位启停、改日和改隶时的分配校验。311 的岗位定义签收提供已有能力的依据。[118](../specs/m02/118-department-business-requirements.md) · [312](../specs/m03/312-position-contract.md) · [311 岗位签收](311-position-definition-signoff.md) |
| 职类独立验收 | 308 的 AC090 和 314 继续用旧职务岗位证明编制可独立交付。部门额度与调剂由 315 签收。[308 分批验收](../specs/m03/308-job-foundation-contract.md#m03-ac-090) · [314 职类路线图](../plans/314-m03-job-function-roadmap.md) |
| 架构与导航 | 同步 StaffingQuota 的部门和岗位身份、事务、术语及概念图，并从总入口、规格索引和验证索引关联本期文件。[M03 模块](../architecture/target/modules/m03.md) · [事务合同](../architecture/target/10-ddd-and-state-contracts.md#staffing-quota-transaction) · [术语](../architecture/target/17-terminology-glossary.md#term-m03-positioncapacity) · [总入口](../index.md) · [规格索引](../specs/README.md) · [验证索引](README.md) |
| 实施时沿用的共用合同 | 012 管理后台交互，120、121 管业务日期、独立修订和接入，X03 管动作、对象和字段授权。实际开发沿用这些合同。[012](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-expression) · [120](../specs/m02/120-dated-maintenance-technical-contract.md) · [121](../specs/m02/121-dated-object-admission-matrix.md) · [X03](../architecture/target/modules/x03.md) |

## 2 HR 走查与独立预期

| 办理场景 | 按原始额度复算的结果 |
| --- | --- |
| HR 只核部门或只核岗位 | 客服部无岗位，核 20 即有 20 待分配。另一部门只给两岗核 3、1，岗位核定合计 4，部门总额未核、待分配留空。首次给该部门核 3 缺 1，核 5 余 1。空部门核 0 显示 0。 |
| HR 给研发部调剂 | 总额 10、开发 6、测试 3，余 1。只把测试改到 5 缺 1。同笔改两岗为 5、5，或改总额为 11、两岗为 6、5，均分配完。只需具备实际改动对象的写权。 |
| HR 预约五月减编 | 初值仍为 10、6、3。2027-05-01 只降总额到 8 缺 1，同日将开发调到 5 则四月底余 1、五月余 0。另一组样本中测试自 2027-06-01 独立增到 4，六月缺 1，预期整笔拒绝，保留原安排。 |
| 岗位管理员停用、恢复或改隶，只读 HR 查指定日 | 部门总额 6，A 岗 2、B 岗 3、C 岗未核。B 停用前、停用期间、恢复后的岗位合计为 5、2、5，待分配为 1、4、1。只可见 A 时小计三个时点均为 2。B 拟转入只余 2 的部门缺 1。 |
| 两位 HR 分配最后一个名额 | 从 10、6、3 开始，一人给开发加 1，另一人给测试加 1。交换办理顺序，先办理的可分到 1，后一笔缺 1。两次独立复算结束时合计均为 10，余 0。数据库双连接竞争、丢响应后的查回以及重启恢复列入 315 验收。 |

用 Python 逐项核对算例。新旧日期分开取值，独立后继延续自己的额度。复算材料只含合成部门和岗位。[复算程序](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/315-staffing-quota-administration/recalculate.py) · [313 后台产品验收](../specs/m03/313-position-capacity-contract.md#m03-ac-130)

## 3 基线、验证与交接

工作树为 wt-dev-d，HEAD 为 `6bc37020e436274cbb7244a55db85efdb8bc74f1`。开工时工作区、暂存区及 stash 均为空。修订前 `make docs` 通过 128 项检查用例，覆盖 415 份 Markdown、13,051 条本地链接。原 313 的 59 个稳定锚点全部可定位，新设 BR040 和 AC116–119。此前九份 313 复核记录保留原时点证据。[原 HR 推演](313-hr-staffing-review.md)

架构生成检查核对 57 份文本。用 Mermaid CLI 12.0.0 和 Chrome 140.0.7339.186 重新渲染 C01 与 M03，两图分别有 61、35 个非空可见文字元素，浏览器坐标检查均在图框内。源文件和 SVG 的摘要见渲染清单，10 份架构文件的变更见第八次本地修订。[渲染清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/315-staffing-quota-administration/render.json) · [本地修订链](../reference/local-revisions.json)

修订后 `make docs` 的 128 项检查用例全部通过，覆盖 417 份 Markdown、13,109 条本地链接。另核对本次增改行内的 50 处章节链接。已登记的 247 份既有验证记录逐项对照一致。差异与新增文件的空白检查通过。

315 的产品执行状态为待开始。真实数据库升级、事务竞争、API、后台菜单和浏览器人工走查在实施批执行。[315 验证与交接](../plans/315-staffing-quota-administration.md#5-验证与交接)
