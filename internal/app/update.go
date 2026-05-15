package app

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/erikfrish/fprint-menu/internal/fprint"
	"github.com/erikfrish/fprint-menu/internal/sudo"
)

const sudoPreserveEnv = sudo.PreserveDesktopEnv

func (m Model) Init() tea.Cmd {
	return refreshEnrolledCmd(m.user)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.MouseMsg:
		return m.handleMouse(msg)
	case authStartedMsg:
		m.authSession = msg.session
		m.authCancel = msg.cancel
		m.authStatus = m.t.T("auth.waiting")
		m.authFailed = false
		return m, waitAuthEvent(msg.session)
	case sudo.Event:
		if debugLog != nil {
			debugLog.Printf("auth event kind=%d line=%q err=%v", msg.Kind, msg.Line, msg.Err)
		}
		return m.handleAuthEvent(msg)
	case authSuccessMsg:
		if debugLog != nil {
			debugLog.Printf("auth success, starting privileged action pending=%v", m.pending)
		}
		m.authCancel = nil
		return m.startPrivilegedAction()
	case commandDoneMsg:
		if debugLog != nil {
			debugLog.Printf("command done title=%q err=%v output=%q", msg.title, msg.err, truncate(msg.output, 200))
		}
		if msg.err != nil && m.shouldRetryWithPassword(msg.output) {
			return m.startPasswordFallback()
		}
		m.screen = screenOutput
		m.result = msg
		m.message = ""
		m.pm, m.pmFound = fprint.DetectPackageManager()
		m.missingCmds = fprint.MissingCommands()
		m.missing = nil
		if m.pmFound {
			m.missing = fprint.MissingPackages(m.pm)
		}
		return m, refreshEnrolledCmd(m.user)
	case enrollProgressMsg:
		if m.enrollCanceled {
			return m, nil
		}
		if msg.current > 0 {
			m.enrollCurrent = msg.current
			m.enrollTotal = msg.total
			if m.enrollCurrent > m.enrollTotal {
				m.enrollCurrent = m.enrollTotal
			}
		}
		if msg.retry {
			m.enrollRetry = true
			m.enrollRetries++
			m.enrollStatus = m.t.T("enroll.retry")
			if m.enrollRetries >= 2 {
				m.enrollBlink = true
				m.enrollBlinkSeq++
				return m, tea.Batch(m.readNextEnrollMsg(), enrollRetryBlinkCmd(m.enrollBlinkSeq))
			}
			return m, m.readNextEnrollMsg()
		}
		if msg.line != "" && strings.Contains(msg.line, "enroll-completed") {
			m.enrollCh = nil
			m.enrollWaiting = false
			m.enrollRetry = false
			m.enrollRetries = 0
			m.enrollBlink = false
			m.enrollStatus = m.t.T("enroll.completed")
			m.screen = screenEnrollSuccess
			return m, refreshEnrolledCmd(m.user)
		}
		m.enrollRetry = false
		m.enrollRetries = 0
		m.enrollBlink = false
		if m.enrollCurrent == 0 {
			m.enrollStatus = m.t.T("enroll.touch")
		} else {
			m.enrollStatus = enrollProgressText(m.t.T("enroll.progress"), m.enrollCurrent, m.enrollTotal)
		}
		if msg.done {
			m.enrollCh = nil
			m.enrollCancel = nil
			if msg.err != nil {
				output := msg.err.Error()
				if m.shouldRetryWithPassword(output) {
					return m.startPasswordFallback()
				}
				err := fmt.Errorf("enroll failed")
				if isEnrollDuplicateError(output) {
					err = fmt.Errorf("%s", m.t.T("enroll.duplicate_title"))
					output = m.t.T("enroll.duplicate") + "\n\n" + output
				}
				m.screen = screenOutput
				m.result = commandDoneMsg{title: m.running, output: output, err: err}
				m.pm, m.pmFound = fprint.DetectPackageManager()
				m.missingCmds = fprint.MissingCommands()
				m.missing = nil
				if m.pmFound {
					m.missing = fprint.MissingPackages(m.pm)
				}
				return m, refreshEnrolledCmd(m.user)
			}
			m.enrollCurrent = m.enrollTotal
			m.enrollStatus = m.t.T("enroll.completed")
			m.screen = screenEnrollSuccess
			return m, refreshEnrolledCmd(m.user)
		}
		return m, m.readNextEnrollMsg()
	case enrollRetryBlinkMsg:
		if msg.seq == m.enrollBlinkSeq {
			m.enrollBlink = false
		}
		return m, nil
	case enrolledMsg:
		m.enrolled = msg.fingers
		m.enrolledErr = msg.err
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress {
		return m, nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		return m.moveCursor(-1)
	case tea.MouseButtonWheelDown:
		return m.moveCursor(1)
	default:
		return m, nil
	}
}

