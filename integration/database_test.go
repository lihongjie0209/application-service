//go:build integration

package integration

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	applicationdomain "github.com/lihongjie0209/application-service/internal/application"
	"github.com/lihongjie0209/application-service/internal/config"
	appdb "github.com/lihongjie0209/application-service/internal/database"
	"github.com/lihongjie0209/application-service/internal/migration"
	"github.com/lihongjie0209/microservice-platform-go/principal"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestRepositoryAndMigrations(t *testing.T) {
	for _, databaseType := range []string{"postgres", "mysql"} {
		t.Run(databaseType, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
			defer cancel()
			dsn, migrationURL := startDatabase(t, ctx, databaseType)
			migrationPath, err := filepath.Abs(filepath.Join("..", "migrations", databaseType))
			if err != nil {
				t.Fatal(err)
			}
			schema := ""
			if databaseType == "postgres" {
				schema = "integration_postgres"
			}
			migrationCfg := config.Migration{Path: migrationPath, DatabaseURL: migrationURL, Table: "integration_" + databaseType + "_schema_migrations", Schema: schema, CreateSchema: schema != ""}
			migrationErrors := make(chan error, 3)
			var migrations sync.WaitGroup
			for range 3 {
				migrations.Add(1)
				go func() {
					defer migrations.Done()
					migrationErrors <- migration.Run(migrationCfg, "up", 0)
				}()
			}
			migrations.Wait()
			close(migrationErrors)
			for err := range migrationErrors {
				if err != nil {
					t.Fatalf("concurrent migration up: %v", err)
				}
			}

			db, err := appdb.Open(ctx, config.Database{Type: databaseType, DSN: dsn, Schema: schema, MaxOpenConns: 5, MaxIdleConns: 2, ConnMaxLifetime: time.Minute, ConnMaxIdleTime: time.Minute, PingTimeout: 10 * time.Second})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = db.Close() })
			var userTables int
			if databaseType == "postgres" {
				if err := db.GetContext(ctx, &userTables, `SELECT count(*) FROM pg_tables WHERE schemaname = current_schema() AND tablename = 'users'`); err != nil {
					t.Fatal(err)
				}
				var timezone string
				if err := db.GetContext(ctx, &timezone, `SHOW TIMEZONE`); err != nil || timezone != "Asia/Shanghai" {
					t.Fatalf("timezone=%q err=%v", timezone, err)
				}
			} else if err := db.GetContext(ctx, &userTables, `SELECT count(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'users'`); err != nil {
				t.Fatal(err)
			} else {
				var timezone string
				if err := db.GetContext(ctx, &timezone, `SELECT @@session.time_zone`); err != nil || timezone != "+08:00" {
					t.Fatalf("timezone=%q err=%v", timezone, err)
				}
			}
			if userTables != 0 {
				t.Fatal("generic template migration must not create a users table")
			}
			repository := applicationdomain.NewRepository(db)
			ctx = principal.WithContext(ctx, principal.Principal{ID: "integration-test", Type: principal.TypeSystem})
			transactor := appdb.NewTransactor(db)
			write := func(fn func(sqlx.ExtContext) error) error {
				return transactor.Within(ctx, nil, func(tx *sqlx.Tx) error { return fn(tx) })
			}
			now := time.Now().Truncate(time.Microsecond)
			application := applicationdomain.Application{ID: "app-1", Code: "orders", Name: "Orders", SortOrder: 20, Status: "active", MetadataJSON: "{}", Version: 1, CreatedAt: now, UpdatedAt: now, CreatedBy: "test", UpdatedBy: "test"}
			if err := write(func(executor sqlx.ExtContext) error { return repository.CreateApplication(ctx, executor, application) }); err != nil {
				t.Fatal(err)
			}
			var audit struct {
				CreatedBy string `db:"created_by"`
				UpdatedBy string `db:"updated_by"`
				Version   int64  `db:"version"`
			}
			if err := db.GetContext(ctx, &audit, db.Rebind(`SELECT created_by,updated_by,version FROM applications WHERE id=?`), application.ID); err != nil {
				t.Fatal(err)
			}
			if audit.CreatedBy != "integration-test" || audit.UpdatedBy != "integration-test" || audit.Version != 1 {
				t.Fatalf("database-owned audit fields = %+v", audit)
			}
			searchedApplications, searchedTotal, err := repository.SearchApplications(ctx, "order", "active", 20, 0)
			if err != nil || searchedTotal != 1 || len(searchedApplications) != 1 || searchedApplications[0].ID != application.ID {
				t.Fatalf("SearchApplications() = (%+v, %d, %v)", searchedApplications, searchedTotal, err)
			}
			application.Name, application.UpdatedAt = "Order Center", now.Add(time.Second)
			if err := write(func(executor sqlx.ExtContext) error {
				return repository.UpdateApplication(ctx, executor, application, 1)
			}); err != nil {
				t.Fatal(err)
			}
			if err := write(func(executor sqlx.ExtContext) error {
				return repository.UpdateApplication(ctx, executor, application, 1)
			}); err == nil {
				t.Fatal("expected stale application version")
			}
			menu := applicationdomain.Menu{ID: "menu-1", ApplicationID: application.ID, Code: "orders.list", Type: "page", Name: "Orders", Route: "/orders", PermissionCode: "orders.list", PermissionScope: "tenant", Visible: true, Status: "active", Version: 1, CreatedAt: now, UpdatedAt: now, CreatedBy: "test", UpdatedBy: "test"}
			if err := write(func(executor sqlx.ExtContext) error { return repository.UpsertMenu(ctx, executor, menu, 0) }); err != nil {
				t.Fatal(err)
			}
			release := applicationdomain.MenuRelease{ID: "release-1", ApplicationID: application.ID, ReleaseNumber: 1, Status: "published", Version: 1, CreatedAt: now, UpdatedAt: now, CreatedBy: "test", UpdatedBy: "test"}
			if err := write(func(executor sqlx.ExtContext) error {
				return repository.CreateRelease(ctx, executor, release, []applicationdomain.Menu{menu})
			}); err != nil {
				t.Fatal(err)
			}
			grant := applicationdomain.Grant{ID: "grant-1", TenantID: "tenant-1", ApplicationID: application.ID, Status: "active", ValidFrom: now, Source: "manual", EntitlementsJSON: "{}", Version: 1, CreatedAt: now, UpdatedAt: now, CreatedBy: "test", UpdatedBy: "test"}
			if err := write(func(executor sqlx.ExtContext) error { return repository.CreateGrant(ctx, executor, grant) }); err != nil {
				t.Fatal(err)
			}
			prioritizedApplication := applicationdomain.Application{ID: "app-2", Code: "accounts", Name: "Accounts", SortOrder: 10, Status: "active", MetadataJSON: "{}", Version: 1, CreatedAt: now, UpdatedAt: now, CreatedBy: "test", UpdatedBy: "test"}
			if err := write(func(executor sqlx.ExtContext) error {
				return repository.CreateApplication(ctx, executor, prioritizedApplication)
			}); err != nil {
				t.Fatal(err)
			}
			prioritizedGrant := applicationdomain.Grant{ID: "grant-2", TenantID: grant.TenantID, ApplicationID: prioritizedApplication.ID, Status: "active", ValidFrom: now, Source: "manual", EntitlementsJSON: "{}", Version: 1, CreatedAt: now.Add(-time.Second), UpdatedAt: now, CreatedBy: "test", UpdatedBy: "test"}
			if err := write(func(executor sqlx.ExtContext) error { return repository.CreateGrant(ctx, executor, prioritizedGrant) }); err != nil {
				t.Fatal(err)
			}
			grantBatch, err := repository.BatchGrants(ctx, grant.TenantID, []string{application.ID, "missing-app"})
			if err != nil || len(grantBatch) != 1 || grantBatch[0].ID != grant.ID {
				t.Fatalf("BatchGrants() = (%+v, %v)", grantBatch, err)
			}
			grants, applications, total, err := repository.ListGrants(ctx, grant.TenantID, true, now.Add(time.Second), 100, 0)
			if err != nil {
				t.Fatal(err)
			}
			if total != 2 || len(grants) != 2 || len(applications) != 2 {
				t.Fatalf("unexpected grant page total=%d grants=%d applications=%d", total, len(grants), len(applications))
			}
			if grants[0].ApplicationID != prioritizedApplication.ID || applications[0].ID != prioritizedApplication.ID {
				t.Fatalf("tenant applications are not ordered by application sort order: grants=%v applications=%v", grants, applications)
			}
			createdFrom, createdTo := now.Add(-time.Minute), now.Add(time.Minute)
			filteredApplications, filteredTotal, err := repository.PageApplications(ctx, applicationdomain.ApplicationFilter{Keyword: "order", Status: "active", IDs: []string{application.ID}, CreatedFrom: &createdFrom, CreatedTo: &createdTo}, 20, 0)
			if err != nil || filteredTotal != 1 || len(filteredApplications) != 1 || filteredApplications[0].ID != application.ID {
				t.Fatalf("PageApplications() = (%+v, %d, %v)", filteredApplications, filteredTotal, err)
			}
			filteredGrants, filteredGrantApps, filteredGrantTotal, err := repository.PageGrants(ctx, applicationdomain.GrantFilter{TenantID: grant.TenantID, ApplicationIDs: []string{prioritizedApplication.ID}, Statuses: []string{"active"}, UpdatedFrom: &createdFrom, UpdatedTo: &createdTo}, now.Add(time.Second), 20, 0)
			if err != nil || filteredGrantTotal != 1 || len(filteredGrants) != 1 || len(filteredGrantApps) != 1 || filteredGrants[0].ApplicationID != prioritizedApplication.ID {
				t.Fatalf("PageGrants() = (%+v, %+v, %d, %v)", filteredGrants, filteredGrantApps, filteredGrantTotal, err)
			}
			active, err := repository.BatchActiveGrants(ctx, grant.TenantID, []string{application.ID, "missing"}, now.Add(time.Second))
			if err != nil || !active[application.ID] || active["missing"] {
				t.Fatalf("active=%v err=%v", active, err)
			}
			if err := write(func(executor sqlx.ExtContext) error {
				return repository.DeleteMenu(ctx, executor, menu.ID, 1, now.Add(2*time.Second), "ignored")
			}); err != nil {
				t.Fatal(err)
			}
			if _, err := repository.GetMenu(ctx, menu.ID); !errors.Is(err, applicationdomain.ErrNotFound) {
				t.Fatalf("GetMenu() after logical delete error = %v", err)
			}
			protected := applicationdomain.Application{ID: "app-delete-protected", Code: "protected", Name: "Protected", Status: "draft", MetadataJSON: "{}", Version: 1, CreatedAt: now, UpdatedAt: now, CreatedBy: "ignored", UpdatedBy: "ignored"}
			if err := write(func(executor sqlx.ExtContext) error { return repository.CreateApplication(ctx, executor, protected) }); err != nil {
				t.Fatal(err)
			}
			if err := write(func(executor sqlx.ExtContext) error {
				_, err := executor.ExecContext(ctx, db.Rebind(`DELETE FROM applications WHERE id=?`), protected.ID)
				return err
			}); err == nil {
				t.Fatal("physical application delete unexpectedly succeeded")
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			if err := migration.Run(migrationCfg, "down", 0); err != nil {
				t.Fatalf("migration down: %v", err)
			}
		})
	}
}

