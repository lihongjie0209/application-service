package database

import (
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/lihongjie0209/microservice-platform-go/principal"
)

func TestTransactorSetsDialectAuditActor(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, driver, query string
	}{
		{name: "postgres", driver: "pgx", query: "SELECT set_config('app.actor_id', $1, true)"},
		{name: "mysql", driver: "mysql", query: "SET @app_actor_id = ?"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			raw, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = raw.Close() })
			db := sqlx.NewDb(raw, test.driver)
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta(test.query)).WithArgs("user-1").WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()
			ctx := principal.WithContext(t.Context(), principal.Principal{ID: "user-1", Type: principal.TypeUser})
			if err := NewTransactor(db).Within(ctx, nil, func(*sqlx.Tx) error { return nil }); err != nil {
				t.Fatal(err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestTransactorRejectsMissingAuditActor(t *testing.T) {
	t.Parallel()
	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	db := sqlx.NewDb(raw, "pgx")
	mock.ExpectBegin()
	mock.ExpectRollback()
	if err := NewTransactor(db).Within(t.Context(), nil, func(*sqlx.Tx) error { return nil }); !errors.Is(err, ErrMissingAuditActor) {
		t.Fatalf("Within() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
