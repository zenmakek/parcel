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
)

type sendState int

const (
	sendStateInput sendState = iota
	sendStateConfirm
	sendStateConnecting
	sendStateWaitingOTP
	sendStateWaitingReceiver
	sendStateTransferring
	sendStateDone
	sendStateError
)

type SendModel struct {
	state     sendState
	input     textinput.Model
	meta      *transfer.Metadata
	otp       string
	errorMsg  string
	progress  float64
	bytesSent int64
	confirm   string
}

type sendResultMsg struct {
	err error
}

type otpReceivedMsg struct {
	otp string
}

type transferDoneMsg struct {
	bytes int64
}

func NewSendModel() SendModel {
	ti := textinput.New()
	ti.Placeholder = "/path/to/file or folder"
	ti.Focus()
	ti.Width = 40
	ti.CharLimit = 256

	return SendModel{
		state: sendStateInput,
		input: ti,
	}
}

func (m SendModel) Init() tea.Cmd {
	return textinput.Blink
}

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
					m.state = sendStateConnecting
					return m, m.startSend()
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

	case otpReceivedMsg:
		m.otp = msg.otp
		m.state = sendStateWaitingReceiver
		return m, nil

	case transferDoneMsg:
		m.bytesSent = msg.bytes
		m.state = sendStateDone
		return m, nil

	case sendResultMsg:
		if msg.err != nil {
			m.errorMsg = msg.err.Error()
			m.state = sendStateError
		}
		return m, nil
	}

	var cmd tea.Cmd
	if m.state == sendStateInput {
		m.input, cmd = m.input.Update(msg)
	}
	return m, cmd
}

func (m SendModel) startSend() tea.Cmd {
	return func() tea.Msg {
		meta := m.meta

		relayAddr := os.Getenv("PARCEL_RELAY")
		if relayAddr == "" {
			relayAddr = "localhost:8080"
		}
		conn, err := net.Dial("tcp", relayAddr)

		if err != nil {
			return sendResultMsg{err: fmt.Errorf("could not connect to relay: %w", err)}
		}

		if meta.IsArchive {
			archivePath := archive.ArchivePath(meta.OriginalPath)
			if err := archive.CompressFolder(meta.OriginalPath, archivePath); err != nil {
				conn.Close()
				return sendResultMsg{err: fmt.Errorf("failed to archive folder: %w", err)}
			}
			defer archive.CleanupArchive(archivePath)

			newMeta, err := transfer.Inspect(archivePath)
			if err != nil {
				conn.Close()
				return sendResultMsg{err: err}
			}
			newMeta.IsArchive = true
			meta = newMeta
		}

		if err := transfer.SendFile(conn, meta); err != nil {
			conn.Close()
			return sendResultMsg{err: err}
		}

		conn.Close()
		return transferDoneMsg{bytes: meta.Size}
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
			styleMuted.Render("Name  : ") + styleNormal.Render(m.meta.Filename) + "\n" +
				styleMuted.Render("Size  : ") + styleNormal.Render(m.meta.HumanSize()) + "\n" +
				styleMuted.Render("Type  : ") + styleNormal.Render(fileType),
		)
		prompt := stylePrompt.Render("  Send this? [y/n]: ") + m.confirm
		return header + info + "\n" + prompt + footer

	case sendStateConnecting:
		return header + styleMuted.Render("  Connecting to relay...") + footer

	case sendStateWaitingOTP:
		return header + styleMuted.Render("  Waiting for OTP...") + footer

	case sendStateWaitingReceiver:
		otpBox := styleBox.Render(
			styleMuted.Render("Your OTP\n\n") +
				styleOTP.Render(m.otp) + "\n\n" +
				styleMuted.Render("Share this code with the receiver"),
		)
		status := styleMuted.Render("  Waiting for receiver to connect...")
		return header + otpBox + "\n" + status + footer

	case sendStateTransferring:
		bar := renderProgressBar(m.progress, 30)
		return header + bar + footer

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

func renderProgressBar(percent float64, width int) string {
	filled := int(percent * float64(width))
	if filled > width {
		filled = width
	}
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	style := styleProgressBar
	if percent >= 1.0 {
		style = styleProgressDone
	}

	return fmt.Sprintf("  %s %5.1f%%", style.Render("["+bar+"]"), percent*100)
}