func (m Model) moveCursor(delta int) (tea.Model, tea.Cmd) {
	limit := 0
	switch m.screen {
	case screenMenu:
		limit = len(m.menu)
	case screenFinger:
		limit = len(m.picker)
	default:
		return m, nil
	}
	if limit == 0 {
		return m, nil
	}

	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= limit {
		m.cursor = limit - 1
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenMenu:
		return m.handleMenuKey(msg)
	case screenFinger:
		return m.handleFingerKey(msg)
	case screenConfirm:
		return m.handleConfirmKey(msg)
	case screenAuth:
		return m.handleAuthKey(msg)
	case screenOutput:
		switch msg.String() {
		case "left", "h", "р", "esc", "backspace", "enter", " ":
			return m.back()
		case "q", "й":
			return m, tea.Quit
		}
	case screenEnrollSuccess:
		switch msg.String() {
		case "left", "h", "р", "esc", "backspace":
			m.screen = screenFinger
			return m, nil
		case "enter", " ":
			m.screen = screenMenu
			m.cursor = m.menuCursor
			return m, nil
		case "q", "й":
			return m, tea.Quit
		}
	case screenRunning:
		if m.pending == actionEnroll && m.enrollCancel != nil {
			switch msg.String() {
			case "esc", "q", "й", "ctrl+c":
				m.enrollCancel()
				m.enrollCancel = nil
				m.enrollCh = nil
				m.enrollCanceled = true
				m.screen = screenFinger
				return m, nil
			}
		}
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) handleMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "й":
		return m, tea.Quit
	case "up", "k", "л":
		return m.moveCursor(-1)
	case "down", "j", "о":
		return m.moveCursor(1)
	case "right", "l", "д", "enter", " ":
		return m.selectMenuItem(m.cursor)
	}
	return m, nil
}

func (m Model) selectMenuItem(index int) (tea.Model, tea.Cmd) {
	item := m.menu[index]
	m.menuCursor = index
	m.pending = item.act
	switch item.act {
	case actionDiagnostics:
		return m.runDiagnostics()
	case actionRestartService:
		m = m.resetAuth()
		m.screen = screenAuth
		return m, m.startAuth()
	case actionInstallDeps, actionWipe:
		m.confirmMode = confirmDanger
		m.screen = screenConfirm
		return m, nil
	case actionEnroll:
		m.picker = fprint.Fingers
		m.screen = screenFinger
		m.cursor = 0
		return m, nil
	case actionVerify, actionDelete:
		if len(m.enrolled) == 0 {
			m.outputBack = screenMenu
			m.outputCursor = m.menuCursor
			m.result = commandDoneMsg{title: item.title, cmd: []string{"fprintd-list", m.user}, output: m.t.T("result.no_enrolled")}
			m.screen = screenOutput
			return m, nil
		}
		m.picker = m.enrolled
		m.screen = screenFinger
		m.cursor = 0
		return m, nil
	}
	return m, nil
}

func (m Model) handleFingerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "й":
		return m, tea.Quit
	case "left", "h", "р", "esc", "backspace":
		return m.back()
	case "up", "k", "л":
		return m.moveCursor(-1)
	case "down", "j", "о":
		return m.moveCursor(1)
	case "right", "l", "д", "enter", " ":
		return m.selectFinger(m.cursor)
	}
	return m, nil
}

func (m Model) selectFinger(index int) (tea.Model, tea.Cmd) {
	m.finger = m.picker[index]
	if m.pending == actionDelete {
		m.confirmMode = confirmDanger
		m.screen = screenConfirm
		return m, nil
	}
	if m.pending == actionEnroll {
		return m.startPasswordFallback()
	}
	return m.runFingerAction()
}

