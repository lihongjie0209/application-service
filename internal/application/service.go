package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lihongjie0209/application-service/internal/apperror"
	"github.com/lihongjie0209/application-service/internal/cache"
	"github.com/lihongjie0209/application-service/internal/database"
	"github.com/lihongjie0209/application-service/internal/principal"
	platformevents "github.com/lihongjie0209/microservice-platform-go/eventbus"
	applicationv1 "github.com/lihongjie0209/platform-protos/gen/go/platform/application/v1"
	searchv1 "github.com/lihongjie0209/platform-protos/gen/go/platform/search/v1"
	"go.uber.org/fx"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ApplicationInput struct {
	Code, Name, Description, Icon, DefaultRoute, Status, MetadataJSON string
	SortOrder                                                         int32
}
type Service struct {
	repository Repository
	transactor *database.Transactor
	locker     *cache.Locker
	logger     *slog.Logger
	now        func() time.Time
}

func NewService(repository Repository, transactor *database.Transactor, locker *cache.Locker, logger *slog.Logger) *Service {
	return &Service{repository: repository, transactor: transactor, locker: locker, logger: logger, now: time.Now}
}

var codePattern = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,127}$`)

func (s *Service) CreateApplication(ctx context.Context, in ApplicationInput) (Application, error) {
	actor, err := actor(ctx)
	if err != nil {
		return Application{}, err
	}
	in, err = validateApplication(in, true)
	if err != nil {
		return Application{}, err
	}
	now := s.now()
	v := Application{ID: uuid.NewString(), Code: in.Code, Name: in.Name, Description: in.Description, Icon: in.Icon, DefaultRoute: in.DefaultRoute, SortOrder: in.SortOrder, Status: "draft", MetadataJSON: in.MetadataJSON, Version: 1, CreatedAt: now, UpdatedAt: now, CreatedBy: actor, UpdatedBy: actor}
	err = s.transactor.Within(ctx, nil, func(tx *sqlx.Tx) error {
		if err := s.repository.CreateApplication(ctx, tx, v); err != nil {
			return err
		}
		return s.addEvent(ctx, tx, "platform.application.catalog.changed.v1", "platform.application.v1.ApplicationChanged", v.ID, "application", "", actor, now, &applicationv1.ApplicationChangedEvent{Application: toProtoApplication(v), ChangeType: "created"})
	})
	return v, translate(err)
}
func (s *Service) UpdateApplication(ctx context.Context, id string, in ApplicationInput, expected int64) (Application, error) {
	if expected < 1 {
		return Application{}, apperror.Invalid("version must be positive", nil)
	}
	actor, err := actor(ctx)
	if err != nil {
		return Application{}, err
	}
	in, err = validateApplication(in, false)
	if err != nil {
		return Application{}, err
	}
	v, err := s.repository.GetApplication(ctx, id)
	if err != nil {
		return Application{}, translate(err)
	}
	v.Name, v.Description, v.Icon, v.DefaultRoute, v.SortOrder, v.Status, v.MetadataJSON = in.Name, in.Description, in.Icon, in.DefaultRoute, in.SortOrder, in.Status, in.MetadataJSON
	v.UpdatedAt, v.UpdatedBy = s.now(), actor
	err = s.transactor.Within(ctx, nil, func(tx *sqlx.Tx) error {
		if err := s.repository.UpdateApplication(ctx, tx, v, expected); err != nil {
			return err
		}
		v.Version = expected + 1
		if err := s.addEvent(ctx, tx, "platform.application.catalog.changed.v1", "platform.application.v1.ApplicationChanged", v.ID, "application", "", actor, v.UpdatedAt, &applicationv1.ApplicationChangedEvent{Application: toProtoApplication(v), ChangeType: "updated"}); err != nil {
			return err
		}
		grants, err := s.repository.ListActiveGrantsByApplication(ctx, tx, v.ID, v.UpdatedAt)
		if err != nil {
			return err
		}
		for _, grant := range grants {
			if err := s.addSearchProjectionEvent(ctx, tx, v, grant, actor, v.UpdatedAt); err != nil {
				return err
			}
		}
		return nil
	})
	return v, translate(err)
}
func (s *Service) GetApplication(ctx context.Context, id string) (Application, error) {
	v, err := s.repository.GetApplication(ctx, strings.TrimSpace(id))
	return v, translate(err)
}
func (s *Service) ListApplications(ctx context.Context, status string, page, pageSize int) (Page[Application], error) {
	page, pageSize, err := pagination(page, pageSize)
	if err != nil {
		return Page[Application]{}, err
	}
	status = strings.TrimSpace(status)
	items, total, err := s.repository.ListApplications(ctx, status, pageSize, (page-1)*pageSize)
	return Page[Application]{Items: items, Total: total, Page: page, PageSize: pageSize}, translate(err)
}
func (s *Service) UpsertMenu(ctx context.Context, v Menu, expected int64) (Menu, error) {
	actor, err := actor(ctx)
	if err != nil {
		return Menu{}, err
	}
	v, err = s.validateMenu(ctx, v, expected)
	if err != nil {
		return Menu{}, err
	}
	now := s.now()
	if expected == 0 {
		v.ID = uuid.NewString()
		v.Version = 1
		v.Status = "active"
		v.CreatedAt, v.CreatedBy = now, actor
	} else {
		current, e := s.repository.GetMenu(ctx, v.ID)
		if e != nil {
			return Menu{}, translate(e)
		}
		v.ApplicationID = current.ApplicationID
		v.CreatedAt, v.CreatedBy = current.CreatedAt, current.CreatedBy
		v.Version = expected + 1
	}
	v.UpdatedAt, v.UpdatedBy = now, actor
	err = s.transactor.Within(ctx, nil, func(tx *sqlx.Tx) error { return s.repository.UpsertMenu(ctx, tx, v, expected) })
	return v, translate(err)
}
func (s *Service) DeleteMenu(ctx context.Context, id string, expected int64) error {
	actor, err := actor(ctx)
	if err != nil {
		return err
	}
	if expected < 1 {
		return apperror.Invalid("version must be positive", nil)
	}
	menu, err := s.repository.GetMenu(ctx, id)
	if err != nil {
		return translate(err)
	}
	items, err := s.repository.ListDraftMenus(ctx, menu.ApplicationID)
	if err != nil {
		return translate(err)
	}
	for _, item := range items {
		if item.ParentID == id {
			return apperror.Conflict("menu with children cannot be deleted", nil)
		}
	}
	return translate(s.transactor.Within(ctx, nil, func(tx *sqlx.Tx) error { return s.repository.DeleteMenu(ctx, tx, id, expected, s.now(), actor) }))
}
func (s *Service) ListMenuDraft(ctx context.Context, appID string) ([]Menu, error) {
	items, err := s.repository.ListDraftMenus(ctx, strings.TrimSpace(appID))
	return items, translate(err)
}
func (s *Service) PublishMenus(ctx context.Context, appID string, appVersion int64, comment string) (MenuRelease, []Menu, error) {
	actor, err := actor(ctx)
	if err != nil {
		return MenuRelease{}, nil, err
	}
	if s.locker == nil {
		return MenuRelease{}, nil, apperror.Unavailable("menu publish lock unavailable", nil)
	}
	lock, ok, err := s.locker.TryLock(ctx, "application:menu-publish:"+appID, 30*time.Second)
	if err != nil {
		return MenuRelease{}, nil, apperror.Unavailable("acquire menu publish lock", err)
	}
	if !ok {
		return MenuRelease{}, nil, apperror.Conflict("menu publication is already running", nil)
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if e := lock.Unlock(unlockCtx); e != nil {
			s.logger.Error("release menu publish lock", "application_id", appID, "error", e)
		}
	}()
	app, err := s.repository.GetApplication(ctx, appID)
	if err != nil {
		return MenuRelease{}, nil, translate(err)
	}
	menus, err := s.repository.ListDraftMenus(ctx, appID)
	if err != nil {
		return MenuRelease{}, nil, translate(err)
	}
	if len(menus) == 0 {
		return MenuRelease{}, nil, apperror.Conflict("at least one menu is required", nil)
	}
	if err = validateMenuTree(menus); err != nil {
		return MenuRelease{}, nil, err
	}
	now := s.now()
	release := MenuRelease{ID: uuid.NewString(), ApplicationID: appID, ReleaseNumber: app.PublishedRelease + 1, Status: "published", Comment: strings.TrimSpace(comment), Version: 1, CreatedAt: now, UpdatedAt: now, CreatedBy: actor, UpdatedBy: actor}
	err = s.transactor.Within(ctx, nil, func(tx *sqlx.Tx) error {
		if err := s.repository.CreateRelease(ctx, tx, release, menus); err != nil {
			return err
		}
		if err := s.repository.SetPublishedRelease(ctx, tx, appID, release.ReleaseNumber, appVersion, now, actor); err != nil {
			return err
		}
		codes := []string{}
		for _, m := range menus {
			if m.PermissionCode != "" {
				codes = append(codes, m.PermissionCode)
			}
		}
		return s.addEvent(ctx, tx, "platform.application.menu.published.v1", "platform.application.v1.MenuPublished", release.ID, "menu_release", "", actor, now, &applicationv1.MenuPublishedEvent{Release: toProtoRelease(release), PermissionCodes: codes})
	})
	if err != nil {
		return MenuRelease{}, nil, translate(err)
	}
	for i := range menus {
		menus[i].ReleaseNumber = release.ReleaseNumber
	}
	return release, menus, nil
}
func (s *Service) GetPublishedNavigation(ctx context.Context, appID string) (Application, MenuRelease, []Menu, error) {
	app, err := s.repository.GetApplication(ctx, appID)
	if err != nil {
		return Application{}, MenuRelease{}, nil, translate(err)
	}
	if app.PublishedRelease == 0 {
		return Application{}, MenuRelease{}, nil, apperror.NotFound("published navigation not found")
	}
	rel, menus, err := s.repository.GetRelease(ctx, appID, app.PublishedRelease)
	return app, rel, menus, translate(err)
}
func (s *Service) Grant(ctx context.Context, tenantID, appID string, from time.Time, until *time.Time, source, entitlements string, expected int64) (Grant, error) {
	actor, err := actor(ctx)
	if err != nil {
		return Grant{}, err
	}
	tenantID, appID = strings.TrimSpace(tenantID), strings.TrimSpace(appID)
	if tenantID == "" || appID == "" {
		return Grant{}, apperror.Invalid("tenant_id and application_id are required", nil)
	}
	application, err := s.repository.GetApplication(ctx, appID)
	if err != nil {
		return Grant{}, translate(err)
	}
	if from.IsZero() {
		from = s.now()
	}
	if until != nil && !until.After(from) {
		return Grant{}, apperror.Invalid("valid_until must be after valid_from", nil)
	}
	if !json.Valid([]byte(defaultJSON(entitlements))) {
		return Grant{}, apperror.Invalid("entitlements_json must be valid JSON", nil)
	}
	now := s.now()
	current, getErr := s.repository.GetGrant(ctx, tenantID, appID)
	create := errors.Is(getErr, ErrNotFound)
	if getErr != nil && !create {
		return Grant{}, translate(getErr)
	}
	if create {
		if expected != 0 {
			return Grant{}, apperror.Conflict("resource version is stale", ErrStaleVersion)
		}
		current = Grant{ID: uuid.NewString(), TenantID: tenantID, ApplicationID: appID, Version: 1, CreatedAt: now, CreatedBy: actor}
	} else if expected < 1 {
		return Grant{}, apperror.Invalid("expected_version is required for an existing grant", nil)
	}
	current.Status, current.ValidFrom, current.ValidUntil, current.Source, current.EntitlementsJSON, current.UpdatedAt, current.UpdatedBy = "active", from, until, strings.TrimSpace(source), defaultJSON(entitlements), now, actor
	err = s.transactor.Within(ctx, nil, func(tx *sqlx.Tx) error {
		if create {
			if err := s.repository.CreateGrant(ctx, tx, current); err != nil {
				return err
			}
		} else {
			if err := s.repository.UpdateGrant(ctx, tx, current, expected); err != nil {
				return err
			}
			current.Version = expected + 1
		}
		if err := s.addEvent(ctx, tx, "platform.application.tenant-grant.changed.v1", "platform.application.v1.TenantApplicationGrantChanged", current.ID, "tenant_application_grant", tenantID, actor, now, &applicationv1.TenantApplicationGrantChangedEvent{Grant: toProtoGrant(current), ChangeType: "granted"}); err != nil {
			return err
		}
		return s.addSearchProjectionEvent(ctx, tx, application, current, actor, now)
	})
	return current, translate(err)
}
func (s *Service) Revoke(ctx context.Context, tenantID, appID string, expected int64) (Grant, error) {
	actor, err := actor(ctx)
	if err != nil {
		return Grant{}, err
	}
	v, err := s.repository.GetGrant(ctx, tenantID, appID)
	if err != nil {
		return Grant{}, translate(err)
	}
	application, err := s.repository.GetApplication(ctx, appID)
	if err != nil {
		return Grant{}, translate(err)
	}
	v.Status, v.UpdatedAt, v.UpdatedBy = "revoked", s.now(), actor
	err = s.transactor.Within(ctx, nil, func(tx *sqlx.Tx) error {
		if err := s.repository.UpdateGrant(ctx, tx, v, expected); err != nil {
			return err
		}
		v.Version = expected + 1
		if err := s.addEvent(ctx, tx, "platform.application.tenant-grant.changed.v1", "platform.application.v1.TenantApplicationGrantChanged", v.ID, "tenant_application_grant", tenantID, actor, v.UpdatedAt, &applicationv1.TenantApplicationGrantChangedEvent{Grant: toProtoGrant(v), ChangeType: "revoked"}); err != nil {
			return err
		}
		key := &searchv1.DocumentKey{TenantId: tenantID, SourceService: "application-service", DocumentType: "application", SourceId: application.ID, SourceVersion: searchProjectionVersion(application.Version, v.Version)}
		return s.addEvent(ctx, tx, "platform.search.document.deleted.v1", "platform.search.document.deleted.v1", application.ID, "search_document", tenantID, actor, v.UpdatedAt, &searchv1.SearchDocumentDeletedEvent{Document: key})
	})
	return v, translate(err)
}
func (s *Service) addSearchProjectionEvent(ctx context.Context, tx *sqlx.Tx, application Application, grant Grant, actor string, at time.Time) error {
	document := searchDocument(application, grant)
	return s.addEvent(ctx, tx, "platform.search.document.upserted.v1", "platform.search.document.upserted.v1", application.ID, "search_document", grant.TenantID, actor, at, &searchv1.SearchDocumentUpsertedEvent{Document: document})
}

func searchDocument(application Application, grant Grant) *searchv1.SearchDocument {
	return &searchv1.SearchDocument{
		TenantId:         grant.TenantID,
		SourceService:    "application-service",
		DocumentType:     "application",
		SourceId:         application.ID,
		ApplicationId:    application.ID,
		Title:            application.Name,
		Summary:          application.Description,
		Url:              application.DefaultRoute,
		Icon:             application.Icon,
		Keywords:         []string{application.Code, application.Name},
		VisibilityTokens: []string{"tenant:" + grant.TenantID + ":*"},
		SourceVersion:    searchProjectionVersion(application.Version, grant.Version),
		SourceCreatedAt:  timestamppb.New(application.CreatedAt),
		SourceUpdatedAt:  timestamppb.New(application.UpdatedAt),
	}
}

// searchProjectionVersion orders both application catalog and tenant grant changes.
// Database versions are positive int64 values; practical service lifetimes remain
// well below the 32-bit component boundary.
func searchProjectionVersion(applicationVersion, grantVersion int64) int64 {
	return applicationVersion<<32 | grantVersion&0xffffffff
}
func (s *Service) ListTenantApplications(ctx context.Context, tenantID string, active bool, page, pageSize int) (Page[Grant], []Application, error) {
	page, pageSize, err := pagination(page, pageSize)
	if err != nil {
		return Page[Grant]{}, nil, err
	}
	grants, apps, total, err := s.repository.ListGrants(ctx, tenantID, active, s.now(), pageSize, (page-1)*pageSize)
	return Page[Grant]{Items: grants, Total: total, Page: page, PageSize: pageSize}, apps, translate(err)
}
func (s *Service) BatchCheck(ctx context.Context, tenantID string, ids []string, at time.Time) (map[string]bool, error) {
	if at.IsZero() {
		at = s.now()
	}
	v, err := s.repository.BatchActiveGrants(ctx, tenantID, ids, at)
	return v, translate(err)
}
func (s *Service) validateMenu(ctx context.Context, v Menu, expected int64) (Menu, error) {
	v.ApplicationID, v.ID, v.ParentID, v.Code, v.Type, v.Name, v.I18nKey, v.Route, v.Component, v.ExternalURL, v.PermissionCode = strings.TrimSpace(v.ApplicationID), strings.TrimSpace(v.ID), strings.TrimSpace(v.ParentID), strings.TrimSpace(v.Code), strings.TrimSpace(v.Type), strings.TrimSpace(v.Name), strings.TrimSpace(v.I18nKey), strings.TrimSpace(v.Route), strings.TrimSpace(v.Component), strings.TrimSpace(v.ExternalURL), strings.TrimSpace(v.PermissionCode)
	if v.ApplicationID == "" || v.Code == "" || v.Name == "" || !codePattern.MatchString(v.Code) {
		return Menu{}, apperror.Invalid("application_id, valid code, and name are required", nil)
	}
	if expected > 0 && v.ID == "" {
		return Menu{}, apperror.Invalid("id is required for menu update", nil)
	}
	allowed := map[string]bool{"directory": true, "page": true, "action": true, "external": true}
	if !allowed[v.Type] {
		return Menu{}, apperror.Invalid("menu type must be directory, page, action, or external", nil)
	}
	if v.Type == "page" && v.Route == "" {
		return Menu{}, apperror.Invalid("page menu requires route", nil)
	}
	if v.Type == "external" && v.ExternalURL == "" {
		return Menu{}, apperror.Invalid("external menu requires external_url", nil)
	}
	if _, err := s.repository.GetApplication(ctx, v.ApplicationID); err != nil {
		return Menu{}, translate(err)
	}
	items, err := s.repository.ListDraftMenus(ctx, v.ApplicationID)
	if err != nil {
		return Menu{}, translate(err)
	}
	foundParent := v.ParentID == ""
	next := make([]Menu, 0, len(items)+1)
	for _, item := range items {
		if item.ID == v.ParentID {
			foundParent = true
		}
		if item.ID != v.ID {
			next = append(next, item)
		}
	}
	if !foundParent {
		return Menu{}, apperror.Invalid("parent menu does not exist in application", nil)
	}
	next = append(next, v)
	if err := validateMenuTree(next); err != nil {
		return Menu{}, err
	}
	return v, nil
}
func validateMenuTree(items []Menu) error {
	byID := map[string]Menu{}
	for _, v := range items {
		if v.ID != "" {
			byID[v.ID] = v
		}
	}
	for _, v := range items {
		seen := map[string]bool{v.ID: true}
		parent := v.ParentID
		for parent != "" {
			if seen[parent] {
				return apperror.Conflict("menu tree contains a cycle", nil)
			}
			seen[parent] = true
			p, ok := byID[parent]
			if !ok {
				return apperror.Invalid("menu parent does not exist", nil)
			}
			parent = p.ParentID
		}
	}
	return nil
}
func (s *Service) addEvent(ctx context.Context, tx *sqlx.Tx, subject, eventType, aggregateID, aggregateType, tenantID, actor string, at time.Time, payload proto.Message) error {
	envelope, err := platformevents.NewEnvelope(platformevents.Metadata{EventID: uuid.NewString(), EventType: eventType, AggregateID: aggregateID, AggregateType: aggregateType, TenantID: tenantID, SchemaVersion: 1, ActorID: actor, OccurredAt: at}, payload)
	if err != nil {
		return err
	}
	encoded, err := proto.Marshal(envelope)
	if err != nil {
		return err
	}
	return s.repository.AddOutbox(ctx, tx, OutboxEvent{ID: envelope.GetEventId(), Subject: subject, Envelope: encoded, AvailableAt: at, CreatedAt: at, UpdatedAt: at, CreatedBy: actor, UpdatedBy: actor})
}
func validateApplication(in ApplicationInput, create bool) (ApplicationInput, error) {
	in.Code = strings.ToLower(strings.TrimSpace(in.Code))
	in.Name, in.Status, in.MetadataJSON = strings.TrimSpace(in.Name), strings.TrimSpace(in.Status), defaultJSON(in.MetadataJSON)
	if create && !codePattern.MatchString(in.Code) {
		return in, apperror.Invalid("invalid application code", nil)
	}
	if in.Name == "" || !json.Valid([]byte(in.MetadataJSON)) {
		return in, apperror.Invalid("name and valid metadata_json are required", nil)
	}
	if !create {
		allowed := map[string]bool{"draft": true, "active": true, "disabled": true, "archived": true}
		if !allowed[in.Status] {
			return in, apperror.Invalid("invalid application status", nil)
		}
	}
	return in, nil
}
func defaultJSON(v string) string {
	if strings.TrimSpace(v) == "" {
		return "{}"
	}
	return strings.TrimSpace(v)
}
func actor(ctx context.Context) (string, error) {
	v, ok := principal.FromContext(ctx)
	if !ok || v.Subject == "" {
		return "", apperror.Unauthorized("authenticated actor is required")
	}
	return v.Subject, nil
}
func pagination(page, size int) (int, int, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		return 0, 0, apperror.Invalid("page_size must not exceed 100", nil)
	}
	return page, size, nil
}
func translate(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotFound) {
		return apperror.NotFound("application resource not found")
	}
	if errors.Is(err, ErrStaleVersion) {
		return apperror.Conflict("resource version is stale", err)
	}
	return apperror.Internal(fmt.Errorf("application persistence: %w", err))
}

var Module = fx.Module("application", fx.Provide(NewRepository, NewService))
