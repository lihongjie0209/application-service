package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

type fakeAPI struct {
	applications   map[string]Application
	menus          map[string]map[string]Menu
	grants         map[string]map[string]Grant
	grantListCalls map[string]int
}

func newFakeAPI() *fakeAPI {
	return &fakeAPI{applications: map[string]Application{}, menus: map[string]map[string]Menu{}, grants: map[string]map[string]Grant{}, grantListCalls: map[string]int{}}
}

func (f *fakeAPI) ListApplications(context.Context) ([]Application, error) {
	values := make([]Application, 0, len(f.applications))
	for _, value := range f.applications {
		values = append(values, value)
	}
	return values, nil
}
func (f *fakeAPI) CreateApplication(_ context.Context, spec ApplicationSpec) (Application, error) {
	metadata, _ := json.Marshal(nonNilMap(spec.Metadata))
	value := Application{ID: "app-" + spec.Code, Code: spec.Code, Name: spec.Name, Status: "draft", MetadataJSON: string(metadata), Version: 1}
	f.applications[spec.Code] = value
	return value, nil
}
func (f *fakeAPI) UpdateApplication(_ context.Context, current Application, spec ApplicationSpec) (Application, error) {
	metadata, _ := json.Marshal(nonNilMap(spec.Metadata))
	current.Name, current.Description, current.Icon, current.DefaultRoute, current.SortOrder, current.Status, current.MetadataJSON = spec.Name, spec.Description, spec.Icon, spec.DefaultRoute, spec.SortOrder, "active", string(metadata)
	current.Version++
	f.applications[current.Code] = current
	return current, nil
}
func (f *fakeAPI) ListMenus(_ context.Context, applicationID string) ([]Menu, error) {
	values := make([]Menu, 0, len(f.menus[applicationID]))
	for _, value := range f.menus[applicationID] {
		values = append(values, value)
	}
	return values, nil
}
func (f *fakeAPI) UpsertMenu(_ context.Context, menu Menu, expected int64) (Menu, error) {
	if f.menus[menu.ApplicationID] == nil {
		f.menus[menu.ApplicationID] = map[string]Menu{}
	}
	if expected == 0 {
		menu.ID, menu.Version = fmt.Sprintf("menu-%s-%s", menu.ApplicationID, menu.Code), 1
	} else {
		menu.Version = expected + 1
	}
	f.menus[menu.ApplicationID][menu.Code] = menu
	return menu, nil
}
func (f *fakeAPI) PublishMenus(_ context.Context, applicationID string, version int64) error {
	for code, application := range f.applications {
		if application.ID == applicationID {
			application.PublishedRelease++
			application.Version = version + 1
			f.applications[code] = application
		}
	}
	return nil
}
func (f *fakeAPI) ListGrants(_ context.Context, tenantID string) ([]Grant, error) {
	f.grantListCalls[tenantID]++
	values := make([]Grant, 0, len(f.grants[tenantID]))
	for _, value := range f.grants[tenantID] {
		values = append(values, value)
	}
	return values, nil
}
func (f *fakeAPI) Grant(_ context.Context, tenantID, applicationID string, expected int64, _ time.Time) error {
	if f.grants[tenantID] == nil {
		f.grants[tenantID] = map[string]Grant{}
	}
	f.grants[tenantID][applicationID] = Grant{TenantID: tenantID, ApplicationID: applicationID, Status: "active", Version: max(1, expected+1)}
	return nil
}

func TestReconcilerApplyIsIdempotent(t *testing.T) {
	t.Parallel()
	manifest := Manifest{Applications: []ApplicationSpec{{
		Code: "orders", Name: "Orders", DefaultRoute: "/apps/orders/list", Metadata: map[string]any{"owner": "commerce"},
		Menus: []MenuSpec{{Code: "root", Type: "directory", Name: "Orders"}, {Code: "list", Parent: "root", Name: "Order list", Route: "list", Component: "orders.list"}},
	}}}
	api := newFakeAPI()
	reconciler := NewReconciler(api)
	first, err := reconciler.Apply(t.Context(), manifest, []string{"tenant-1", "tenant-1"})
	if err != nil {
		t.Fatal(err)
	}
	if first.ApplicationsCreated != 1 || first.ApplicationsUpdated != 1 || first.MenusCreated != 2 || first.MenusPublished != 1 || first.GrantsApplied != 1 {
		t.Fatalf("first result = %+v", first)
	}
	if api.menus["app-orders"]["list"].ParentID != api.menus["app-orders"]["root"].ID {
		t.Fatal("child menu did not resolve its parent ID")
	}
	second, err := reconciler.Apply(t.Context(), manifest, []string{"tenant-1"})
	if err != nil {
		t.Fatal(err)
	}
	if second != (Result{}) {
		t.Fatalf("second result = %+v, want no changes", second)
	}
	if api.grantListCalls["tenant-1"] != 2 {
		t.Fatalf("grant list calls = %d, want one per reconciliation", api.grantListCalls["tenant-1"])
	}
}

func TestManifestRejectsCrossApplicationPageAndCycles(t *testing.T) {
	t.Parallel()
	for name, manifest := range map[string]Manifest{
		"cross namespace": {Applications: []ApplicationSpec{{Code: "orders", Name: "Orders", Menus: []MenuSpec{{Code: "list", Name: "List", Component: "billing.list"}}}}},
		"cycle":           {Applications: []ApplicationSpec{{Code: "orders", Name: "Orders", Menus: []MenuSpec{{Code: "a", Parent: "b", Name: "A"}, {Code: "b", Parent: "a", Name: "B"}}}}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := manifest.Validate(); err == nil {
				t.Fatal("Validate() accepted invalid manifest")
			}
		})
	}
}
