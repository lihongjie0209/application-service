package httptransport

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/lihongjie0209/application-service/internal/application"
)

func TestApplicationMetadataAcceptsJSONObject(t *testing.T) {
	t.Parallel()
	var request CreateApplicationRequest
	if err := json.Unmarshal([]byte(`{"code":"console","name":"Console","metadata_json":{"owner":"platform","features":["menus"]}}`), &request); err != nil {
		t.Fatal(err)
	}
	encoded, err := encodeJSONObject(request.MetadataJSON)
	if err != nil {
		t.Fatal(err)
	}
	if encoded != `{"features":["menus"],"owner":"platform"}` {
		t.Fatalf("metadata = %s", encoded)
	}
}

func TestMenuInputIgnoresServerOwnedFields(t *testing.T) {
	t.Parallel()

	var request UpsertMenuRequest
	if err := json.Unmarshal([]byte(`{
		"menu":{"id":"menu-1","application_id":"app-1","code":"orders","type":"page","name":"Orders",
		"created_by":"attacker","updated_by":"attacker","release_id":"secret-release","version":999},
		"expected_version":2
	}`), &request); err != nil {
		t.Fatal(err)
	}
	menu := request.Menu.applicationMenu()
	if menu.CreatedBy != "" || menu.UpdatedBy != "" || menu.ReleaseID != "" || menu.Version != 0 {
		t.Fatalf("server-owned fields reached domain input: %+v", menu)
	}
}

func TestApplicationBodyPublicJSONContract(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.September, 2, 8, 0, 0, 0, time.UTC)
	encoded, err := json.Marshal(applicationBody(application.Application{
		ID: "app-1", Code: "orders", Name: "Orders", Description: "Order management", Icon: "mdi:cart",
		DefaultRoute: "orders", SortOrder: 1, Status: "active", MetadataJSON: `{}`, PublishedRelease: 3,
		Version: 4, CreatedAt: now, UpdatedAt: now, CreatedBy: "user-1", UpdatedBy: "user-2",
	}))
	if err != nil {
		t.Fatalf("marshal application body: %v", err)
	}
	assertPublicJSONKeys(t, encoded, []string{
		"code", "created_at", "created_by", "default_route", "description", "icon", "id", "metadata_json", "name",
		"published_release", "sort_order", "status", "updated_at", "updated_by", "version",
	})
}

func TestMenuBodyDoesNotExposeReleaseStorageID(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(menuBody(application.Menu{ID: "menu-1", ReleaseID: "internal-release-row"}))
	if err != nil {
		t.Fatalf("marshal menu body: %v", err)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &body); err != nil {
		t.Fatalf("unmarshal menu body: %v", err)
	}
	if _, exists := body["release_id"]; exists {
		t.Fatal("menu body exposed release_id")
	}
	if strings.Contains(string(encoded), "internal-release-row") {
		t.Fatal("menu body exposed internal release identifier")
	}
}

func assertPublicJSONKeys(t *testing.T, encoded []byte, expected []string) {
	t.Helper()

	var body map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &body); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}
	actual := make([]string, 0, len(body))
	for key := range body {
		actual = append(actual, key)
	}
	sort.Strings(actual)
	sort.Strings(expected)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("public json keys = %v, want %v", actual, expected)
	}
}

func TestApplicationMetadataRejectsDoubleEncodedJSON(t *testing.T) {
	t.Parallel()
	var request CreateApplicationRequest
	if err := json.Unmarshal([]byte(`{"code":"console","name":"Console","metadata_json":"{}"}`), &request); err == nil {
		t.Fatal("json.Unmarshal() error = nil")
	}
}

func TestGrantEntitlementsAcceptJSONObject(t *testing.T) {
	t.Parallel()
	var request GrantRequest
	if err := json.Unmarshal([]byte(`{"tenant_id":"tenant-1","application_id":"app-1","entitlements_json":{"plan":"pro"}}`), &request); err != nil {
		t.Fatal(err)
	}
	encoded, err := encodeJSONObject(request.EntitlementsJSON)
	if err != nil {
		t.Fatal(err)
	}
	if encoded != `{"plan":"pro"}` {
		t.Fatalf("entitlements = %s", encoded)
	}
}
