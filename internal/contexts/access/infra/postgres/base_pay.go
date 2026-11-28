package postgres

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"context"
	"errors"
)

func (s *Store) BasePayRights(ctx context.Context, subject, assignment string) (out access.BasePayRights, err error) {
	tx, ok := dbtx.From(ctx)
	if !ok {
		return out, errors.New("salary authorization requires guarded transaction")
	}
	rows, err := tx.Query(ctx, "SELECT action,coalesce(plan_id::text,'') FROM access.base_pay_grant WHERE instance_id=$1 AND principal_id=$2 AND assignment_id=$3", s.Instance, subject, assignment)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	out.Plans = map[string]bool{}
	for rows.Next() {
		var action, plan string
		if err = rows.Scan(&action, &plan); err != nil {
			return out, err
		}
		switch action {
		case "read":
			out.Read = true
		case "history":
			out.History = true
		case "first":
			out.First = true
		case "adjust":
			out.Adjust = true
		case "correct":
			out.Correct = true
		case "withdraw":
			out.Withdraw = true
		case "use":
			if plan == "" {
				out.UseAll = true
			} else {
				out.Plans[plan] = true
			}
		}
	}
	return out, rows.Err()
}
