# CHNHCM-506 同日人事办理方案修订复核

## 1 受审范围

2026-11-24，Codex 以 wt-dev-f 的 `87c4b07dd26acb9b4f21a0bb441a5a7cc3f1a519` 为基线，复核工作区中 401、403、500、506 和 507 的修订。106 第 8 节已采纳按职务首次入职的日级模型。基线的 00037 首单动作类型为 `Hire` 和 `CancelHire`。本次依据 106 的反例补充 T04 的业务、页面及写入合同。[106 反例](../adr/106-same-day-personnel-actions-and-daily-state.md#43-后续必须证明的反例) · [106 首单决定](../adr/106-same-day-personnel-actions-and-daily-state.md#first-hire-adoption) · [首单存储](../../db/migrations/00037_employee_hire.sql) · [上一轮办理列表复核](507-personnel-action-register-design-review.md) · [登记表](../governance/registry.json)

## 2 五项评审意见的处置

| 评审问题 | 可从业务入口核对的结果 |
| --- | --- |
| 晋升的职级从何而来、怎样保存 | 从原任期办理，M04 保留该员工原职级、拟任职级及部门，按 M03 在相应期间的职务与岗位资格重核。样本在研发部从 P5 晋至 P6，随后调入产品部，两个部门都可在相应位置核清。[实际职级与任职资格](../specs/m04/506-job-information-records.md#m04-br-013) · [页面字段](../specs/m04/507-job-information-tab.md#same-day-job-form) |
| 任一单项保存都会产生非法任职时怎样办理 | 从原任职直接进入「晋升并调动」。原岗只准 P5、新岗只准 P6 时，两次单办都应被拒，专用表单一次保存新岗 P6、两笔原件和同一回执。另用一项晋升决定附带换岗，核对只计一次晋升。[两组验收](../specs/m04/506-job-information-records.md#m04-ac-019) · [提交与恢复](../specs/m04/403-hire-technical-contract.md#same-day-action-write) |
| 误记晋升撤销后，调动及旧来源还能否查清 | 目标岗位接受 P5 时，撤销后当前为产品部、已转正、P5，原调动仍有效。含撤销原笔共三行，有效事项两笔。撤销前来源仍为 P6、晋升一次。若调入岗位只准 P6，单撤整笔拒绝。[五步来源对照](../specs/m04/506-job-information-records.md#m04-scn-018) · [原笔撤销](../specs/m04/401-direct-hire-requirements.md#m04-ac-006) |
| 迟到的通知如何补在当天已登记调动之前 | 从「当日办理」选中调动补晋升，再在晋升前补转正。任职详情按确切业务先后显示三笔，事务列表按首次登记时刻倒序。对独立事项交换录入顺序，两笔各自可读。[双顺序验收](../specs/m04/507-job-information-tab.md#m04-ac-018) |
| 从不同任职发起转正时，发生部门和笔数如何认定 | 通知载明实际发生部门时优先采用。未载时，按同一关系转正前的人事管理主任职取部门。从培训兼岗或研发主岗进入，都能查到同一关系的一笔转正。无法唯一认定时，区分允许记未确定与必须补足依据两种情况。[部门规则](../specs/m04/506-job-information-records.md#m04-br-014) · [归属、计数和受限样本](../specs/m04/506-job-information-records.md#m04-scn-018) |

## 3 验证与交接

将五次办理的日状态、转正／晋升／调动次数与独立计算逐行比对。穷举三笔的六种登记顺序，按相同的业务前序均得产品部、已转正、P6。另核两笔互无先后的事项、四种岗位与职级组合、撤销前后两份来源，以及通知明确、按本关系主任职推定和归属未确定的情况。以上均为合成资料的设计预期。

`make docs` 的 180 项回归和仓库文档检查通过，`make boundaries` 与 `git diff --check` 通过。补查本次新增及改写文句中的仓内链接、标题锚点和新文稿空白，均可定位并符合格式。

T04 实施时从正常菜单办理，并用 Go、真实 PostgreSQL 和 API 验证职级资格、事务回滚、不同经办人并发及同号恢复。页面验收另核受限原件、切换旧来源和返回焦点。按职务先验证可用职级集合，按岗位另行验证岗位资格和占编。[T04 交付与签收](../archive/plans/500-personnel-roadmap.md#t04-same-day-actions)
