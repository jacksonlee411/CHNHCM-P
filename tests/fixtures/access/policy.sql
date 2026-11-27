-- Disposable test databases only. Business regression cases describe concrete
-- permissions with this projection. Every edit creates real policy versions
-- and a real revision, so the product Guard exercises the production schema.
CREATE OR REPLACE VIEW access.subject_grant AS
 SELECT instance_id,grant_id,principal_id,scope_kind,scope_id,action,read_fields,write_fields
 FROM access.bound_permission;
CREATE OR REPLACE FUNCTION access.edit_fixture_grant() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE v uuid:=gen_random_uuid(); n bigint; op text;
BEGIN
 IF TG_OP='DELETE' THEN
  UPDATE access.grant_assignment SET revision=revision+1
   WHERE instance_id=OLD.instance_id AND grant_id=OLD.grant_id RETURNING revision INTO STRICT n;
  INSERT INTO access.grant_revision(instance_id,grant_id,revision,role_id,scope_id,fields_id,operation)
   SELECT instance_id,grant_id,n,role_id,scope_id,fields_id,'revoke' FROM access.grant_revision
   WHERE instance_id=OLD.instance_id AND grant_id=OLD.grant_id AND revision=n-1;
  RETURN OLD;
 END IF;
 INSERT INTO access.policy_version(instance_id,kind,version_id,definition_id,version_no,actions)
  VALUES(NEW.instance_id,'role',v,v,1,ARRAY[NEW.action]);
 INSERT INTO access.policy_version(instance_id,kind,version_id,definition_id,version_no,scope_kind,scope_id)
  VALUES(NEW.instance_id,'scope',v,v,1,NEW.scope_kind,NEW.scope_id);
 INSERT INTO access.policy_version(instance_id,kind,version_id,definition_id,version_no,read_fields,write_fields)
  VALUES(NEW.instance_id,'fields',v,v,1,NEW.read_fields,NEW.write_fields);
 INSERT INTO access.grant_assignment(instance_id,grant_id,principal_id,source,purpose,revision)
 VALUES(NEW.instance_id,NEW.grant_id,NEW.principal_id,'development',split_part(NEW.action,'.',1),1)
 ON CONFLICT(instance_id,grant_id) DO UPDATE SET revision=access.grant_assignment.revision+1
 RETURNING revision INTO n;
 op:=CASE WHEN n=1 THEN 'issue' ELSE 'adopt' END;
 INSERT INTO access.grant_revision(instance_id,grant_id,revision,role_id,scope_id,fields_id,operation)
 VALUES(NEW.instance_id,NEW.grant_id,n,v,v,v,op);
 RETURN NEW;
END $$;
CREATE OR REPLACE TRIGGER edit_fixture INSTEAD OF INSERT OR UPDATE OR DELETE ON access.subject_grant
 FOR EACH ROW EXECUTE FUNCTION access.edit_fixture_grant();
