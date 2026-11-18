package engine

import (
	"context"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestConnectorRejectsUnknownEngine(t *testing.T) {
	_, err := NewConnector().Connect(context.Background(), domain.Profile{
		Engine: domain.Engine("sqlite"), Host: "h", Port: 1,
	}, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if domain.KindOf(err) != domain.KindConfig {
		t.Fatalf("kind = %v", domain.KindOf(err))
	}
}
