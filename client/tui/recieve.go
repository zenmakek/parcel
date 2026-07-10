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
	"github.com/zenmakek/parcel/shared/hash"
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
	state     receiveState
	input     textinput.Model
	errorMsg  string
	savedPath string
}

func NewReceiveModel() ReceiveModel {
	ti := textinput.New()
	ti.Placeholder = "64-character file hash"
	ti.Focus()
	ti.Width = 66
	ti.CharLimit = 64
	return ReceiveModel{state: receiveStateInput, input: ti}
}

func (m ReceiveModel) Init() tea.Cmd { return textinput.Blink }

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
				h := strings.TrimSpace(m.input.Value())
				if err := hash.Validate(h); err != nil {
					m.errorMsg = "Invalid hash: must be 64 hex characters"
					m.state = receiveStateError
					return m, nil
				}
				m.state = receiveStateConnecting
				return m, m.startReceive(h)
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

func (m ReceiveModel) startReceive(fileHash string) tea.Cmd {
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

		homeDir, _ := os.UserHomeDir()
		downloadDir := filepath.Join(homeDir, "Downloads")
		os.MkdirAll(downloadDir, 0755)

		// fall back to relay-based transfer for now
		// direct P2P wired in Phase 29
		receivedPath, isArchive, err := transfer.ReceiveFile(conn, fileHash, downloadDir)
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

		return receiveResultMsg{path: receivedPath}
	}
}

func (m ReceiveModel) View() string {
	header := styleTitle.Render("  [ Receive ]") + "\n\n"
	footer := "\n" + styleMuted.Render("  esc to go back  •  ctrl+c to quit")

	switch m.state {
	case receiveStateInput:
		prompt := stylePrompt.Render("  Enter file hash:\n")
		input := "  " + styleInput.Render(m.input.View())
		hint := "\n" + styleMuted.Render("  64-character SHA256 hash shared by the sender")
		return header + prompt + input + hint + footer

	case receiveStateConnecting:
		return header + styleMuted.Render("  Connecting to peers...") + footer

	case receiveStateReceiving:
		return header + styleMuted.Render("  Downloading chunks...") + footer

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