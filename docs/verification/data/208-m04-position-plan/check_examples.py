"""复算岗位用量与原岗承诺的虚构日期样本。"""

from collections import Counter
from datetime import date, timedelta


def require_equal(label, actual, expected):
    if actual != expected:
        raise ValueError(f"{label}: {actual!r} != {expected!r}")
    print(f"{label}: {actual}")


def check_position_days():
    # 三个独立来源。区间的右端是第一天不再使用该岗位的日期。
    hire = [("小林", "A", "2027-04-01", None)]
    cancelled = []
    transfer = [
        ("小林", "A", "2027-04-01", "2027-05-01"),
        ("小林", "B", "2027-05-01", None),
    ]
    cases = [
        ("入职前", hire, "2027-03-31", (0, 0, 1, 0)),
        ("入职日", hire, "2027-04-01", (1, 1, 0, 0)),
        ("撤销后查入职日", cancelled, "2027-04-01", (0, 0, 1, 0)),
        ("调动前", transfer, "2027-04-30", (1, 1, 0, 0)),
        ("调动日", transfer, "2027-05-01", (1, 0, 1, 1)),
    ]
    for label, source, day, expected in cases:
        people = set()
        occupied = Counter()
        for person, position, start, until in source:
            if start <= day and (until is None or day < until):
                people.add(person)
                occupied[position] += 1
        actual = len(people), occupied["A"], 1 - occupied["A"], occupied["B"]
        require_equal(label, actual, expected)

    first = date(2027, 5, 1)
    delayed = date(2027, 5, 5)
    replacement = date(2027, 5, 3)
    overlap = []
    for offset in range((delayed - first).days):
        day = first + timedelta(days=offset)
        if day >= replacement:
            overlap.append(day.isoformat())
    require_equal("原岗承诺重叠日", overlap, ["2027-05-03", "2027-05-04"])


if __name__ == "__main__":
    check_position_days()
