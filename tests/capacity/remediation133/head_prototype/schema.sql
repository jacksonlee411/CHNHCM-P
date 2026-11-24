-- PRIVATE EXPERIMENT ONLY. Never a product migration.
CREATE TABLE organization.proof133_node (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, payload bytea NOT NULL,
 hash bytea GENERATED ALWAYS AS (sha256(payload)) STORED,
 first_slot integer NOT NULL CHECK(first_slot>=0), members integer NOT NULL CHECK(members>0),
 first_date date NOT NULL,last_date date NOT NULL CHECK(last_date>=first_date),
 left_hash bytea,right_hash bytea,
 PRIMARY KEY(instance_id,object_id,hash),
 FOREIGN KEY(instance_id,object_id) REFERENCES organization.business_unit(instance_id,bu_id),
 FOREIGN KEY(instance_id,object_id,left_hash) REFERENCES organization.proof133_node(instance_id,object_id,hash),
 FOREIGN KEY(instance_id,object_id,right_hash) REFERENCES organization.proof133_node(instance_id,object_id,hash),
 CHECK((left_hash IS NULL)=(right_hash IS NULL)),
 CHECK((convert_from(payload,'UTF8')::jsonb->>'Instance')::uuid=instance_id AND
       (convert_from(payload,'UTF8')::jsonb->>'Object')::uuid=object_id AND
       (convert_from(payload,'UTF8')::jsonb->>'Start')::integer=first_slot AND
       (convert_from(payload,'UTF8')::jsonb->>'Count')::integer=members AND
       (convert_from(payload,'UTF8')::jsonb->>'Low')::date=first_date AND
       (convert_from(payload,'UTF8')::jsonb->>'High')::date=last_date)
);
CREATE FUNCTION organization.proof133_node_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE l organization.proof133_node; r organization.proof133_node; j jsonb;
BEGIN
 IF TG_OP<>'INSERT' THEN RAISE EXCEPTION 'immutable proof node' USING ERRCODE='23514'; END IF;
 j:=convert_from(NEW.payload,'UTF8')::jsonb;
 IF NEW.left_hash IS NULL THEN
  IF NEW.members>128 OR jsonb_array_length(j->'Records')<>NEW.members OR j->'Left'<>'null'::jsonb OR j->'Right'<>'null'::jsonb
   THEN RAISE EXCEPTION 'leaf framing' USING ERRCODE='23514'; END IF;
 ELSE
  SELECT * INTO STRICT l FROM organization.proof133_node WHERE instance_id=NEW.instance_id AND object_id=NEW.object_id AND hash=NEW.left_hash;
  SELECT * INTO STRICT r FROM organization.proof133_node WHERE instance_id=NEW.instance_id AND object_id=NEW.object_id AND hash=NEW.right_hash;
  IF NEW.first_slot<>l.first_slot OR r.first_slot<>l.first_slot+l.members OR NEW.members<>l.members+r.members
    OR NEW.first_date<>l.first_date OR NEW.last_date<>r.last_date OR l.last_date>=r.first_date
    OR j->'Left'->>'Hash'<>encode(l.hash,'hex') OR j->'Right'->>'Hash'<>encode(r.hash,'hex')
    OR (j->'Left'->>'Start')::integer<>l.first_slot OR (j->'Right'->>'Start')::integer<>r.first_slot
    OR (j->'Left'->>'Count')::integer<>l.members OR (j->'Right'->>'Count')::integer<>r.members
    OR (j->'Left'->>'Low')::date<>l.first_date OR (j->'Right'->>'Low')::date<>r.first_date
    OR (j->'Left'->>'High')::date<>l.last_date OR (j->'Right'->>'High')::date<>r.last_date
   THEN RAISE EXCEPTION 'branch framing' USING ERRCODE='23514'; END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER proof133_node_guard BEFORE INSERT OR UPDATE OR DELETE ON organization.proof133_node FOR EACH ROW EXECUTE FUNCTION organization.proof133_node_guard();
