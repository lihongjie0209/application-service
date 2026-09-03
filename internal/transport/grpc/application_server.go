package grpctransport

import (
	"context"
	"time"

	applicationdomain "github.com/lihongjie0209/application-service/internal/application"
	applicationv1 "github.com/lihongjie0209/platform-protos/gen/go/platform/application/v1"
	commonv1 "github.com/lihongjie0209/platform-protos/gen/go/platform/common/v1"
)

type applicationServer struct {
	applicationv1.UnimplementedApplicationServiceServer
	service *applicationdomain.Service
}

func (s *applicationServer) CreateApplication(ctx context.Context, r *applicationv1.CreateApplicationRequest) (*applicationv1.CreateApplicationResponse, error) {
	v, e := s.service.CreateApplication(ctx, applicationdomain.ApplicationInput{Code: r.GetCode(), Name: r.GetName(), Description: r.GetDescription(), Icon: r.GetIcon(), DefaultRoute: r.GetDefaultRoute(), SortOrder: r.GetSortOrder(), MetadataJSON: r.GetMetadataJson()})
	return &applicationv1.CreateApplicationResponse{Application: applicationdomain.ToProtoApplication(v)}, e
}
func (s *applicationServer) UpdateApplication(ctx context.Context, r *applicationv1.UpdateApplicationRequest) (*applicationv1.UpdateApplicationResponse, error) {
	v, e := s.service.UpdateApplication(ctx, r.GetId(), applicationdomain.ApplicationInput{Name: r.GetName(), Description: r.GetDescription(), Icon: r.GetIcon(), DefaultRoute: r.GetDefaultRoute(), SortOrder: r.GetSortOrder(), Status: r.GetStatus(), MetadataJSON: r.GetMetadataJson()}, r.GetVersion())
	return &applicationv1.UpdateApplicationResponse{Application: applicationdomain.ToProtoApplication(v)}, e
}
func (s *applicationServer) GetApplication(ctx context.Context, r *applicationv1.GetApplicationRequest) (*applicationv1.GetApplicationResponse, error) {
	v, e := s.service.GetApplication(ctx, r.GetId())
	return &applicationv1.GetApplicationResponse{Application: applicationdomain.ToProtoApplication(v)}, e
}
func (s *applicationServer) ListApplications(ctx context.Context, r *applicationv1.ListApplicationsRequest) (*applicationv1.ListApplicationsResponse, error) {
	page, size := pageValues(r.GetPage())
	v, e := s.service.ListApplications(ctx, r.GetStatus(), page, size)
	out := make([]*applicationv1.Application, 0, len(v.Items))
	for _, x := range v.Items {
		out = append(out, applicationdomain.ToProtoApplication(x))
	}
	return &applicationv1.ListApplicationsResponse{Applications: out, Page: &commonv1.PageResult{Page: uint32(v.Page), PageSize: uint32(v.PageSize), Total: uint64(v.Total)}}, e
}
func (s *applicationServer) UpsertMenu(ctx context.Context, r *applicationv1.UpsertMenuRequest) (*applicationv1.UpsertMenuResponse, error) {
	v, e := s.service.UpsertMenu(ctx, applicationdomain.FromProtoMenu(r.GetMenu()), r.GetExpectedVersion())
	return &applicationv1.UpsertMenuResponse{Menu: applicationdomain.ToProtoMenu(v)}, e
}
func (s *applicationServer) GetMenu(ctx context.Context, r *applicationv1.GetMenuRequest) (*applicationv1.GetMenuResponse, error) {
	v, e := s.service.GetMenu(ctx, r.GetId())
	return &applicationv1.GetMenuResponse{Menu: applicationdomain.ToProtoMenu(v)}, e
}
func (s *applicationServer) DeleteMenu(ctx context.Context, r *applicationv1.DeleteMenuRequest) (*applicationv1.DeleteMenuResponse, error) {
	return &applicationv1.DeleteMenuResponse{}, s.service.DeleteMenu(ctx, r.GetId(), r.GetVersion())
}
func (s *applicationServer) ListMenuDraft(ctx context.Context, r *applicationv1.ListMenuDraftRequest) (*applicationv1.ListMenuDraftResponse, error) {
	v, e := s.service.ListMenuDraft(ctx, r.GetApplicationId())
	out := make([]*applicationv1.Menu, 0, len(v))
	for _, x := range v {
		out = append(out, applicationdomain.ToProtoMenu(x))
	}
	return &applicationv1.ListMenuDraftResponse{Menus: out}, e
}
func (s *applicationServer) PublishMenus(ctx context.Context, r *applicationv1.PublishMenusRequest) (*applicationv1.PublishMenusResponse, error) {
	rel, menus, e := s.service.PublishMenus(ctx, r.GetApplicationId(), r.GetApplicationVersion(), r.GetComment())
	out := make([]*applicationv1.Menu, 0, len(menus))
	for _, x := range menus {
		out = append(out, applicationdomain.ToProtoMenu(x))
	}
	return &applicationv1.PublishMenusResponse{Release: applicationdomain.ToProtoRelease(rel), Menus: out}, e
}
func (s *applicationServer) GetPublishedNavigation(ctx context.Context, r *applicationv1.GetPublishedNavigationRequest) (*applicationv1.GetPublishedNavigationResponse, error) {
	app, rel, menus, e := s.service.GetPublishedNavigation(ctx, r.GetApplicationId())
	out := make([]*applicationv1.Menu, 0, len(menus))
	for _, x := range menus {
		out = append(out, applicationdomain.ToProtoMenu(x))
	}
	return &applicationv1.GetPublishedNavigationResponse{Application: applicationdomain.ToProtoApplication(app), Release: applicationdomain.ToProtoRelease(rel), Menus: out}, e
}
func (s *applicationServer) GrantTenantApplication(ctx context.Context, r *applicationv1.GrantTenantApplicationRequest) (*applicationv1.GrantTenantApplicationResponse, error) {
	var from time.Time
	if r.GetValidFrom() != nil {
		from = r.GetValidFrom().AsTime()
	}
	var until *time.Time
	if r.GetValidUntil() != nil {
		v := r.GetValidUntil().AsTime()
		until = &v
	}
	grant, e := s.service.Grant(ctx, r.GetTenantId(), r.GetApplicationId(), from, until, r.GetSource(), r.GetEntitlementsJson(), r.GetExpectedVersion())
	return &applicationv1.GrantTenantApplicationResponse{Grant: applicationdomain.ToProtoGrant(grant)}, e
}
func (s *applicationServer) GetTenantApplicationGrant(ctx context.Context, r *applicationv1.GetTenantApplicationGrantRequest) (*applicationv1.GetTenantApplicationGrantResponse, error) {
	v, e := s.service.GetGrant(ctx, r.GetTenantId(), r.GetApplicationId())
	return &applicationv1.GetTenantApplicationGrantResponse{Grant: applicationdomain.ToProtoGrant(v)}, e
}
func (s *applicationServer) RevokeTenantApplication(ctx context.Context, r *applicationv1.RevokeTenantApplicationRequest) (*applicationv1.RevokeTenantApplicationResponse, error) {
	v, e := s.service.Revoke(ctx, r.GetTenantId(), r.GetApplicationId(), r.GetVersion())
	return &applicationv1.RevokeTenantApplicationResponse{Grant: applicationdomain.ToProtoGrant(v)}, e
}
func (s *applicationServer) ListTenantApplications(ctx context.Context, r *applicationv1.ListTenantApplicationsRequest) (*applicationv1.ListTenantApplicationsResponse, error) {
	page, size := pageValues(r.GetPage())
	grants, apps, e := s.service.ListTenantApplications(ctx, r.GetTenantId(), r.GetActiveOnly(), page, size)
	pg := make([]*applicationv1.TenantApplicationGrant, 0, len(grants.Items))
	for _, x := range grants.Items {
		pg = append(pg, applicationdomain.ToProtoGrant(x))
	}
	pa := make([]*applicationv1.Application, 0, len(apps))
	for _, x := range apps {
		pa = append(pa, applicationdomain.ToProtoApplication(x))
	}
	return &applicationv1.ListTenantApplicationsResponse{Grants: pg, Applications: pa, Page: &commonv1.PageResult{Page: uint32(grants.Page), PageSize: uint32(grants.PageSize), Total: uint64(grants.Total)}}, e
}
func (s *applicationServer) BatchCheckTenantApplications(ctx context.Context, r *applicationv1.BatchCheckTenantApplicationsRequest) (*applicationv1.BatchCheckTenantApplicationsResponse, error) {
	var at time.Time
	if r.GetAt() != nil {
		at = r.GetAt().AsTime()
	}
	active, e := s.service.BatchCheck(ctx, r.GetTenantId(), r.GetApplicationIds(), at)
	out := make([]*applicationv1.TenantApplicationDecision, 0, len(r.GetApplicationIds()))
	for _, id := range r.GetApplicationIds() {
		granted := active[id]
		reason := "not_granted"
		if granted {
			reason = "granted"
		}
		out = append(out, &applicationv1.TenantApplicationDecision{ApplicationId: id, Granted: granted, Reason: reason})
	}
	return &applicationv1.BatchCheckTenantApplicationsResponse{Decisions: out}, e
}
func pageValues(v *commonv1.PageRequest) (int, int) {
	if v == nil {
		return 1, 20
	}
	return int(v.GetPage()), int(v.GetPageSize())
}
