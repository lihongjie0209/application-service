package grpctransport

import (
	"testing"
	"time"

	"github.com/lihongjie0209/application-service/internal/auth"
	"github.com/lihongjie0209/application-service/internal/config"
	"github.com/lihongjie0209/microservice-platform-go/principal"
	applicationv1 "github.com/lihongjie0209/platform-protos/gen/go/platform/application/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthenticateGRPCOptional_PSK(t *testing.T) {
	t.Parallel()
	const key = "01234567890123456789012345678901"
	authService := auth.New(config.Config{JWT: config.JWT{Issuer: "test", Secret: key, TTL: time.Hour}})
	cfg := config.Config{App: config.App{Name: "application-service"}, Auth: config.Auth{PSK: config.PSK{Enabled: true, Key: key}}}
	for _, test := range []struct {
		name   string
		header string
		code   codes.Code
	}{
		{name: "valid", header: "PSK " + key, code: codes.OK},
		{name: "anonymous is deferred to policy", code: codes.OK},
		{name: "bearer rejected", header: "Bearer " + key, code: codes.Unauthenticated},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs("authorization", test.header))
			authenticated, err := authenticateGRPCOptional(ctx, authService, cfg)
			if got := status.Code(err); got != test.code {
				t.Fatalf("status code = %s, want %s", got, test.code)
			}
			if test.name == "valid" {
				value, ok := principal.FromContext(authenticated)
				if !ok || value.ID != "application-service:psk" || value.Type != principal.TypeServiceAccount {
					t.Fatalf("principal = %#v, %v", value, ok)
				}
			}
		})
	}
}

func TestPlatformAdministrationMethodSeparatesManagementFromPrincipalReads(t *testing.T) {
	t.Parallel()
	for _, method := range []string{
		applicationv1.ApplicationService_CreateApplication_FullMethodName,
		applicationv1.ApplicationService_PublishMenus_FullMethodName,
		applicationv1.ApplicationService_GrantTenantApplication_FullMethodName,
		applicationv1.ApplicationService_GetTenantApplicationGrant_FullMethodName,
	} {
		if !platformAdministrationMethod(method) {
			t.Fatalf("method %q must be platform-scoped", method)
		}
	}
	for _, method := range []string{
		applicationv1.ApplicationService_GetPublishedNavigation_FullMethodName,
		applicationv1.ApplicationService_ListTenantApplications_FullMethodName,
		applicationv1.ApplicationService_BatchCheckTenantApplications_FullMethodName,
	} {
		if platformAdministrationMethod(method) {
			t.Fatalf("method %q must remain principal-scoped", method)
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
	ctx, err = authenticateGRPCOptional(ctx, service, config.Config{})
	if err != nil {
		t.Fatal(err)
	}
	value, ok := principal.FromContext(ctx)
	if !ok || value.ID != "user-1" || value.Type != principal.TypeServiceAccount {
		t.Fatalf("principal = %#v, %v", value, ok)
	}
}
