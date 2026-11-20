package tui

import (
	"strings"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func selectUsersGroup(t *testing.T, m Model) Model {
	t.Helper()
	m = selectKind(t, m, domain.KindSecurity)
	m, cmd := press(t, m, "right")
	m = settle(t, m, cmd)
	m = selectKind(t, m, domain.KindUsers)
	m, cmd = press(t, m, "right")
	return settle(t, m, cmd)
}

func TestUserCreateBlockedInReadOnly(t *testing.T) {
	m := workspace(t)
	m = selectUsersGroup(t, m)
	m, _ = press(t, m, "+")
	if m.modal != modalNone {
		t.Fatalf("read-only + opened modal %v", m.modal)
	}
	if !strings.Contains(m.message, "read-write mode") {
		t.Fatalf("message = %q", m.message)
	}
}

func TestUserCreateWritesCredentialAndOffersProfile(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	admin := &fakeUserAdmin{result: domain.MutationResult{Statement: `CREATE ROLE "new_user" LOGIN PASSWORD '***'`}}
	e.connector.user = admin
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-ro"))
	m = enableReadWrite(t, m)
	m = selectUsersGroup(t, m)
	m, _ = press(t, m, "+")
	if m.modal != modalUserForm {
		t.Fatalf("form modal = %v", m.modal)
	}
	m = typeString(t, m, "new_user")
	m, _ = press(t, m, "tab")
	m = typeString(t, m, "user-secret")
	if strings.Contains(StripANSI(m.View().Content), "user-secret") {
		t.Fatal("password was rendered in clear text")
	}
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if admin.calls != 1 || admin.lastSpec.Name != "new_user" || admin.lastSpec.Password != "user-secret" {
		t.Fatalf("admin = %+v", admin)
	}
	if m.modal != modalUserSave {
		t.Fatalf("save modal = %v", m.modal)
	}
	if m.user.saveProfile.User != "new_user" || m.user.saveProfile.Database != "orders" {
		t.Fatalf("save profile = %+v", m.user.saveProfile)
	}
	m, cmd = press(t, m, "enter")
	m = runCmd(t, m, cmd)
	views, err := e.svc.Profiles(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, view := range views {
		if view.Profile.Label == "new_user@orders" {
			found = true
		}
	}
	if !found {
		t.Fatalf("saved profile missing: %+v", views)
	}
	if strings.Contains(m.message, "user-secret") {
		t.Fatal("password leaked into status")
	}
}

func TestMySQLUserCreateUsesMyPass(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	admin := &fakeUserAdmin{result: domain.MutationResult{Statement: "CREATE USER 'new_mysql'@'%' IDENTIFIED BY '***'"}}
	e.connector.user = admin
	m := workspaceFrom(t, selectLabel(t, e.model(t), "billing"))
	m = enableReadWrite(t, m)
	m = selectUsersGroup(t, m)
	m, _ = press(t, m, "+")
	if m.engine() != domain.EngineMySQL {
		t.Fatalf("engine = %v", m.engine())
	}
	m = typeString(t, m, "new_mysql")
	m, _ = press(t, m, "tab") // host
	m, _ = press(t, m, "tab") // password
	m = typeString(t, m, "mysql-secret")
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if m.modal != modalUserSave {
		t.Fatalf("save modal = %v", m.modal)
	}
	if m.user.saveProfile.User != "new_mysql" || m.user.saveProfile.Source == nil ||
		m.user.saveProfile.Source.Kind != domain.SourceMySQLPass {
		t.Fatalf("save profile = %+v", m.user.saveProfile)
	}
}

func TestMongoUserCreateCanTargetAnotherDatabase(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	admin := &fakeUserAdmin{result: domain.MutationResult{Statement: `db.getSiblingDB("admin").createUser({user:"new_mongo", pwd:"***", roles:[]})`}}
	e.connector.user = admin
	m := workspaceFrom(t, selectLabel(t, e.model(t), "analytics"))
	m = enableReadWrite(t, m)
	m = selectUsersGroup(t, m)
	m, _ = press(t, m, "+")
	if m.modal != modalUserForm || m.engine() != domain.EngineMongoDB {
		t.Fatalf("modal=%v engine=%v", m.modal, m.engine())
	}
	if m.user.database.Value() != "analytics" {
		t.Fatalf("default database = %q", m.user.database.Value())
	}
	view := StripANSI(m.viewModal())
	if !strings.Contains(view, "database") || !strings.Contains(view, "analytics") {
		t.Fatalf("editable database field missing:\n%s", view)
	}
	m = typeString(t, m, "new_mongo")
	m, _ = press(t, m, "tab")
	if m.user.focus != 3 {
		t.Fatalf("focus after name = %d, want database", m.user.focus)
	}
	m.user.database.SetValue("admin")
	m.user.database.CursorEnd()
	m, _ = press(t, m, "tab")
	m = typeString(t, m, "mongo-secret")
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if admin.calls != 1 || admin.lastSpec.Name != "new_mongo" || admin.lastSpec.Database != "admin" {
		t.Fatalf("admin = %+v", admin)
	}
	if m.modal != modalUserSave {
		t.Fatalf("save modal = %v", m.modal)
	}
	if m.user.saveProfile.Database != "admin" || m.user.saveProfile.Label != "new_mongo@admin" {
		t.Fatalf("save profile = %+v", m.user.saveProfile)
	}
	if m.user.saveProfile.Source == nil || m.user.saveProfile.Source.Kind != domain.SourceMongoFile ||
		!strings.Contains(m.user.saveProfile.Source.Path, ".mongo.new_mongo-in-admin") {
		t.Fatalf("credential source = %+v", m.user.saveProfile.Source)
	}
}

func TestUserDropConfirmation(t *testing.T) {
	e := newTestEnv(t)
	seedStandard(t, e)
	admin := &fakeUserAdmin{result: domain.MutationResult{Statement: `DROP ROLE "app_ro"`}}
	e.connector.user = admin
	profiles := sampleProfiles()
	dropAdmin := profiles[0]
	dropAdmin.Label = "prod-admin"
	dropAdmin.User = "admin"
	dropAdmin.Source = nil
	e.seed(t, append(profiles, dropAdmin)...)
	m := workspaceFrom(t, selectLabel(t, e.model(t), "prod-admin"))
	m = enableReadWrite(t, m)
	m = selectUsersGroup(t, m)
	m = selectKind(t, m, domain.KindUser)
	m, _ = press(t, m, "-")
	if m.modal != modalUserConfirm {
		t.Fatalf("confirm modal = %v", m.modal)
	}
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if admin.calls != 1 || admin.lastUser.Name == "" {
		t.Fatalf("admin calls = %d, user = %+v", admin.calls, admin.lastUser)
	}
	if m.modal != modalNone || !strings.Contains(m.message, "user removed") {
		t.Fatalf("modal=%v message=%q", m.modal, m.message)
	}
}