func (m Model) runFingerAction() (tea.Model, tea.Cmd) {
	switch m.pending {
	case actionVerify:
		m.outputBack = screenFinger
		m.outputCursor = m.cursor
		return m.run(m.t.T("menu.verify")+" "+m.finger, "fprintd-verify", "-f", m.finger, m.user)
	}
	return m, nil
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "й":
		return m, tea.Quit
	case "n", "N", "т", "Т", "esc":
		return m.back()
	case "y", "Y", "н", "Н", "enter":
		return m.confirm()
	}
	return m, nil
}

func (m Model) confirm() (tea.Model, tea.Cmd) {
	switch m.pending {
	case actionDelete, actionWipe, actionInstallDeps:
		m = m.resetAuth()
		m.screen = screenAuth
		return m, m.startAuth()
	}
	return m, nil
}

func (m Model) startAuth() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		session, err := sudo.StartValidatePTY(ctx)
		if err != nil {
			cancel()
			return sudo.Event{Kind: sudo.EventFailed, Err: err}
		}
		return authStartedMsg{session: session, cancel: cancel}
	}
}

func (m Model) resetAuth() Model {
	m.authSession = nil
	m.authCancel = nil
	m.authStatus = m.t.T("auth.starting")
	m.authWaitPass = false
	m.authPassword = ""
	m.authFailed = false
	m.authSavedPwd = ""
	m.authPasswordOnly = false
	return m
}

func waitAuthEvent(session *sudo.Session) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-session.Events()
		if !ok {
			return sudo.Event{Kind: sudo.EventFailed, Err: fmt.Errorf("auth session closed")}
		}
		return event
	}
}

func waitAuthEventAfter(session *sudo.Session, delay time.Duration) tea.Cmd {
	if session == nil {
		return nil
	}
	return tea.Tick(delay, func(time.Time) tea.Msg {
		event, ok := <-session.Events()
		if !ok {
			return sudo.Event{Kind: sudo.EventFailed, Err: fmt.Errorf("auth session closed")}
		}
		return event
	})
}

func (m Model) handleAuthEvent(event sudo.Event) (tea.Model, tea.Cmd) {
	switch event.Kind {
	case sudo.EventFingerprintPrompt:
		m.authWaitPass = false
		m.authPassword = ""
		m.authStatus = m.t.T("auth.fingerprint")
	case sudo.EventFingerprintFailed:
		m.authWaitPass = false
		m.authStatus = m.t.T("auth.fp_failed")
		return m, waitAuthEventAfter(m.authSession, 400*time.Millisecond)
	case sudo.EventPasswordPrompt:
		m.authWaitPass = true
		m.authPassword = ""
		m.authStatus = m.t.T("auth.password_prompt")
	case sudo.EventPasswordFailed:
		m.authWaitPass = false
		m.authPassword = ""
		m.authStatus = m.t.T("auth.pw_failed")
		return m, waitAuthEventAfter(m.authSession, 400*time.Millisecond)
	case sudo.EventNoPassword:
		m.authStatus = m.t.T("auth.no_password")
	case sudo.EventSuccess:
		m.authWaitPass = false
		m.authFailed = false
		m.authStatus = m.t.T("auth.success")
		if debugLog != nil {
			debugLog.Printf("auth success, savedPwd=%v", m.authSavedPwd != "")
		}
		return m, func() tea.Msg { return authSuccessMsg{} }
	case sudo.EventFailed:
		m.authWaitPass = false
		m.authSession = nil
		m.authFailed = true
		if m.authCancel != nil {
			m.authCancel()
			m.authCancel = nil
		}
		m.authStatus = m.t.T("auth.failed")
		return m, nil
	case sudo.EventRaw:
	}
	if m.authSession != nil {
		return m, waitAuthEvent(m.authSession)
	}
	return m, nil
}

