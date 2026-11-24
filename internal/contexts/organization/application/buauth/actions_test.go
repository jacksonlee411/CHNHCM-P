package buauth

import "testing"

func TestRedateDedicatedCategoryMatrix(t *testing.T) {
	kinds := []string{"CREATE", "CHANGE", "DEACTIVATE", "REACTIVATE"}
	grants := []string{"bu.first.redate", "bu.record.redate", "bu.deactivation.redate", "bu.reactivation.redate"}
	// Explicit policy expectation: no category's grant substitutes another's.
	want := [4][4]bool{{true, false, false, false}, {false, true, false, false}, {false, false, true, false}, {false, false, false, true}}
	for i, kind := range kinds {
		for j, grant := range grants {
			t.Run(kind+"/"+grant, func(t *testing.T) {
				if (RedateAction(kind) == grant) != want[i][j] {
					t.Fatal("wrong category permission", kind, grant)
				}
			})
		}
	}
}
func TestCategoryActionsFailClosed(t *testing.T) {
	for _, category := range []string{"", "INVALID"} {
		if DeleteAction(category) != "" || RedateAction(category) != "" || ReclassifyAction(category) != "" {
			t.Fatal(category)
		}
	}
	if DeleteAction("CREATE") != "" {
		t.Fatal("first must not be deleted")
	}
	for category, want := range map[string]string{"CHANGE": "bu.record.delete", "DEACTIVATE": "bu.deactivation.delete", "REACTIVATE": "bu.reactivation.delete"} {
		if DeleteAction(category) != want {
			t.Fatal(category)
		}
	}
	for category, want := range map[string]string{"CREATE": "bu.first.reclassify", "CHANGE": "bu.record.reclassify", "DEACTIVATE": "bu.deactivation.reclassify", "REACTIVATE": "bu.reactivation.reclassify"} {
		if ReclassifyAction(category) != want {
			t.Fatal(category)
		}
	}
}
