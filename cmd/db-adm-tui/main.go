package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/mcgru/db-adm-tui/internal/app"
	"github.com/mcgru/db-adm-tui/internal/engine"
	"github.com/mcgru/db-adm-tui/internal/profile"
	"github.com/mcgru/db-adm-tui/internal/tui"
)

func main() {
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

	program := tea.NewProgram(tui.New(service))
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
