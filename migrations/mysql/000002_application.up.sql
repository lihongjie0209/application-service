CREATE TABLE applications (
    id VARCHAR(36) PRIMARY KEY, code VARCHAR(128) NOT NULL UNIQUE, name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL, icon TEXT NOT NULL, default_route TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0, status VARCHAR(32) NOT NULL, metadata_json TEXT NOT NULL,
    published_release BIGINT NOT NULL DEFAULT 0, version BIGINT NOT NULL DEFAULT 1,
    created_at DATETIME(6) NOT NULL, updated_at DATETIME(6) NOT NULL,
    created_by VARCHAR(255) NOT NULL, updated_by VARCHAR(255) NOT NULL,
    INDEX idx_applications_status_sort (status, sort_order, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE application_menu_drafts (
    id VARCHAR(36) PRIMARY KEY, application_id VARCHAR(36) NOT NULL, parent_id VARCHAR(36) NOT NULL DEFAULT '',
    menu_code VARCHAR(128) NOT NULL, menu_type VARCHAR(32) NOT NULL, name VARCHAR(255) NOT NULL,
    i18n_key VARCHAR(255) NOT NULL DEFAULT '', route TEXT NOT NULL, component TEXT NOT NULL, icon TEXT NOT NULL,
    external_url TEXT NOT NULL, permission_code VARCHAR(255) NOT NULL DEFAULT '', sort_order INTEGER NOT NULL DEFAULT 0,
    visible BOOLEAN NOT NULL DEFAULT TRUE, status VARCHAR(32) NOT NULL DEFAULT 'active', version BIGINT NOT NULL DEFAULT 1,
    created_at DATETIME(6) NOT NULL, updated_at DATETIME(6) NOT NULL,
    created_by VARCHAR(255) NOT NULL, updated_by VARCHAR(255) NOT NULL,
    UNIQUE KEY uq_menu_draft_code (application_id, menu_code),
    INDEX idx_menu_drafts_tree (application_id, parent_id, sort_order, id),
    CONSTRAINT fk_menu_draft_application FOREIGN KEY (application_id) REFERENCES applications(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE application_menu_releases (
    id VARCHAR(36) PRIMARY KEY, application_id VARCHAR(36) NOT NULL, release_number BIGINT NOT NULL,
    status VARCHAR(32) NOT NULL, comment TEXT NOT NULL, version BIGINT NOT NULL DEFAULT 1,
    created_at DATETIME(6) NOT NULL, updated_at DATETIME(6) NOT NULL,
    created_by VARCHAR(255) NOT NULL, updated_by VARCHAR(255) NOT NULL,
    UNIQUE KEY uq_menu_release_number (application_id, release_number),
    CONSTRAINT fk_menu_release_application FOREIGN KEY (application_id) REFERENCES applications(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE application_menu_release_items (
    id VARCHAR(36) NOT NULL, release_id VARCHAR(36) NOT NULL, application_id VARCHAR(36) NOT NULL,
    release_number BIGINT NOT NULL, parent_id VARCHAR(36) NOT NULL DEFAULT '', menu_code VARCHAR(128) NOT NULL,
    menu_type VARCHAR(32) NOT NULL, name VARCHAR(255) NOT NULL, i18n_key VARCHAR(255) NOT NULL DEFAULT '',
    route TEXT NOT NULL, component TEXT NOT NULL, icon TEXT NOT NULL, external_url TEXT NOT NULL,
    permission_code VARCHAR(255) NOT NULL DEFAULT '', sort_order INTEGER NOT NULL DEFAULT 0,
    visible BOOLEAN NOT NULL DEFAULT TRUE, status VARCHAR(32) NOT NULL, version BIGINT NOT NULL DEFAULT 1,
    created_at DATETIME(6) NOT NULL, updated_at DATETIME(6) NOT NULL,
    created_by VARCHAR(255) NOT NULL, updated_by VARCHAR(255) NOT NULL,
    PRIMARY KEY (release_id, id), UNIQUE KEY uq_menu_release_item_code (application_id, release_number, menu_code),
    INDEX idx_menu_release_tree (application_id, release_number, parent_id, sort_order, id),
    CONSTRAINT fk_menu_release_item_release FOREIGN KEY (release_id) REFERENCES application_menu_releases(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE tenant_application_grants (
    id VARCHAR(36) PRIMARY KEY, tenant_id VARCHAR(36) NOT NULL, application_id VARCHAR(36) NOT NULL,
    status VARCHAR(32) NOT NULL, valid_from DATETIME(6) NOT NULL, valid_until DATETIME(6) NULL,
    source VARCHAR(64) NOT NULL, entitlements_json TEXT NOT NULL, version BIGINT NOT NULL DEFAULT 1,
    created_at DATETIME(6) NOT NULL, updated_at DATETIME(6) NOT NULL,
    created_by VARCHAR(255) NOT NULL, updated_by VARCHAR(255) NOT NULL,
    UNIQUE KEY uq_tenant_application_grant (tenant_id, application_id),
    INDEX idx_tenant_application_active (tenant_id, status, valid_from, valid_until),
    CONSTRAINT fk_tenant_grant_application FOREIGN KEY (application_id) REFERENCES applications(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE application_outbox_events (
    id VARCHAR(36) PRIMARY KEY, subject VARCHAR(255) NOT NULL, envelope LONGBLOB NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0, available_at DATETIME(6) NOT NULL, published_at DATETIME(6) NULL,
    last_error TEXT NOT NULL, version BIGINT NOT NULL DEFAULT 1,
    created_at DATETIME(6) NOT NULL, updated_at DATETIME(6) NOT NULL,
    created_by VARCHAR(255) NOT NULL, updated_by VARCHAR(255) NOT NULL,
    INDEX idx_application_outbox_pending (published_at, available_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
