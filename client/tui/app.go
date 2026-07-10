package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	screenHome screen = iota
	screenSend
	screenReceive
	screenSeeds
)

type App struct {
	current screen
	home    HomeModel
	send    SendModel
	receive ReceiveModel
	seeds   SeedsModel
	width   int
	height  int
}

func NewApp(peerID string, storeSize int64) App {
	return App{
		current: screenHome,
		home:    NewHomeModel(peerID, storeSize),
		send:    NewSendModel(),
		receive: NewReceiveModel(),
		seeds:   NewSeedsModel(),
	}
}

func (a App) Init() tea.Cmd {
	return nil
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case navigateMsg:
		switch msg.to {
		case screenHome:
			a.current = screenHome
			a.send = NewSendModel()
			a.receive = NewReceiveModel()
		case screenSend:
			a.current = screenSend
		case screenReceive:
			a.current = screenReceive
		case screenSeeds:
			a.current = screenSeeds
		}
		return a, nil
	}

	switch a.current {
	case screenHome:
		updated, cmd := a.home.Update(msg)
		a.home = updated.(HomeModel)
		return a, cmd
	case screenSend:
		updated, cmd := a.send.Update(msg)
		a.send = updated.(SendModel)
		return a, cmd
	case screenReceive:
		updated, cmd := a.receive.Update(msg)
		a.receive = updated.(ReceiveModel)
		return a, cmd
	case screenSeeds:
		updated, cmd := a.seeds.Update(msg)
		a.seeds = updated.(SeedsModel)
		return a, cmd
	}

	return a, nil
}

func (a App) View() string {
	switch a.current {
	case screenSend:
		return a.send.View()
	case screenReceive:
		return a.receive.View()
	case screenSeeds:
		return a.seeds.View()
	default:
		return a.home.View()
	}
}

type navigateMsg struct {
	to screen
}

func navigateTo(s screen) tea.Cmd {
	return func() tea.Msg {
		return navigateMsg{to: s}
	}
}