package tui

import (
	"strings"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func selectRoleNode(t *testing.T, m Model) Model {
	t.Helper()
	m = selectKind(t, m, domain.KindSecurity)
	m, cmd := press(t, m, "right")
	m = settle(t, m, cmd)
	m = selectKind(t, m, domain.KindRoles)
	m, cmd = press(t, m, "right")
	m = settle(t, m, cmd)
	return selectKind(t, m, domain.KindRole)
}

func enableReadWrite(t *testing.T, m Model) Model {
	t.Helper()
	m, _ = press(t, m, "ctrl+x")
	m, _ = press(t, m, "ctrl+w")
	m, cmd := press(t, m, "enter")
	return runCmd(t, m, cmd)
}

func openMatrix(t *testing.T, m Model) Model {
	t.Helper()
	m, _ = press(t, m, "ctrl+x")
	m, cmd := press(t, m, "ctrl+r")
	m = runCmd(t, m, cmd)
	if m.screen != ScreenMatrix {
		t.Fatalf("screen = %v, message=%q", m.screen, m.message)
	}
	return m
}

func TestRoleActionsAreBlockedInReadOnlyMode(t *testing.T) {
	m := workspace(t)
	m = selectRoleNode(t, m)
	m, _ = press(t, m, "+")
	if m.modal != modalNone {
		t.Fatalf("read-only + opened modal %v", m.modal)
	}
	if !strings.Contains(m.message, "read-write mode") {
		t.Fatalf("message = %q", m.message)
	}
}

func TestMatrixIsBlockedInReadOnlyMode(t *testing.T) {
	m := workspace(t)
	m, _ = press(t, m, "ctrl+x")
	m, _ = press(t, m, "ctrl+r")
	if m.screen != ScreenWorkspace {
		t.Fatalf("screen = %v", m.screen)
	}
	if !strings.Contains(m.message, "read-write mode") {
		t.Fatalf("message = %q", m.message)
	}
}

func TestMatrixOpensAndSwitches(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	matrix := defaultFakeMatrix(domain.EnginePostgreSQL)
	e.connector.matrix = matrix
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	m = enableReadWrite(t, m)
	m = openMatrix(t, m)
	if !m.matrix.membershipLoaded || !m.matrix.privilegesLoaded {
		t.Fatalf("matrix not loaded: %+v", m.matrix)
	}
	if m.matrix.active != matrixMembership {
		t.Fatalf("active = %v", m.matrix.active)
	}
	view := StripANSI(m.viewMatrix())
	for _, want := range []string{"Role × Principal", "u:app_ro", "r:reporting", "[x]"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
	m, _ = press(t, m, "2")
	if m.matrix.active != matrixPrivileges {
		t.Fatalf("active = %v", m.matrix.active)
	}
	if !strings.Contains(StripANSI(m.viewMatrix()), "Principal/Role × Object") {
		t.Fatal("privilege matrix not shown")
	}
	m, _ = press(t, m, "tab")
	if m.matrix.active != matrixMembership {
		t.Fatalf("tab did not switch back: %v", m.matrix.active)
	}
}

func TestRoleMenuGrantOpensPrivilegeMatrix(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	m = enableReadWrite(t, m)
	m = selectRoleNode(t, m)
	m, _ = press(t, m, "+")
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if m.screen != ScreenMatrix || m.matrix.active != matrixPrivileges {
		t.Fatalf("screen=%v active=%v", m.screen, m.matrix.active)
	}
	if m.matrixRoleRows()[m.matrix.cursors[matrixPrivileges].row].Name != "reporting" {
		t.Fatalf("selected role = %+v", m.matrixRoleRows()[m.matrix.cursors[matrixPrivileges].row])
	}
}

func TestMongoMatrixUsesRoleRowsAndShowsBuiltinRoles(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	e.connector.matrix = defaultFakeMatrix(domain.EngineMongoDB)
	m := workspaceFrom(t, selectLabel(t, e.model(t), "analytics"))
	m = enableReadWrite(t, m)
	m = openMatrix(t, m)
	wantBuiltin := false
	for _, role := range m.matrix.membership.Roles {
		if role.Builtin {
			wantBuiltin = true
		}
	}
	if !wantBuiltin {
		t.Fatalf("built-in role missing: %+v", m.matrix.membership.Roles)
	}
	m, _ = press(t, m, "2")
	for _, principal := range m.matrix.privileges.Principals {
		if principal.Kind != domain.KindRole {
			t.Fatalf("Mongo privilege row is not a role: %+v", principal)
		}
	}
}

func TestMatrixPrincipalLabels(t *testing.T) {
	cases := []struct {
		ref  domain.RoleRef
		want string
	}{
		{domain.RoleRef{Engine: domain.EngineMongoDB, Kind: domain.KindUser, Database: "admin", Name: "root@admin"}, "u:root"},
		{domain.RoleRef{Engine: domain.EngineMongoDB, Kind: domain.KindUser, Database: "appdb", Name: "app_user@appdb"}, "u:app_user@appdb"},
		{domain.RoleRef{Engine: domain.EngineMongoDB, Kind: domain.KindRole, Database: "appdb", Name: "readWrite@appdb"}, "r:readWrite@appdb"},
		{domain.RoleRef{Engine: domain.EngineMySQL, Kind: domain.KindUser, Name: "app@%"}, "u:app"},
		{domain.RoleRef{Engine: domain.EngineMySQL, Kind: domain.KindUser, Name: "app@db1"}, "u:app@db1"},
		{domain.RoleRef{Engine: domain.EnginePostgreSQL, Kind: domain.KindUser, Name: "app_ro"}, "u:app_ro"},
	}
	for _, tc := range cases {
		if got := matrixPrincipalLabel(tc.ref); got != tc.want {
			t.Fatalf("label(%+v) = %q, want %q", tc.ref, got, tc.want)
		}
	}
}

func TestSecurityDetailsShowsMatrixHint(t *testing.T) {
	m := workspace(t)
	m = selectKind(t, m, domain.KindSecurity)
	if view := StripANSI(m.View().Content); !strings.Contains(view, accessMatrixHint) {
		t.Fatalf("security details missing matrix hint:\n%s", view)
	}
	m, cmd := press(t, m, "right")
	m = settle(t, m, cmd)
	for _, kind := range []domain.ObjectKind{domain.KindUsers, domain.KindRoles} {
		m = selectKind(t, m, kind)
		if view := StripANSI(m.View().Content); !strings.Contains(view, accessMatrixHint) {
			t.Fatalf("%s details missing matrix hint:\n%s", kind, view)
		}
	}
}

func TestMatrixShowsFullTruncatedColumnLabel(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	engine := domain.EnginePostgreSQL
	user := domain.RoleRef{Engine: engine, Name: "very_long_principal_name", Kind: domain.KindUser}
	role := domain.RoleRef{Engine: engine, Name: "reporting", Kind: domain.KindRole}
	e.connector.matrix = &fakeRoleMatrix{
		membership: domain.RoleMatrix{
			Principals: []domain.RoleRef{user},
			Roles:      []domain.RoleRef{role},
			Members:    map[string]map[string]bool{user.Key(): {role.Key(): true}},
		},
	}
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	m = enableReadWrite(t, m)
	m = openMatrix(t, m)
	view := StripANSI(m.viewMatrix())
	if !strings.Contains(view, "column u:very_long_principal_name") {
		t.Fatalf("full truncated column label missing:\n%s", view)
	}
}

func TestMatrixSpaceTogglesMembership(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	admin := &fakeRoleAdmin{result: domain.MutationResult{Statement: `REVOKE "reporting" FROM "app_ro"`}}
	e.connector.role = admin
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	m = enableReadWrite(t, m)
	m = openMatrix(t, m)
	row := m.matrix.cursors[matrixMembership].row
	col := m.matrix.cursors[matrixMembership].col
	if got := m.matrixCellText(row, col); got != "[x]" {
		t.Fatalf("initial cell = %q", got)
	}
	m, cmd := press(t, m, "space")
	if got := m.matrixCellText(row, col); got != "[ ]" {
		t.Fatalf("cell was not updated immediately: %q", got)
	}
	m = runCmd(t, m, cmd)
	if admin.calls != 1 || admin.last != "RevokeRole" {
		t.Fatalf("calls=%d last=%q", admin.calls, admin.last)
	}
	if admin.lastMember.Name != "app_ro" || admin.lastRole.Name != "reporting" {
		t.Fatalf("membership = %+v", admin.lastMember)
	}
	if !strings.Contains(m.message, "change applied") {
		t.Fatalf("message = %q", m.message)
	}
}

func TestMatrixSpaceTogglesPrivilege(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	admin := &fakeRoleAdmin{result: domain.MutationResult{Statement: `REVOKE CONNECT ON DATABASE "orders" FROM "reporting"`}}
	e.connector.role = admin
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	m = enableReadWrite(t, m)
	m = openMatrix(t, m)
	m, _ = press(t, m, "2")
	m, cmd := press(t, m, "space")
	m = runCmd(t, m, cmd)
	if admin.calls != 1 || admin.last != "RevokePrivileges" {
		t.Fatalf("calls=%d last=%q", admin.calls, admin.last)
	}
	if admin.lastGrant.Role.Name != "reporting" || admin.lastGrant.Target.Kind != domain.KindDatabase ||
		len(admin.lastGrant.Privileges) != 1 || admin.lastGrant.Privileges[0] != "CONNECT" {
		t.Fatalf("grant = %+v", admin.lastGrant)
	}
}

func TestMatrixDetailsCtrlATogglesAllPrivileges(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	m = enableReadWrite(t, m)
	m = openMatrix(t, m)
	m, _ = press(t, m, "2")
	m, _ = press(t, m, "e")
	if m.modal != modalMatrixDetails {
		t.Fatalf("modal = %v", m.modal)
	}
	if len(m.matrix.details.choices) < 2 {
		t.Fatalf("choices = %+v", m.matrix.details.choices)
	}
	m, _ = press(t, m, "ctrl+a")
	for _, choice := range m.matrix.details.choices {
		if !m.matrix.details.chosen[choice] {
			t.Fatalf("Ctrl+A did not select %q", choice)
		}
	}
	m, _ = press(t, m, "ctrl+a")
	for _, choice := range m.matrix.details.choices {
		if m.matrix.details.chosen[choice] {
			t.Fatalf("second Ctrl+A did not clear %q", choice)
		}
	}
	if view := StripANSI(m.viewModal()); !strings.Contains(view, "Ctrl+A toggle all") {
		t.Fatalf("Ctrl+A hint missing:\n%s", view)
	}
}

func TestMatrixDetailsAppliesPrivileges(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	admin := &fakeRoleAdmin{result: domain.MutationResult{Statement: `GRANT CREATE ON DATABASE "orders" TO "reporting"`}}
	e.connector.role = admin
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	m = enableReadWrite(t, m)
	m = openMatrix(t, m)
	m, _ = press(t, m, "2")
	m, _ = press(t, m, "e")
	if m.modal != modalMatrixDetails {
		t.Fatalf("modal = %v", m.modal)
	}
	m, _ = press(t, m, "down")
	m, _ = press(t, m, "space")
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if admin.calls != 1 || admin.last != "GrantPrivileges" {
		t.Fatalf("calls=%d last=%q", admin.calls, admin.last)
	}
	if len(admin.lastGrant.Privileges) != 1 || admin.lastGrant.Privileges[0] != "CREATE" {
		t.Fatalf("grant = %+v", admin.lastGrant)
	}
}

func TestMatrixFilterKeepsPerMatrixState(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	m = enableReadWrite(t, m)
	m = openMatrix(t, m)
	m, _ = press(t, m, "/")
	m = typeString(t, m, "app_role")
	m, _ = press(t, m, "enter")
	if m.matrix.cursors[matrixMembership].filter != "app_role" {
		t.Fatalf("filter = %q", m.matrix.cursors[matrixMembership].filter)
	}
	if got := len(m.matrixRoleRows()); got != 1 {
		t.Fatalf("rows = %d", got)
	}
	m, _ = press(t, m, "2")
	m, _ = press(t, m, "1")
	if m.matrix.cursors[matrixMembership].filter != "app_role" {
		t.Fatal("membership filter was not kept")
	}
	m, _ = press(t, m, "esc")
	if m.screen != ScreenWorkspace {
		t.Fatalf("screen = %v", m.screen)
	}
}

func TestRoleCreateFromRolesGroup(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	admin := &fakeRoleAdmin{result: domain.MutationResult{Statement: `CREATE ROLE "app_role"`}}
	e.connector.role = admin
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	m = enableReadWrite(t, m)
	m = selectKind(t, m, domain.KindSecurity)
	m, cmd := press(t, m, "right")
	m = settle(t, m, cmd)
	m = selectKind(t, m, domain.KindRoles)
	m, _ = press(t, m, "+")
	if m.modal != modalRoleCreate {
		t.Fatalf("modal = %v", m.modal)
	}
	m = typeString(t, m, "app_role")
	m, cmd = press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if admin.calls != 1 || admin.last != "CreateRole" {
		t.Fatalf("calls = %d last=%q", admin.calls, admin.last)
	}
	if m.modal != modalNone || !strings.Contains(m.message, "role change applied") {
		t.Fatalf("modal=%v message=%q", m.modal, m.message)
	}
}

func TestRoleDropConfirmation(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	admin := &fakeRoleAdmin{result: domain.MutationResult{Statement: `DROP ROLE "reporting"`}}
	e.connector.role = admin
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	m = enableReadWrite(t, m)
	m = selectRoleNode(t, m)
	m, _ = press(t, m, "-")
	m, _ = press(t, m, "down")
	m, _ = press(t, m, "down")
	m, _ = press(t, m, "enter")
	if m.modal != modalRoleConfirm {
		t.Fatalf("confirm modal = %v", m.modal)
	}
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if admin.calls != 1 || admin.last != "DropRole" || m.modal != modalNone {
		t.Fatalf("calls=%d last=%q modal=%v", admin.calls, admin.last, m.modal)
	}
}
