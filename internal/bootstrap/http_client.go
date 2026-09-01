package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

type HTTPClient struct {
	baseURL       *url.URL
	authorization string
	client        *http.Client
}

type APIError struct {
	Status    int
	Code      int    `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("application service: status=%d code=%d message=%s request_id=%s", e.Status, e.Code, e.Message, e.RequestID)
}

func NewHTTPClient(baseURL, authorization string, timeout time.Duration) (*HTTPClient, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("application service URL must be an absolute HTTP(S) URL")
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &HTTPClient{baseURL: parsed, authorization: strings.TrimSpace(authorization), client: &http.Client{Timeout: timeout}}, nil
}

func (c *HTTPClient) ListApplications(ctx context.Context) ([]Application, error) {
	result := make([]Application, 0)
	for page := 1; ; page++ {
		var body struct {
			Items    []Application `json:"items"`
			Total    int           `json:"total"`
			PageSize int           `json:"page_size"`
		}
		if err := c.post(ctx, "/api/v1/applications/list", map[string]any{"page": page, "page_size": 100}, &body); err != nil {
			return nil, err
		}
		result = append(result, body.Items...)
		if len(result) >= body.Total || len(body.Items) == 0 {
			return result, nil
		}
	}
}

func (c *HTTPClient) CreateApplication(ctx context.Context, spec ApplicationSpec) (Application, error) {
	var result Application
	err := c.post(ctx, "/api/v1/applications/create", map[string]any{"code": spec.Code, "name": spec.Name, "description": spec.Description, "icon": spec.Icon, "default_route": spec.DefaultRoute, "sort_order": spec.SortOrder, "metadata_json": nonNilMap(spec.Metadata)}, &result)
	return result, err
}

func (c *HTTPClient) UpdateApplication(ctx context.Context, current Application, spec ApplicationSpec) (Application, error) {
	var result Application
	err := c.post(ctx, "/api/v1/applications/update", map[string]any{"id": current.ID, "name": spec.Name, "description": spec.Description, "icon": spec.Icon, "default_route": spec.DefaultRoute, "sort_order": spec.SortOrder, "status": "active", "metadata_json": nonNilMap(spec.Metadata), "version": current.Version}, &result)
	return result, err
}

func (c *HTTPClient) ListMenus(ctx context.Context, applicationID string) ([]Menu, error) {
	var result []Menu
	err := c.post(ctx, "/api/v1/applications/menus/draft/list", map[string]any{"application_id": applicationID}, &result)
	return result, err
}

func (c *HTTPClient) UpsertMenu(ctx context.Context, menu Menu, expected int64) (Menu, error) {
	var result Menu
	err := c.post(ctx, "/api/v1/applications/menus/upsert", map[string]any{"menu": menu, "expected_version": expected}, &result)
	return result, err
}

func (c *HTTPClient) PublishMenus(ctx context.Context, applicationID string, version int64) error {
	return c.post(ctx, "/api/v1/applications/menus/publish", map[string]any{"application_id": applicationID, "application_version": version, "comment": "platform-bootstrap reconciliation"}, &struct{}{})
}

func (c *HTTPClient) ListGrants(ctx context.Context, tenantID string) ([]Grant, error) {
	result := make([]Grant, 0)
	for page := 1; ; page++ {
		var body struct {
			Grants struct {
				Items []Grant `json:"items"`
				Total int     `json:"total"`
			} `json:"grants"`
		}
		if err := c.post(ctx, "/api/v1/applications/tenant-grants/list", map[string]any{"tenant_id": tenantID, "active_only": false, "page": page, "page_size": 100}, &body); err != nil {
			return nil, err
		}
		result = append(result, body.Grants.Items...)
		if len(result) >= body.Grants.Total || len(body.Grants.Items) == 0 {
			return result, nil
		}
	}
}

func (c *HTTPClient) Grant(ctx context.Context, tenantID, applicationID string, expected int64, validFrom time.Time) error {
	return c.post(ctx, "/api/v1/applications/tenant-grants/grant", map[string]any{"tenant_id": tenantID, "application_id": applicationID, "valid_from": validFrom, "source": "platform-bootstrap", "entitlements_json": map[string]any{}, "expected_version": expected}, &struct{}{})
}

func (c *HTTPClient) post(ctx context.Context, path string, input, output any) error {
	payload, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}
	target := c.baseURL.ResolveReference(&url.URL{Path: path})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", uuid.NewString())
	if c.authorization != "" {
		request.Header.Set("Authorization", c.authorization)
	}
	response, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("call application service: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, 16<<20))
	if err != nil {
		return fmt.Errorf("read application service response: %w", err)
	}
	var envelope struct {
		Code      int             `json:"code"`
		Message   string          `json:"message"`
		Body      json.RawMessage `json:"body"`
		RequestID string          `json:"request_id"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("decode application service response: %w", err)
	}
	if response.StatusCode != http.StatusOK || envelope.Code != 0 {
		return &APIError{Status: response.StatusCode, Code: envelope.Code, Message: envelope.Message, RequestID: envelope.RequestID}
	}
	if output == nil || len(envelope.Body) == 0 || bytes.Equal(envelope.Body, []byte("null")) {
		return nil
	}
	if err := json.Unmarshal(envelope.Body, output); err != nil {
		return fmt.Errorf("decode application service response body: %w", err)
	}
	return nil
}

func nonNilMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}
