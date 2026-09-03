package application

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

var ErrNotFound = errors.New("application resource not found")
var ErrStaleVersion = errors.New("stale application resource version")

type Repository interface {
	CreateApplication(context.Context, sqlx.ExtContext, Application) error
	UpdateApplication(context.Context, sqlx.ExtContext, Application, int64) error
	GetApplication(context.Context, string) (Application, error)
	ListApplications(context.Context, string, int, int) ([]Application, int64, error)
	SearchApplications(context.Context, string, string, int, int) ([]Application, int64, error)
	GetMenu(context.Context, string) (Menu, error)
	UpsertMenu(context.Context, sqlx.ExtContext, Menu, int64) error
	DeleteMenu(context.Context, sqlx.ExtContext, string, int64, time.Time, string) error
	ListDraftMenus(context.Context, string) ([]Menu, error)
	CreateRelease(context.Context, sqlx.ExtContext, MenuRelease, []Menu) error
	GetRelease(context.Context, string, int64) (MenuRelease, []Menu, error)
	SetPublishedRelease(context.Context, sqlx.ExtContext, string, int64, int64, time.Time, string) error
	GetGrant(context.Context, string, string) (Grant, error)
	CreateGrant(context.Context, sqlx.ExtContext, Grant) error
	UpdateGrant(context.Context, sqlx.ExtContext, Grant, int64) error
	ListGrants(context.Context, string, bool, time.Time, int, int) ([]Grant, []Application, int64, error)
	ListActiveGrantsByApplication(context.Context, sqlx.ExtContext, string, time.Time) ([]Grant, error)
	BatchActiveGrants(context.Context, string, []string, time.Time) (map[string]bool, error)
	BatchGrants(context.Context, string, []string) ([]Grant, error)
	AddOutbox(context.Context, sqlx.ExtContext, OutboxEvent) error
}
type SQLRepository struct{ db *sqlx.DB }

func NewRepository(db *sqlx.DB) Repository { return &SQLRepository{db: db} }

const applicationColumns = `id,code,name,description,icon,default_route,sort_order,status,metadata_json,published_release,version,created_at,updated_at,created_by,updated_by`
const menuColumns = `id,application_id,parent_id,menu_code,menu_type,name,i18n_key,route,component,icon,external_url,permission_code,permission_scope,sort_order,visible,status,version,created_at,updated_at,created_by,updated_by`
const grantColumns = `id,tenant_id,application_id,status,valid_from,valid_until,source,entitlements_json,version,created_at,updated_at,created_by,updated_by`

