CREATE OR REPLACE FUNCTION app_audit_row()
RETURNS trigger LANGUAGE plpgsql AS $audit$
DECLARE actor_id text := NULLIF(current_setting('app.actor_id', true), '');
BEGIN
 IF actor_id IS NULL THEN RAISE EXCEPTION 'app.actor_id must be set for audited writes'; END IF;
 IF TG_OP = 'INSERT' THEN NEW.created_at := statement_timestamp(); NEW.updated_at := NEW.created_at; NEW.created_by := actor_id; NEW.updated_by := actor_id; NEW.version := 1; NEW.deleted_at := NULL; NEW.deleted_by := NULL; RETURN NEW; END IF;
 IF TG_OP = 'UPDATE' THEN NEW.created_at := OLD.created_at; NEW.created_by := OLD.created_by; NEW.updated_at := statement_timestamp(); NEW.updated_by := actor_id; NEW.version := OLD.version + 1; IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN NEW.deleted_at := statement_timestamp(); NEW.deleted_by := actor_id; ELSIF OLD.deleted_at IS NOT NULL AND NEW.deleted_at IS NULL THEN NEW.deleted_by := NULL; ELSE NEW.deleted_at := OLD.deleted_at; NEW.deleted_by := OLD.deleted_by; END IF; RETURN NEW; END IF;
 RAISE EXCEPTION 'physical DELETE is forbidden on audited table %, use deleted_at', TG_TABLE_NAME;
END;
$audit$;

ALTER TABLE applications ADD COLUMN deleted_at TIMESTAMPTZ, ADD COLUMN deleted_by TEXT;
ALTER TABLE application_menu_drafts ADD COLUMN deleted_at TIMESTAMPTZ, ADD COLUMN deleted_by TEXT;
ALTER TABLE application_menu_releases ADD COLUMN deleted_at TIMESTAMPTZ, ADD COLUMN deleted_by TEXT;
ALTER TABLE application_menu_release_items ADD COLUMN deleted_at TIMESTAMPTZ, ADD COLUMN deleted_by TEXT;
ALTER TABLE tenant_application_grants ADD COLUMN deleted_at TIMESTAMPTZ, ADD COLUMN deleted_by TEXT;
ALTER TABLE application_outbox_events ADD COLUMN deleted_at TIMESTAMPTZ, ADD COLUMN deleted_by TEXT;

UPDATE application_menu_drafts SET deleted_at=updated_at,deleted_by=updated_by WHERE status='deleted';
CREATE TRIGGER applications_audit_row BEFORE INSERT OR UPDATE OR DELETE ON applications FOR EACH ROW EXECUTE FUNCTION app_audit_row();
CREATE TRIGGER application_menu_drafts_audit_row BEFORE INSERT OR UPDATE OR DELETE ON application_menu_drafts FOR EACH ROW EXECUTE FUNCTION app_audit_row();
CREATE TRIGGER application_menu_releases_audit_row BEFORE INSERT OR UPDATE OR DELETE ON application_menu_releases FOR EACH ROW EXECUTE FUNCTION app_audit_row();
CREATE TRIGGER application_menu_release_items_audit_row BEFORE INSERT OR UPDATE OR DELETE ON application_menu_release_items FOR EACH ROW EXECUTE FUNCTION app_audit_row();
CREATE TRIGGER tenant_application_grants_audit_row BEFORE INSERT OR UPDATE OR DELETE ON tenant_application_grants FOR EACH ROW EXECUTE FUNCTION app_audit_row();
CREATE TRIGGER application_outbox_events_audit_row BEFORE INSERT OR UPDATE ON application_outbox_events FOR EACH ROW EXECUTE FUNCTION app_audit_row();
CREATE INDEX application_menu_drafts_active_tree_idx ON application_menu_drafts(application_id,parent_id,sort_order,id) WHERE deleted_at IS NULL;

