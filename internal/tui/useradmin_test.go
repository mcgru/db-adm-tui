package tui

import (
	"os"
	"path/filepath"
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
	if m.user.database.Value() != "admin" {
		t.Fatalf("default database = %q", m.user.database.Value())
	}
	if m.user.hosts.Value() != "mongo1:27017" {
		t.Fatalf("default hosts = %q", m.user.hosts.Value())
	}
	choices := m.mongoUserDatabases()
	if len(choices) < 2 || choices[0] != "admin" {
		t.Fatalf("database choices = %+v", choices)
	}
	view := StripANSI(m.viewModal())
	if !strings.Contains(view, "auth.db") || !strings.Contains(view, "admin") ||
		!strings.Contains(view, "auth database: stores the MongoDB user") ||
		!strings.Contains(view, "Space cycles: admin") ||
		!strings.Contains(view, "hosts") || !strings.Contains(view, "comma-separated host:port") {
		t.Fatalf("editable auth fields missing:\n%s", view)
	}
	nameAt := strings.Index(view, "name")
	passwordAt := strings.Index(view, "password")
	authAt := strings.Index(view, "auth.db")
	hostsAt := strings.Index(view, "hosts")
	if nameAt < 0 || passwordAt < 0 || authAt < 0 || hostsAt < 0 ||
		!(nameAt < passwordAt && passwordAt < authAt && authAt < hostsAt) {
		t.Fatalf("form order is not name, password, auth.db, hosts:\n%s", view)
	}
	m = typeString(t, m, "new_mongo")
	m, _ = press(t, m, "tab")
	if m.user.focus != 2 {
		t.Fatalf("focus after name = %d, want password", m.user.focus)
	}
	m = typeString(t, m, "mongo-secret")
	m, _ = press(t, m, "tab")
	if m.user.focus != 3 {
		t.Fatalf("focus after password = %d, want database", m.user.focus)
	}
	m, _ = press(t, m, "space")
	target := choices[1]
	if m.user.database.Value() != target {
		t.Fatalf("Space selected %q, want %q", m.user.database.Value(), target)
	}
	m, _ = press(t, m, "tab")
	if m.user.focus != 4 {
		t.Fatalf("focus after database = %d, want hosts", m.user.focus)
	}
	m.user.hosts.SetValue("mongo1:27017,mongo2:27018")
	m.user.hosts.CursorEnd()
	m, cmd := press(t, m, "enter")
	m = runCmd(t, m, cmd)
	if admin.calls != 1 || admin.lastSpec.Name != "new_mongo" || admin.lastSpec.Database != target ||
		admin.lastSpec.MongoHosts != "mongo1:27017,mongo2:27018" {
		t.Fatalf("admin = %+v", admin)
	}
	if m.modal != modalUserSave {
		t.Fatalf("save modal = %v", m.modal)
	}
	if m.user.saveProfile.Database != target || m.user.saveProfile.Label != "new_mongo@"+target {
		t.Fatalf("save profile = %+v", m.user.saveProfile)
	}
	if m.user.saveProfile.Source == nil || m.user.saveProfile.Source.Kind != domain.SourceMongoFile ||
		!strings.Contains(m.user.saveProfile.Source.Path, ".mongo.new_mongo-in-"+target) {
		t.Fatalf("credential source = %+v", m.user.saveProfile.Source)
	}
	credentialPath := filepath.Join(e.home, ".mongo.new_mongo-in-"+target)
	data, err := os.ReadFile(credentialPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "mongo1:27017,mongo2:27018") {
		t.Fatalf("credential URI lost replica-set hosts:\n%s", data)
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
