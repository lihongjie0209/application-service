CREATE TABLE applications (
 id TEXT PRIMARY KEY, code TEXT NOT NULL UNIQUE, name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', icon TEXT NOT NULL DEFAULT '', default_route TEXT NOT NULL DEFAULT '', sort_order INTEGER NOT NULL DEFAULT 0, status TEXT NOT NULL, metadata_json TEXT NOT NULL DEFAULT '{}', published_release BIGINT NOT NULL DEFAULT 0, version BIGINT NOT NULL DEFAULT 1, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL, created_by TEXT NOT NULL, updated_by TEXT NOT NULL
);
CREATE INDEX idx_applications_status_sort ON applications(status, sort_order, id);
CREATE TABLE application_menu_drafts (
 id TEXT PRIMARY KEY, application_id TEXT NOT NULL REFERENCES applications(id), parent_id TEXT NOT NULL DEFAULT '', menu_code TEXT NOT NULL, menu_type TEXT NOT NULL, name TEXT NOT NULL, i18n_key TEXT NOT NULL DEFAULT '', route TEXT NOT NULL DEFAULT '', component TEXT NOT NULL DEFAULT '', icon TEXT NOT NULL DEFAULT '', external_url TEXT NOT NULL DEFAULT '', permission_code TEXT NOT NULL DEFAULT '', sort_order INTEGER NOT NULL DEFAULT 0, visible BOOLEAN NOT NULL DEFAULT TRUE, status TEXT NOT NULL DEFAULT 'active', version BIGINT NOT NULL DEFAULT 1, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL, created_by TEXT NOT NULL, updated_by TEXT NOT NULL, UNIQUE(application_id, menu_code)
);
CREATE INDEX idx_menu_drafts_tree ON application_menu_drafts(application_id, parent_id, sort_order, id);
CREATE TABLE application_menu_releases (
 id TEXT PRIMARY KEY, application_id TEXT NOT NULL REFERENCES applications(id), release_number BIGINT NOT NULL, status TEXT NOT NULL, comment TEXT NOT NULL DEFAULT '', version BIGINT NOT NULL DEFAULT 1, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL, created_by TEXT NOT NULL, updated_by TEXT NOT NULL, UNIQUE(application_id, release_number)
);
CREATE TABLE application_menu_release_items (
 id TEXT NOT NULL, release_id TEXT NOT NULL REFERENCES application_menu_releases(id), application_id TEXT NOT NULL, release_number BIGINT NOT NULL, parent_id TEXT NOT NULL DEFAULT '', menu_code TEXT NOT NULL, menu_type TEXT NOT NULL, name TEXT NOT NULL, i18n_key TEXT NOT NULL DEFAULT '', route TEXT NOT NULL DEFAULT '', component TEXT NOT NULL DEFAULT '', icon TEXT NOT NULL DEFAULT '', external_url TEXT NOT NULL DEFAULT '', permission_code TEXT NOT NULL DEFAULT '', sort_order INTEGER NOT NULL DEFAULT 0, visible BOOLEAN NOT NULL DEFAULT TRUE, status TEXT NOT NULL, version BIGINT NOT NULL DEFAULT 1, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL, created_by TEXT NOT NULL, updated_by TEXT NOT NULL, PRIMARY KEY(release_id,id), UNIQUE(application_id,release_number,menu_code)
);
CREATE INDEX idx_menu_release_tree ON application_menu_release_items(application_id, release_number, parent_id, sort_order, id);
CREATE TABLE tenant_application_grants (
 id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, application_id TEXT NOT NULL REFERENCES applications(id), status TEXT NOT NULL, valid_from TIMESTAMPTZ NOT NULL, valid_until TIMESTAMPTZ NULL, source TEXT NOT NULL, entitlements_json TEXT NOT NULL DEFAULT '{}', version BIGINT NOT NULL DEFAULT 1, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL, created_by TEXT NOT NULL, updated_by TEXT NOT NULL, UNIQUE(tenant_id,application_id)
);
CREATE INDEX idx_tenant_application_active ON tenant_application_grants(tenant_id,status,valid_from,valid_until);
CREATE TABLE application_outbox_events (
 id TEXT PRIMARY KEY, subject TEXT NOT NULL, envelope BYTEA NOT NULL, attempts INTEGER NOT NULL DEFAULT 0, available_at TIMESTAMPTZ NOT NULL, published_at TIMESTAMPTZ NULL, last_error TEXT NOT NULL DEFAULT '', version BIGINT NOT NULL DEFAULT 1, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL, created_by TEXT NOT NULL, updated_by TEXT NOT NULL
);
CREATE INDEX idx_application_outbox_pending ON application_outbox_events(published_at,available_at);

