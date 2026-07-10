package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zenmakek/parcel/client/identity"
	"github.com/zenmakek/parcel/client/store"
	"github.com/zenmakek/parcel/client/tui"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println("Parcel", version)
		os.Exit(0)
	}

	if os.Getenv("PARCEL_RELAY") == "" {
		os.Setenv("PARCEL_RELAY", "139.59.60.82:8080")
	}

	id, err := identity.New()
	if err != nil {
		fmt.Println("failed to load identity:", err)
		os.Exit(1)
	}

	s, err := store.New()
	if err != nil {
		fmt.Println("failed to open store:", err)
		os.Exit(1)
	}

	storeSize, _ := s.StoreSize()

	app := tui.NewApp(id.PeerID, storeSize)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}
