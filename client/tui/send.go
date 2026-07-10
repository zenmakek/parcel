package tui

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zenmakek/parcel/client/archive"
	"github.com/zenmakek/parcel/client/transfer"
	"github.com/zenmakek/parcel/shared/hash"
)

type sendState int

const (
	sendStateInput sendState = iota
	sendStateConfirm
	sendStateHashing
	sendStateWaitingReceiver
	sendStateTransferring
	sendStateDone
	sendStateError
)

type SendModel struct {
	state    sendState
	input    textinput.Model
	meta     *transfer.Metadata
	fileHash string
	errorMsg string
	bytesSent int64
	confirm  string
}

type hashDoneMsg struct {
	fileHash string
	err      error
}

type sendDoneMsg struct {
	bytes int64
	err   error
}

func NewSendModel() SendModel {
	ti := textinput.New()
	ti.Placeholder = "/path/to/file or folder"
	ti.Focus()
	ti.Width = 40
	ti.CharLimit = 256
	return SendModel{state: sendStateInput, input: ti}
}

func (m SendModel) Init() tea.Cmd { return textinput.Blink }

func (m SendModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, navigateTo(screenHome)
		case "enter":
			switch m.state {
			case sendStateInput:
				path := strings.TrimSpace(m.input.Value())
				if err := transfer.ValidatePath(path); err != nil {
					m.errorMsg = err.Error()
					m.state = sendStateError
					return m, nil
				}
				meta, err := transfer.Inspect(path)
				if err != nil {
					m.errorMsg = err.Error()
					m.state = sendStateError
					return m, nil
				}
				m.meta = meta
				m.state = sendStateConfirm
				return m, nil

			case sendStateConfirm:
				if m.confirm == "y" || m.confirm == "Y" {
					m.state = sendStateHashing
					return m, m.hashFile()
				}
				return m, navigateTo(screenHome)

			case sendStateError, sendStateDone:
				return m, navigateTo(screenHome)
			}

		default:
			if m.state == sendStateConfirm {
				if msg.String() == "backspace" && len(m.confirm) > 0 {
					m.confirm = m.confirm[:len(m.confirm)-1]
				} else if len(msg.String()) == 1 {
					m.confirm = msg.String()
				}
				return m, nil
			}
		}

	case hashDoneMsg:
		if msg.err != nil {
			m.errorMsg = msg.err.Error()
			m.state = sendStateError
			return m, nil
		}
		m.fileHash = msg.fileHash
		m.state = sendStateWaitingReceiver
		return m, m.startSeed()

	case sendDoneMsg:
		if msg.err != nil {
			m.errorMsg = msg.err.Error()
			m.state = sendStateError
			return m, nil
		}
		m.bytesSent = msg.bytes
		m.state = sendStateDone
		return m, nil
	}

	var cmd tea.Cmd
	if m.state == sendStateInput {
		m.input, cmd = m.input.Update(msg)
	}
	return m, cmd
}

func (m SendModel) hashFile() tea.Cmd {
	return func() tea.Msg {
		path := m.meta.OriginalPath

		if m.meta.IsArchive {
			archivePath := archive.ArchivePath(path)
			if err := archive.CompressFolder(path, archivePath); err != nil {
				return hashDoneMsg{err: err}
			}
			path = archivePath
		}

		h, err := hash.HashFile(path)
		if err != nil {
			return hashDoneMsg{err: err}
		}
		return hashDoneMsg{fileHash: h}
	}
}

func (m SendModel) startSeed() tea.Cmd {
	return func() tea.Msg {
		relayAddr := os.Getenv("PARCEL_RELAY")
		if relayAddr == "" {
			relayAddr = "localhost:8080"
		}

		conn, err := net.Dial("tcp", relayAddr)
		if err != nil {
			return sendDoneMsg{err: fmt.Errorf("could not connect to relay: %w", err)}
		}
		defer conn.Close()

		meta := m.meta
		if meta.IsArchive {
			archivePath := archive.ArchivePath(meta.OriginalPath)
			newMeta, err := transfer.Inspect(archivePath)
			if err != nil {
				return sendDoneMsg{err: err}
			}
			newMeta.IsArchive = true
			meta = newMeta
		}

		if err := transfer.SendFile(conn, meta); err != nil {
			return sendDoneMsg{err: err}
		}

		return sendDoneMsg{bytes: meta.Size}
	}
}

func (m SendModel) View() string {
	header := styleTitle.Render("  [ Send ]") + "\n\n"
	footer := "\n" + styleMuted.Render("  esc to go back  •  ctrl+c to quit")

	switch m.state {
	case sendStateInput:
		prompt := stylePrompt.Render("  File or folder path:\n")
		input := "  " + styleInput.Render(m.input.View())
		return header + prompt + input + footer

	case sendStateConfirm:
		fileType := "File"
		if m.meta.IsArchive {
			fileType = "Folder → will archive as .tar.gz"
		}
		info := styleBox.Render(
			styleMuted.Render("Name : ") + styleNormal.Render(m.meta.Filename) + "\n" +
				styleMuted.Render("Size : ") + styleNormal.Render(m.meta.HumanSize()) + "\n" +
				styleMuted.Render("Type : ") + styleNormal.Render(fileType),
		)
		prompt := stylePrompt.Render("  Send this? [y/n]: ") + m.confirm
		return header + info + "\n" + prompt + footer

	case sendStateHashing:
		return header + styleMuted.Render("  Computing file hash...") + footer

	case sendStateWaitingReceiver:
		hashBox := styleBox.Render(
			styleMuted.Render("Share this hash\n\n") +
				styleOTP.Render(m.fileHash[:32]+"\n"+m.fileHash[32:]) + "\n\n" +
				styleMuted.Render("Receiver enters this to download"),
		)
		return header + hashBox + "\n" + styleMuted.Render("  Seeding...") + footer

	case sendStateTransferring:
		return header + styleMuted.Render("  Transferring...") + footer

	case sendStateDone:
		msg := styleSuccess.Render("  ✓ Transfer complete!") + "\n" +
			styleMuted.Render(fmt.Sprintf("  %d bytes sent", m.bytesSent)) + "\n\n" +
			styleMuted.Render("  Press Enter to go back")
		return header + msg + footer

	case sendStateError:
		msg := styleError.Render("  ✗ Error") + "\n" +
			styleNormal.Render("  "+m.errorMsg) + "\n\n" +
			styleMuted.Render("  Press Enter to go back")
		return header + msg + footer
	}

	return header + footer
}