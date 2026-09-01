package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

type Reconciler struct {
	api API
	now func() time.Time
}

func NewReconciler(api API) *Reconciler { return &Reconciler{api: api, now: time.Now} }

func (r *Reconciler) Apply(ctx context.Context, manifest Manifest, tenantIDs []string) (Result, error) {
	if err := manifest.Validate(); err != nil {
		return Result{}, err
	}
	existing, err := r.api.ListApplications(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("list applications: %w", err)
	}
	byCode := make(map[string]Application, len(existing))
	for _, application := range existing {
		byCode[application.Code] = application
	}
	tenants := normalizedTenantIDs(tenantIDs)
	grantsByTenant := make(map[string]map[string]Grant, len(tenants))
	for _, tenantID := range tenants {
		grants, listErr := r.api.ListGrants(ctx, tenantID)
		if listErr != nil {
			return Result{}, fmt.Errorf("list tenant %q grants: %w", tenantID, listErr)
		}
		grantsByTenant[tenantID] = make(map[string]Grant, len(grants))
		for _, grant := range grants {
			grantsByTenant[tenantID][grant.ApplicationID] = grant
		}
	}
	var result Result
	for _, spec := range manifest.Applications {
		application, exists := byCode[spec.Code]
		if !exists {
			application, err = r.api.CreateApplication(ctx, spec)
			if err != nil {
				return result, fmt.Errorf("create application %q: %w", spec.Code, err)
			}
			result.ApplicationsCreated++
		}
		if !applicationEqual(application, spec) {
			application, err = r.api.UpdateApplication(ctx, application, spec)
			if err != nil {
				return result, fmt.Errorf("update application %q: %w", spec.Code, err)
			}
			result.ApplicationsUpdated++
		}
		menuChanged, menuResult, reconcileErr := r.reconcileMenus(ctx, application, spec.Menus)
		result.MenusCreated += menuResult.MenusCreated
		result.MenusUpdated += menuResult.MenusUpdated
		if reconcileErr != nil {
			return result, fmt.Errorf("reconcile application %q menus: %w", spec.Code, reconcileErr)
		}
		if menuChanged || application.PublishedRelease == 0 {
			if err := r.api.PublishMenus(ctx, application.ID, application.Version); err != nil {
				return result, fmt.Errorf("publish application %q menus: %w", spec.Code, err)
			}
			result.MenusPublished++
		}
		for _, tenantID := range tenants {
			grant, exists := grantsByTenant[tenantID][application.ID]
			expected, active := grant.Version, exists && grant.Status == "active"
			if !active {
				if err := r.api.Grant(ctx, tenantID, application.ID, expected, r.now()); err != nil {
					return result, fmt.Errorf("grant application %q to tenant %q: %w", spec.Code, tenantID, err)
				}
				grantsByTenant[tenantID][application.ID] = Grant{TenantID: tenantID, ApplicationID: application.ID, Status: "active", Version: max(1, expected+1)}
				result.GrantsApplied++
			}
		}
	}
	return result, nil
}

func (r *Reconciler) reconcileMenus(ctx context.Context, application Application, specs []MenuSpec) (bool, Result, error) {
	existing, err := r.api.ListMenus(ctx, application.ID)
	if err != nil {
		return false, Result{}, err
	}
	byCode := make(map[string]Menu, len(existing))
	for _, menu := range existing {
		byCode[menu.Code] = menu
	}
	ordered, err := orderedMenus(specs)
	if err != nil {
		return false, Result{}, err
	}
	ids := make(map[string]string, len(existing))
	for code, menu := range byCode {
		ids[code] = menu.ID
	}
	changed := false
	var result Result
	for _, spec := range ordered {
		menu, exists := byCode[spec.Code]
		desired := menuFromSpec(application.ID, ids[spec.Parent], spec)
		expected := int64(0)
		if exists {
			desired.ID, expected = menu.ID, menu.Version
			if menuEqual(menu, desired) {
				ids[spec.Code] = menu.ID
				continue
			}
		}
		menu, err = r.api.UpsertMenu(ctx, desired, expected)
		if err != nil {
			return changed, result, err
		}
		ids[spec.Code] = menu.ID
		changed = true
		if exists {
			result.MenusUpdated++
		} else {
			result.MenusCreated++
		}
	}
	return changed, result, nil
}

func applicationEqual(value Application, spec ApplicationSpec) bool {
	metadata, _ := json.Marshal(nonNilMap(spec.Metadata))
	var current map[string]any
	_ = json.Unmarshal([]byte(value.MetadataJSON), &current)
	var desired map[string]any
	_ = json.Unmarshal(metadata, &desired)
	return value.Name == spec.Name && value.Description == spec.Description && value.Icon == spec.Icon &&
		value.DefaultRoute == spec.DefaultRoute && value.SortOrder == spec.SortOrder && value.Status == "active" && reflect.DeepEqual(current, desired)
}

func menuFromSpec(applicationID, parentID string, spec MenuSpec) Menu {
	menuType := strings.TrimSpace(spec.Type)
	if menuType == "" {
		menuType = "page"
	}
	return Menu{ApplicationID: applicationID, ParentID: parentID, Code: spec.Code, Type: menuType, Name: spec.Name, I18nKey: spec.I18nKey, Route: spec.Route, Component: spec.Component, Icon: spec.Icon, ExternalURL: spec.ExternalURL, PermissionCode: spec.PermissionCode, SortOrder: spec.SortOrder, Visible: visible(spec.Visible), Status: "active"}
}

func menuEqual(left, right Menu) bool {
	left.ID, left.Version = "", 0
	right.ID, right.Version = "", 0
	return reflect.DeepEqual(left, right)
}

func normalizedTenantIDs(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
