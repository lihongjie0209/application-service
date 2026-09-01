package bootstrap

import (
	"context"
	"time"
)

type Application struct {
	ID               string `json:"id"`
	Code             string `json:"code"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	Icon             string `json:"icon"`
	DefaultRoute     string `json:"default_route"`
	SortOrder        int32  `json:"sort_order"`
	Status           string `json:"status"`
	MetadataJSON     string `json:"metadata_json"`
	PublishedRelease int64  `json:"published_release"`
	Version          int64  `json:"version"`
}

type Menu struct {
	ID              string `json:"id"`
	ApplicationID   string `json:"application_id"`
	ParentID        string `json:"parent_id"`
	Code            string `json:"code"`
	Type            string `json:"type"`
	Name            string `json:"name"`
	I18nKey         string `json:"i18n_key"`
	Route           string `json:"route"`
	Component       string `json:"component"`
	Icon            string `json:"icon"`
	ExternalURL     string `json:"external_url"`
	PermissionCode  string `json:"permission_code"`
	PermissionScope string `json:"permission_scope"`
	Status          string `json:"status"`
	SortOrder       int32  `json:"sort_order"`
	Visible         bool   `json:"visible"`
	Version         int64  `json:"version"`
}

type Grant struct {
	TenantID      string `json:"tenant_id"`
	ApplicationID string `json:"application_id"`
	Status        string `json:"status"`
	Version       int64  `json:"version"`
}

type API interface {
	ListApplications(context.Context) ([]Application, error)
	CreateApplication(context.Context, ApplicationSpec) (Application, error)
	UpdateApplication(context.Context, Application, ApplicationSpec) (Application, error)
	ListMenus(context.Context, string) ([]Menu, error)
	UpsertMenu(context.Context, Menu, int64) (Menu, error)
	PublishMenus(context.Context, string, int64) error
	ListGrants(context.Context, string) ([]Grant, error)
	Grant(context.Context, string, string, int64, time.Time) error
}

type Result struct {
	ApplicationsCreated int `json:"applications_created"`
	ApplicationsUpdated int `json:"applications_updated"`
	MenusCreated        int `json:"menus_created"`
	MenusUpdated        int `json:"menus_updated"`
	MenusPublished      int `json:"menus_published"`
	GrantsApplied       int `json:"grants_applied"`
}
