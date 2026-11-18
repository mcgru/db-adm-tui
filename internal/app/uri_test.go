package app

import (
	"strings"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestParseURI(t *testing.T) {
	cases := []struct {
		name     string
		raw      string
		label    string
		engine   domain.Engine
		host     string
		port     int
		database string
		user     string
		labelOut string
	}{
		{
			name: "postgres explicit port and label", raw: "postgres://app_ro@db1:6543/orders",
			label: "prod-ro", engine: domain.EnginePostgreSQL, host: "db1", port: 6543,
			database: "orders", user: "app_ro", labelOut: "prod-ro",
		},
		{
			name: "postgresql default port and label", raw: "postgresql://app_ro@db1/orders",
			engine: domain.EnginePostgreSQL, host: "db1", port: 5432, database: "orders",
			user: "app_ro", labelOut: "orders@db1",
		},
		{
			name: "mysql", raw: "mysql://billing_ro@mysql1:3306/billing",
			engine: domain.EngineMySQL, host: "mysql1", port: 3306, database: "billing",
			user: "billing_ro", labelOut: "billing@mysql1",
		},
		{
			name: "mongodb", raw: "mongodb://analyst@mongo1/analytics",
			engine: domain.EngineMongoDB, host: "mongo1", port: 27017, database: "analytics",
			user: "analyst", labelOut: "analytics@mongo1",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := ParseURI(tc.raw, tc.label)
			if err != nil {
				t.Fatal(err)
			}
			if p.Engine != tc.engine || p.Host != tc.host || p.Port != tc.port ||
				p.Database != tc.database || p.User != tc.user || p.Label != tc.labelOut {
				t.Fatalf("profile = %+v", p)
			}
			if p.Source != nil {
				t.Fatalf("new URI must not attach a source before matching: %+v", p.Source)
			}
		})
	}
}

func TestParseURIRejectsPasswordsInline(t *testing.T) {
	for _, raw := range []string{
		"postgres://app:secret@db1:5432/orders",
		"mysql://app:@mysql1/billing",
		"postgres://app_ro@db1/orders?password=secret",
		"postgres://app_ro@db1/orders?pwd=secret",
	} {
		_, err := ParseURI(raw, "")
		if err == nil {
			t.Fatalf("%s accepted", raw)
		}
		if err.Error() != "the URI must not contain a password" {
			t.Fatalf("err = %q", err.Error())
		}
		if strings.Contains(err.Error(), "secret") {
			t.Fatal("validation error leaked password")
		}
	}
}

func TestParseURIRejectsMissingParts(t *testing.T) {
	cases := map[string]string{
		"":                                  "the URI is not valid",
		"sqlite://u@h/db":                   "the engine is not supported",
		"postgres://u@h:bad/db":             "the port is not valid",
		"postgres://u@h":                    "the URI needs a database path",
		"postgres://h/db":                   "the URI needs a user",
		"postgres://u@h/db/extra":           "the URI is not valid",
		"postgres://u@h/":                   "the URI needs a database path",
		"postgres://u@h/db?sslmode=require": "URI options are not supported in this story",
	}
	for raw, want := range cases {
		_, err := ParseURI(raw, "")
		if err == nil || err.Error() != want {
			t.Errorf("ParseURI(%q) = %v, want %q", raw, err, want)
		}
	}
}

func TestURIErrorMessage(t *testing.T) {
	_, err := ParseURI("postgres://u:p@h/db", "")
	if got := URIErrorMessage(err); got != "Could not connect: the URI must not contain a password." {
		t.Fatalf("message = %q", got)
	}
	connectErr := &domain.Error{Kind: domain.KindAuth, Message: "Could not connect to prod-ro: authentication failed."}
	if got := URIErrorMessage(connectErr); got != connectErr.Message {
		t.Fatalf("connect message = %q", got)
	}
}
