package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type HomeModel struct {
	cursor    int
	peerID    string
	storeSize int64
}

type menuItem struct {
	label       string
	description string
}

var menuItems = []menuItem{
	{"Send", "Share a file or folder"},
	{"Receive", "Enter a hash to download"},
	{"Seeds", "View seeding status"},
	{"Quit", "Exit Parcel"},
}

func NewHomeModel(peerID string, storeSize int64) HomeModel {
	return HomeModel{cursor: 0, peerID: peerID, storeSize: storeSize}
}

func (m HomeModel) Init() tea.Cmd { return nil }

func (m HomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(menuItems)-1 {
				m.cursor++
			}
		case "enter", " ":
			switch m.cursor {
			case 0:
				return m, navigateTo(screenSend)
			case 1:
				return m, navigateTo(screenReceive)
			case 2:
				return m, navigateTo(screenSeeds)
			case 3:
				return m, tea.Quit
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m HomeModel) View() string {
	banner := styleTitle.Render(`
  ██████╗  █████╗ ██████╗  ██████╗███████╗██╗     
  ██╔══██╗██╔══██╗██╔══██╗██╔════╝██╔════╝██║     
  ██████╔╝███████║██████╔╝██║     █████╗  ██║     
  ██╔═══╝ ██╔══██║██╔══██╗██║     ██╔══╝  ██║     
  ██║     ██║  ██║██║  ██║╚██████╗███████╗███████╗
  ╚═╝     ╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝╚══════╝╚══════╝`)

	subtitle := styleSubtitle.Render("  No accounts. No links. Just a code.") + "\n"

	peerLine := styleMuted.Render(fmt.Sprintf("  Peer ID: %s", shortID(m.peerID)))
	storeLine := styleMuted.Render(fmt.Sprintf("  Store:   %s", humanBytes(m.storeSize)))
	info := peerLine + "\n" + storeLine + "\n\n"

	menu := ""
	for i, item := range menuItems {
		cursor := "  "
		label := styleNormal.Render(item.label)
		desc := styleMuted.Render("  " + item.description)
		if i == m.cursor {
			cursor = styleSelected.Render("▶ ")
			label = styleSelected.Render(item.label)
		}
		menu += fmt.Sprintf("%s%s\n%s\n\n", cursor, label, desc)
	}

	help := styleMuted.Render("  ↑/↓ navigate  •  enter select  •  q quit")
	return banner + "\n" + subtitle + "\n" + info + menu + help
}

func shortID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12] + "..."
}

func humanBytes(b int64) string {
	const MB = 1024 * 1024
	const KB = 1024
	switch {
	case b >= MB:
		return fmt.Sprintf("%.1f MB", float64(b)/MB)
	case b >= KB:
		return fmt.Sprintf("%.1f KB", float64(b)/KB)
	default:
		return fmt.Sprintf("%d B", b)
	}
}