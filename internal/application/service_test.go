package application

import (
	"testing"
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
