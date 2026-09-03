package grpctransport

import (
	"testing"
	"time"

	"github.com/lihongjie0209/application-service/internal/auth"
	"github.com/lihongjie0209/application-service/internal/config"
	platformauthz "github.com/lihongjie0209/microservice-platform-go/authz"
	"github.com/lihongjie0209/microservice-platform-go/principal"
	applicationv1 "github.com/lihongjie0209/platform-protos/gen/go/platform/application/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthenticateGRPC_PSKWildcard(t *testing.T) {
	t.Parallel()
	const key = "01234567890123456789012345678901"
	authService := auth.New(config.Config{JWT: config.JWT{Issuer: "test", Secret: key, TTL: time.Hour}})
	cfg := config.Auth{
		SkipGRPCMethods: []string{"/hello.v1.UserService/*"},
		PSK:             config.PSK{Enabled: true, Key: key, GRPCMethods: []string{"/hello.v1.UserService/*"}},
	}
	for _, test := range []struct {
		name   string
		header string
		code   codes.Code
	}{
		{name: "valid", header: "PSK " + key, code: codes.OK},
		{name: "PSK precedes skip", code: codes.Unauthenticated},
		{name: "bearer rejected", header: "Bearer " + key, code: codes.Unauthenticated},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs("authorization", test.header))
			authenticated, err := authenticateGRPC(ctx, "/hello.v1.UserService/GetUser", authService, cfg)
			if got := status.Code(err); got != test.code {
				t.Fatalf("status code = %s, want %s", got, test.code)
			}
			if test.code == codes.OK {
				value, ok := principal.FromContext(authenticated)
				if !ok || value.ID != "psk" || value.Type != principal.TypeSystem {
					t.Fatalf("principal = %#v, %v", value, ok)
				}
			}
		})
	}
}

func TestApplicationGRPCRequirementCoversEveryBusinessMethod(t *testing.T) {
	t.Parallel()
	resolve := applicationGRPCRequirement(true)
	methods := []string{
		applicationv1.ApplicationService_CreateApplication_FullMethodName,
		applicationv1.ApplicationService_UpdateApplication_FullMethodName,
		applicationv1.ApplicationService_GetApplication_FullMethodName,
		applicationv1.ApplicationService_ListApplications_FullMethodName,
		applicationv1.ApplicationService_UpsertMenu_FullMethodName,
		applicationv1.ApplicationService_GetMenu_FullMethodName,
		applicationv1.ApplicationService_DeleteMenu_FullMethodName,
		applicationv1.ApplicationService_ListMenuDraft_FullMethodName,
		applicationv1.ApplicationService_PublishMenus_FullMethodName,
		applicationv1.ApplicationService_GetPublishedNavigation_FullMethodName,
		applicationv1.ApplicationService_GrantTenantApplication_FullMethodName,
		applicationv1.ApplicationService_GetTenantApplicationGrant_FullMethodName,
		applicationv1.ApplicationService_RevokeTenantApplication_FullMethodName,
		applicationv1.ApplicationService_ListTenantApplications_FullMethodName,
		applicationv1.ApplicationService_BatchCheckTenantApplications_FullMethodName,
	}
	for _, method := range methods {
		if requirement, ok := resolve(method); !ok || requirement.Resource == "" || requirement.Action == "" {
			t.Fatalf("method %q requirement = %+v, %v", method, requirement, ok)
		}
	}
	if _, ok := applicationGRPCRequirement(false)(methods[0]); ok {
		t.Fatal("disabled authorization must not call the decision service")
	}
}

func TestApplicationGRPCRequirementSeparatesPlatformManagementFromPrincipalReads(t *testing.T) {
	t.Parallel()
	resolve := applicationGRPCRequirement(true)
	for _, method := range []string{
		applicationv1.ApplicationService_CreateApplication_FullMethodName,
		applicationv1.ApplicationService_PublishMenus_FullMethodName,
		applicationv1.ApplicationService_GrantTenantApplication_FullMethodName,
		applicationv1.ApplicationService_GetTenantApplicationGrant_FullMethodName,
	} {
		requirement, _ := resolve(method)
		if requirement.Scope != platformauthz.ScopePlatform {
			t.Fatalf("method %q scope = %v, want platform", method, requirement.Scope)
		}
	}
	for _, method := range []string{
		applicationv1.ApplicationService_GetPublishedNavigation_FullMethodName,
		applicationv1.ApplicationService_ListTenantApplications_FullMethodName,
		applicationv1.ApplicationService_BatchCheckTenantApplications_FullMethodName,
	} {
		requirement, _ := resolve(method)
		if requirement.Scope != platformauthz.ScopePrincipal {
			t.Fatalf("method %q scope = %v, want principal-derived", method, requirement.Scope)
		}
	}
}

func TestAuthenticateGRPC_JWTInjectsPrincipal(t *testing.T) {
	t.Parallel()
	const key = "01234567890123456789012345678901"
	service := auth.New(config.Config{JWT: config.JWT{Issuer: "test", Secret: key, TTL: time.Hour}, Auth: config.Auth{ClientID: "client", ClientSecret: "secret"}})
	token, err := service.Issue("user-1")
	if err != nil {
		t.Fatal(err)
	}
	ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs("authorization", "Bearer "+token))
	ctx, err = authenticateGRPC(ctx, "/hello.v1.UserService/GetUser", service, config.Auth{})
	if err != nil {
		t.Fatal(err)
	}
	value, ok := principal.FromContext(ctx)
	if !ok || value.ID != "user-1" || value.Type != principal.TypeServiceAccount {
		t.Fatalf("principal = %#v, %v", value, ok)
	}
}
