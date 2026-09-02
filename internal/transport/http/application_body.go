package httptransport

import (
	"time"

	"github.com/lihongjie0209/application-service/internal/application"
)

type ApplicationBody struct {
	ID               string    `json:"id"`
	Code             string    `json:"code"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	Icon             string    `json:"icon"`
	DefaultRoute     string    `json:"default_route"`
	SortOrder        int32     `json:"sort_order"`
	Status           string    `json:"status"`
	MetadataJSON     string    `json:"metadata_json"`
	PublishedRelease int64     `json:"published_release"`
	Version          int64     `json:"version"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	CreatedBy        string    `json:"created_by"`
	UpdatedBy        string    `json:"updated_by"`
}

type MenuBody struct {
	ID              string    `json:"id"`
	ApplicationID   string    `json:"application_id"`
	ReleaseNumber   int64     `json:"release_number"`
	ParentID        string    `json:"parent_id"`
	Code            string    `json:"code"`
	Type            string    `json:"type"`
	Name            string    `json:"name"`
	I18nKey         string    `json:"i18n_key"`
	Route           string    `json:"route"`
	Component       string    `json:"component"`
	Icon            string    `json:"icon"`
	ExternalURL     string    `json:"external_url"`
	PermissionCode  string    `json:"permission_code"`
	PermissionScope string    `json:"permission_scope"`
	SortOrder       int32     `json:"sort_order"`
	Visible         bool      `json:"visible"`
	Status          string    `json:"status"`
	Version         int64     `json:"version"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	CreatedBy       string    `json:"created_by"`
	UpdatedBy       string    `json:"updated_by"`
}

type MenuInputBody struct {
	ID              string `json:"id"`
	ApplicationID   string `json:"application_id" binding:"required"`
	ParentID        string `json:"parent_id"`
	Code            string `json:"code" binding:"required"`
	Type            string `json:"type" binding:"required"`
	Name            string `json:"name" binding:"required"`
	I18nKey         string `json:"i18n_key"`
	Route           string `json:"route"`
	Component       string `json:"component"`
	Icon            string `json:"icon"`
	ExternalURL     string `json:"external_url"`
	PermissionCode  string `json:"permission_code"`
	PermissionScope string `json:"permission_scope"`
	SortOrder       int32  `json:"sort_order"`
	Visible         bool   `json:"visible"`
}

type MenuReleaseBody struct {
	ID            string    `json:"id"`
	ApplicationID string    `json:"application_id"`
	ReleaseNumber int64     `json:"release_number"`
	Status        string    `json:"status"`
	Comment       string    `json:"comment"`
	Version       int64     `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	CreatedBy     string    `json:"created_by"`
	UpdatedBy     string    `json:"updated_by"`
}

type GrantBody struct {
	ID               string     `json:"id"`
	TenantID         string     `json:"tenant_id"`
	ApplicationID    string     `json:"application_id"`
	Status           string     `json:"status"`
	ValidFrom        time.Time  `json:"valid_from"`
	ValidUntil       *time.Time `json:"valid_until,omitempty"`
	Source           string     `json:"source"`
	EntitlementsJSON string     `json:"entitlements_json"`
	Version          int64      `json:"version"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	CreatedBy        string     `json:"created_by"`
	UpdatedBy        string     `json:"updated_by"`
}

type ApplicationPageBody struct {
	Items    []ApplicationBody `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

func applicationBody(value application.Application) ApplicationBody {
	return ApplicationBody{
		ID: value.ID, Code: value.Code, Name: value.Name, Description: value.Description, Icon: value.Icon,
		DefaultRoute: value.DefaultRoute, SortOrder: value.SortOrder, Status: value.Status,
		MetadataJSON: value.MetadataJSON, PublishedRelease: value.PublishedRelease, Version: value.Version,
		CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt, CreatedBy: value.CreatedBy, UpdatedBy: value.UpdatedBy,
	}
}

func menuBody(value application.Menu) MenuBody {
	return MenuBody{
		ID: value.ID, ApplicationID: value.ApplicationID, ReleaseNumber: value.ReleaseNumber,
		ParentID: value.ParentID, Code: value.Code, Type: value.Type, Name: value.Name, I18nKey: value.I18nKey,
		Route: value.Route, Component: value.Component, Icon: value.Icon, ExternalURL: value.ExternalURL,
		PermissionCode: value.PermissionCode, PermissionScope: value.PermissionScope, SortOrder: value.SortOrder,
		Visible: value.Visible, Status: value.Status, Version: value.Version, CreatedAt: value.CreatedAt,
		UpdatedAt: value.UpdatedAt, CreatedBy: value.CreatedBy, UpdatedBy: value.UpdatedBy,
	}
}

func (value MenuInputBody) applicationMenu() application.Menu {
	return application.Menu{
		ID: value.ID, ApplicationID: value.ApplicationID, ParentID: value.ParentID, Code: value.Code,
		Type: value.Type, Name: value.Name, I18nKey: value.I18nKey, Route: value.Route, Component: value.Component,
		Icon: value.Icon, ExternalURL: value.ExternalURL, PermissionCode: value.PermissionCode,
		PermissionScope: value.PermissionScope, SortOrder: value.SortOrder, Visible: value.Visible,
	}
}

func menuReleaseBody(value application.MenuRelease) MenuReleaseBody {
	return MenuReleaseBody{
		ID: value.ID, ApplicationID: value.ApplicationID, ReleaseNumber: value.ReleaseNumber,
		Status: value.Status, Comment: value.Comment, Version: value.Version, CreatedAt: value.CreatedAt,
		UpdatedAt: value.UpdatedAt, CreatedBy: value.CreatedBy, UpdatedBy: value.UpdatedBy,
	}
}

func grantBody(value application.Grant) GrantBody {
	return GrantBody{
		ID: value.ID, TenantID: value.TenantID, ApplicationID: value.ApplicationID, Status: value.Status,
		ValidFrom: value.ValidFrom, ValidUntil: value.ValidUntil, Source: value.Source,
		EntitlementsJSON: value.EntitlementsJSON, Version: value.Version, CreatedAt: value.CreatedAt,
		UpdatedAt: value.UpdatedAt, CreatedBy: value.CreatedBy, UpdatedBy: value.UpdatedBy,
	}
}

func applicationBodies(values []application.Application) []ApplicationBody {
	result := make([]ApplicationBody, len(values))
	for index := range values {
		result[index] = applicationBody(values[index])
	}
	return result
}

func menuBodies(values []application.Menu) []MenuBody {
	result := make([]MenuBody, len(values))
	for index := range values {
		result[index] = menuBody(values[index])
	}
	return result
}

func grantBodies(values []application.Grant) []GrantBody {
	result := make([]GrantBody, len(values))
	for index := range values {
		result[index] = grantBody(values[index])
	}
	return result
}

func applicationPageBody(value application.Page[application.Application]) ApplicationPageBody {
	return ApplicationPageBody{Items: applicationBodies(value.Items), Total: value.Total, Page: value.Page, PageSize: value.PageSize}
}
