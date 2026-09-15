CREATE INDEX applications_page_created_idx ON applications(status, created_at DESC, id) WHERE deleted_at IS NULL;
CREATE INDEX applications_page_updated_idx ON applications(status, updated_at DESC, id) WHERE deleted_at IS NULL;
CREATE INDEX tenant_application_grants_page_created_idx ON tenant_application_grants(tenant_id, status, created_at DESC, application_id) WHERE deleted_at IS NULL;
CREATE INDEX tenant_application_grants_page_updated_idx ON tenant_application_grants(tenant_id, status, updated_at DESC, application_id) WHERE deleted_at IS NULL;
