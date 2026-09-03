package bootstrap

import (
	"path/filepath"
	"testing"
)

func TestDefaultPlatformManifestIsValid(t *testing.T) {
	t.Parallel()
	manifest, err := LoadManifest(filepath.Join("..", "..", "bootstrap", "platform-applications.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Applications) != 17 {
		t.Fatalf("applications = %d, want 17", len(manifest.Applications))
	}
	components := map[string]struct{}{}
	actionPermissions := map[string]struct{}{}
	actionCount := 0
	platformManagementPages := map[string]bool{
		"platform-admin.applications":       true,
		"platform-admin.menus":              true,
		"platform-admin.application-grants": true,
		"billing-center.plans":              true,
		"dictionary-center.providers":       true,
		"platform-admin.tenants":            true,
		"metering-center.meters":            true,
	}
	for _, application := range manifest.Applications {
		for _, menu := range application.Menus {
			if menu.Type == "action" {
				actionCount++
				if menu.Parent == "" || menu.PermissionCode == "" || visible(menu.Visible) {
					t.Fatalf("action %q must be hidden and reference a parent permission", menu.Code)
				}
				actionPermissions[menu.PermissionCode] = struct{}{}
			}
			if menu.Component == "" {
				continue
			}
			if _, exists := components[menu.Component]; exists {
				t.Fatalf("duplicate component %q", menu.Component)
			}
			components[menu.Component] = struct{}{}
			if menu.PermissionCode == "" || (menu.PermissionScope != "tenant" && menu.PermissionScope != "platform") {
				t.Fatalf("component %q has incomplete permission reference %q/%q", menu.Component, menu.PermissionScope, menu.PermissionCode)
			}
			if platformManagementPages[menu.Component] && menu.PermissionScope != "platform" {
				t.Fatalf("global management component %q has scope %q, want platform", menu.Component, menu.PermissionScope)
			}
		}
	}
	if len(components) != 43 {
		t.Fatalf("page components = %d, want 43", len(components))
	}
	if len(actionPermissions) != 120 {
		t.Fatalf("action permissions = %d, want 120", len(actionPermissions))
	}
	if actionCount != 128 {
		t.Fatalf("action nodes = %d, want 128", actionCount)
	}
	if _, exists := actionPermissions["identity.user.update-profile"]; !exists {
		t.Fatal("user profile update permission is missing")
	}
	if _, exists := actionPermissions["identity.service-account.rotate-secret"]; !exists {
		t.Fatal("service account secret rotation permission is missing")
	}
	if _, exists := actionPermissions["notification.provider.update"]; !exists {
		t.Fatal("notification provider update permission is missing")
	}
}
