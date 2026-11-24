package domain

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"errors"
	"strings"
	"testing"
)

func TestPositionHierarchyDates(t *testing.T) {
	const jan, feb, apr, may = "2027-01-01", "2027-02-01", "2027-04-01", "2027-05-01"
	positions := map[string]Timeline{}
	for _, id := range []string{"A", "B", "C"} {
		positions[id] = Timeline{ID: id, Records: []Record{{Date: jan, Type: "CREATE"}}}
	}
	versionOne := HierarchyEdges{{Child: "A", Parent: "B", Start: jan}}
	versionTwo := versionOne.Assign("A", "C", apr)
	if err := versionTwo.Validate(positions); err != nil {
		t.Fatal(err)
	}
	if versionOne.At(apr)["A"] != "B" || versionTwo.At("2027-03-31")["A"] != "B" || versionTwo.At(apr)["A"] != "C" {
		t.Fatal("未来换上级改写了旧版或错过生效边界")
	}
	future := versionTwo.Assign("B", "A", feb)
	requireConflict := func(edges HierarchyEdges, date string) {
		t.Helper()
		var fault *foundation.Fault
		if err := edges.Validate(positions); !errors.As(err, &fault) || fault.Status != 409 || !strings.Contains(fault.Message, date) {
			t.Fatalf("预期 %s 冲突，实际 %v", date, err)
		}
	}
	requireConflict(future, feb)
	requireConflict(versionOne.Assign("B", "C", jan).Assign("C", "A", may), may)
	requireConflict(append(versionOne.Assign("B", "", apr), HierarchyEdge{Child: "A", Parent: "C", Start: feb}), feb)
	requireConflict(versionOne.Assign("A", "A", feb), feb)
	if err := versionOne.Assign("A", "", apr).Assign("B", "A", apr).Validate(positions); err != nil {
		t.Fatal("首尾相接的相反方向关系应可发布", err)
	}
	requireConflict(versionOne.Assign("A", "missing", apr), apr)
	requireConflict(versionOne.Assign("missing", "B", feb), feb)
	positions["C"] = Timeline{ID: "C", Records: []Record{{Date: jan, End: may, Type: "CREATE"}, {Date: may, Type: "DEACTIVATE"}}}
	requireConflict(versionTwo, may)
	requireConflict(versionOne.Assign("C", "B", jan), may)
	detached := versionTwo.Assign("A", "", may)
	if err := detached.Validate(positions); err != nil {
		t.Fatal(err)
	}
	inserted := detached.Assign("A", "B", "2027-04-15")
	if inserted.At(may)["A"] != "" || detached.At("2027-04-15")["A"] != "C" {
		t.Fatal("插入覆盖了独立后继或原版本")
	}
}
