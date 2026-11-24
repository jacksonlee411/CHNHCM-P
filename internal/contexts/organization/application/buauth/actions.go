// Package buauth maps accurate BU categories to X03 action codes. It does not grant access.
package buauth

// DeleteAction is selected from the accurate target category, never from a
// client's classification or the current business-date state.
func DeleteAction(operationType string) string {
	return map[string]string{"CHANGE": "bu.record.delete", "DEACTIVATE": "bu.deactivation.delete", "REACTIVATE": "bu.reactivation.delete"}[operationType]
}

func RedateAction(operationType string) string {
	return map[string]string{"CREATE": "bu.first.redate", "CHANGE": "bu.record.redate", "DEACTIVATE": "bu.deactivation.redate", "REACTIVATE": "bu.reactivation.redate"}[operationType]
}

func ReclassifyAction(t string) string {
	return map[string]string{"CREATE": "bu.first.reclassify", "CHANGE": "bu.record.reclassify", "DEACTIVATE": "bu.deactivation.reclassify", "REACTIVATE": "bu.reactivation.reclassify"}[t]
}
