package domain

import (
	"slices"
	"testing"
)

func TestAllocationDatesFollowVisibleRosterAndQuota(t *testing.T) {
	const dep = "研发"
	const jan, feb, mar, apr, may, jun, jul, aug = "2027-01-01", "2027-02-01", "2027-03-01", "2027-04-01", "2027-05-01", "2027-06-01", "2027-07-01", "2027-08-01"
	zero, two := 0, 2
	p := AllocatedPosition{
		Position: QuotaPeriods{Records: []QuotaPeriod{
			{Date: jan, End: feb, DepartmentID: dep, Active: true},
			{Date: feb, End: may, DepartmentID: dep, Active: true, Name: "更名"},
			{Date: may, End: jun, DepartmentID: dep},
			{Date: jun, End: jul, DepartmentID: dep, Active: true},
			{Date: jul, DepartmentID: "财务", Active: true},
		}},
		Quota: StaffingQuota{Records: []QuotaRecord{
			{Date: mar, Amount: &zero}, {Date: "2027-03-15", Amount: &zero}, {Date: apr}, {Date: may, Amount: &two}, {Date: aug, Amount: &zero},
		}},
	}
	check := func(department string, amount bool, want ...string) {
		t.Helper()
		got := p.AllocationDates(department, amount)
		slices.Sort(got)
		if !slices.Equal(slices.Compact(got), want) {
			t.Fatalf("%s 分配日期 %v，预期 %v", department, got, want)
		}
	}
	check(dep, true, jan, mar, "2027-03-15", apr, may, jun, jul)
	check(dep, false, jan, may, jun, jul)
	check("财务", true, jul, aug)
	p.Position.Records = []QuotaPeriod{{Date: jan, End: feb, DepartmentID: dep, Active: true}, {Date: jun, DepartmentID: dep, Active: true}}
	check(dep, true, jan, feb, jun, aug)
}

func TestDepartmentCapacityBoundaries(t *testing.T) {
	const jan = "2027-01-01"
	const may = "2027-05-01"
	const jun = "2027-06-01"
	record := func(day string, n int) QuotaRecord { return QuotaRecord{Date: day, Amount: &n} }
	quota := func(records ...QuotaRecord) StaffingQuota { return StaffingQuota{Records: records} }
	assignment := func(day, dep, action string) QuotaPeriod {
		return QuotaPeriod{Date: day, Active: action != "DEACTIVATE", DepartmentID: dep}
	}
	member := func(dep string, records ...QuotaRecord) AllocatedPosition {
		return AllocatedPosition{Position: QuotaPeriods{Records: []QuotaPeriod{assignment(jan, dep, "CREATE")}}, Quota: quota(records...)}
	}
	cap := quota(record(jan, 10), record(may, 8))
	a, b := member("研发", record(jan, 6), record(may, 5)), member("研发", record(jan, 3), record(jun, 4))
	check := func(expectedDay, shortfall string, positions ...AllocatedPosition) {
		t.Helper()
		actual := DepartmentCapacity("研发", cap, positions)
		if expectedDay == "" {
			if actual != nil {
				t.Fatal(actual)
			}
			return
		}
		want := AllocationConflict{"研发", expectedDay, shortfall}
		if actual == nil || *actual != want {
			t.Fatalf("冲突 %v，预期 %v", actual, want)
		}
	}
	check(jun, "1", a, b, member("财务", record(jan, 99)))
	b.Position.Records[0].End = jun
	b.Position.Records = append(b.Position.Records, assignment(jun, "研发", "DEACTIVATE"))
	check("", "", a, b)
	b.Position.Records[1].End = "2027-07-01"
	b.Position.Records = append(b.Position.Records, assignment("2027-07-01", "研发", "REACTIVATE"))
	check("2027-07-01", "1", a, b)
	b.Position.Records[2] = assignment("2027-07-01", "财务", "REACTIVATE")
	check("", "", a, b)
	cap = quota(record(jan, 0), QuotaRecord{Date: jun})
	check(jan, "9", a, b)
	cap = StaffingQuota{}
	check("", "", a, b)
}
