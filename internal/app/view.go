package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	minWidth     = 72
	minHeight    = 24
	sidebarWidth = 26
	contentGap   = 2
	rightPadding = 2
	maxLineWidth = 96
)

func (m Model) View() string {
	if m.width > 0 && m.height > 0 && (m.width < minWidth || m.height < minHeight) {
		return m.viewTooSmall()
	}

	var body string
	switch m.screen {
	case screenMenu:
		body = m.viewMenu()
	case screenFinger:
		body = m.viewFinger()
	case screenConfirm:
		body = m.viewConfirm()
	case screenAuth:
		body = m.viewAuthBase()
	case screenEnrollSuccess:
		body = m.viewEnrollSuccess()
	case screenRunning:
		body = m.viewRunning()
	case screenOutput:
		body = m.viewOutput()
	default:
		return ""
	}
	view := m.chrome(body)
	if m.screen == screenAuth {
		return m.overlay(view, m.viewAuthModal())
	}
	return view
}

func (m Model) chrome(body string) string {
	width := m.width
	if width < minWidth {
		width = minWidth
	}
	stylePadding := appStyle.GetHorizontalPadding()
	mainWidth := width - sidebarWidth - contentGap - stylePadding - rightPadding
	if mainWidth < 34 {
		mainWidth = 34
	}

	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		badgeStyle.Render(" FPRINT "),
		" ",
		titleStyle.Render(appName),
		" ",
		subtleStyle.Render("v"+m.version),
	)

	main := panelStyle.Width(mainWidth).MaxWidth(mainWidth).Render(body)
	side := sidebarStyle.Width(sidebarWidth).MaxWidth(sidebarWidth).Render(m.sidebar())
	content := lipgloss.JoinHorizontal(lipgloss.Top, main, strings.Repeat(" ", contentGap), side)

	lineWidth := min(width-stylePadding-rightPadding, maxLineWidth)
	return appStyle.Render(header + "\n" + subtleStyle.Render(strings.Repeat("─", lineWidth)) + "\n" + content)
}

func (m Model) viewTooSmall() string {
	message := fmt.Sprintf(
		"Window too small\n\nCurrent: %dx%d\nRequired: %dx%d\n\nIncrease the terminal size to continue.",
		m.width,
		m.height,
		minWidth,
		minHeight,
	)
	return appStyle.Render(boxStyle.Render(warnStyle.Render(message)))
}

