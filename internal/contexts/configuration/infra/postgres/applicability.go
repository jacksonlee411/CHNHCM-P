package postgres

import (
	"context"
	"encoding/json"
	"errors"

	app "chnhcm.local/chnhcm/internal/contexts/configuration/application"
	config "chnhcm.local/chnhcm/internal/contexts/configuration/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"github.com/jackc/pgx/v5"
)

type Store struct{ Instance string }

func (s *Store) SavePlanScope(ctx context.Context, enterprise, owner string, spec config.Scope, period foundation.Period, members []config.Member) (out config.Policy, err error) {
	tx, ok := dbtx.From(ctx)
	if !ok {
		return out, errors.New("plan scope requires a transaction")
	}
	spec, err = app.Normalize(spec)
	if err != nil {
		return out, err
	}
	if !period.Valid() || len(spec.Choices) != len(members) {
		return out, app.Invalid("范围的实际来源与适用期间不完整")
	}
	proofs := make(map[config.Choice]config.Member, len(members))
	for _, m := range members {
		proofs[m.Choice] = m
	}
	out = config.Policy{Version: foundation.NewID(), Mode: spec.Mode, ObservedFor: period}
	for _, choice := range spec.Choices {
		m := proofs[choice]
		if !foundation.ValidID(m.Source) || !foundation.ValidID(m.Record) || !foundation.ValidID(m.Revision) {
			return config.Policy{}, app.Invalid("适用对象缺少有效来源版本")
		}
		out.Members = append(out.Members, m)
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return out, err
	}
	_, err = tx.Exec(ctx, "INSERT INTO configuration.plan_scope_version(instance_id,enterprise_id,plan_id,version,policy) VALUES($1,$2,$3,$4,$5)", s.Instance, enterprise, owner, out.Version, raw)
	return out, err
}

func (s *Store) ReadPlanScope(ctx context.Context, enterprise, owner, version string) (out config.Policy, err error) {
	tx, ok := dbtx.From(ctx)
	if !ok {
		return out, errors.New("plan scope requires a transaction")
	}
	var raw []byte
	err = tx.QueryRow(ctx, "SELECT policy FROM configuration.plan_scope_version WHERE instance_id=$1 AND enterprise_id=$2 AND plan_id=$3 AND version=$4", s.Instance, enterprise, owner, version).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, app.Invalid("计划采用的范围版本不可用")
	}
	if err == nil {
		err = json.Unmarshal(raw, &out)
	}
	return out, err
}
func (*Store) Matches(p config.Policy, subjects map[string]string) bool {
	return app.Matches(p, subjects)
}

func (*Store) Normalize(v config.Scope) (config.Scope, error) { return app.Normalize(v) }
func (*Store) PermitsDimension(v config.Scope, dimension, id string) bool {
	return app.WithinDimension(v, dimension, id)
}
