package department

import (
	"context"
	"testing"
)

func TestHistoricalNameDoesNotRelaxNormalAdmission(t *testing.T) {
	original := " 组织?甲 "
	value := Data{Name: original, BU: "00000000-0000-4000-8000-000000000001"}
	if _, e := Kernel.Policy.Normalize(value); e == nil {
		t.Fatal("normal admission accepted imported spelling")
	}
	k := HistoricalNameKernel()
	created, e := k.PrepareCreate(context.Background(), "00000000-0000-4000-8000-000000000002", "1984-01-01", "1984-01-01", "NEW_ORGANIZATION", value)
	if e != nil || created.Records[0].Data.Name != original {
		t.Fatal("source spelling changed", e)
	}
	parent := "00000000-0000-4000-8000-000000000003"
	patch := Patch{"parentDepartmentId": &parent}
	got, e := (Policy{}).Apply(value, &patch)
	if e != nil || got.Name != original || got.Parent != parent {
		t.Fatal("unrelated maintenance revalidated inherited name", e)
	}
	patch = Patch{"name": &original}
	if _, e = (Policy{}).Apply(value, &patch); e == nil {
		t.Fatal("explicit name resubmission bypassed admission")
	}
	bad := "组织\n甲"
	value.Name = bad
	if _, e = k.Policy.Normalize(value); e == nil {
		t.Fatal("historical name accepted a control character")
	}
}
