package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lihongjie0209/application-service/internal/apperror"
	"github.com/lihongjie0209/microservice-platform-go/principal"
	applicationv1 "github.com/lihongjie0209/platform-protos/gen/go/platform/application/v1"
)

type navigationRepository struct {
	Repository
	active map[string]bool
}

func (r navigationRepository) BatchActiveGrants(context.Context, string, []string, time.Time) (map[string]bool, error) {
	return r.active, nil
}

func (navigationRepository) GetApplication(context.Context, string) (Application, error) {
	return Application{ID: "app-1", PublishedRelease: 1}, nil
}

func (navigationRepository) GetRelease(context.Context, string, int64) (MenuRelease, []Menu, error) {
	return MenuRelease{ID: "release-1"}, []Menu{{ID: "menu-1"}}, nil
}

func TestGetPublishedNavigationRequiresActiveTenantGrant(t *testing.T) {
	t.Parallel()
	ctx := principal.WithContext(t.Context(), principal.Principal{ID: "user-1", Type: principal.TypeUser, TenantID: "tenant-1"})

	denied := &Service{repository: navigationRepository{active: map[string]bool{}}, now: time.Now}
	if _, _, _, err := denied.GetPublishedNavigation(ctx, "app-1"); appErrorCode(err) != apperror.CodeForbidden {
		t.Fatalf("GetPublishedNavigation() error = %#v, want forbidden", err)
	}

	allowed := &Service{repository: navigationRepository{active: map[string]bool{"app-1": true}}, now: time.Now}
	app, release, menus, err := allowed.GetPublishedNavigation(ctx, "app-1")
	if err != nil || app.ID != "app-1" || release.ID != "release-1" || len(menus) != 1 {
		t.Fatalf("GetPublishedNavigation() = (%+v, %+v, %+v, %v)", app, release, menus, err)
	}
}

func appErrorCode(err error) int {
	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return 0
}

func TestAuthorizeTenant(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		identity *principal.Principal
		tenantID string
		wantCode int
	}{
		{name: "matching user tenant", identity: &principal.Principal{ID: "user-1", Type: principal.TypeUser, TenantID: "tenant-1"}, tenantID: "tenant-1"},
		{name: "different user tenant", identity: &principal.Principal{ID: "user-1", Type: principal.TypeUser, TenantID: "tenant-1"}, tenantID: "tenant-2", wantCode: apperror.CodeForbidden},
		{name: "user without active tenant", identity: &principal.Principal{ID: "user-1", Type: principal.TypeUser}, tenantID: "tenant-1", wantCode: apperror.CodeForbidden},
		{name: "service account cross tenant", identity: &principal.Principal{ID: "service-1", Type: principal.TypeServiceAccount}, tenantID: "tenant-2"},
		{name: "system cross tenant", identity: &principal.Principal{ID: "system", Type: principal.TypeSystem}, tenantID: "tenant-2"},
		{name: "unknown principal type", identity: &principal.Principal{ID: "unknown"}, tenantID: "tenant-1", wantCode: apperror.CodeForbidden},
		{name: "missing principal", tenantID: "tenant-1", wantCode: apperror.CodeUnauthorized},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			if test.identity != nil {
				ctx = principal.WithContext(ctx, *test.identity)
			}
			err := authorizeTenant(ctx, test.tenantID)
			if test.wantCode == 0 {
				if err != nil {
					t.Fatalf("authorizeTenant() error = %v", err)
				}
				return
			}
			if appErrorCode(err) != test.wantCode {
				t.Fatalf("authorizeTenant() error = %#v, want code %d", err, test.wantCode)
			}
		})
	}
}

func TestValidateMenuTree(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		if err := validateMenuTree([]Menu{{ID: "root"}, {ID: "child", ParentID: "root"}}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("cycle", func(t *testing.T) {
		if err := validateMenuTree([]Menu{{ID: "a", ParentID: "b"}, {ID: "b", ParentID: "a"}}); err == nil {
			t.Fatal("expected cycle to fail")
		}
	})
	t.Run("missing parent", func(t *testing.T) {
		if err := validateMenuTree([]Menu{{ID: "child", ParentID: "missing"}}); err == nil {
			t.Fatal("expected missing parent to fail")
		}
	})
}

func TestSearchDocumentUsesTenantVisibilityAndCompositeVersion(t *testing.T) {
	createdAt := time.Date(2026, time.August, 31, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	updatedAt := createdAt.Add(time.Hour)
	document := searchDocument(
		Application{ID: "app-1", Code: "orders", Name: "Orders", Description: "Order center", DefaultRoute: "/orders", Version: 7, CreatedAt: createdAt, UpdatedAt: updatedAt},
		Grant{TenantID: "tenant-1", Version: 11},
	)

	if document.GetSourceVersion() != searchProjectionVersion(7, 11) {
		t.Fatalf("source_version=%d", document.GetSourceVersion())
	}
	if len(document.GetVisibilityTokens()) != 1 || document.GetVisibilityTokens()[0] != "tenant:tenant-1:*" {
		t.Fatalf("visibility_tokens=%v", document.GetVisibilityTokens())
	}
	if got := document.GetSourceUpdatedAt().AsTime(); !got.Equal(updatedAt) {
		t.Fatalf("source_updated_at=%s", got)
	}
	if searchProjectionVersion(8, 1) <= searchProjectionVersion(7, 100) {
		t.Fatal("an application update must sort after every earlier grant version")
	}
	if searchProjectionVersion(7, 12) <= searchProjectionVersion(7, 11) {
		t.Fatal("a grant update must increase the projection version")
	}
}

func TestNewEventEnvelopeCarriesApplicationScope(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, time.September, 1, 18, 0, 0, 0, time.FixedZone("CST", 8*60*60))

	envelope, err := newEventEnvelope(
		"platform.application.v1.TenantApplicationGrantChanged",
		"grant-1",
		"tenant_application_grant",
		"tenant-1",
		"app-1",
		"actor-1",
		at,
		&applicationv1.TenantApplicationGrantChangedEvent{},
	)
	if err != nil {
		t.Fatalf("newEventEnvelope() error = %v", err)
	}
	if envelope.GetTenantId() != "tenant-1" || envelope.GetApplicationId() != "app-1" {
		t.Fatalf("event scope = tenant %q application %q", envelope.GetTenantId(), envelope.GetApplicationId())
	}
	if envelope.GetAggregateId() != "grant-1" || envelope.GetContext().GetActorId() != "actor-1" {
		t.Fatalf("event metadata = aggregate %q actor %q", envelope.GetAggregateId(), envelope.GetContext().GetActorId())
	}
}

func TestValidateApplication(t *testing.T) {
	for name, input := range map[string]ApplicationInput{
		"bad code":     {Code: "Bad Code", Name: "App", MetadataJSON: "{}"},
		"empty name":   {Code: "valid-app", MetadataJSON: "{}"},
		"invalid json": {Code: "valid-app", Name: "App", MetadataJSON: "{"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := validateApplication(input, true); err == nil {
				t.Fatal("expected validation failure")
			}
		})
	}
	value, err := validateApplication(ApplicationInput{Code: "  Valid-App ", Name: "App"}, true)
	if err != nil || value.Code != "valid-app" || value.MetadataJSON != "{}" {
		t.Fatalf("value=%+v err=%v", value, err)
	}
}
