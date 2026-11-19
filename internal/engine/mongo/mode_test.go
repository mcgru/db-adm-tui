package mongo

import "testing"

func TestMongoElevated(t *testing.T) {
	status := mongoConnectionStatus{}
	status.AuthInfo.AuthenticatedUserRoles = []mongoRole{{Role: "readWrite", DB: "analytics"}}
	if mongoElevated(status) {
		t.Fatal("readWrite reported elevated")
	}
	status.AuthInfo.AuthenticatedUserRoles = append(status.AuthInfo.AuthenticatedUserRoles, mongoRole{Role: "root", DB: "admin"})
	if !mongoElevated(status) {
		t.Fatal("root role not elevated")
	}
	other := mongoConnectionStatus{}
	other.AuthInfo.AuthenticatedUserPrivileges = []mongoPrivilege{{
		Actions: []string{"anyAction"},
	}}
	if !mongoElevated(other) {
		t.Fatal("anyAction not elevated")
	}
}
