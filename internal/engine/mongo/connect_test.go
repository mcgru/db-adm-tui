package mongo

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestBuildURIIsSafeAndComplete(t *testing.T) {
	p := domain.Profile{
		Engine: domain.EngineMongoDB, Host: "mongo1", Port: 27017,
		Database: "analytics", User: "analyst",
	}
	uri := buildURI(p, "p@ss word")
	for _, want := range []string{"mongodb://", "analyst", "mongo1:27017", "/analytics"} {
		if !strings.Contains(uri, want) {
			t.Fatalf("uri missing %q: %s", want, uri)
		}
	}
	if strings.Contains(uri, "p@ss word") {
		t.Fatal("password was not URL-encoded")
	}
	parsed, err := url.Parse(uri)
	if err != nil {
		t.Fatal(err)
	}
	password, ok := parsed.User.Password()
	if !ok || password != "p@ss word" {
		t.Fatalf("decoded password = %q, %v", password, ok)
	}
}

func TestBuildURIPreservesSourceSelector(t *testing.T) {
	p := domain.Profile{
		Engine: domain.EngineMongoDB, Host: "mongo1", Port: 27017,
		Database: "analytics", User: "analyst",
		Source: &domain.Source{
			Kind:     domain.SourceMongoFile,
			Path:     "~/.mongo.analyst",
			Selector: "mongodb://analyst@mongo1:27017,mongo2:27018/analytics?authSource=admin&tls=true",
		},
	}
	uri := buildURI(p, "p@ss word")
	for _, want := range []string{"mongo2:27018", "authSource=admin", "tls=true"} {
		if !strings.Contains(uri, want) {
			t.Fatalf("uri lost %q: %s", want, uri)
		}
	}
	if strings.Contains(uri, "p@ss word") {
		t.Fatal("password was not encoded")
	}
	if !strings.Contains(uri, url.UserPassword("analyst", "p@ss word").String()) {
		t.Fatalf("selector URI lost the encoded password: %s", uri)
	}
}

func TestSafeErrorRedactsSecretAndMapsKinds(t *testing.T) {
	err := safeError("connect", "hunter2", errors.New("server selection failed for hunter2"))
	if strings.Contains(err.Error(), "hunter2") {
		t.Fatalf("error leaked password: %v", err)
	}
	timeout := safeError("connect", "", context.DeadlineExceeded)
	if timeout.Kind != domain.KindTimeout {
		t.Fatalf("timeout kind = %v", timeout.Kind)
	}
}
