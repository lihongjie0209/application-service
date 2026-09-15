package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEveryApplicationTableRegistersDatabaseAuditMaintenance(t *testing.T) {
	t.Parallel()
	tables := []string{"applications", "application_menu_drafts", "application_menu_releases", "application_menu_release_items", "tenant_application_grants", "application_outbox_events"}
	for _, dialect := range []string{"postgres", "kingbase", "mysql"} {
		dialect := dialect
		t.Run(dialect, func(t *testing.T) {
			t.Parallel()
			content, err := os.ReadFile(filepath.Join("..", "..", "migrations", dialect, "000005_audit_soft_delete.up.sql"))
			if err != nil {
				t.Fatal(err)
			}
			ddl := strings.ToLower(string(content))
			for _, table := range tables {
				if !strings.Contains(ddl, "alter table "+table+" add column deleted_at") && !strings.Contains(ddl, "alter table "+table+" modify column created_at") {
					t.Errorf("%s does not add logical-delete fields for %s", dialect, table)
				}
				if dialect == "mysql" {
					if !strings.Contains(ddl, "create trigger "+table+"_audit_bi before insert") || !strings.Contains(ddl, "create trigger "+table+"_audit_bu before update") {
						t.Errorf("mysql audit insert/update triggers missing for %s", table)
					}
					if table != "application_outbox_events" && !strings.Contains(ddl, "create trigger "+table+"_audit_bd before delete") {
						t.Errorf("mysql delete protection missing for %s", table)
					}
					if !strings.Contains(ddl, "alter table "+table+" modify column created_at timestamp(6) not null, modify column updated_at timestamp(6) not null") {
						t.Errorf("mysql microsecond audit timestamps missing for %s", table)
					}
				} else if !strings.Contains(ddl, "create trigger "+table+"_audit_row before insert or update") {
					t.Errorf("%s audit trigger missing for %s", dialect, table)
				}
			}
		})
	}
}

func TestPageFilterIndexesExistForEveryDialect(t *testing.T) {
	t.Parallel()
	indexes := []string{"applications_page_created_idx", "applications_page_updated_idx", "tenant_application_grants_page_created_idx", "tenant_application_grants_page_updated_idx"}
	for _, dialect := range []string{"postgres", "kingbase", "mysql"} {
		dialect := dialect
		t.Run(dialect, func(t *testing.T) {
			t.Parallel()
			content, err := os.ReadFile(filepath.Join("..", "..", "migrations", dialect, "000006_page_filter_indexes.up.sql"))
			if err != nil {
				t.Fatal(err)
			}
			ddl := strings.ToLower(string(content))
			for _, index := range indexes {
				if !strings.Contains(ddl, "create index "+index) {
					t.Errorf("%s is missing %s", dialect, index)
				}
			}
		})
	}
}
