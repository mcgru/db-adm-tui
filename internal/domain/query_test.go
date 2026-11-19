package domain

import (
	"strings"
	"testing"
)

func TestRedactSensitive(t *testing.T) {
	cases := map[string]string{
		"ALTER USER app PASSWORD 'hunter2'":          "ALTER USER app PASSWORD '***'",
		"CREATE USER x IDENTIFIED BY 'hunter2'":      "CREATE USER x IDENTIFIED BY '***'",
		`{"pwd": "hunter2", "user": "app"}`:          `{"pwd": "***", "user": "app"}`,
		"postgres://app:hunter2@db1/orders":          "postgres://app:***@db1/orders",
		"password=hunter2&sslmode=require":           "password=***&sslmode=require",
		`db.createUser({user: "a", pwd: "hunter2"})`: `db.createUser({user: "a", pwd: "***"})`,
	}
	for input, want := range cases {
		if got := RedactSensitive(input); got != want {
			t.Fatalf("RedactSensitive(%q) = %q, want %q", input, got, want)
		}
	}
	if got := RedactSecrets("boom hunter2 and PASSWORD 'x'", "hunter2"); strings.Contains(got, "hunter2") || strings.Contains(got, "'x'") {
		t.Fatalf("RedactSecrets = %q", got)
	}
}

func TestUniqueColumnNames(t *testing.T) {
	got := UniqueColumnNames([]string{"id", "name", "id", "id"})
	want := []string{"id", "name", "id (2)", "id (3)"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}
