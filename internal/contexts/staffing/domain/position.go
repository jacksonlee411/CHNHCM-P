package domain

import "slices"

type PositionAdoption struct {
	ReferenceAdoption
	Name string `json:"name"`
}

func PositionFields() []string { return []string{"departmentId", "jobId", "locationId", "gradeId"} }
func PositionLabel(field string) string {
	return map[string]string{"departmentId": "所属部门", "jobId": "职务", "locationId": "工作地点", "gradeId": "职级"}[field]
}

type PositionUseBoundary struct {
	Field, ID, Date string
	WithGrade       bool
}

func ChangedPositionUses(before, after Timeline) []PositionUseBoundary {
	dates := make([]string, 0, len(before.Records)+len(after.Records))
	for _, timeline := range []Timeline{before, after} {
		for _, record := range timeline.Records {
			dates = append(dates, record.Date)
		}
	}
	slices.Sort(dates)
	var previous, next Record
	var beforeIndex, afterIndex int
	var changes []PositionUseBoundary
	for _, date := range slices.Compact(dates) {
		if beforeIndex < len(before.Records) && before.Records[beforeIndex].Date == date {
			previous = before.Records[beforeIndex]
			beforeIndex++
		}
		if afterIndex < len(after.Records) && after.Records[afterIndex].Date == date {
			next = after.Records[afterIndex]
			afterIndex++
		}
		for _, field := range PositionFields() {
			var oldID, newID string
			if previous.Active() {
				oldID = previous.Data[field]
			}
			if next.Active() {
				newID = next.Data[field]
			}
			if oldID == newID {
				continue
			}
			if oldID != "" {
				changes = append(changes, PositionUseBoundary{field, oldID, date, field == "jobId" && previous.Data["gradeId"] != ""})
			}
			if newID != "" {
				changes = append(changes, PositionUseBoundary{field, newID, date, field == "jobId" && next.Data["gradeId"] != ""})
			}
		}
	}
	return changes
}

func PositionConflict(source Timeline, start, end, gradeID string) string {
	index := At(source, start)
	if index < 0 {
		return start
	}
	for _, record := range source.Records[index:] {
		if end != "" && record.Date >= end {
			break
		}
		if !record.Active() || gradeID != "" && !slices.ContainsFunc(record.Grades, func(member GradeMember) bool { return member.GradeID == gradeID }) {
			return max(start, record.Date)
		}
	}
	return ""
}
