# CHNHCM-313 提前补招与限期交接方案复核

## 1 范围与基线

2026-11-21，按用户确认的业务方向修订[编制管理合同](../specs/m03/313-position-capacity-contract.md)。开工时固定工作树为 wt-dev-d，HEAD 为 `362239f5fae11bcbd6d3249742d6f296f4fc7978`，工作区、未跟踪文件和 stash 均为空。受测输入为该基线加本轮文档差异。Codex 主代理从用人部门、招聘、编制和人事的办理顺序复核，沿用[上一版独立评审](313-staffing-establishment-independent-review.md)作为旧稿记录。本轮结论限于方案及日期算例。

## 2 业务反例与修订

| 复核的情形 | 修订后的办理依据 |
| --- | --- |
| 原员工表达离职意向，岗位满编，用人部门当天就想启动寻访 | M05 可以发布和面试。当日实际占编仍为 1。承诺未来到岗日期需要有效预留，意向不能替代确定的任职结束记录。[提前招聘](../specs/m03/313-position-capacity-contract.md#m03-br-034) |
| 接替者提前到岗，月末恰有独立生效的正式增编 | 常规编制与指定补招的交接额度按来源分别计算，一笔落实交接段及后续常规段。核对截止日次日的常规数可检出“把岗位改回旧值”的错误。[交接示例](../specs/m03/313-position-capacity-contract.md#m03-br-035) |
| 先为两人安排十天交接，但第十一天的常规编制已经给了他人 | 单日看有交接额度仍不足以承诺长期录用。须在落账前覆盖全期。同一原任职的两份补招、两方争同期人数上限均进入岗位和需求共同守卫。[后续覆盖与授权](../specs/m03/313-position-capacity-contract.md#m03-ac-101) |
| 已发录用后候选人改期或撤回，另有人拟使用该份额 | 招聘据实处置录用和绑定。重算未用期间，缩减需求同时释放对应额度，已履行期间及旧录用可追溯。[改期与取消](../specs/m03/313-position-capacity-contract.md#m03-ac-102) |
| 原员工想多留三天，以及两人事实上已多工作三天 | 拟议延期先取得三天额度。已经发生的在岗事实由人事核实，与占编差额一起记录。原计划无缝接替、已有证据证明两人重叠时，也可据实登记。未来离职撤销仍受足额校验。[实况与处置](../specs/m03/313-position-capacity-contract.md#m03-br-036) |
| 部门有余量，但余量属于另一岗位 | 首批继续按岗位控编。企业采用岗位调剂时，转出、转入一次保存并核对两岗权限、全部使用与部门总数。组织共享总编保留独立选型。[控编层级](../specs/m03/313-position-capacity-contract.md#7-大陆企业控编粒度的建议) |

复核中细化了两处会影响办理结果的口径。交接期间的两名员工都计入实际已占，检查常规减编时只与常规来源的预留和占编比较。录用改期发生冲突的算例使用有明确结束日的原预留，区分“原已承诺”与“新期间没有余量”。[计量口径](../specs/m03/313-position-capacity-contract.md#m03-br-025) · [未来预留](../specs/m03/313-position-capacity-contract.md#m03-ac-099)

## 3 按自然日复算

下例使用财务主管的最后在岗日和接任者的到岗日作为输入。用两份在岗日期的交集求出真实交接天数，再与计划的名额分段比较。观察窗口到 2027-04-04，产品办理开放任期时仍须核对更远的未来节点。

```python
from datetime import date, timedelta


def dates(first, last):
    first, last = date.fromisoformat(first), date.fromisoformat(last)
    return {first + timedelta(days=i) for i in range((last - first).days + 1)}


march_end = dates("2027-03-21", "2027-03-31")  # 原员工实际在岗
incoming = dates("2027-03-22", "2027-04-04")  # 接任者实际在岗
handover = march_end & incoming
regular = incoming - handover
assert len(handover) == 10 and regular == dates("2027-04-01", "2027-04-04")
for day in march_end | incoming:
    actual = int(day in march_end) + int(day in incoming)
    assert actual == (2 if day in handover else 1)
    # 入职前，未来承诺分别落在专用和常规上。转换后同一日只用其中一种。
    assert int(day in march_end) + int(day in regular) == 1
    assert int(day in handover) + int(day in regular) == int(day in incoming)

later_arrival = dates("2027-03-25", "2027-04-04")
later_exit = dates("2027-03-21", "2027-04-03")
earlier_exit = dates("2027-03-21", "2027-03-28")
assert len(later_arrival & march_end) == len(incoming & earlier_exit) == 7
assert len(later_exit & incoming) == 13
uncovered = (later_exit & incoming) - handover
assert uncovered == dates("2027-04-01", "2027-04-03")
other_booking = dates("2027-04-01", "2027-04-04")
assert min(regular & other_booking) == date(2027, 4, 1)
late_hire = dates("2027-04-04", "2027-04-04")
vacant = dates("2027-03-21", "2027-04-04") - (march_end | late_hire)
assert vacant == dates("2027-04-01", "2027-04-03")
```

## 4 检查与后续验收

| 实际检查 | 结果 |
| --- | --- |
| 运行第 3 节 Python 日期算例 | 正常交接 10 天，新人晚来或原员工早离均为 7 天，延后离任为 13 天。未覆盖期间和推迟入职造成的空缺均为 3 天。后续常规名额另有预留时首个冲突日为 2027-04-01 |
| 运行上一版独立算例 | 部门及可见范围、规则采用日期、招聘份额与撤销边界复算通过。[原算例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/313-staffing-establishment-review/recalculate.py) |
| 核对登记与原件 | 基线原有 BR、AC 锚点均保留，新增 3 条规则和 9 项验收条件。既有 313 验证报告及附件共 9 份与 Git 基线逐字节一致。本次 6 份架构文件的前后摘要登记在[本地修订链](../reference/local-revisions.json) |
| 仓库检查 | 修订前 `make docs` 通过 128 项脚本测试、410 份 Markdown 和 12950 条本地链接检查。修订后通过同组测试及 411 份文档的登记、链接和来源检查。架构生成器 `--check` 核对 57 份输出无差异，`git diff --check` 和新增报告空白检查通过 |

本次主代理复核接受该文档修订，313 正文 SHA-256 为 `207f2b7fb6d3a2713be5a1afcce5fe06210759e0f8a90ff1e6458172c1dbdace`。招聘、人事和编制产品需按[新增验收](../specs/m03/313-position-capacity-contract.md#61-补招与交接的业务验收)分别通过真实数据库、正常菜单和跨模块联验。本轮未办理真实业务或更新持久验收环境。
