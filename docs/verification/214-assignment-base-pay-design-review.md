# CHNHCM-214 员工首次定薪与按日调薪方案评审

## 1 审阅对象与基线

2026-11-28，Codex 按用户委托评审 [214 实施计划](../plans/214-assignment-base-pay-rollout.md)、[215 业务与页面](../specs/m09/215-assignment-base-pay.md)、[216 技术合同](../specs/m09/216-assignment-base-pay-contract.md)。受评基线为 `wt-dev-c` 的 `f1b9a79f8c1d1c9c56fa677a361aeb5549eb569b`，基线外为本次方案及登记。制度输入取[4380 重启后结果](data/210-default-port/after-restart.json)，其 SHA-256 为 `04730b3c45dafe3cb919dbde4ee27529cbc6233019011c6293745a46f3e67db5`。范围为方案、来源及验收预期的书面评审。[登记表](../governance/registry.json)

## 2 首轮发现与修订复查

| 触发条件 | 初次核对结果 | 修订与复查结果 |
| --- | --- | --- |
| 用十一月 12,000、一月 13,000 验证是否正确选择薪带版本 | 这两个数均可通过旧区间检验。即使一月始终读错薪带，光查金额也分辨不出来。 | 增设 15,500 跨界与 16,250 检测误用更正前上限，核定原件明确定位薪标修订。十二月查询另核原采用 15,000 及当日 16,000。[跨界预期](../specs/m09/215-assignment-base-pay.md#m09-ac-010) |
| 一月的 13,000 已提前录入，现须补录十一月的独立决定 | 初稿只写两笔之间的插入，首笔之前的办理入口及撤去后的十二月金额无法确定。 | 允许在同一实际任期内补录更早决定。补录后十二月为 12,000，撤去更早误建笔后为尚未定薪。一月原身份、起日及回执始终有效。[办理与反例](../specs/m09/215-assignment-base-pay.md#m09-br-011) |
| 同日收到两份金额不同的决定，或原笔的任职依据后来被撤回 | 初稿给了同日误录更正入口，未区分两份真实决定的冲突，撤去已失资格原笔的处置条件也不清楚。 | 同日金额冲突须查明业务决定，拒绝按提交先后覆盖。已证实取消或误建的原笔按处置权撤去，可处理已失效来源，仍保留原件与回执。[处置合同](../specs/m09/216-assignment-base-pay-contract.md#5-权限恢复与实施验证) |
| 开表时只匹配 PRO-M，提交时并发新增第二份合格计划 | 初稿要求提交时重新判断唯一性。另一计划生效，即使屏幕上已显示的 PRO-M 仍合法，也会打断保存。 | 按 132 的实际依赖校验和 213 的一次确认，保存屏幕显示且仍合法的具体计划。打开表单即有多份时需先选。复核包括新出现第二份、原计划失效、跨页候选和仅有津贴的计划。[候选与事务](../specs/m09/216-assignment-base-pay-contract.md#pay-commit) |

## 3 独立复算

在仓库根目录以 Python 标准库运行下列内容。第一组从首期结果读取计划金额。第二组只用人工给定的决定及日期推演输入顺序和撤去后的预期，不调用未来的产品判断函数。

```python
import json
from decimal import Decimal
from pathlib import Path

snapshot = json.loads(Path("docs/verification/data/210-default-port/after-restart.json").read_text())
bands = {day: tuple(Decimal(value.replace(",", "")) for value in amounts)
         for code, day, *amounts in snapshot["pages"] if code == "PRO-M"}
nov, dec, jan = "2026-11-30", "2026-12-01", "2027-01-01"
assert bands[nov] == (10000, 15000, 300)
assert bands[dec] == bands["2026-12-31"] == (11000, 16000, 300)
assert bands[jan] == (11000, 16000, 350)

def within(day, value):
    low, high, _ = bands[day]
    return low <= Decimal(value) <= high

assert all(within(d, v) for d in (nov, dec, jan) for v in ("12000", "13000"))
assert [within(d, "15500") for d in (nov, dec)] == [False, True]
assert [within(d, "10500") for d in (nov, dec)] == [True, False]
assert all(within(dec, v) for v in ("11000", "16000"))
assert not within(dec, "16250") and Decimal("16250") <= Decimal("16500")

dates = ["2026-11-29", nov, dec, "2026-12-14", "2026-12-15", "2026-12-31", jan]
def amounts_on(decisions):
    result = []
    for day in dates:
        covered = [start for start in decisions if start <= day]
        result.append(decisions[max(covered)] if covered else None)
    return result

later = {jan: 13000}
original = {**later, nov: 12000}
inserted = {**original, "2026-12-15": 12800}
assert amounts_on(later) == [None] * 6 + [13000]
assert amounts_on(original) == [None, 12000, 12000, 12000, 12000, 12000, 13000]
assert amounts_on(inserted) == [None, 12000, 12000, 12000, 12800, 12800, 13000]
assert amounts_on({d: v for d, v in inserted.items() if d != "2026-12-15"}) == amounts_on(original)
assert amounts_on({d: v for d, v in original.items() if d != nov}) == amounts_on(later)
print("PRO-M 薪带、津贴和七个业务日的个人金额预期均相符")
```

## 4 跨域、页面与权限复审

| 阅读路径或角色 | 核对依据和设计结论 |
| --- | --- |
| 经办人选专业主岗、运营兼岗或再聘的 0 号 | 对照 [M04 实际职级与任期](../specs/m04/506-job-information-records.md#m04-br-013) 及[现有范围匹配](../../internal/bootstrap/compensation_plans.go)。当前接口只校验调用方提供的维度。实施计划将从真实任职取得事实列为第一阶段。仅有目录职级、仅有津贴及受限任职均有确定结果。 |
| 经办人查十二月，再追溯十一月原笔 | 对照 [M09 核定归属](../architecture/target/modules/m09.md) 和[计划采用](../specs/m09/211-compensation-configuration.md#m09-br-006)。当日工资、当日制度、核定时采用均有各自取值路径，金额更正按原生效日重核。独立后继或一月津贴变更不会改写十一月原件。 |
| 仅有制度权、仅有兼岗范围、个人金额读权被撤、仅来源读权被撤 | 按 [X03](../architecture/target/modules/x03.md)、[历史读取](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-history-access) 将目录、任职、个人金额、制度和原件分别核对。读写、旧地址、差额、原回执和未保存草稿覆盖到 [215 权限反例](../specs/m09/215-assignment-base-pay.md#m09-ac-013)。 |
| 首页进入、选日、输入、取消、保存后查原件并返回 | 对照[任职页](../specs/m04/507-job-information-tab.md#job-tab-layout)、[薪标详情与日志](../specs/m09/212-compensation-configuration-pages.md#reference-pages)、[共享表单和返回](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-expression)。普通人员名录仍使用原查询，所选任职内才提供工资查询日。真实页面的桌面、短屏、键盘和原生缩放验收交给实施阶段。 |
| 确认草稿、同日竞争、重送及数据库失败 | [216 事务合同](../specs/m09/216-assignment-base-pay-contract.md#pay-commit) 已落实到同一任期首次争用、来源在途变化、一次返回及原号查回的独立预期。M09 核定和未来 M10 的工资采用分属各自所有者。 |

## 5 结论与接续

**方案评审通过，可按 214 启动产品实施。** 六组验收场景覆盖个人首次核定与真实调薪、薪带版本边界、任职及候选、误录处置、授权和恢复。按第三节复算，15,500 在十一月被旧带排除，十二月落入新带。七个查询日的金额与插入、撤去预期一致。个人核定的 Go、数据库、浏览器、持久环境升级与业务人员签收由 [214 三个实施阶段](../plans/214-assignment-base-pay-rollout.md#2-实施顺序与出口) 接续。本次工作读取首期留存证据，未创建运行资源。

送审定稿按下列 SHA-256 定位。在工作树运行 `make docs`，195 项检查用例及仓库文档检查通过。`git diff --check` 及四份新文档的空白、末尾换行核对通过。另逐条核对四份文档的本地链接与标题锚点。

| 受评文档 | SHA-256 |
| --- | --- |
| 214 | `4696426dfaa70898ddeeee9bae59526a9db02cce1c7aa04937bc4ce5c98cde4a` |
| 215 | `8e2352a2d990fb8cf55b3eb047a4aaa36164f33f4558c9f741a4bfe7ade42689` |
| 216 | `18225b43b5a18af6b44aa1c8a5d6b4eeb7175b4db5e91565fcc8a54c83296eaf` |