func (m Model) handleAuthKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		if m.authCancel != nil {
			m.authCancel()
			m.authCancel = nil
		}
		return m, tea.Quit
	case "esc":
		return m.cancelAuth()
	case "enter":
		if m.authWaitPass && m.authPasswordOnly {
			m.authSavedPwd = m.authPassword
			m.authPassword = ""
			m.authWaitPass = false
			m.authPasswordOnly = false
			m.authStatus = m.t.T("auth.sending")
			return m, func() tea.Msg { return authSuccessMsg{} }
		}
		if m.authWaitPass && m.authSession != nil {
			password := m.authPassword
			m.authPassword = ""
			m.authWaitPass = false
			m.authSavedPwd = password
			m.authStatus = m.t.T("auth.sending")
			session := m.authSession
			return m, func() tea.Msg {
				_ = session.SendPassword(password)
				return nil
			}
		}
		if m.authSession == nil || m.authFailed {
			return m.cancelAuth()
		}
	case "backspace":
		if m.authWaitPass && len(m.authPassword) > 0 {
			m.authPassword = m.authPassword[:len(m.authPassword)-1]
		}
	default:
		if m.authWaitPass && len(msg.String()) == 1 {
			m.authPassword += msg.String()
		}
	}
	return m, nil
}

func (m Model) cancelAuth() (tea.Model, tea.Cmd) {
	if m.authCancel != nil {
		m.authCancel()
		m.authCancel = nil
	}
	m.authSession = nil
	m.authWaitPass = false
	m.authPassword = ""
	m.authPasswordOnly = false
	switch m.pending {
	case actionEnroll:
		m.screen = screenFinger
	case actionRestartService:
		m.screen = screenMenu
		m.cursor = m.menuCursor
	default:
		m.screen = screenConfirm
	}
	return m, nil
}

func (m Model) startPasswordFallback() (tea.Model, tea.Cmd) {
	if debugLog != nil {
		debugLog.Printf("starting password fallback pending=%v", m.pending)
	}
	m = m.resetAuth()
	m.screen = screenAuth
	m.authWaitPass = true
	m.authPasswordOnly = true
	m.authStatus = m.t.T("auth.password_prompt")
	return m, nil
}

func (m Model) shouldRetryWithPassword(output string) bool {
	if !m.needsPrivilege() {
		return false
	}
	return isSudoPasswordRequired(output)
}

func (m Model) needsPrivilege() bool {
	switch m.pending {
	case actionEnroll, actionDelete, actionWipe, actionInstallDeps, actionRestartService:
		return true
	default:
		return false
	}
}

func (m Model) startPrivilegedAction() (tea.Model, tea.Cmd) {
	switch m.pending {
	case actionDelete:
		return m.runSudoCached(m.t.T("menu.delete"), "fprintd-delete", m.user, "-f", m.finger)
	case actionWipe:
		return m.runSudoCached(m.t.T("menu.wipe"), "fprintd-delete", fprint.LocalFingerprintUsers(m.user)...)
	case actionInstallDeps:
		if !m.pmFound {
			m.result = commandDoneMsg{title: m.t.T("menu.install"), output: m.t.T("result.no_pkg_manager")}
			m.screen = screenOutput
			return m, nil
		}
		args := append([]string{}, m.pm.Install...)
		args = append(args, m.missing...)
		return m.runSudoCached(m.t.T("menu.install"), args[0], args[1:]...)
	case actionEnroll:
		return m.runSudoEnrollCached(m.t.T("menu.enroll")+" "+m.finger, "fprintd-enroll", "-f", m.finger, m.user)
	case actionRestartService:
		return m.runSudoCached(m.t.T("menu.restart"), "systemctl", "restart", "fprintd")
	}
	return m, nil
}

func (m Model) back() (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenOutput:
		m.screen = m.outputBack
		if m.outputBack == screenFinger {
			m.cursor = m.outputCursor
		} else {
			m.cursor = m.menuCursor
		}
	default:
		m.screen = screenMenu
		m.cursor = m.menuCursor
	}
	return m, nil
}

func (m Model) run(title string, name string, args ...string) (tea.Model, tea.Cmd) {
	m.screen = screenRunning
	m.running = title
	m.enrollStatus = ""
	cmd := exec.Command(name, args...)
	return m, tea.Batch(m.spinner.Tick, commandCmd(title, append([]string{name}, args...), cmd))
}

func (m Model) runSudoCached(title string, name string, args ...string) (tea.Model, tea.Cmd) {
	sudoArgs := sudoArgs(m.authSavedPwd, name, args...)
	if debugLog != nil {
		debugLog.Printf("runSudoCached: sudo %v savedPwd=%v", sudoArgs, m.authSavedPwd != "")
	}
	if name == "fprintd-delete" {
		m.screen = screenRunning
		m.running = title
		m.enrollCurrent = 0
		m.enrollTotal = 0
		m.enrollStatus = ""
		m.enrollWaiting = false
		m.outputBack = screenMenu
		m.outputCursor = m.menuCursor
		return m, tea.Batch(m.spinner.Tick, sudoDeleteCmd(title, sudoArgs, m.authSavedPwd))
	}
	if m.authSavedPwd == "" {
		return m.run(title, "sudo", sudoArgs...)
	}
	return m.runWithStdin(title, m.authSavedPwd, "sudo", sudoArgs...)
}

