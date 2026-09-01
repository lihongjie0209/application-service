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
	for _, application := range manifest.Applications {
		for _, menu := range application.Menus {
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
		}
	}
	if len(components) != 41 {
		t.Fatalf("page components = %d, want 41", len(components))
	}
}
