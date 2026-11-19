package main

import (
	"strings"
	"testing"
)

func TestVersionText(t *testing.T) {
	oldVersion, oldCommit, oldDate := version, commit, date
	defer func() {
		version, commit, date = oldVersion, oldCommit, oldDate
	}()
	version, commit, date = "1.2.3", "abc123", "2026-11-19T00:00:00Z"
	text := versionText()
	for _, want := range []string{"db-adm-tui 1.2.3", "commit abc123", "built 2026-11-19T00:00:00Z"} {
		if !strings.Contains(text, want) {
			t.Fatalf("version text missing %q: %q", want, text)
		}
	}
}
