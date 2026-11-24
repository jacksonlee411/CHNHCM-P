package dated

import (
	"reflect"
	"testing"
)

func TestOperationDirectoryIsObjectScopedAndDetached(t *testing.T) {
	want := map[string]string{"NAME_ADJUSTMENT": "名称调整", "OTHER_DATA_ADJUSTMENT": "其他数据调整"}
	if !reflect.DeepEqual(OperationReasons(BusinessUnit, "CHANGE"), want) {
		t.Fatal("BU directory")
	}
	got := OperationReasons(Department, "CHANGE")
	if len(got) != 3 || got["ORGANIZATION_REASSIGNMENT"] != "变更归属组织" {
		t.Fatal(got)
	}
	delete(got, "NAME_ADJUSTMENT")
	if OperationReasons(Department, "CHANGE")["NAME_ADJUSTMENT"] == "" {
		t.Fatal("mutable catalog leaked")
	}
	if !reflect.DeepEqual(OperationReasons(BusinessUnit, "CREATE"), map[string]string{"NEW_ORGANIZATION": "新设组织"}) {
		t.Fatal("old create reasons exposed")
	}
	if len(OperationReasons("unknown", "CREATE")) != 0 || len(OperationReasons(BusinessUnit, "unknown")) != 0 {
		t.Fatal("unknown catalog did not fail closed")
	}
	if OperationLabel("CHANGE") != "数据更改" {
		t.Fatal("old type label")
	}
}
