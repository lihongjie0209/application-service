package application

import (
	applicationv1 "github.com/lihongjie0209/platform-protos/gen/go/platform/application/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoApplication(v Application) *applicationv1.Application {
	return &applicationv1.Application{Id: v.ID, Code: v.Code, Name: v.Name, Description: v.Description, Icon: v.Icon, DefaultRoute: v.DefaultRoute, SortOrder: v.SortOrder, Status: v.Status, MetadataJson: v.MetadataJSON, PublishedRelease: v.PublishedRelease, Version: v.Version, CreatedAt: timestamppb.New(v.CreatedAt), UpdatedAt: timestamppb.New(v.UpdatedAt), CreatedBy: v.CreatedBy, UpdatedBy: v.UpdatedBy}
}
func ToProtoApplication(v Application) *applicationv1.Application { return toProtoApplication(v) }
func toProtoMenu(v Menu) *applicationv1.Menu {
	return &applicationv1.Menu{Id: v.ID, ApplicationId: v.ApplicationID, ReleaseNumber: v.ReleaseNumber, ParentId: v.ParentID, Code: v.Code, Type: v.Type, Name: v.Name, I18NKey: v.I18nKey, Route: v.Route, Component: v.Component, Icon: v.Icon, ExternalUrl: v.ExternalURL, PermissionCode: v.PermissionCode, SortOrder: v.SortOrder, Visible: v.Visible, Status: v.Status, Version: v.Version, CreatedAt: timestamppb.New(v.CreatedAt), UpdatedAt: timestamppb.New(v.UpdatedAt), CreatedBy: v.CreatedBy, UpdatedBy: v.UpdatedBy}
}
func ToProtoMenu(v Menu) *applicationv1.Menu { return toProtoMenu(v) }
func toProtoRelease(v MenuRelease) *applicationv1.MenuRelease {
	return &applicationv1.MenuRelease{Id: v.ID, ApplicationId: v.ApplicationID, ReleaseNumber: v.ReleaseNumber, Status: v.Status, Comment: v.Comment, Version: v.Version, CreatedAt: timestamppb.New(v.CreatedAt), UpdatedAt: timestamppb.New(v.UpdatedAt), CreatedBy: v.CreatedBy, UpdatedBy: v.UpdatedBy}
}
func ToProtoRelease(v MenuRelease) *applicationv1.MenuRelease { return toProtoRelease(v) }
func toProtoGrant(v Grant) *applicationv1.TenantApplicationGrant {
	p := &applicationv1.TenantApplicationGrant{Id: v.ID, TenantId: v.TenantID, ApplicationId: v.ApplicationID, Status: v.Status, ValidFrom: timestamppb.New(v.ValidFrom), Source: v.Source, EntitlementsJson: v.EntitlementsJSON, Version: v.Version, CreatedAt: timestamppb.New(v.CreatedAt), UpdatedAt: timestamppb.New(v.UpdatedAt), CreatedBy: v.CreatedBy, UpdatedBy: v.UpdatedBy}
	if v.ValidUntil != nil {
		p.ValidUntil = timestamppb.New(*v.ValidUntil)
	}
	return p
}
func ToProtoGrant(v Grant) *applicationv1.TenantApplicationGrant { return toProtoGrant(v) }
func FromProtoMenu(v *applicationv1.Menu) Menu {
	if v == nil {
		return Menu{}
	}
	return Menu{ID: v.GetId(), ApplicationID: v.GetApplicationId(), ParentID: v.GetParentId(), Code: v.GetCode(), Type: v.GetType(), Name: v.GetName(), I18nKey: v.GetI18NKey(), Route: v.GetRoute(), Component: v.GetComponent(), Icon: v.GetIcon(), ExternalURL: v.GetExternalUrl(), PermissionCode: v.GetPermissionCode(), SortOrder: v.GetSortOrder(), Visible: v.GetVisible(), Status: v.GetStatus()}
}
