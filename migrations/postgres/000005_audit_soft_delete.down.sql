DROP INDEX IF EXISTS application_menu_drafts_active_tree_idx;
DROP TRIGGER IF EXISTS application_outbox_events_audit_row ON application_outbox_events;
ALTER TABLE application_outbox_events DROP COLUMN deleted_by, DROP COLUMN deleted_at;
DROP TRIGGER IF EXISTS tenant_application_grants_audit_row ON tenant_application_grants;
ALTER TABLE tenant_application_grants DROP COLUMN deleted_by, DROP COLUMN deleted_at;
DROP TRIGGER IF EXISTS application_menu_release_items_audit_row ON application_menu_release_items;
ALTER TABLE application_menu_release_items DROP COLUMN deleted_by, DROP COLUMN deleted_at;
DROP TRIGGER IF EXISTS application_menu_releases_audit_row ON application_menu_releases;
ALTER TABLE application_menu_releases DROP COLUMN deleted_by, DROP COLUMN deleted_at;
DROP TRIGGER IF EXISTS application_menu_drafts_audit_row ON application_menu_drafts;
ALTER TABLE application_menu_drafts DROP COLUMN deleted_by, DROP COLUMN deleted_at;
DROP TRIGGER IF EXISTS applications_audit_row ON applications;
ALTER TABLE applications DROP COLUMN deleted_by, DROP COLUMN deleted_at;
DROP FUNCTION IF EXISTS app_audit_row();

