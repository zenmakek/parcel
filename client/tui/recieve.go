package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"net"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zenmakek/parcel/client/archive"
	"github.com/zenmakek/parcel/client/transfer"
)

type receiveState int

const (
	receiveStateInput receiveState = iota
	receiveStateConnecting
	receiveStateReceiving
	receiveStateDone
	receiveStateError
)

type receiveResultMsg struct {
	path      string
	isArchive bool
	err       error
}

type ReceiveModel struct {
	state         receiveState
	input         textinput.Model
	errorMsg      string
	savedPath     string
	bytesReceived int64
}

func NewReceiveModel() ReceiveModel {
	ti := textinput.New()
	ti.Placeholder = "6-digit OTP"
	ti.Focus()
	ti.Width = 20
	ti.CharLimit = 6

	return ReceiveModel{
		state: receiveStateInput,
		input: ti,
	}
}

func (m ReceiveModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m ReceiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, navigateTo(screenHome)
		case "enter":
			switch m.state {
			case receiveStateInput:
				otp := strings.TrimSpace(m.input.Value())

				if len(otp) != 6 {
					m.errorMsg = "OTP must be exactly 6 digits"
					m.state = receiveStateError
					return m, nil
				}

				for _, c := range otp {
					if c < '0' || c > '9' {
						m.errorMsg = "OTP must be numeric"
						m.state = receiveStateError
						return m, nil
					}
				}

				m.state = receiveStateConnecting
				return m, m.startReceive(otp)

			case receiveStateDone, receiveStateError:
				return m, navigateTo(screenHome)
			}
		}

	case receiveResultMsg:
		if msg.err != nil {
			m.errorMsg = msg.err.Error()
			m.state = receiveStateError
			return m, nil
		}
		m.savedPath = msg.path
		m.state = receiveStateDone
		return m, nil
	}

	var cmd tea.Cmd
	if m.state == receiveStateInput {
		m.input, cmd = m.input.Update(msg)
	}
	return m, cmd
}

func (m ReceiveModel) startReceive(otp string) tea.Cmd {
	return func() tea.Msg {
		relayAddr := os.Getenv("PARCEL_RELAY")
		if relayAddr == "" {
			relayAddr = "localhost:8080"
		}
		conn, err := net.Dial("tcp", relayAddr)
		if err != nil {
			return receiveResultMsg{err: fmt.Errorf("could not connect to relay: %w", err)}
		}
		defer conn.Close()

		homeDir, err := os.UserHomeDir()
		if err != nil {
			return receiveResultMsg{err: fmt.Errorf("could not determine home directory: %w", err)}
		}

		downloadDir := filepath.Join(homeDir, "Downloads")
		if err := os.MkdirAll(downloadDir, 0755); err != nil {
			return receiveResultMsg{err: fmt.Errorf("could not create download directory: %w", err)}
		}

		receivedPath, isArchive, err := transfer.ReceiveFile(conn, otp, downloadDir)
		if err != nil {
			return receiveResultMsg{err: err}
		}

		if isArchive {
			if err := archive.ExtractArchive(receivedPath, downloadDir); err != nil {
				return receiveResultMsg{err: fmt.Errorf("extraction failed: %w", err)}
			}
			archive.CleanupArchive(receivedPath)
			folderName := strings.TrimSuffix(filepath.Base(receivedPath), ".tar.gz")
			return receiveResultMsg{path: filepath.Join(downloadDir, folderName)}
		}

		return receiveResultMsg{path: receivedPath, isArchive: false}
	}
}

func (m ReceiveModel) View() string {
	header := styleTitle.Render("  [ Receive ]") + "\n\n"
	footer := "\n" + styleMuted.Render("  esc to go back  •  ctrl+c to quit")

	switch m.state {
	case receiveStateInput:
		prompt := stylePrompt.Render("  Enter OTP:\n")
		input := "  " + styleInput.Render(m.input.View())
		return header + prompt + input + footer

	case receiveStateConnecting:
		return header + styleMuted.Render("  Connecting to relay...") + footer

	case receiveStateReceiving:
		return header + styleMuted.Render("  Receiving file...") + footer

	case receiveStateDone:
		msg := styleSuccess.Render("  ✓ Transfer complete!") + "\n\n" +
			styleMuted.Render("  Saved to:") + "\n" +
			styleNormal.Render("  "+m.savedPath) + "\n\n" +
			styleMuted.Render("  Press Enter to go back")
		return header + msg + footer

	case receiveStateError:
		msg := styleError.Render("  ✗ Error") + "\n" +
			styleNormal.Render("  "+m.errorMsg) + "\n\n" +
			styleMuted.Render("  Press Enter to go back")
		return header + msg + footer
	}

	return header + footer
}