func startDatabase(t *testing.T, ctx context.Context, databaseType string) (string, string) {
	t.Helper()
	switch databaseType {
	case "postgres":
		container, err := postgres.Run(ctx, "postgres:17-alpine", postgres.WithDatabase("app"), postgres.WithUsername("app"), postgres.WithPassword("app"), postgres.BasicWaitStrategies(), postgres.WithSQLDriver("pgx"))
		if err != nil {
			t.Fatal(err)
		}
		testcontainers.CleanupContainer(t, container)
		dsn, err := container.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			t.Fatal(err)
		}
		return dsn, dsn
	case "mysql":
		configPath, err := filepath.Abs(filepath.Join("testdata", "mysql.cnf"))
		if err != nil {
			t.Fatal(err)
		}
		container, err := mysql.Run(ctx, "mysql:8.4", mysql.WithDatabase("app"), mysql.WithUsername("app"), mysql.WithPassword("app"), mysql.WithConfigFile(configPath))
		if err != nil {
			t.Fatal(err)
		}
		testcontainers.CleanupContainer(t, container)
		dsn, err := container.ConnectionString(ctx, "parseTime=true")
		if err != nil {
			t.Fatal(err)
		}
		migrationDSN, err := container.ConnectionString(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return dsn, "mysql://" + migrationDSN
	default:
		t.Fatal(fmt.Errorf("unsupported database %q", databaseType))
		return "", ""
	}
}