func (m Model) sidebar() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(m.t.T("status.title")))
	b.WriteString("\n\n")
	b.WriteString(kv(m.t.T("status.user"), m.user))
	b.WriteString(kv(m.t.T("status.lang"), m.t.Lang()))
	if m.finger != "" {
		b.WriteString(kv(m.t.T("status.finger"), m.t.Finger(m.finger)))
	}
	if m.running != "" && m.screen == screenRunning {
		b.WriteString(kv(m.t.T("status.task"), m.running))
	}

	b.WriteString("\n")
	b.WriteString(titleStyle.Render(m.t.T("status.health")))
	b.WriteString("\n")
	if len(m.missing) == 0 {
		if len(m.missingCmds) == 0 {
			b.WriteString(goodStyle.Render(m.t.T("status.deps.ok")))
		} else {
			b.WriteString(warnStyle.Render(m.t.T("status.commands.missing")))
			b.WriteString("\n")
			b.WriteString(subtleStyle.Render(strings.Join(m.missingCmds, ", ")))
		}
		b.WriteString("\n")
	} else {
		b.WriteString(warnStyle.Render(m.t.T("status.deps.missing")))
		b.WriteString("\n")
		b.WriteString(subtleStyle.Render(strings.Join(m.missing, ", ")))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(titleStyle.Render(m.t.T("status.enrolled")))
	b.WriteString("\n")
	if m.enrolledErr != nil {
		b.WriteString(warnStyle.Render(m.t.T("status.enrolled.unavailable")))
		b.WriteString("\n")
		b.WriteString(subtleStyle.Render(m.t.T("status.enrolled.diagnostics")))
		b.WriteString("\n")
	} else if len(m.enrolled) == 0 {
		b.WriteString(subtleStyle.Render(m.t.T("status.enrolled.none")))
		b.WriteString("\n")
	} else {
		for _, finger := range m.enrolled {
			marker := "•"
			if finger == m.finger {
				marker = "›"
			}
			b.WriteString(goodStyle.Render(marker))
			b.WriteString(" ")
			b.WriteString(m.t.Finger(finger))
			b.WriteString("\n")
		}
	}

	if m.result.title != "" {
		b.WriteString("\n")
		b.WriteString(titleStyle.Render(m.t.T("status.last_run")))
		b.WriteString("\n")
		if m.result.err != nil {
			b.WriteString(errorStyle.Render("✗ " + m.result.title))
		} else {
			b.WriteString(goodStyle.Render("✓ " + m.result.title))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(titleStyle.Render(m.t.T("status.keys")))
	b.WriteString("\n")
	for _, hint := range m.keyHints() {
		for _, line := range strings.Split(hint, `\n`) {
			b.WriteString(subtleStyle.Render(line))
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m Model) viewMenu() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(m.t.T("menu.title")))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render(m.t.T("menu.hint")))
	b.WriteString("\n\n")
	if m.message != "" {
		b.WriteString(warnStyle.Render(m.message))
		b.WriteString("\n\n")
	}
	for i, item := range m.menu {
		cursor := "  "
		cursorStyle := lipgloss.NewStyle()
		titleStyle := lipgloss.NewStyle()
		if i == m.cursor {
			cursor = "› "
			cursorStyle = selectedStyle
			titleStyle = selectedStyle
			if item.danger {
				cursorStyle = dangerSelectedStyle
				titleStyle = dangerSelectedStyle
			}
		} else if item.danger {
			titleStyle = dangerStyle
		}
		b.WriteString(cursorStyle.Render(cursor) + titleStyle.Render(item.title))
		b.WriteString("\n")
		b.WriteString("    " + subtleStyle.Render(item.desc) + "\n")
	}
	return b.String()
}

func (m Model) viewFinger() string {
	var b strings.Builder
	b.WriteString(m.backButton())
	b.WriteString("\n\n")
	b.WriteString(titleStyle.Render(m.actionTitle()))
	b.WriteString("\n")
	b.WriteString(titleStyle.Render(m.t.T("finger.title")))
	b.WriteString("\n")
	if m.pending == actionEnroll {
		b.WriteString(subtleStyle.Render(m.t.T("finger.enroll_hint")))
	}
	b.WriteString("\n\n")
	for i, finger := range m.picker {
		cursor := "  "
		style := lipgloss.NewStyle()
		if i == m.cursor {
			cursor = "› "
			style = selectedStyle
		}
		b.WriteString(style.Render(cursor + m.t.Finger(finger)))
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) viewConfirm() string {
	var text string
	switch m.pending {
	case actionDelete:
		text = fmt.Sprintf(m.t.T("confirm.delete"), m.finger, m.user)
	case actionWipe:
		text = fmt.Sprintf(m.t.T("confirm.wipe"), m.user)
	case actionInstallDeps:
		text = fmt.Sprintf(m.t.T("confirm.install"), strings.Join(m.missing, ", "))
	default:
		text = "Continue?"
	}
	return m.backButton() + "\n\n" + titleStyle.Render(m.actionTitle()) + "\n" + boxStyle.Render(warnStyle.Render(text)+"\n\n"+subtleStyle.Render(m.t.T("confirm.hint")))
}

func (m Model) viewAuthBase() string {
	var b strings.Builder
	b.WriteString(m.backButton())
	b.WriteString("\n\n")
	b.WriteString(titleStyle.Render(m.actionTitle()))
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render(wrapText(m.t.T("auth.base_hint"), 48)))
	return b.String()
}

func (m Model) viewAuthModal() string {
	modalWidth := 72
	innerWidth := 52
	modalTitle := titleStyle.Copy().Background(bg)
	modalWarn := warnStyle.Copy().Background(bg)
	modalSubtle := subtleStyle.Copy().Background(bg)
	modalSelected := selectedStyle.Copy().Background(bg)
	modalError := errorStyle.Copy().Background(bg)
	modalGood := goodStyle.Copy().Background(bg)
	var b strings.Builder
	b.WriteString(modalTitle.Render(m.t.T("auth.title")))
	b.WriteString("\n\n")
	if strings.Contains(m.authStatus, m.t.T("auth.success")) {
		b.WriteString(modalGood.Render(m.authStatus))
	} else if m.authFailed || strings.Contains(m.authStatus, m.t.T("auth.pw_failed")) || strings.Contains(m.authStatus, m.t.T("auth.fp_failed")) {
		b.WriteString(modalError.Render(m.authStatus))
	} else {
		b.WriteString(modalWarn.Render(m.authStatus))
	}
	b.WriteString("\n\n")
	if m.authWaitPass {
		mask := strings.Repeat("•", len(m.authPassword))
		b.WriteString(modalSelected.Render("› " + mask + "│"))
		b.WriteString("\n\n")
		b.WriteString(modalSubtle.Render(wrapText(m.t.T("auth.hint_password"), 44)))
	} else if m.authFailed {
		b.WriteString(modalSubtle.Render(wrapText(m.t.T("auth.hint_failed"), 44)))
	} else {
		b.WriteString(modalSubtle.Render(wrapText(m.t.T("auth.hint_wait"), 44)))
	}
	content := lipgloss.NewStyle().Width(innerWidth).MaxWidth(innerWidth).Background(bg).Render(b.String())
	return boxStyle.Width(modalWidth).MaxWidth(modalWidth).Background(bg).Render(content)
}

func (m Model) overlay(base string, modal string) string {
	canvasWidth := m.width
	if canvasWidth < lipgloss.Width(base) {
		canvasWidth = lipgloss.Width(base)
	}
	if canvasWidth < lipgloss.Width(modal)+2 {
		canvasWidth = lipgloss.Width(modal) + 2
	}

	canvasHeight := m.height
	if canvasHeight < lipgloss.Height(base) {
		canvasHeight = lipgloss.Height(base)
	}
	if canvasHeight < lipgloss.Height(modal)+2 {
		canvasHeight = lipgloss.Height(modal) + 2
	}

	baseCanvas := lipgloss.Place(canvasWidth, canvasHeight, lipgloss.Left, lipgloss.Top, base)
	baseLines := strings.Split(baseCanvas, "\n")
	modalLines := strings.Split(modal, "\n")
	modalWidth := lipgloss.Width(modal)
	modalHeight := len(modalLines)
	startX := (canvasWidth - modalWidth) / 2
	if startX < 0 {
		startX = 0
	}
	startY := (canvasHeight - modalHeight) / 2
	if startY < 0 {
		startY = 0
	}

	for i, modalLine := range modalLines {
		y := startY + i
		if y < 0 || y >= len(baseLines) {
			continue
		}

		line := padANSI(baseLines[y], canvasWidth)
		modalLine = padANSI(modalLine, modalWidth)

		left := ansi.Cut(line, 0, startX)
		right := ansi.Cut(line, startX+modalWidth, canvasWidth)
		baseLines[y] = left + modalLine + right
	}
	return strings.Join(baseLines, "\n")
}

func padANSI(line string, targetWidth int) string {
	currentWidth := ansi.StringWidth(line)
	if currentWidth >= targetWidth {
		return line
	}
	return line + strings.Repeat(" ", targetWidth-currentWidth)
}

func (m Model) viewEnrollSuccess() string {
	fp := fingerprintArt(m.enrollTotal, m.enrollTotal)
	progress := enrollProgressBar(m.enrollTotal, m.enrollTotal)
	return fmt.Sprintf(
		"%s\n\n%s\n%s\n\n%s\n\n%s",
		goodStyle.Render(m.t.T("enroll.success")),
		fp,
		progress,
		fmt.Sprintf(goodStyle.Render(m.t.T("enroll.finger_added")), m.t.Finger(m.finger)),
		subtleStyle.Render(wrapText(m.t.T("enroll.success_hint"), 42)),
	)
}

func (m Model) actionTitle() string {
	for _, item := range m.menu {
		if item.act == m.pending {
			return item.title
		}
	}
	return appName
}

func (m Model) viewRunning() string {
	if m.pending == actionEnroll && m.enrollTotal > 0 {
		return m.viewEnrolling()
	}
	return fmt.Sprintf(
		"%s %s\n\n%s\n\n%s",
		m.spinner.View(),
		titleStyle.Render(m.running),
		subtleStyle.Render(m.t.T("running.hint")),
		boxStyle.Render(m.t.T("running.sensor")),
	)
}

func (m Model) viewEnrolling() string {
	fp := fingerprintArt(m.enrollCurrent, m.enrollTotal)
	progress := enrollProgressBar(m.enrollCurrent, m.enrollTotal)
	status := m.enrollStatus
	if status == "" {
		status = m.t.T("enroll.touch")
	}
	statusStyle := warnStyle
	if m.enrollRetry && !m.enrollBlink {
		statusStyle = errorStyle
	}

	return fmt.Sprintf(
		"%s %s\n\n%s\n%s\n\n%s\n\n%s",
		m.spinner.View(),
		titleStyle.Render(m.running),
		fp,
		progress,
		statusStyle.Render(status),
		subtleStyle.Render(m.t.T("enroll.cancel_hint")),
	)
}

func fingerprintArt(current, total int) string {
	lines := []string{
		"     ╭─────────╮     ",
		"     │  ╭───╮  │     ",
		"     │  │   │  │     ",
		"     │  │ ╷ │  │     ",
		"     │  │ │ │  │     ",
		"     │  │ ╵ │  │     ",
		"     │  ╰───╯  │     ",
		"     ╰─────────╯     ",
	}
	if total <= 0 {
		total = 5
	}
	if current > total {
		current = total
	}
	filled := (current * len(lines)) / total
	var b strings.Builder
	for i, line := range lines {
		if i < filled {
			b.WriteString(goodStyle.Render(line))
		} else {
			b.WriteString(subtleStyle.Render(line))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func enrollProgressBar(current, total int) string {
	if total <= 0 {
		total = 5
	}
	if current > total {
		current = total
	}
	width := 20
	filled := (current * width) / total
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	pct := 0
	if total > 0 {
		pct = (current * 100) / total
	}
	return fmt.Sprintf("  %s %d%%", bar, pct)
}

func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}
	var b strings.Builder
	lineLen := 0
	for _, word := range words {
		wordLen := len([]rune(word))
		if lineLen > 0 && lineLen+1+wordLen > width {
			b.WriteString("\n")
			lineLen = 0
		}
		if lineLen > 0 {
			b.WriteString(" ")
			lineLen++
		}
		b.WriteString(word)
		lineLen += wordLen
	}
	return b.String()
}

func (m Model) viewOutput() string {
	var b strings.Builder
	b.WriteString(m.backButton())
	b.WriteString("\n\n")
	if m.result.err != nil {
		errMsg := m.result.err.Error()
		if isAlreadyInUseError(errMsg) {
			b.WriteString(errorStyle.Render(m.t.T("result.already_in_use")))
		} else {
			b.WriteString(errorStyle.Render(fmt.Sprintf(m.t.T("result.failed"), errMsg)))
		}
	} else {
		b.WriteString(goodStyle.Render(m.t.T("result.completed")))
	}
	b.WriteString("\n")
	if len(m.result.cmd) > 0 {
		b.WriteString(subtleStyle.Render("$ " + strings.Join(m.result.cmd, " ")))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	output := strings.TrimSpace(m.result.output)
	if output == "" {
		output = m.t.T("result.empty")
	}
	b.WriteString(boxStyle.Render(output))
	return b.String()
}

func (m Model) backButton() string {
	return selectedStyle.Render(m.t.T("back"))
}

func kv(key, value string) string {
	return subtleStyle.Render(key+":") + " " + value + "\n"
}

func (m Model) keyHints() []string {
	switch m.screen {
	case screenMenu:
		return []string{m.t.T("keys.menu")}
	case screenFinger:
		return []string{m.t.T("keys.sub")}
	case screenConfirm:
		return []string{m.t.T("keys.confirm")}
	case screenAuth:
		return []string{m.t.T("keys.auth")}
	case screenEnrollSuccess:
		return []string{m.t.T("keys.success")}
	case screenOutput:
		return []string{m.t.T("keys.output")}
	case screenRunning:
		return []string{m.t.T("keys.running")}
	default:
		return nil
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
