package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	screenHome screen = iota
	screenSend
	screenReceive
)

type App struct {
	current screen
	home    HomeModel
	send    SendModel
	receive ReceiveModel
	width   int
	height  int
}

func NewApp() App {
	return App{
		current: screenHome,
		home:    NewHomeModel(),
		send:    NewSendModel(),
		receive: NewReceiveModel(),
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
	}

	return a, nil
}

func (a App) View() string {
	switch a.current {
	case screenSend:
		return a.send.View()
	case screenReceive:
		return a.receive.View()
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
