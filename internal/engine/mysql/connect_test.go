package mysql

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestBuildConfigCarriesProfileAndSecret(t *testing.T) {
	p := domain.Profile{
		Engine: domain.EngineMySQL, Host: "mysql1", Port: 3306,
		Database: "billing", User: "billing_ro",
	}
	cfg := buildConfig(p, "hunter2")
	if cfg.User != "billing_ro" || cfg.Passwd != "hunter2" {
		t.Fatalf("credentials = %q/%q", cfg.User, cfg.Passwd)
	}
	if cfg.Addr != "mysql1:3306" || cfg.DBName != "billing" || cfg.Net != "tcp" {
		t.Fatalf("target = %+v", cfg)
	}
	if cfg.Timeout != 10*time.Second || !cfg.ParseTime {
		t.Fatalf("timeouts = %v parseTime=%v", cfg.Timeout, cfg.ParseTime)
	}
}

func TestSafeErrorRedactsSecretAndMapsKinds(t *testing.T) {
	err := safeError("connect", "hunter2", errors.New("dial failed for hunter2"))
	if strings.Contains(err.Error(), "hunter2") {
		t.Fatalf("error leaked password: %v", err)
	}
	timeout := safeError("connect", "", context.DeadlineExceeded)
	if timeout.Kind != domain.KindTimeout {
		t.Fatalf("timeout kind = %v", timeout.Kind)
	}
}
