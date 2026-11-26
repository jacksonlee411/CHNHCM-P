package postgres

import (
	"context"

	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	"github.com/jackc/pgx/v5"
)

// The original numbers, promises and revision trail are evidence. Report
// ambiguous cases to an authorized reviewer without changing those facts.
func (u *unit) retainedIdentities(ctx context.Context, report *h.ProbationRepairReport) error {
	rows, err := u.tx.Query(ctx, `WITH original AS (
	SELECT act.id,act.kind,t.person_id,t.employment_id,e.employer_id,t.period,t.department_id,t.job_id,t.bu_id,t.location_id,
	    (SELECT jsonb_agg(DISTINCT jsonb_build_array(coalesce(seg->>'position',''),CASE WHEN coalesce(seg->>'position','')='' THEN coalesce(nullif(seg->>'departmentId',''),t.department_id::text) END, CASE WHEN coalesce(seg->>'position','')='' THEN coalesce(nullif(seg->>'jobId',''),t.job_id::text) END)) FROM jsonb_array_elements(r.adoption#>'{secondary,segments}') seg) AS arrangement,coalesce(r.adoption#>>'{secondary,grade,id}','') AS grade
	FROM staffing.employment_action act JOIN staffing.assignment t ON (t.instance_id,t.id)=(act.instance_id,act.assignment_id)
	JOIN staffing.employment e ON (e.instance_id,e.id)=(t.instance_id,t.employment_id)
	JOIN staffing.employment_action_revision r ON (r.instance_id,r.action_id,r.revision)=(act.instance_id,act.id,act.revision)
	WHERE act.instance_id=$1 AND act.kind IN ('AdditionalAssignment','ParallelEmployment')),
	numbered AS (SELECT id,count(*) OVER (PARTITION BY person_id,kind,employer_id,CASE WHEN kind='AdditionalAssignment' THEN employment_id END,
	period,arrangement,grade,bu_id,location_id) AS n FROM original),
	hired AS (SELECT act.id,count(*) OVER(PARTITION BY t.person_id) AS n FROM staffing.employment_action act
	JOIN staffing.assignment t ON (t.instance_id,t.id)=(act.instance_id,act.assignment_id)
	WHERE act.instance_id=$1 AND act.kind='Hire' AND t.record_number=0 AND NOT EXISTS
	(SELECT FROM staffing.employment_action c WHERE c.instance_id=act.instance_id AND c.original_id=act.id AND c.kind='CancelHire')),
	items AS (
	SELECT id,CASE WHEN n>1 THEN '相同初办安排已发多个兼岗号，请核对原任用与编号' ELSE '' END AS reason FROM numbered
	UNION ALL SELECT id,CASE WHEN n>1 THEN '同一人员有多份有效首聘原件，请核对离职及再聘依据' ELSE '' END FROM hired
	UNION ALL SELECT DISTINCT p.action_id,'原交接段已使用专额，请按当时的核定、原任和第三方承诺核对名额'
	FROM staffing.assignment_head hd JOIN staffing.assignment_segment seg USING(instance_id,assignment_id,version)
	JOIN staffing.position_succession p ON (p.instance_id,p.id)=(seg.instance_id,nullif(seg.adoption->>'handoverSource','')::uuid)
	LEFT JOIN staffing.quota_source qs ON qs.instance_id=seg.instance_id AND qs.source_id=nullif(seg.adoption->>'quotaSource','')::uuid
	LEFT JOIN LATERAL (SELECT (row->>'quota')::int AS amount FROM jsonb_array_elements(qs.snapshot->'records') row
	  WHERE row->>'revision'=seg.adoption->>'quotaRevision') adopted ON true
	WHERE hd.instance_id=$1 AND NOT seg.adoption ? 'successionSource' AND (adopted.amount IS NULL OR adopted.amount>1)
	UNION ALL SELECT DISTINCT r.action_id,'晋升更正与前版通知、职级及依赖相同，请核对已发布的更正审计'
	FROM staffing.employment_action_revision r JOIN staffing.employment_action a ON (a.instance_id,a.id)=(r.instance_id,r.action_id)
	JOIN staffing.employment_action_revision before ON (before.instance_id,before.action_id,before.revision)=(r.instance_id,r.action_id,r.previous_revision)
	WHERE r.instance_id=$1 AND a.kind='Promotion' AND r.operation='Correct' AND r.business_date=before.business_date AND r.notice=before.notice
	AND (r.adoption #- '{promotion,priorSource}')=(before.adoption #- '{promotion,priorSource}'))
	SELECT id,reason FROM items ORDER BY id,reason`, u.store.Instance)
	if err != nil {
		return err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[struct{ ID, Reason string }])
	if err != nil {
		return err
	}
	for _, item := range items {
		a, err := u.action(ctx, item.ID)
		if err != nil {
			return err
		}
		disclosure, err := u.actionDisclosure(ctx, a, true)
		if err != nil {
			return err
		}
		if !disclosure.to {
			continue
		}
		report.Scanned++
		if item.Reason == "" {
			continue
		}
		report.Differences++
		report.Unresolved++
		report.Items = append(report.Items, h.ProbationRepairItem{Action: item.ID, Result: "unresolved", Reason: item.Reason})
	}
	return u.retainedRecordModes(ctx, report)
}