func (m Model) runWithStdin(title string, stdinData string, name string, args ...string) (tea.Model, tea.Cmd) {
	m.screen = screenRunning
	m.running = title
	m.enrollCurrent = 0
	m.enrollTotal = 0
	m.enrollStatus = ""
	m.enrollWaiting = false
	m.enrollRetry = false
	m.enrollRetries = 0
	m.enrollBlink = false
	m.enrollBlinkSeq = 0
	m.outputBack = screenMenu
	m.outputCursor = m.menuCursor
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(stdinData + "\n")
	return m, tea.Batch(m.spinner.Tick, commandCmd(title, append([]string{name}, args...), cmd))
}

func (m Model) runSudoEnrollCached(title string, name string, args ...string) (tea.Model, tea.Cmd) {
	m.screen = screenRunning
	m.running = title
	m.enrollCurrent = 0
	m.enrollTotal = 6
	m.enrollStatus = m.t.T("enroll.touch")
	m.enrollWaiting = false
	m.enrollRetry = false
	m.enrollRetries = 0
	m.enrollBlink = false
	m.enrollBlinkSeq = 0
	m.outputBack = screenFinger
	m.outputCursor = m.cursor
	m.enrollCanceled = false
	ch := make(chan enrollProgressMsg, 32)
	m.enrollCh = ch
	ctx, cancel := context.WithCancel(context.Background())
	m.enrollCancel = cancel
	if debugLog != nil {
		debugLog.Printf("runSudoEnrollCached: sudo %s %v", name, args)
	}
	go streamEnrollProgressCached(ctx, m.authSavedPwd, name, args, ch, containsString(m.enrolled, m.finger), m.user, m.finger)
	return m, tea.Batch(m.spinner.Tick, m.readNextEnrollMsg())
}

func (m Model) readNextEnrollMsg() tea.Cmd {
	ch := m.enrollCh
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return enrollProgressMsg{done: true}
		}
		return msg
	}
}

func enrollRetryBlinkCmd(seq int) tea.Cmd {
	return tea.Tick(220*time.Millisecond, func(time.Time) tea.Msg {
		return enrollRetryBlinkMsg{seq: seq}
	})
}

func streamEnrollProgressCached(ctx context.Context, password string, name string, args []string, ch chan enrollProgressMsg, replace bool, user string, finger string) {
	defer close(ch)
	if debugLog != nil {
		debugLog.Printf("streamEnrollProgressCached: starting, name=%s args=%v", name, args)
	}
	if err := restartFprintdCached(ctx, password); err != nil {
		if ctx.Err() != nil {
			return
		}
		if debugLog != nil {
			debugLog.Printf("streamEnrollProgressCached: restartFprintd failed: %v", err)
		}
		ch <- enrollProgressMsg{done: true, err: err}
		return
	}
	if replace {
		if err := deleteFingerCached(ctx, password, user, finger); err != nil {
			if ctx.Err() != nil {
				return
			}
			ch <- enrollProgressMsg{done: true, err: err}
			return
		}
	}
	time.Sleep(750 * time.Millisecond)
	if ctx.Err() != nil {
		return
	}
	for attempt := 0; attempt < 2; attempt++ {
		err := streamEnrollAttemptCached(ctx, password, name, args, ch)
		if ctx.Err() != nil {
			return
		}
		if err == nil {
			ch <- enrollProgressMsg{done: true, current: 6, total: 6}
			return
		}
		if attempt == 0 && isAlreadyInUseError(err.Error()) {
			if restartErr := restartFprintdCached(ctx, password); restartErr != nil {
				ch <- enrollProgressMsg{done: true, err: restartErr}
				return
			}
			continue
		}
		ch <- enrollProgressMsg{done: true, err: err}
		return
	}
}

