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
	if admin.calls != 1 {
		t.Fatalf("calls = %d", admin.calls)
	}
	if m.modal != modalNone || !strings.Contains(m.message, "role change applied") {
		t.Fatalf("modal=%v message=%q", m.modal, m.message)
	}
}

func TestRoleGrantFormApplies(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	admin := &fakeRoleAdmin{result: domain.MutationResult{Statement: `GRANT CONNECT ON DATABASE "orders" TO "reporting"`}}
	e.connector.role = admin
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	m = enableReadWrite(t, m)
	m = selectRoleNode(t, m)
	m, _ = press(t, m, "+")
	if m.modal != modalRoleMenu {
		t.Fatalf("menu modal = %v", m.modal)
	}
	m, _ = press(t, m, "enter")
	if m.modal != modalRoleForm {
		t.Fatalf("form modal = %v", m.modal)
	}
	for i := 0; i < 4; i++ {
		m, _ = press(t, m, "tab")
	}
	m, _ = press(t, m, "space")
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if admin.calls != 1 {
		t.Fatalf("calls = %d", admin.calls)
	}
	if m.modal != modalNone {
		t.Fatalf("modal = %v", m.modal)
	}
}

func TestRoleAssignFormApplies(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	admin := &fakeRoleAdmin{result: domain.MutationResult{Statement: `GRANT "reporting" TO "app_ro"`}}
	e.connector.role = admin
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	m = enableReadWrite(t, m)
	m = selectRoleNode(t, m)
	m, _ = press(t, m, "+")
	m, _ = press(t, m, "down")
	m, _ = press(t, m, "enter")
	if m.modal != modalRoleForm {
		t.Fatalf("form modal = %v", m.modal)
	}
	m, _ = press(t, m, "tab")
	m = typeString(t, m, "app_ro")
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if admin.calls != 1 || m.modal != modalNone {
		t.Fatalf("calls=%d modal=%v", admin.calls, m.modal)
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
	if admin.calls != 1 || m.modal != modalNone {
		t.Fatalf("calls=%d modal=%v", admin.calls, m.modal)
	}
}
