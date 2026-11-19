package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/mcgru/db-adm-tui/internal/app"
	"github.com/mcgru/db-adm-tui/internal/engine"
	"github.com/mcgru/db-adm-tui/internal/history"
	"github.com/mcgru/db-adm-tui/internal/profile"
	"github.com/mcgru/db-adm-tui/internal/tui"
)

// These values are replaced through -ldflags by make build/release.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-version") {
		fmt.Print(versionText())
		return
	}
	store, err := profile.NewDefaultStore()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not determine the profile directory:", err)
		os.Exit(1)
	}
	resolver, err := profile.NewResolver()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not determine the credential source paths:", err)
		os.Exit(1)
	}
	connector := engine.NewConnector()
	defer connector.Close()
	service := app.NewService(store, resolver, connector)
	if historyStore, err := history.NewDefaultStore(); err != nil {
		fmt.Fprintln(os.Stderr, "Could not determine the history directory:", err)
		os.Exit(1)
	} else {
		service.SetHistory(historyStore)
	}

	program := tea.NewProgram(tui.New(service))
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func versionText() string {
	return fmt.Sprintf("db-adm-tui %s\ncommit %s\nbuilt %s\n", version, commit, date)
}
