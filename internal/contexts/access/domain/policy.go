package domain

import "strings"

// NormalizeLogin is shared by account initialization and authentication.
func NormalizeLogin(input string) (string, bool) {
	login := strings.TrimSpace(input)
	return login, len(login) > 0 && len(login) <= 100
}

func Contains(fields []string, field string) bool {
	for _, f := range fields {
		if f == field {
			return true
		}
	}
	return false
}
func ContainsAll(allowed, requested []string) bool {
	for _, f := range requested {
		if !Contains(allowed, f) {
			return false
		}
	}
	return true
}
func Matches(scopeKind, scopeID, action, enterpriseID, buID, requestedAction string) bool {
	return action == requestedAction && ((scopeKind == "enterprise" && scopeID == enterpriseID) || (scopeKind == "bu" && buID != "" && scopeID == buID))
}

func MatchesDepartment(scopeKind, scopeID, action, enterpriseID, departmentID, currentBU, requestedAction string) bool {
	return action == requestedAction && ((scopeKind == "enterprise" && scopeID == enterpriseID) ||
		(scopeKind == "bu" && currentBU != "" && scopeID == currentBU) ||
		(scopeKind == "department" && departmentID != "" && scopeID == departmentID))
}
