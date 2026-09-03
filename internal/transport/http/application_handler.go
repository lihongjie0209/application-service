package httptransport

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lihongjie0209/application-service/internal/apperror"
	"github.com/lihongjie0209/application-service/internal/application"
)

type JSONObject map[string]any

type CreateApplicationRequest struct {
	Code         string     `json:"code" binding:"required"`
	Name         string     `json:"name" binding:"required"`
	Description  string     `json:"description"`
	Icon         string     `json:"icon"`
	DefaultRoute string     `json:"default_route"`
	SortOrder    int32      `json:"sort_order"`
	MetadataJSON JSONObject `json:"metadata_json" swaggertype:"object"`
}
type UpdateApplicationRequest struct {
	ID           string     `json:"id" binding:"required"`
	Name         string     `json:"name" binding:"required"`
	Description  string     `json:"description"`
	Icon         string     `json:"icon"`
	DefaultRoute string     `json:"default_route"`
	SortOrder    int32      `json:"sort_order"`
	Status       string     `json:"status" binding:"required"`
	MetadataJSON JSONObject `json:"metadata_json" swaggertype:"object"`
	Version      int64      `json:"version" binding:"required"`
}
type IDRequest struct {
	ID string `json:"id" binding:"required"`
}
type ListApplicationsRequest struct {
	Keyword  string `json:"keyword"`
	Status   string `json:"status"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}
type BatchTenantGrantsRequest struct {
	TenantID       string   `json:"tenant_id" binding:"required"`
	ApplicationIDs []string `json:"application_ids" binding:"required"`
}
type BatchTenantGrantsResponse struct {
	Items []GrantBody `json:"items"`
}
type UpsertMenuRequest struct {
	Menu            MenuInputBody `json:"menu" binding:"required"`
	ExpectedVersion int64         `json:"expected_version"`
}
type DeleteMenuRequest struct {
	ID      string `json:"id" binding:"required"`
	Version int64  `json:"version" binding:"required"`
}
type ApplicationIDRequest struct {
	ApplicationID string `json:"application_id" binding:"required"`
}
type PublishMenusRequest struct {
	ApplicationID      string `json:"application_id" binding:"required"`
	ApplicationVersion int64  `json:"application_version" binding:"required"`
	Comment            string `json:"comment"`
}
type PublishMenusResponse struct {
	Release MenuReleaseBody `json:"release"`
	Menus   []MenuBody      `json:"menus"`
}
type NavigationResponse struct {
	Application ApplicationBody `json:"application"`
	Release     MenuReleaseBody `json:"release"`
	Menus       []MenuBody      `json:"menus"`
}
type BatchNavigationRequest struct {
	ApplicationIDs []string `json:"application_ids" binding:"required"`
}
type BatchNavigationResponse struct {
	Items []NavigationResponse `json:"items"`
}
type GrantRequest struct {
	TenantID         string     `json:"tenant_id" binding:"required"`
	ApplicationID    string     `json:"application_id" binding:"required"`
	ValidFrom        time.Time  `json:"valid_from"`
	ValidUntil       *time.Time `json:"valid_until"`
	Source           string     `json:"source"`
	EntitlementsJSON JSONObject `json:"entitlements_json" swaggertype:"object"`
	ExpectedVersion  int64      `json:"expected_version"`
}
type RevokeGrantRequest struct {
	TenantID      string `json:"tenant_id" binding:"required"`
	ApplicationID string `json:"application_id" binding:"required"`
	Version       int64  `json:"version" binding:"required"`
}
type ListTenantApplicationsRequest struct {
	TenantID   string `json:"tenant_id" binding:"required"`
	ActiveOnly bool   `json:"active_only"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
}
type TenantApplicationsResponse struct {
	Grants       GrantPageBody     `json:"grants"`
	Applications []ApplicationBody `json:"applications"`
}
type GrantPageBody struct {
	Items    []GrantBody `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}
type BatchCheckRequest struct {
	TenantID       string    `json:"tenant_id" binding:"required"`
	ApplicationIDs []string  `json:"application_ids" binding:"required"`
	At             time.Time `json:"at"`
}
type Decision struct {
	ApplicationID string `json:"application_id"`
	Granted       bool   `json:"granted"`
	Reason        string `json:"reason"`
}

// CreateApplication godoc
// @Summary Create an application
// @Tags applications
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body CreateApplicationRequest true "Application"
// @Success 200 {object} Response{body=ApplicationBody}
// @Router /api/v1/applications/create [post]
func (h *Handler) CreateApplication(c *gin.Context) {
	var r CreateApplicationRequest
	if !bind(c, h.logger, &r) {
		return
	}
	metadata, err := encodeJSONObject(r.MetadataJSON)
	if err != nil {
		Fail(c, h.logger, apperror.Invalid("metadata_json must be a JSON object", err))
		return
	}
	v, err := h.applications.CreateApplication(c.Request.Context(), application.ApplicationInput{Code: r.Code, Name: r.Name, Description: r.Description, Icon: r.Icon, DefaultRoute: r.DefaultRoute, SortOrder: r.SortOrder, MetadataJSON: metadata})
	respond(c, h.logger, applicationBody(v), err)
}

// UpdateApplication godoc
// @Summary Update an application with optimistic locking
// @Tags applications
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body UpdateApplicationRequest true "Application and version"
// @Success 200 {object} Response{body=ApplicationBody}
// @Failure 409 {object} Response "Code 30009: version conflict"
// @Router /api/v1/applications/update [post]
func (h *Handler) UpdateApplication(c *gin.Context) {
	var r UpdateApplicationRequest
	if !bind(c, h.logger, &r) {
		return
	}
	metadata, err := encodeJSONObject(r.MetadataJSON)
	if err != nil {
		Fail(c, h.logger, apperror.Invalid("metadata_json must be a JSON object", err))
		return
	}
	v, err := h.applications.UpdateApplication(c.Request.Context(), r.ID, application.ApplicationInput{Name: r.Name, Description: r.Description, Icon: r.Icon, DefaultRoute: r.DefaultRoute, SortOrder: r.SortOrder, Status: r.Status, MetadataJSON: metadata}, r.Version)
	respond(c, h.logger, applicationBody(v), err)
}

// GetApplication godoc
// @Summary Get an application
// @Tags applications
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body IDRequest true "Application ID"
// @Success 200 {object} Response{body=ApplicationBody}
// @Router /api/v1/applications/get [post]
func (h *Handler) GetApplication(c *gin.Context) {
	var r IDRequest
	if !bind(c, h.logger, &r) {
		return
	}
	v, err := h.applications.GetApplication(c.Request.Context(), r.ID)
	respond(c, h.logger, applicationBody(v), err)
}

// ListApplications godoc
// @Summary List applications
// @Tags applications
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body ListApplicationsRequest true "Filters and pagination"
// @Success 200 {object} Response{body=ApplicationPageBody}
// @Router /api/v1/applications/list [post]
func (h *Handler) ListApplications(c *gin.Context) {
	var r ListApplicationsRequest
	if !bind(c, h.logger, &r) {
		return
	}
	v, err := h.applications.SearchApplications(c.Request.Context(), r.Keyword, r.Status, r.Page, r.PageSize)
	respond(c, h.logger, applicationPageBody(v), err)
}

// BatchManagedTenantGrants godoc
// @Summary Get grant state for a bounded application set
// @Tags tenant-applications
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body BatchTenantGrantsRequest true "Tenant and application IDs (maximum 100)"
// @Success 200 {object} Response{body=BatchTenantGrantsResponse}
// @Router /api/v1/applications/tenant-grants/manage/batch-get [post]
func (h *Handler) BatchManagedTenantGrants(c *gin.Context) {
	var request BatchTenantGrantsRequest
	if !bind(c, h.logger, &request) {
		return
	}
	items, err := h.applications.BatchGrants(c.Request.Context(), request.TenantID, request.ApplicationIDs)
	respond(c, h.logger, BatchTenantGrantsResponse{Items: grantBodies(items)}, err)
}

// UpsertMenu godoc
// @Summary Create or update a menu draft
// @Tags menus
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body UpsertMenuRequest true "Menu and expected version"
// @Success 200 {object} Response{body=MenuBody}
// @Router /api/v1/applications/menus/upsert [post]
func (h *Handler) UpsertMenu(c *gin.Context) {
	var r UpsertMenuRequest
	if !bind(c, h.logger, &r) {
		return
	}
	v, err := h.applications.UpsertMenu(c.Request.Context(), r.Menu.applicationMenu(), r.ExpectedVersion)
	respond(c, h.logger, menuBody(v), err)
}

// DeleteMenu godoc
// @Summary Delete a menu draft with optimistic locking
// @Tags menus
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body DeleteMenuRequest true "Menu ID and version"
// @Success 200 {object} Response
// @Router /api/v1/applications/menus/delete [post]
func (h *Handler) DeleteMenu(c *gin.Context) {
	var r DeleteMenuRequest
	if !bind(c, h.logger, &r) {
		return
	}
	err := h.applications.DeleteMenu(c.Request.Context(), r.ID, r.Version)
	respond(c, h.logger, gin.H{}, err)
}

// ListMenuDraft godoc
// @Summary List menu drafts
// @Tags menus
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body ApplicationIDRequest true "Application ID"
// @Success 200 {object} Response{body=[]MenuBody}
// @Router /api/v1/applications/menus/draft/list [post]
func (h *Handler) ListMenuDraft(c *gin.Context) {
	var r ApplicationIDRequest
	if !bind(c, h.logger, &r) {
		return
	}
	v, err := h.applications.ListMenuDraft(c.Request.Context(), r.ApplicationID)
	respond(c, h.logger, menuBodies(v), err)
}

// PublishMenus godoc
// @Summary Publish an immutable menu release
// @Tags menus
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body PublishMenusRequest true "Application version and comment"
// @Success 200 {object} Response{body=PublishMenusResponse}
// @Failure 409 {object} Response "Code 30009: version conflict"
// @Router /api/v1/applications/menus/publish [post]
func (h *Handler) PublishMenus(c *gin.Context) {
	var r PublishMenusRequest
	if !bind(c, h.logger, &r) {
		return
	}
	rel, menus, err := h.applications.PublishMenus(c.Request.Context(), r.ApplicationID, r.ApplicationVersion, r.Comment)
	respond(c, h.logger, PublishMenusResponse{Release: menuReleaseBody(rel), Menus: menuBodies(menus)}, err)
}

// GetNavigation godoc
// @Summary Get published application navigation
// @Tags menus
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body ApplicationIDRequest true "Application ID"
// @Success 200 {object} Response{body=NavigationResponse}
// @Router /api/v1/applications/navigation/get [post]
func (h *Handler) GetNavigation(c *gin.Context) {
	var r ApplicationIDRequest
	if !bind(c, h.logger, &r) {
		return
	}
	app, rel, menus, err := h.applications.GetPublishedNavigation(c.Request.Context(), r.ApplicationID)
	respond(c, h.logger, NavigationResponse{Application: applicationBody(app), Release: menuReleaseBody(rel), Menus: menuBodies(menus)}, err)
}

// ListNavigations godoc
// @Summary Get published navigation for multiple granted applications
// @Tags menus
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body BatchNavigationRequest true "Application IDs"
// @Success 200 {object} Response{body=BatchNavigationResponse}
// @Router /api/v1/applications/navigation/batch-get [post]
func (h *Handler) ListNavigations(c *gin.Context) {
	var request BatchNavigationRequest
	if !bind(c, h.logger, &request) {
		return
	}
	items, err := h.applications.ListPublishedNavigations(c.Request.Context(), request.ApplicationIDs)
	response := BatchNavigationResponse{Items: make([]NavigationResponse, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, NavigationResponse{
			Application: applicationBody(item.Application),
			Release:     menuReleaseBody(item.Release),
			Menus:       menuBodies(item.Menus),
		})
	}
	respond(c, h.logger, response, err)
}

// Grant godoc
// @Summary Grant an application to a tenant
// @Tags tenant-applications
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body GrantRequest true "Tenant application grant"
// @Success 200 {object} Response{body=GrantBody}
// @Router /api/v1/applications/tenant-grants/grant [post]
func (h *Handler) Grant(c *gin.Context) {
	var r GrantRequest
	if !bind(c, h.logger, &r) {
		return
	}
	entitlements, err := encodeJSONObject(r.EntitlementsJSON)
	if err != nil {
		Fail(c, h.logger, apperror.Invalid("entitlements_json must be a JSON object", err))
		return
	}
	v, err := h.applications.Grant(c.Request.Context(), r.TenantID, r.ApplicationID, r.ValidFrom, r.ValidUntil, r.Source, entitlements, r.ExpectedVersion)
	respond(c, h.logger, grantBody(v), err)
}

// Revoke godoc
// @Summary Revoke a tenant application grant
// @Tags tenant-applications
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body RevokeGrantRequest true "Grant version"
// @Success 200 {object} Response{body=GrantBody}
// @Router /api/v1/applications/tenant-grants/revoke [post]
func (h *Handler) Revoke(c *gin.Context) {
	var r RevokeGrantRequest
	if !bind(c, h.logger, &r) {
		return
	}
	v, err := h.applications.Revoke(c.Request.Context(), r.TenantID, r.ApplicationID, r.Version)
	respond(c, h.logger, grantBody(v), err)
}

// ListTenantApplications godoc
// @Summary List applications granted to a tenant
// @Tags tenant-applications
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body ListTenantApplicationsRequest true "Tenant and pagination"
// @Success 200 {object} Response{body=TenantApplicationsResponse}
// @Router /api/v1/applications/tenant-grants/list [post]
func (h *Handler) ListTenantApplications(c *gin.Context) {
	h.listTenantApplications(c)
}

// ListManagedTenantApplications godoc
// @Summary List tenant application grants as a platform administrator
// @Tags tenant-applications
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body ListTenantApplicationsRequest true "Target tenant and pagination"
// @Success 200 {object} Response{body=TenantApplicationsResponse}
// @Router /api/v1/applications/tenant-grants/manage/list [post]
func (h *Handler) ListManagedTenantApplications(c *gin.Context) {
	h.listTenantApplications(c)
}

func (h *Handler) listTenantApplications(c *gin.Context) {
	var r ListTenantApplicationsRequest
	if !bind(c, h.logger, &r) {
		return
	}
	grants, apps, err := h.applications.ListTenantApplications(c.Request.Context(), r.TenantID, r.ActiveOnly, r.Page, r.PageSize)
	respond(c, h.logger, TenantApplicationsResponse{Grants: GrantPageBody{Items: grantBodies(grants.Items), Total: grants.Total, Page: grants.Page, PageSize: grants.PageSize}, Applications: applicationBodies(apps)}, err)
}

// BatchCheck godoc
// @Summary Batch-check tenant application grants
// @Tags tenant-applications
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body BatchCheckRequest true "Tenant and application IDs"
// @Success 200 {object} Response{body=[]Decision}
// @Router /api/v1/applications/tenant-grants/batch-check [post]
func (h *Handler) BatchCheck(c *gin.Context) {
	var r BatchCheckRequest
	if !bind(c, h.logger, &r) {
		return
	}
	active, err := h.applications.BatchCheck(c.Request.Context(), r.TenantID, r.ApplicationIDs, r.At)
	items := make([]Decision, 0, len(r.ApplicationIDs))
	for _, id := range r.ApplicationIDs {
		granted := active[id]
		reason := "not_granted"
		if granted {
			reason = "granted"
		}
		items = append(items, Decision{ApplicationID: id, Granted: granted, Reason: reason})
	}
	respond(c, h.logger, items, err)
}
func bind(c *gin.Context, logger *slog.Logger, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		Fail(c, logger, apperror.Invalid("invalid request", err))
		return false
	}
	return true
}
func respond(c *gin.Context, logger *slog.Logger, body any, err error) {
	if err != nil {
		Fail(c, logger, err)
		return
	}
	OK(c, body)
}

func encodeJSONObject(value JSONObject) (string, error) {
	if value == nil {
		return "", nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