CREATE TABLE organization.proof133_source (
 instance_id uuid NOT NULL,object_id uuid NOT NULL,source_id uuid NOT NULL,source_no bigint NOT NULL,
 chunk_no integer NOT NULL,state_shape integer NOT NULL,root bytea NOT NULL,members integer NOT NULL,
 first_id uuid NOT NULL,first_date date NOT NULL,
 PRIMARY KEY(instance_id,object_id,source_id),
 UNIQUE(instance_id,object_id,source_id,root,members),
 FOREIGN KEY(instance_id,object_id,chunk_no) REFERENCES organization.bu_history(instance_id,object_id,chunk_no),
 FOREIGN KEY(instance_id,object_id,root) REFERENCES organization.proof133_node(instance_id,object_id,hash)
);
CREATE FUNCTION organization.proof133_source_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE h organization.bu_history; n organization.proof133_node; i integer;
BEGIN
 IF TG_OP<>'INSERT' THEN RAISE EXCEPTION 'immutable source proof' USING ERRCODE='23514'; END IF;
 SELECT * INTO STRICT h FROM organization.bu_history WHERE instance_id=NEW.instance_id AND object_id=NEW.object_id AND chunk_no=NEW.chunk_no;
 i:=NEW.source_no-h.first_no+1;
 IF i<1 OR i>cardinality(h.source_refs) OR h.ids[h.source_refs[i]]<>NEW.source_id OR h.states[i]<>NEW.state_shape
  THEN RAISE EXCEPTION 'proof must bind actual source' USING ERRCODE='23514'; END IF;
 SELECT * INTO STRICT n FROM organization.proof133_node WHERE instance_id=NEW.instance_id AND object_id=NEW.object_id AND hash=NEW.root;
 IF n.first_slot<>0 OR n.members<>NEW.members OR n.first_date<>NEW.first_date
  THEN RAISE EXCEPTION 'source root framing' USING ERRCODE='23514'; END IF;
 WHILE n.left_hash IS NOT NULL LOOP
  SELECT * INTO STRICT n FROM organization.proof133_node WHERE instance_id=NEW.instance_id AND object_id=NEW.object_id AND hash=n.left_hash;
 END LOOP;
 IF (convert_from(n.payload,'UTF8')::jsonb->'Records'->0->>'ID')::uuid<>NEW.first_id
  THEN RAISE EXCEPTION 'real first identity required' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER proof133_source_guard BEFORE INSERT OR UPDATE OR DELETE ON organization.proof133_source FOR EACH ROW EXECUTE FUNCTION organization.proof133_source_guard();
ALTER TABLE organization.business_unit ADD COLUMN proof133_root bytea,ADD COLUMN proof133_count integer;
ALTER TABLE organization.business_unit ADD CONSTRAINT proof133_head_source FOREIGN KEY(instance_id,bu_id,head_source_id,proof133_root,proof133_count) REFERENCES organization.proof133_source(instance_id,object_id,source_id,root,members) DEFERRABLE INITIALLY DEFERRED;
GRANT SELECT,INSERT ON organization.proof133_node,organization.proof133_source TO chnhcm_app;

GRANT UPDATE(proof133_root,proof133_count) ON organization.business_unit TO chnhcm_app;


 CREATE FUNCTION organization.proof133_history_guard() RETURNS trigger LANGUAGE plpgsql AS $$
 DECLARE first_new integer; i integer;
 BEGIN
  first_new:=1;
  IF TG_OP='UPDATE' THEN first_new:=cardinality(OLD.source_refs)+1;END IF;
  IF first_new>cardinality(NEW.source_refs) THEN RETURN NEW;END IF;
  FOR i IN first_new..cardinality(NEW.source_refs) LOOP
   IF NOT EXISTS(SELECT FROM organization.proof133_source s WHERE(s.instance_id,s.object_id,s.source_id,s.source_no,s.chunk_no,s.state_shape)=(NEW.instance_id,NEW.object_id,NEW.ids[NEW.source_refs[i]],NEW.first_no+i-1,NEW.chunk_no,NEW.states[i]))
    THEN RAISE EXCEPTION 'every real new source needs an immutable proof root' USING ERRCODE='23514';END IF;
  END LOOP;
  RETURN NEW;
 END $$;
 CREATE CONSTRAINT TRIGGER proof133_history_guard AFTER INSERT OR UPDATE ON organization.bu_history DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION organization.proof133_history_guard();
