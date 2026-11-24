-- During cutover, replace the two predecessors' mutually exclusive quota checks. Other grants keep their original constraints.
-- +goose StatementBegin
DO $$ DECLARE c record; expr text; ext text; BEGIN
 FOR c IN SELECT conname,pg_get_constraintdef(oid) def FROM pg_constraint
 WHERE conrelid='access.subject_grant'::regclass AND contype='c' LOOP
  expr:=substring(c.def from 7);
  ext:=NULL;
  IF position('jobfamily.' IN expr)>0 AND position('read_fields' IN expr)>0 THEN
   ext:=$p$action LIKE 'staffingquota.%' AND read_fields <@ ARRAY['code','name','departmentId','status','effectiveDate','quota','reason','audit','usageState','sourceVersion']::text[] AND write_fields <@ ARRAY['effectiveDate','quota','reason']::text[]$p$;
  ELSIF position('jobfamily.read' IN expr)>0 AND position('action' IN expr)>0 THEN
   ext:=$p$action IN ('staffingquota.read','staffingquota.history.read','staffingquota.create','staffingquota.adjust','staffingquota.correct','staffingquota.redate','staffingquota.delete','staffingquota.withdraw')$p$;
  ELSIF position('scope_kind' IN expr)>0 AND (position('instance' IN expr)>0 OR position('staffingquota.%' IN expr)>0) THEN
   ext:=$p$scope_kind IN ('position','position_catalog','defined_position') AND action LIKE 'staffingquota.%'$p$;
  END IF;
  IF ext IS NOT NULL THEN
   EXECUTE format('ALTER TABLE access.subject_grant DROP CONSTRAINT %I',c.conname);
   EXECUTE format('ALTER TABLE access.subject_grant ADD CONSTRAINT %I CHECK ((%s) OR (%s))',c.conname,expr,ext);
  END IF;
 END LOOP;
END $$;
-- +goose StatementEnd

-- Preserve the original grant ID for read, history and the first write action.
-- The two catalog scopes also require the corresponding position-definition authority in X03.
CREATE TEMP TABLE quota_grants_to_convert ON COMMIT DROP AS
SELECT g.* FROM access.subject_grant g WHERE action IN ('quota.read','quota.history.read','quota.change');
DELETE FROM access.subject_grant WHERE action IN ('quota.read','quota.history.read','quota.change');
INSERT INTO access.subject_grant(instance_id,grant_id,principal_id,scope_kind,scope_id,action,read_fields,write_fields)
SELECT instance_id,CASE WHEN n=1 THEN grant_id ELSE gen_random_uuid() END,principal_id,
 CASE scope_kind WHEN 'enterprise' THEN 'position_catalog' ELSE 'defined_position' END,scope_id,'staffingquota.'||op,
 array_replace(array_replace(read_fields,'quantity','quota'),'operator','audit') ||
 CASE WHEN read_fields @> ARRAY['quantity','effectiveDate'] THEN ARRAY['sourceVersion']::text[] ELSE '{}'::text[] END,
 array_replace(write_fields,'quantity','quota')
FROM quota_grants_to_convert CROSS JOIN LATERAL unnest(
 CASE action WHEN 'quota.read' THEN ARRAY['read'] WHEN 'quota.history.read' THEN ARRAY['history.read']
 ELSE ARRAY['create','adjust','withdraw','correct','redate','delete'] END) WITH ORDINALITY AS op(op,n);
UPDATE access.policy_guard p SET policy_no=policy_no+1 WHERE EXISTS (
 SELECT FROM quota_grants_to_convert g WHERE (g.instance_id,g.principal_id)=(p.instance_id,p.principal_id));
ALTER TABLE access.subject_grant ADD CONSTRAINT quota_catalog_scope CHECK (
 action NOT LIKE 'quota.%' AND (action NOT LIKE 'staffingquota.%' OR scope_kind IN ('enterprise','position','position_catalog','defined_position'))
 AND (scope_kind NOT IN ('position_catalog','defined_position') OR action LIKE 'staffingquota.%'));
-- +goose Down
SELECT 1/0;
