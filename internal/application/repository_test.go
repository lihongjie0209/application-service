package application

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestListGrantsCountJoinsApplicationsForDeletedFilter(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	repository := &SQLRepository{db: sqlx.NewDb(database, "sqlmock")}

	query := `SELECT COUNT(*) FROM tenant_application_grants g JOIN applications a ON a.id=g.application_id WHERE g.tenant_id=? AND g.deleted_at IS NULL AND a.deleted_at IS NULL`
	mock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs("tenant-1").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	listQuery := `SELECT g.id,g.tenant_id,g.application_id,g.status,g.valid_from,g.valid_until,g.source,g.entitlements_json,g.version,g.created_at,g.updated_at,g.created_by,g.updated_by FROM tenant_application_grants g JOIN applications a ON a.id=g.application_id WHERE g.tenant_id=? AND g.deleted_at IS NULL AND a.deleted_at IS NULL ORDER BY a.sort_order,a.id LIMIT ? OFFSET ?`
	mock.ExpectQuery(regexp.QuoteMeta(listQuery)).WithArgs("tenant-1", 20, 0).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	grants, applications, total, err := repository.ListGrants(t.Context(), "tenant-1", false, time.Time{}, 20, 0)
	if err != nil || total != 0 || len(grants) != 0 || len(applications) != 0 {
		t.Fatalf("ListGrants() = (%v, %v, %d, %v)", grants, applications, total, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
