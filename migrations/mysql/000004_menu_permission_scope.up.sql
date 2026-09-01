ALTER TABLE application_menu_drafts
    ADD COLUMN permission_scope VARCHAR(16) NOT NULL DEFAULT 'tenant';
ALTER TABLE application_menu_release_items
    ADD COLUMN permission_scope VARCHAR(16) NOT NULL DEFAULT 'tenant';

ALTER TABLE application_menu_drafts
    ADD CONSTRAINT application_menu_drafts_permission_scope_check CHECK (permission_scope IN ('tenant', 'platform'));
ALTER TABLE application_menu_release_items
    ADD CONSTRAINT application_menu_release_items_permission_scope_check CHECK (permission_scope IN ('tenant', 'platform'));
