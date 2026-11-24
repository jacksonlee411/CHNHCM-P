package domain

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"errors"
)

// GradeData is Grade's complete scalar value. The dated engine carries scalar
// fields in Record.Data; only this adapter maps Grade values into that format.
// Relationships belong to their consumers and are not Grade fields.
type GradeData struct {
	Name, NameEn, Description string
}

type GradePatch struct {
	Name, NameEn, Description *string
}

func (p GradePatch) Apply(base GradeData) (GradeData, error) {
	for _, field := range []struct {
		name   string
		value  *string
		target *string
	}{{"name", p.Name, &base.Name}, {"nameEn", p.NameEn, &base.NameEn}, {"description", p.Description, &base.Description}} {
		if field.value == nil {
			continue
		}
		var value string
		var err error
		if field.name == "description" {
			value, err = Text(*field.value, 1000)
		} else {
			value, err = foundation.NormalizeName(*field.value, field.name == "name")
		}
		if err != nil {
			message := err.Error()
			var fault *foundation.Fault
			if errors.As(err, &fault) {
				message = fault.Message
			}
			return base, fieldInvalid(field.name, message)
		}
		*field.target = value
	}
	if base.Name == "" {
		return base, fieldInvalid("name", "请输入名称")
	}
	return base, nil
}

func applyGrade(base map[string]string, changes map[string]*string) (map[string]string, error) {
	patch := GradePatch{}
	for key, value := range changes {
		// Optional null explicitly clears. A missing key preserves the snapshot.
		if value == nil {
			empty := ""
			value = &empty
		}
		switch key {
		case "name":
			patch.Name = value
		case "nameEn":
			patch.NameEn = value
		case "description":
			patch.Description = value
		default:
			return nil, fieldInvalid(key, "职级不接受此字段")
		}
	}
	data, err := patch.Apply(GradeData{Name: base["name"], NameEn: base["nameEn"], Description: base["description"]})
	if err != nil {
		return nil, err
	}
	return map[string]string{"name": data.Name, "nameEn": data.NameEn, "description": data.Description}, nil
}