func streamEnrollAttemptCached(ctx context.Context, password string, name string, args []string, ch chan enrollProgressMsg) error {
	sudoArgs := sudoArgs(password, name, args...)
	if debugLog != nil {
		debugLog.Printf("streamEnrollAttemptCached: sudo %v savedPwd=%v", sudoArgs, password != "")
	}
	cmd := exec.CommandContext(ctx, "sudo", sudoArgs...)
	if password != "" {
		cmd.Stdin = strings.NewReader(password + "\n")
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s\n%s", err.Error(), stderr.String())
	}
	current := 0
	total := 6
	started := false
	var allOutput strings.Builder
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		allOutput.WriteString(line)
		allOutput.WriteString("\n")
		if !strings.HasPrefix(line, "Enroll result:") {
			if strings.HasPrefix(line, "Enrolling ") {
				ch <- enrollProgressMsg{line: line, current: 0, total: total}
			}
			continue
		}
		switch {
		case strings.Contains(line, "enroll-stage-passed"):
			if !started {
				started = true
				ch <- enrollProgressMsg{line: line, current: 0, total: total}
				continue
			}
			if current < total-1 {
				current++
			}
			ch <- enrollProgressMsg{line: line, current: current, total: total}
		case strings.Contains(line, "enroll-completed"):
			current = total
			ch <- enrollProgressMsg{line: line, current: current, total: total}
			go func() {
				_, _ = io.Copy(io.Discard, stdout)
				_ = cmd.Wait()
			}()
			return nil
		case strings.Contains(line, "enroll-retry"):
			ch <- enrollProgressMsg{line: line, current: current, total: total, retry: true}
		case strings.Contains(line, "enroll-duplicate"):
			_ = cmd.Wait()
			return fmt.Errorf("%s", line)
		case strings.Contains(line, "enroll-fail"):
			_ = cmd.Wait()
			return fmt.Errorf("%s", line)
		}
	}
	_ = cmd.Wait()
	exitCode := cmd.ProcessState.ExitCode()
	errOut := strings.TrimSpace(stderr.String())
	combined := strings.TrimSpace(allOutput.String())
	if exitCode != 0 {
		detail := combined
		if errOut != "" {
			detail = errOut + "\n" + combined
		}
		if detail == "" {
			detail = fmt.Sprintf("exit code %d", exitCode)
		}
		return fmt.Errorf("%s", detail)
	}
	if current < total && combined != "" {
		return fmt.Errorf("%s", combined)
	}
	return nil
}