func (r *SQLRepository) CreateApplication(ctx context.Context, e sqlx.ExtContext, v Application) error {
	_, err := e.ExecContext(ctx, r.db.Rebind(`INSERT INTO applications (`+applicationColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`), v.ID, v.Code, v.Name, v.Description, v.Icon, v.DefaultRoute, v.SortOrder, v.Status, v.MetadataJSON, v.PublishedRelease, v.Version, v.CreatedAt, v.UpdatedAt, v.CreatedBy, v.UpdatedBy)
	return err
}
func (r *SQLRepository) UpdateApplication(ctx context.Context, e sqlx.ExtContext, v Application, expected int64) error {
	res, err := e.ExecContext(ctx, r.db.Rebind(`UPDATE applications SET name=?,description=?,icon=?,default_route=?,sort_order=?,status=?,metadata_json=?,version=version+1,updated_at=?,updated_by=? WHERE id=? AND version=?`), v.Name, v.Description, v.Icon, v.DefaultRoute, v.SortOrder, v.Status, v.MetadataJSON, v.UpdatedAt, v.UpdatedBy, v.ID, expected)
	return stale(res, err)
}
func (r *SQLRepository) GetApplication(ctx context.Context, id string) (Application, error) {
	var v Application
	err := r.db.GetContext(ctx, &v, r.db.Rebind(`SELECT `+applicationColumns+` FROM applications WHERE id=?`), id)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return v, err
}
func (r *SQLRepository) ListApplications(ctx context.Context, status string, limit, offset int) ([]Application, int64, error) {
	where := `1=1`
	args := []any{}
	if status != "" {
		where = `status=?`
		args = append(args, status)
	}
	var total int64
	if err := r.db.GetContext(ctx, &total, r.db.Rebind(`SELECT COUNT(*) FROM applications WHERE `+where), args...); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	items := []Application{}
	err := r.db.SelectContext(ctx, &items, r.db.Rebind(`SELECT `+applicationColumns+` FROM applications WHERE `+where+` ORDER BY sort_order,id LIMIT ? OFFSET ?`), args...)
	return items, total, err
}
func (r *SQLRepository) SearchApplications(ctx context.Context, keyword, status string, limit, offset int) ([]Application, int64, error) {
	where := `1=1`
	args := []any{}
	if status != "" {
		where += ` AND status=?`
		args = append(args, status)
	}
	if keyword != "" {
		where += ` AND (LOWER(code) LIKE LOWER(?) OR LOWER(name) LIKE LOWER(?))`
		pattern := "%" + keyword + "%"
		args = append(args, pattern, pattern)
	}
	var total int64
	if err := r.db.GetContext(ctx, &total, r.db.Rebind(`SELECT COUNT(*) FROM applications WHERE `+where), args...); err != nil {
		return nil, 0, err
	}
	queryArgs := append(append([]any(nil), args...), limit, offset)
	items := []Application{}
	err := r.db.SelectContext(ctx, &items, r.db.Rebind(`SELECT `+applicationColumns+` FROM applications WHERE `+where+` ORDER BY sort_order,id LIMIT ? OFFSET ?`), queryArgs...)
	return items, total, err
}
func (r *SQLRepository) GetMenu(ctx context.Context, id string) (Menu, error) {
	var v Menu
	err := r.db.GetContext(ctx, &v, r.db.Rebind(`SELECT `+menuColumns+` FROM application_menu_drafts WHERE id=? AND status<>'deleted'`), id)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return v, err
}
func (r *SQLRepository) UpsertMenu(ctx context.Context, e sqlx.ExtContext, v Menu, expected int64) error {
	if expected == 0 {
		_, err := e.ExecContext(ctx, r.db.Rebind(`INSERT INTO application_menu_drafts (`+menuColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`), v.ID, v.ApplicationID, v.ParentID, v.Code, v.Type, v.Name, v.I18nKey, v.Route, v.Component, v.Icon, v.ExternalURL, v.PermissionCode, v.PermissionScope, v.SortOrder, v.Visible, v.Status, v.Version, v.CreatedAt, v.UpdatedAt, v.CreatedBy, v.UpdatedBy)
		return err
	}
	res, err := e.ExecContext(ctx, r.db.Rebind(`UPDATE application_menu_drafts SET parent_id=?,menu_code=?,menu_type=?,name=?,i18n_key=?,route=?,component=?,icon=?,external_url=?,permission_code=?,permission_scope=?,sort_order=?,visible=?,status=?,version=version+1,updated_at=?,updated_by=? WHERE id=? AND version=? AND status<>'deleted'`), v.ParentID, v.Code, v.Type, v.Name, v.I18nKey, v.Route, v.Component, v.Icon, v.ExternalURL, v.PermissionCode, v.PermissionScope, v.SortOrder, v.Visible, v.Status, v.UpdatedAt, v.UpdatedBy, v.ID, expected)
	return stale(res, err)
}
func (r *SQLRepository) DeleteMenu(ctx context.Context, e sqlx.ExtContext, id string, expected int64, now time.Time, actor string) error {
	res, err := e.ExecContext(ctx, r.db.Rebind(`UPDATE application_menu_drafts SET status='deleted',version=version+1,updated_at=?,updated_by=? WHERE id=? AND version=? AND status<>'deleted'`), now, actor, id, expected)
	return stale(res, err)
}
func (r *SQLRepository) ListDraftMenus(ctx context.Context, appID string) ([]Menu, error) {
	items := []Menu{}
	err := r.db.SelectContext(ctx, &items, r.db.Rebind(`SELECT `+menuColumns+` FROM application_menu_drafts WHERE application_id=? AND status<>'deleted' ORDER BY parent_id,sort_order,id`), appID)
	return items, err
}
func (r *SQLRepository) CreateRelease(ctx context.Context, e sqlx.ExtContext, v MenuRelease, menus []Menu) error {
	_, err := e.ExecContext(ctx, r.db.Rebind(`INSERT INTO application_menu_releases (id,application_id,release_number,status,comment,version,created_at,updated_at,created_by,updated_by) VALUES (?,?,?,?,?,?,?,?,?,?)`), v.ID, v.ApplicationID, v.ReleaseNumber, v.Status, v.Comment, v.Version, v.CreatedAt, v.UpdatedAt, v.CreatedBy, v.UpdatedBy)
	if err != nil {
		return err
	}
	for _, m := range menus {
		_, err = e.ExecContext(ctx, r.db.Rebind(`INSERT INTO application_menu_release_items (id,release_id,application_id,release_number,parent_id,menu_code,menu_type,name,i18n_key,route,component,icon,external_url,permission_code,permission_scope,sort_order,visible,status,version,created_at,updated_at,created_by,updated_by) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`), m.ID, v.ID, m.ApplicationID, v.ReleaseNumber, m.ParentID, m.Code, m.Type, m.Name, m.I18nKey, m.Route, m.Component, m.Icon, m.ExternalURL, m.PermissionCode, m.PermissionScope, m.SortOrder, m.Visible, m.Status, 1, v.CreatedAt, v.UpdatedAt, v.CreatedBy, v.UpdatedBy)
		if err != nil {
			return err
		}
	}
	return nil
}
func (r *SQLRepository) GetRelease(ctx context.Context, appID string, n int64) (MenuRelease, []Menu, error) {
	var rel MenuRelease
	err := r.db.GetContext(ctx, &rel, r.db.Rebind(`SELECT id,application_id,release_number,status,comment,version,created_at,updated_at,created_by,updated_by FROM application_menu_releases WHERE application_id=? AND release_number=?`), appID, n)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	if err != nil {
		return rel, nil, err
	}
	items := []Menu{}
	err = r.db.SelectContext(ctx, &items, r.db.Rebind(`SELECT id,application_id,release_number,parent_id,menu_code,menu_type,name,i18n_key,route,component,icon,external_url,permission_code,permission_scope,sort_order,visible,status,version,created_at,updated_at,created_by,updated_by FROM application_menu_release_items WHERE application_id=? AND release_number=? ORDER BY parent_id,sort_order,id`), appID, n)
	return rel, items, err
}
func (r *SQLRepository) SetPublishedRelease(ctx context.Context, e sqlx.ExtContext, id string, n, expected int64, now time.Time, actor string) error {
	res, err := e.ExecContext(ctx, r.db.Rebind(`UPDATE applications SET published_release=?,version=version+1,updated_at=?,updated_by=? WHERE id=? AND version=?`), n, now, actor, id, expected)
	return stale(res, err)
}
func (r *SQLRepository) GetGrant(ctx context.Context, tenantID, appID string) (Grant, error) {
	var v Grant
	err := r.db.GetContext(ctx, &v, r.db.Rebind(`SELECT `+grantColumns+` FROM tenant_application_grants WHERE tenant_id=? AND application_id=?`), tenantID, appID)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return v, err
}
func (r *SQLRepository) CreateGrant(ctx context.Context, e sqlx.ExtContext, v Grant) error {
	_, err := e.ExecContext(ctx, r.db.Rebind(`INSERT INTO tenant_application_grants (`+grantColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`), v.ID, v.TenantID, v.ApplicationID, v.Status, v.ValidFrom, v.ValidUntil, v.Source, v.EntitlementsJSON, v.Version, v.CreatedAt, v.UpdatedAt, v.CreatedBy, v.UpdatedBy)
	return err
}
func (r *SQLRepository) UpdateGrant(ctx context.Context, e sqlx.ExtContext, v Grant, expected int64) error {
	res, err := e.ExecContext(ctx, r.db.Rebind(`UPDATE tenant_application_grants SET status=?,valid_from=?,valid_until=?,source=?,entitlements_json=?,version=version+1,updated_at=?,updated_by=? WHERE id=? AND version=?`), v.Status, v.ValidFrom, v.ValidUntil, v.Source, v.EntitlementsJSON, v.UpdatedAt, v.UpdatedBy, v.ID, expected)
	return stale(res, err)
}
func (r *SQLRepository) ListGrants(ctx context.Context, tenantID string, active bool, at time.Time, limit, offset int) ([]Grant, []Application, int64, error) {
	where := `g.tenant_id=?`
	args := []any{tenantID}
	if active {
		where += ` AND g.status='active' AND g.valid_from<=? AND (g.valid_until IS NULL OR g.valid_until>?)`
		args = append(args, at, at)
	}
	var total int64
	if err := r.db.GetContext(ctx, &total, r.db.Rebind(`SELECT COUNT(*) FROM tenant_application_grants g WHERE `+where), args...); err != nil {
		return nil, nil, 0, err
	}
	args = append(args, limit, offset)
	grants := []Grant{}
	if err := r.db.SelectContext(ctx, &grants, r.db.Rebind(`SELECT g.`+strings.ReplaceAll(grantColumns, `,`, `,g.`)+` FROM tenant_application_grants g JOIN applications a ON a.id=g.application_id WHERE `+where+` ORDER BY a.sort_order,a.id LIMIT ? OFFSET ?`), args...); err != nil {
		return nil, nil, 0, err
	}
	if len(grants) == 0 {
		return grants, []Application{}, total, nil
	}
	ids := make([]string, 0, len(grants))
	for _, g := range grants {
		ids = append(ids, g.ApplicationID)
	}
	query, queryArgs, err := sqlx.In(`SELECT `+applicationColumns+` FROM applications WHERE id IN (?)`, ids)
	if err != nil {
		return nil, nil, 0, err
	}
	loaded := []Application{}
	if err = r.db.SelectContext(ctx, &loaded, r.db.Rebind(query), queryArgs...); err != nil {
		return nil, nil, 0, err
	}
	byID := make(map[string]Application, len(loaded))
	for _, app := range loaded {
		byID[app.ID] = app
	}
	apps := make([]Application, 0, len(grants))
	for _, grant := range grants {
		app, ok := byID[grant.ApplicationID]
		if !ok {
			return nil, nil, 0, ErrNotFound
		}
		apps = append(apps, app)
	}
	return grants, apps, total, nil
}

func (r *SQLRepository) BatchGrants(ctx context.Context, tenantID string, applicationIDs []string) ([]Grant, error) {
	query, args, err := sqlx.In(`SELECT `+grantColumns+` FROM tenant_application_grants WHERE tenant_id=? AND application_id IN (?) ORDER BY id`, tenantID, applicationIDs)
	if err != nil {
		return nil, err
	}
	items := []Grant{}
	err = r.db.SelectContext(ctx, &items, r.db.Rebind(query), args...)
	return items, err
}
func (r *SQLRepository) ListActiveGrantsByApplication(ctx context.Context, e sqlx.ExtContext, appID string, at time.Time) ([]Grant, error) {
	items := []Grant{}
	err := sqlx.SelectContext(ctx, e, &items, r.db.Rebind(`SELECT `+grantColumns+` FROM tenant_application_grants WHERE application_id=? AND status='active' AND valid_from<=? AND (valid_until IS NULL OR valid_until>?) ORDER BY tenant_id`), appID, at, at)
	return items, err
}
func (r *SQLRepository) BatchActiveGrants(ctx context.Context, tenantID string, ids []string, at time.Time) (map[string]bool, error) {
	out := map[string]bool{}
	if len(ids) == 0 {
		return out, nil
	}
	q, args, err := sqlx.In(`SELECT application_id FROM tenant_application_grants WHERE tenant_id=? AND application_id IN (?) AND status='active' AND valid_from<=? AND (valid_until IS NULL OR valid_until>?)`, tenantID, ids, at, at)
	if err != nil {
		return nil, err
	}
	var active []string
	if err = r.db.SelectContext(ctx, &active, r.db.Rebind(q), args...); err != nil {
		return nil, err
	}
	for _, id := range active {
		out[id] = true
	}
	return out, nil
}
func (r *SQLRepository) AddOutbox(ctx context.Context, e sqlx.ExtContext, v OutboxEvent) error {
	_, err := e.ExecContext(ctx, r.db.Rebind(`INSERT INTO application_outbox_events (id,subject,envelope,attempts,available_at,published_at,last_error,version,created_at,updated_at,created_by,updated_by) VALUES (?,?,?,0,?,NULL,'',1,?,?,?,?)`), v.ID, v.Subject, v.Envelope, v.AvailableAt, v.CreatedAt, v.UpdatedAt, v.CreatedBy, v.UpdatedBy)
	return err
}
func stale(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, e := res.RowsAffected()
	if e == nil && n == 0 {
		return ErrStaleVersion
	}
	return e
}
