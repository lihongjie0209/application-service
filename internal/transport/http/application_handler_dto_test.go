package httptransport

import (
	"encoding/json"
	"testing"
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
