package postgres

import "chnhcm.local/chnhcm/internal/contexts/organization/infra/history/model"

// VerifyHistoryBU and VerifyHistoryDepartment expose the same read/write closure
// checks to privileged offline verification. They grant no persistence access.
func VerifyHistoryBU(s model.BUState, op model.BUOperation) error { return validateBUFacts(s, op) }
func VerifyHistoryDepartment(s model.DepartmentState, op model.DepartmentOperation, proof model.DepartmentEvidence) error {
	return validateDepartmentFacts(s, op, proof)
}
