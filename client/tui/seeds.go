package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zenmakek/parcel/shared/chunk"
)

type SeedsModel struct {
	seeds []*chunk.Manifest
}

func NewSeedsModel() SeedsModel {
	return SeedsModel{}
}

type seedsLoadedMsg struct {
	seeds []*chunk.Manifest
}

func (m SeedsModel) Init() tea.Cmd { return nil }

func (m SeedsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return m, navigateTo(screenHome)
		case "ctrl+c":
			return m, tea.Quit
		}
	case seedsLoadedMsg:
		m.seeds = msg.seeds
	}
	return m, nil
}

func (m SeedsModel) View() string {
	header := styleTitle.Render("  [ Seeds ]") + "\n\n"
	footer := "\n" + styleMuted.Render("  esc to go back  •  ctrl+c to quit")

	if len(m.seeds) == 0 {
		empty := styleMuted.Render("  No files being seeded yet.\n") +
			styleMuted.Render("  Files you download will be seeded automatically.")
		return header + empty + footer
	}

	content := ""
	for i, s := range m.seeds {
		line := fmt.Sprintf("  %d. %s\n", i+1, s.Filename)
		line += fmt.Sprintf("     %s  %s\n",
			styleMuted.Render(s.FileHash[:16]+"..."),
			styleMuted.Render(humanBytes(s.TotalSize)),
		)
		content += styleNormal.Render(line) + "\n"
	}

	return header + content + footer
}