# CHNHCM-313 人员调动与两岗编制方案复核

## 1 范围与基线

2026-11-21，在固定工作树 wt-dev-d 复核调出补缺、内部接任、两岗并行交接及满编对调。开工 HEAD 为 `362239f5fae11bcbd6d3249742d6f296f4fc7978`。工作区已有上一轮补招与交接稿的 12 个已跟踪文件修改及 1 份未跟踪的复核报告，stash 为空。本次由 Codex 主代理做方案复核。[修订合同](../specs/m03/313-position-capacity-contract.md#m03-br-037) · [前一轮补招与交接复核](313-backfill-and-handover-review.md)

开工差异以 gzip 保存，SHA-256 为 `90737da6eb49ad879dc70f35ba5da78d6850e3750567723396d6bc7699e29df6`。在独立临时目录用上述 HEAD 的文件应用解压后的补丁，13 份输入的逐份摘要均与开工时相同。前一轮 313 正文 SHA-256 为 `207f2b7fb6d3a2713be5a1afcce5fe06210759e0f8a90ff1e6458172c1dbdace`。[开工差异原件](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/313-transfer/before-transfer.patch.gz) · [架构修订链](../reference/local-revisions.json)

## 2 影响办理结果的情形

| 人事和用人部门遇到的情况 | 合同处理与验收依据 |
| --- | --- |
| 部门得知员工要调走，原岗今天满编 | 可立即提出补缺并启动招聘。人事保存未来调动后，拟到岗期间的原岗空缺可预留。同一期间的调入岗已被新任职占用。调动、招聘争最后一个名额时，先成功的安排受保护。[未来调动与补缺](../specs/m03/313-position-capacity-contract.md#m03-ac-107) |
| 同一员工晋级、换岗、借调或正式兼岗 | 岗位及占编性质均未变时沿用原占编。换岗按两岗规则，正式兼岗按两份任职计算。借调按实际保留的任职处理，承诺返回已释放的原岗须核实返岗期间。整岗改隶按日迁移部门小计，人员调动沿用两岗各自核定数。[调动分类](../specs/m03/313-position-capacity-contract.md#m03-br-037) · [借调与返回](../specs/m03/313-position-capacity-contract.md#m03-ac-113) |
| 新岗直接由内部员工接任，或该员工已取得内部竞聘的录用 | 前者以正式调动和接任任职办理交接，可查到全期招聘预留为零。后者核实原招聘份额、录用和绑定后，在原工作关系内转换一次。两条路径均须为交接结束后的任期落实常规名额。[内部接任](../specs/m03/313-position-capacity-contract.md#m03-ac-109) |
| 调出和调入部门都需要交接 | 调出岗为原岗接替者、调入岗为内部调入者，分别落实有限期间的名额。正式兼任期间，两岗可同时交接。一人横跨两岗在企业人数中计一次。[两岗办理示例](../specs/m03/313-position-capacity-contract.md#m03-br-038) |
| 经办人要推迟或撤销已办调动，接替者已获录用 | 列明每岗受影响的日期、人数和原承诺。延期可在授权内延长交接，撤销未来调动须能恢复原岗并处置后继。实况有误按有依据的实际到、离岗日更正，保留原件。[延期、撤销与实况](../specs/m03/313-position-capacity-contract.md#m03-ac-111) |
| 两个满编岗位同日互换，但两人的占编性质可能不同 | 按交换后的结果逐岗逐段核实。其他人的任职和招聘预留继续计入。只要一岗有缺口、被冻结或经办人缺少一岗权限，整组保留原安排。保存、查回、改期和撤销均可追溯两人的动作。[满编对调](../specs/m03/313-position-capacity-contract.md#m03-ac-112) · [跨岗抵扣反例](../specs/m03/313-position-capacity-contract.md#m03-ac-114) |

## 3 独立日期与人数算例

销售、运营各核 1 编。小陈于 2027-03-25 到销售，小周在运营工作至 2027-04-10。以下按各人的在岗日期计算两岗人数、额外交接日及两岗范围内去重人数。日期均含首尾。样本只使用合成的任职安排。

```python
from datetime import date, timedelta


def days(first, last):
    return {first + timedelta(days=n) for n in range((last - first).days + 1)}


sample_days = days(date(2027, 3, 24), date(2027, 4, 11))
april_1 = date(2027, 4, 1)
april_4 = date(2027, 4, 4)
three_days = days(april_1, date(2027, 4, 3))


def roster(sales_last, operations_first):
    by_day = {}
    for day in sample_days:
        sales, operations = set(), set()
        if day <= sales_last:
            sales.add("小林")
        if day >= date(2027, 3, 25):
            sales.add("小陈")
        if day <= date(2027, 4, 10):
            operations.add("小周")
        if day >= operations_first:
            operations.add("小林")
        by_day[day] = sales, operations
    return by_day


def overlap(by_day):
    sales_days = {day for day, (sales, _) in by_day.items() if len(sales) > 1}
    operations_days = {day for day, (_, ops) in by_day.items() if len(ops) > 1}
    return sales_days, operations_days


normal = roster(date(2027, 3, 31), april_1)
for day, expected in [
    (date(2027, 3, 24), (1, 1, 2)),
    (date(2027, 3, 25), (2, 1, 3)),
    (april_1, (1, 2, 3)),
    (date(2027, 4, 11), (1, 1, 2)),
]:
    sales, ops = normal[day]
    assert (len(sales), len(ops), len(sales | ops)) == expected
sales_before, ops_before = overlap(normal)
assert (len(sales_before), len(ops_before)) == (7, 10)

sales_later, ops_later = overlap(roster(date(2027, 4, 3), april_4))
assert (len(sales_later), len(ops_later)) == (10, 7)
assert sales_later - sales_before == ops_before - ops_later == three_days

dual = roster(date(2027, 4, 3), april_1)
dual_sales, dual_ops = overlap(dual)
assert dual_sales & dual_ops == three_days
for day in three_days:
    sales, ops = dual[day]
    assert (len(sales), len(ops), len(sales | ops)) == (2, 2, 3)


def loads(posts, units):
    return tuple(sum(units[p] for p in people) for people in posts)


ordinary = {"甲": 1, "乙": 1}
assert loads(({"甲"}, {"乙"}), ordinary) == (1, 1)
assert loads((set(), {"甲", "乙"}), ordinary) == (0, 2)
assert loads(({"乙"}, {"甲"}), ordinary) == (1, 1)
with_third = {"甲": 0, "乙": 1, "丁": 1}
assert loads(({"甲", "丁"}, {"乙"}), with_third) == (1, 1)
assert loads(({"乙", "丁"}, {"甲"}), with_third) == (2, 0)
future_days = days(date(2027, 5, 1), date(2027, 5, 31))
proposed = {}
for day in future_days:
    units = {"丁": 1, "甲": 0, "乙": int(day > date(2027, 5, 14))}
    proposed[day] = loads(({"丁", "乙"}, {"甲"}), units)
assert proposed[date(2027, 5, 1)] == (1, 0)
first_conflict = min(day for day, occupied in proposed.items() if max(occupied) > 1)
assert first_conflict == date(2027, 5, 15)
```

## 4 验证与交付范围

| 实际检查 | 结果 |
| --- | --- |
| 第 3 节逐日复算 | 原安排销售、运营分别交接 7 天和 10 天，延期后三天的缺口全部落在销售，结果为 10 天和 7 天。改为如期调入并正式兼任，三天内两岗各有 2 份占编任职，两岗范围内为 3 人 |
| 对调反例 | 普通对调后两岗各占 1。存在第三人并有免占任职时，拟对调后为 2 和 0。把调入任职的占编起日放在未来时，逐日复算定位到 2027-05-15 的首个缺口 |
| 上轮算例与旧稿 | 重新运行补招日期、部门汇总、规则采用、预留及撤销算例，全部通过。前一轮未提交的复核报告与开工时逐字节一致，另有 9 份历史报告及附件与 HEAD 相同 |
| 登记、来源及差异 | 原有规则及验收锚点全部保留，新登记 3 条规则、8 项验收。5 份架构文件的前后摘要与连续修订记录一致。生成器校验 57 份输出无差异。`git diff --check` 及新报告的空白检查通过 |
| 仓库文档 | 开工时 `make docs` 通过 128 项脚本测试、411 份文档和 12979 条本地链接检查。修订后通过 128 项脚本测试及 412 份文档的登记、链接与来源检查 |

复核后的 313 正文 SHA-256 为 `4e227b0bcb14c08e1c2bb49bffe27f6cf59de8d954d9e023e22af4221bbac504`。本次结论限于上述方案和算例。

核编首批可独立验收。人事及编制接入时，先以已核齐用量的岗位验证纯内部接任、正式兼任、借调返回及同日对调。有真实招聘份额、已发录用或两条路径争用最后一个名额时，再从招聘与人事正常入口验证先后顺序、中途失败及重启查回。产品实施、真实数据库并发、浏览器和人工业务签收均待相应批次执行。[八项产品验收条件](../specs/m03/313-position-capacity-contract.md#62-人员调动的业务验收)
