package application

import "time"

type Application struct {
	ID               string    `db:"id" json:"id"`
	Code             string    `db:"code" json:"code"`
	Name             string    `db:"name" json:"name"`
	Description      string    `db:"description" json:"description"`
	Icon             string    `db:"icon" json:"icon"`
	DefaultRoute     string    `db:"default_route" json:"default_route"`
	SortOrder        int32     `db:"sort_order" json:"sort_order"`
	Status           string    `db:"status" json:"status"`
	MetadataJSON     string    `db:"metadata_json" json:"metadata_json"`
	PublishedRelease int64     `db:"published_release" json:"published_release"`
	Version          int64     `db:"version" json:"version"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time `db:"updated_at" json:"updated_at"`
	CreatedBy        string    `db:"created_by" json:"created_by"`
	UpdatedBy        string    `db:"updated_by" json:"updated_by"`
}
type Menu struct {
	ID              string    `db:"id" json:"id"`
	ApplicationID   string    `db:"application_id" json:"application_id"`
	ReleaseID       string    `db:"release_id" json:"-"`
	ReleaseNumber   int64     `db:"release_number" json:"release_number"`
	ParentID        string    `db:"parent_id" json:"parent_id"`
	Code            string    `db:"menu_code" json:"code"`
	Type            string    `db:"menu_type" json:"type"`
	Name            string    `db:"name" json:"name"`
	I18nKey         string    `db:"i18n_key" json:"i18n_key"`
	Route           string    `db:"route" json:"route"`
	Component       string    `db:"component" json:"component"`
	Icon            string    `db:"icon" json:"icon"`
	ExternalURL     string    `db:"external_url" json:"external_url"`
	PermissionCode  string    `db:"permission_code" json:"permission_code"`
	PermissionScope string    `db:"permission_scope" json:"permission_scope"`
	SortOrder       int32     `db:"sort_order" json:"sort_order"`
	Visible         bool      `db:"visible" json:"visible"`
	Status          string    `db:"status" json:"status"`
	Version         int64     `db:"version" json:"version"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
	CreatedBy       string    `db:"created_by" json:"created_by"`
	UpdatedBy       string    `db:"updated_by" json:"updated_by"`
}
type MenuRelease struct {
	ID            string    `db:"id" json:"id"`
	ApplicationID string    `db:"application_id" json:"application_id"`
	ReleaseNumber int64     `db:"release_number" json:"release_number"`
	Status        string    `db:"status" json:"status"`
	Comment       string    `db:"comment" json:"comment"`
	Version       int64     `db:"version" json:"version"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
	CreatedBy     string    `db:"created_by" json:"created_by"`
	UpdatedBy     string    `db:"updated_by" json:"updated_by"`
}
type Grant struct {
	ID               string     `db:"id" json:"id"`
	TenantID         string     `db:"tenant_id" json:"tenant_id"`
	ApplicationID    string     `db:"application_id" json:"application_id"`
	Status           string     `db:"status" json:"status"`
	ValidFrom        time.Time  `db:"valid_from" json:"valid_from"`
	ValidUntil       *time.Time `db:"valid_until" json:"valid_until,omitempty"`
	Source           string     `db:"source" json:"source"`
	EntitlementsJSON string     `db:"entitlements_json" json:"entitlements_json"`
	Version          int64      `db:"version" json:"version"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updated_at"`
	CreatedBy        string     `db:"created_by" json:"created_by"`
	UpdatedBy        string     `db:"updated_by" json:"updated_by"`
}
type OutboxEvent struct {
	ID, Subject                       string
	Envelope                          []byte
	AvailableAt, CreatedAt, UpdatedAt time.Time
	CreatedBy, UpdatedBy              string
}
type Page[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}