func deleteFingerCached(ctx context.Context, password string, user string, finger string) error {
	if debugLog != nil {
		debugLog.Printf("deleteFingerCached: user=%s finger=%s savedPwd=%v", user, finger, password != "")
	}
	cmd := exec.CommandContext(ctx, "sudo", sudoArgs(password, "fprintd-delete", user, "-f", finger)...)
	if password != "" {
		cmd.Stdin = strings.NewReader(password + "\n")
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s\n%s", err.Error(), strings.TrimSpace(buf.String()))
	}
	return nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func isEnrollDuplicateError(output string) bool {
	return strings.Contains(output, "enroll-duplicate")
}

func isSudoPasswordRequired(output string) bool {
	for _, event := range sudo.ParseEvents(output) {
		switch event.Kind {
		case sudo.EventPasswordPrompt, sudo.EventPasswordFailed, sudo.EventNoPassword:
			return true
		}
	}
	text := strings.ToLower(output)
	return strings.Contains(text, "a password is required") || strings.Contains(text, "no password was provided")
}

func sudoArgs(password string, name string, args ...string) []string {
	prefix := []string{"-n", sudoPreserveEnv, name}
	if password != "" {
		prefix = []string{"-S", "-p", "", sudoPreserveEnv, name}
	}
	return append(prefix, args...)
}

func restartFprintdCached(ctx context.Context, password string) error {
	if debugLog != nil {
		debugLog.Printf("restartFprintdCached: starting savedPwd=%v", password != "")
	}
	cmd := exec.CommandContext(ctx, "sudo", sudoArgs(password, "systemctl", "restart", "fprintd")...)
	if password != "" {
		cmd.Stdin = strings.NewReader(password + "\n")
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	if debugLog != nil {
		debugLog.Printf("restartFprintdCached: err=%v output=%q", err, truncate(strings.TrimSpace(buf.String()), 200))
	}
	if err != nil {
		text := strings.TrimSpace(buf.String())
		if text != "" {
			return fmt.Errorf("%s", text)
		}
		return err
	}
	return nil
}

func isAlreadyInUseError(text string) bool {
	return strings.Contains(text, "AlreadyInUse") || strings.Contains(text, "AlreadyInUser")
}

func enrollProgressText(format string, current, total int) string {
	if strings.Contains(format, "%") {
		return fmt.Sprintf(format, current, total)
	}
	return fmt.Sprintf("%s %d/%d", format, current, total)
}

func (m Model) runDiagnostics() (tea.Model, tea.Cmd) {
	m.screen = screenRunning
	m.running = m.t.T("menu.diagnostics")
	m.outputBack = screenMenu
	m.outputCursor = m.menuCursor
	return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
		var out strings.Builder
		out.WriteString(m.t.T("diag.packages") + "\n")
		if m.pmFound {
			out.WriteString("  manager: " + m.pm.Name + "\n")
			for _, pkg := range m.pm.Pkgs {
				args := append([]string{}, m.pm.Query[1:]...)
				args = append(args, pkg)
				line, err := fprint.CommandOutput(m.pm.Query[0], args...)
				if err != nil {
					out.WriteString("  missing: " + pkg + "\n")
					continue
				}
				out.WriteString("  " + strings.TrimSpace(line) + "\n")
			}
		} else {
			out.WriteString("  no supported package manager found\n")
		}
		missingCmds := fprint.MissingCommands()
		if len(missingCmds) > 0 {
			out.WriteString("\nMissing commands\n")
			for _, name := range missingCmds {
				out.WriteString("  " + name + "\n")
			}
		}

		out.WriteString("\n" + m.t.T("diag.service") + "\n")
		service, err := fprint.CommandOutput("systemctl", "status", "fprintd.service", "--no-pager", "--lines=0")
		if err != nil {
			out.WriteString(m.t.T("diag.service.inactive"))
		} else {
			out.WriteString(fprint.Indent(service))
		}

		out.WriteString("\n" + m.t.T("diag.devices") + "\n")
		if _, err := exec.LookPath("lsusb"); err != nil {
			out.WriteString(m.t.T("diag.usbutils"))
		} else {
			usb, _ := fprint.CommandOutput("lsusb")
			matches := fprint.FilterLines(usb, []string{"finger", "fprint", "validity", "synaptics", "goodix", "elan"})
			if matches == "" {
				out.WriteString(m.t.T("diag.no_device"))
			} else {
				out.WriteString(fprint.Indent(matches))
			}
		}

		return commandDoneMsg{title: m.t.T("menu.diagnostics"), cmd: []string{"diagnostics"}, output: out.String()}
	})
}

func commandCmd(title string, argv []string, cmd *exec.Cmd) tea.Cmd {
	return func() tea.Msg {
		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf
		err := cmd.Run()
		return commandDoneMsg{title: title, cmd: argv, output: buf.String(), err: err}
	}
}

func sudoDeleteCmd(title string, sudoArgs []string, password string) tea.Cmd {
	return func() tea.Msg {
		output, err := runSudoCapture(sudoArgs, password)
		if err == nil || !isAlreadyInUseError(output) {
			return commandDoneMsg{title: title, cmd: append([]string{"sudo"}, sudoArgs...), output: output, err: err}
		}
		if debugLog != nil {
			debugLog.Printf("sudoDeleteCmd: already in use, restarting fprintd and retrying")
		}
		if restartErr := restartFprintdCached(context.Background(), password); restartErr != nil {
			return commandDoneMsg{title: title, cmd: []string{"sudo", "systemctl", "restart", "fprintd"}, output: restartErr.Error(), err: restartErr}
		}
		output, err = runSudoCapture(sudoArgs, password)
		return commandDoneMsg{title: title, cmd: append([]string{"sudo"}, sudoArgs...), output: output, err: err}
	}
}

func runSudoCapture(sudoArgs []string, password string) (string, error) {
	cmd := exec.Command("sudo", sudoArgs...)
	if password != "" {
		cmd.Stdin = strings.NewReader(password + "\n")
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

func refreshEnrolledCmd(target string) tea.Cmd {
	return func() tea.Msg {
		items, err := fprint.Enrolled(target)
		return enrolledMsg{fingers: items, err: err}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
