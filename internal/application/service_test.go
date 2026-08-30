package application

import (
	"testing"
	"time"
)

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
