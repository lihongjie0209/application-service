package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPClientUsesUniformEnvelopeAndAuthorization(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/applications/list" || request.Method != http.MethodPost || request.Header.Get("Authorization") != "PSK secret" || request.Header.Get("Idempotency-Key") == "" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "message": "success", "body": map[string]any{"items": []map[string]any{{"id": "app-1", "code": "orders"}}, "total": 1, "page": 1, "page_size": 100}})
	}))
	t.Cleanup(server.Close)
	client, err := NewHTTPClient(server.URL+"/ignored-prefix", "PSK secret", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	applications, err := client.ListApplications(t.Context())
	if err != nil || len(applications) != 1 || applications[0].Code != "orders" {
		t.Fatalf("applications = %+v, error = %v", applications, err)
	}
}

func TestHTTPClientReturnsStructuredAPIError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 20003, "message": "permission denied", "request_id": "request-1"})
	}))
	t.Cleanup(server.Close)
	client, err := NewHTTPClient(server.URL, "Bearer token", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListApplications(t.Context())
	apiError, ok := err.(*APIError)
	if !ok || apiError.Code != 20003 || apiError.RequestID != "request-1" {
		t.Fatalf("error = %#v", err)
	}
}
