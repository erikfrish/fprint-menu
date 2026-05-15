package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/erikfrish/fprint-menu/internal/sudo"
)

type sudoEventMsg sudo.Event

type quitMsg struct{}

type model struct {
	ctx      context.Context
	cancel   context.CancelFunc
	session  *sudo.Session
	debug    *log.Logger
	status   string
	password string
	waitPass bool
	done     bool
	err      error
	width    int
	height   int
	fpFails  int
	pwFails  int
}

func main() {
	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newModel() model {
	ctx, cancel := context.WithCancel(context.Background())
	file, err := os.OpenFile("/tmp/sudo-tui.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	var logger *log.Logger
	if err == nil {
		logger = log.New(file, "", log.LstdFlags|log.Lmicroseconds)
	}
	return model{ctx: ctx, cancel: cancel, debug: logger, status: "Starting sudo authentication..."}
}

func (m model) Init() tea.Cmd {
	return m.startSudo()
}

func (m model) startSudo() tea.Cmd {
	return func() tea.Msg {
		session, err := sudo.StartValidatePTY(m.ctx)
		if err != nil {
			return sudoEventMsg{Kind: sudo.EventFailed, Err: err}
		}
		return sessionStartedMsg{session: session}
	}
}

type sessionStartedMsg struct {
	session *sudo.Session
}

func waitEvent(session *sudo.Session) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-session.Events()
		if !ok {
			return sudoEventMsg{Kind: sudo.EventFailed, Err: fmt.Errorf("sudo event stream closed")}
		}
		return sudoEventMsg(event)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case sessionStartedMsg:
		m.session = msg.session
		m.status = "Waiting for sudo/PAM..."
		return m, waitEvent(msg.session)
	case sudoEventMsg:
		event := sudo.Event(msg)
		if m.debug != nil {
			m.debug.Printf("event kind=%d line=%q err=%v", event.Kind, event.Line, event.Err)
		}
		switch event.Kind {
		case sudo.EventRaw:
			return m, waitEvent(m.session)
		case sudo.EventFingerprintPrompt:
			m.waitPass = false
			m.password = ""
			m.status = "Touch the fingerprint reader."
		case sudo.EventFingerprintFailed:
			m.fpFails++
			m.waitPass = false
			m.status = fmt.Sprintf("Fingerprint did not match (%d failed scan%s).", m.fpFails, plural(m.fpFails))
		case sudo.EventPasswordPrompt:
			m.waitPass = true
			m.password = ""
			m.status = "Sudo is asking for your password now."
		case sudo.EventPasswordFailed:
			m.pwFails++
			m.waitPass = false
			m.password = ""
			m.status = fmt.Sprintf("Password was rejected (%d failed password attempt%s).", m.pwFails, plural(m.pwFails))
		case sudo.EventNoPassword:
			m.status = "Sudo did not receive a password. Waiting for the next PAM step."
		case sudo.EventSuccess:
			m.done = true
			m.status = "Sudo authentication passed."
			return m, quitSoon()
		case sudo.EventFailed:
			m.done = true
			m.err = event.Err
			if m.err == nil {
				m.err = fmt.Errorf("sudo failed")
			}
			m.status = m.err.Error()
			return m, tea.Quit
		}
		if m.session == nil || m.done {
			return m, nil
		}
		return m, waitEvent(m.session)
	case quitMsg:
		return m, tea.Quit
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancel()
			return m, tea.Quit
		case "enter":
			if m.waitPass && m.session != nil {
				password := m.password
				m.password = ""
				m.waitPass = false
				m.status = "Password sent. Waiting for sudo/PAM..."
				return m, func() tea.Msg {
					if err := m.session.SendPassword(password); err != nil {
						return sudoEventMsg{Kind: sudo.EventFailed, Err: err}
					}
					return nil
				}
			}
		case "backspace":
			if m.waitPass && len(m.password) > 0 {
				m.password = m.password[:len(m.password)-1]
			}
		default:
			if m.waitPass && len(msg.String()) == 1 {
				m.password += msg.String()
			}
		}
	}
	return m, nil
}

func quitSoon() tea.Cmd {
	return tea.Tick(1200*time.Millisecond, func(time.Time) tea.Msg { return quitMsg{} })
}

func (m model) View() string {
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("39")).Padding(1, 2).Width(64)
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render("sudo-tui")
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	warn := lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	good := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")
	if m.done && m.err == nil {
		b.WriteString(good.Render(m.status))
	} else if m.err != nil {
		b.WriteString(warn.Render(m.status))
	} else {
		b.WriteString(m.status)
	}
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Fingerprint failures: %d\n", m.fpFails))
	b.WriteString(fmt.Sprintf("Password failures: %d\n", m.pwFails))
	b.WriteString("\n")
	if m.waitPass {
		b.WriteString(warn.Render("Password: "))
		b.WriteString(strings.Repeat("•", len(m.password)))
		b.WriteString("│\n")
		b.WriteString(muted.Render("[enter] submit password, [esc] cancel"))
	} else {
		b.WriteString(muted.Render("Waiting for fingerprint/PAM. [esc] cancel"))
	}

	content := box.Render(b.String())
	if m.width <= 0 || m.height <= 0 {
		return content
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
