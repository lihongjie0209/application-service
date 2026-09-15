package routepolicy

import (
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	appdb "github.com/lihongjie0209/application-service/internal/database"
	"github.com/lihongjie0209/microservice-platform-go/authz"
	"github.com/lihongjie0209/microservice-platform-go/principal"
	platformpolicy "github.com/lihongjie0209/microservice-platform-go/routepolicy"
)

func TestRepositoryLoadsDenormalizedPermissionReferences(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	db := sqlx.NewDb(database, "sqlmock")
	repository := NewRepository(db, appdb.NewTransactor(db))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id,route_id,expression,version FROM route_policy_definitions WHERE status='active' AND deleted_at IS NULL ORDER BY route_id`)).WillReturnRows(sqlmock.NewRows([]string{"id", "route_id", "expression", "version"}).AddRow("policy-1", "route-1", `permissions["application.read"]`, 2))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT r.policy_id,r.id,r.permission_key,r.resource,r.action,r.scope FROM route_policy_permission_refs r JOIN route_policy_definitions d ON d.id=r.policy_id AND d.deleted_at IS NULL AND d.status='active' WHERE r.deleted_at IS NULL ORDER BY r.policy_id,r.permission_key`)).WillReturnRows(sqlmock.NewRows([]string{"policy_id", "id", "permission_key", "resource", "action", "scope"}).AddRow("policy-1", "ref-1", "application.read", "application.catalog", "read", "platform"))
	definitions, err := repository.Load(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	permission := definitions[0].Permissions["application.read"]
	if len(definitions) != 1 || definitions[0].RouteID != "route-1" || permission.Scope != authz.ScopePlatform || permission.Resource != "application.catalog" {
		t.Fatalf("definitions = %+v", definitions)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestServiceRejectsInvalidExpressionBeforePersistence(t *testing.T) {
	compiler, err := platformpolicy.NewCompiler()
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(nil, nil, compiler, nil, nil, nil)
	ctx := principal.WithContext(t.Context(), principal.Principal{ID: "admin", Type: principal.TypeUser})
	_, err = service.Set(ctx, SetInput{RouteID: "route-1", Expression: `permissions["missing"]`, Status: "active"})
	if err == nil {
		t.Fatal("invalid expression unexpectedly reached persistence")
	}
	if !errors.Is(err, platformpolicy.ErrInvalid) {
		t.Fatalf("error = %v", err)
	}
}

func TestRoutePolicyWhereKeepsCountAndItemsFiltersEquivalent(t *testing.T) {
	where, args := routePolicyWhere(Filter{Keyword: "Menu", Protocol: "http", RouteStatus: "active", PolicyStatus: "disabled"})
	want := `r.deleted_at IS NULL AND (LOWER(r.path) LIKE ? OR LOWER(r.operation) LIKE ?) AND r.protocol=? AND r.status=? AND p.status=?`
	if where != want {
		t.Fatalf("where = %q", where)
	}
	if len(args) != 5 || args[0] != "%menu%" || args[2] != "http" || args[4] != "disabled" {
		t.Fatalf("args = %#v", args)
	}
}

func TestBootstrapRouteIDsMatchSharedStableIdentity(t *testing.T) {
	t.Parallel()
	for path, want := range map[string]string{
		"/api/v1/version":             "68c5fcb6-2d64-57b7-b393-a753da4b712a",
		"/api/v1/me":                  "8e4038d9-155e-5320-8e9b-30dcbe5de3b4",
		"/api/v1/route-policies/page": "05ddeb9e-0ba6-53d6-a4a6-58ee83005188",
		"/api/v1/route-policies/get":  "fd87b9a5-18ea-5d7e-8d30-c6c62d9d8eda",
		"/api/v1/route-policies/set":  "986d037e-007e-57d7-910e-64932ff0269a",
	} {
		route, err := platformpolicy.NewRoute("http", "post", path, "application-service", "")
		if err != nil || route.ID != want {
			t.Fatalf("route %s id = %q, %v; want %q", path, route.ID, err, want)
		}
	}
}
