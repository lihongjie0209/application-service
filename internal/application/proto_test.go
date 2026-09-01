package application

import (
	"testing"

	applicationv1 "github.com/lihongjie0209/platform-protos/gen/go/platform/application/v1"
)

func TestMenuPermissionScopeProtoRoundTrip(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		domain    string
		proto     applicationv1.MenuPermissionScope
		wantProto applicationv1.MenuPermissionScope
		want      string
	}{
		{name: "tenant", domain: "tenant", proto: applicationv1.MenuPermissionScope_MENU_PERMISSION_SCOPE_TENANT, wantProto: applicationv1.MenuPermissionScope_MENU_PERMISSION_SCOPE_TENANT, want: "tenant"},
		{name: "platform", domain: "platform", proto: applicationv1.MenuPermissionScope_MENU_PERMISSION_SCOPE_PLATFORM, wantProto: applicationv1.MenuPermissionScope_MENU_PERMISSION_SCOPE_PLATFORM, want: "platform"},
		{name: "legacy unspecified", domain: "", proto: applicationv1.MenuPermissionScope_MENU_PERMISSION_SCOPE_UNSPECIFIED, wantProto: applicationv1.MenuPermissionScope_MENU_PERMISSION_SCOPE_TENANT, want: "tenant"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			encoded := ToProtoMenu(Menu{PermissionScope: test.domain})
			if encoded.GetPermissionScope() != test.wantProto {
				t.Fatalf("encoded scope = %v", encoded.GetPermissionScope())
			}
			decoded := FromProtoMenu(&applicationv1.Menu{PermissionScope: test.proto})
			if decoded.PermissionScope != test.want {
				t.Fatalf("decoded scope = %q", decoded.PermissionScope)
			}
		})
	}
}
