package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type HomeModel struct {
	cursor int
}

type menuItem struct {
	label       string
	description string
}

var menuItems = []menuItem{
	{"Send", "Transfer files or folders"},
	{"Receive", "Enter an OTP to receive"},
	{"Quit", "Exit Parcel"},
}

func NewHomeModel() HomeModel {
	return HomeModel{cursor: 0}
}

func (m HomeModel) Init() tea.Cmd {
	return nil
}

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

	subtitle := styleSubtitle.Render("  No accounts. No links. Just a code.") + "\n\n"

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

	return banner + "\n" + subtitle + menu + help
}
