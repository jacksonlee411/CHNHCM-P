package postgres

import "testing"

func TestDepartmentRelationGuardFixedContractVector(t *testing.T) {
	const want int64 = 7405740448247810073
	if got := administrativeGuardKey("11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222", "33333333-3333-4333-8333-333333333333"); got != want {
		t.Fatalf("guard namespace/encoding changed: %d", got)
	}
}
