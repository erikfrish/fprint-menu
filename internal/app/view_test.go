package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/erikfrish/fprint-menu/internal/fprint"
)

func TestViewFitsCommonWindowSizes(t *testing.T) {
	sizes := []struct {
		width  int
		height int
	}{
		{84, 24},
		{96, 28},
		{120, 32},
		{160, 40},
	}

	for _, size := range sizes {
		m := New("test", "en")
		m.width = size.width
		m.height = size.height

		for _, line := range strings.Split(m.View(), "\n") {
			if got := len([]rune(line)); got > size.width {
				t.Fatalf("view line width %d exceeds terminal width %d for %dx%d: %q", got, size.width, size.width, size.height, line)
			}
		}
	}
}

func TestViewWarnsWhenWindowTooSmall(t *testing.T) {
	m := New("test", "en")
	m.width = minWidth - 1
	m.height = minHeight

	view := m.View()
	if !strings.Contains(view, "Window too small") {
		t.Fatalf("expected too-small warning, got %q", view)
	}
}

func TestMouseOnlyVerticalScrollMovesCursor(t *testing.T) {
	m := New("test", "en")
	m.cursor = 1

	next, _ := m.handleMouse(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
	m = next.(Model)
	if m.cursor != 2 {
		t.Fatalf("wheel down cursor = %d, want 2", m.cursor)
	}

	next, _ = m.handleMouse(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp})
	m = next.(Model)
	if m.cursor != 1 {
		t.Fatalf("wheel up cursor = %d, want 1", m.cursor)
	}

	next, _ = m.handleMouse(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = next.(Model)
	if m.cursor != 1 {
		t.Fatalf("left click cursor = %d, want unchanged 1", m.cursor)
	}
}

func TestEnrollGoesToAuthScreen(t *testing.T) {
	m := New("test", "en")
	m.pending = actionEnroll
	m.finger = "right-index-finger"
	m.picker = fprint.Fingers

	next, _ := m.selectFinger(0)
	m = next.(Model)
	if m.screen != screenAuth {
		t.Fatalf("expected screenAuth, got screen=%v", m.screen)
	}
}

func TestDeleteConfirmGoesToAuthScreen(t *testing.T) {
	m := New("test", "en")
	m.pending = actionDelete
	m.finger = "right-index-finger"

	next, cmd := m.confirm()
	m = next.(Model)
	if m.screen != screenAuth {
		t.Fatalf("expected screenAuth, got screen=%v", m.screen)
	}
	_ = cmd
}

func TestAuthScreenAcceptsQAsPasswordCharacter(t *testing.T) {
	m := New("test", "en")
	m.screen = screenAuth
	m.authPassword = "abc"
	m.authWaitPass = true

	next, cmd := m.handleAuthKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = next.(Model)
	if cmd != nil {
		t.Fatal("expected q to be handled as input, not quit")
	}
	if m.authPassword != "abcq" {
		t.Fatalf("expected password abcq, got %q", m.authPassword)
	}
}

func TestAuthEscapeFromEnrollReturnsToFingerPicker(t *testing.T) {
	m := New("test", "en")
	m.pending = actionEnroll
	m.screen = screenAuth
	m.authPassword = "testpass"

	next, _ := m.handleAuthKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(Model)
	if m.screen != screenFinger {
		t.Fatalf("expected screenFinger, got screen=%v", m.screen)
	}
}

func TestOutputBackReturnsToFingerPickerForVerify(t *testing.T) {
	m := New("test", "en")
	m.screen = screenOutput
	m.outputBack = screenFinger
	m.outputCursor = 3

	next, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if m.screen != screenFinger {
		t.Fatalf("expected screenFinger, got screen=%v", m.screen)
	}
	if m.cursor != 3 {
		t.Fatalf("expected cursor 3, got %d", m.cursor)
	}
}

func TestSuccessScreenBackAndEnter(t *testing.T) {
	m := New("test", "en")
	m.screen = screenEnrollSuccess
	m.menuCursor = 2

	next, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(Model)
	if m.screen != screenFinger {
		t.Fatalf("expected screenFinger, got screen=%v", m.screen)
	}

	m.screen = screenEnrollSuccess
	next, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if m.screen != screenMenu {
		t.Fatalf("expected screenMenu, got screen=%v", m.screen)
	}
	if m.cursor != 2 {
		t.Fatalf("expected menu cursor 2, got %d", m.cursor)
	}
}

func TestDangerSelectionUsesYellowSelectedStyle(t *testing.T) {
	m := New("test", "en")
	var dangerIndexes []int
	for i, item := range m.menu {
		if item.danger {
			dangerIndexes = append(dangerIndexes, i)
		}
	}
	if len(dangerIndexes) < 2 {
		t.Fatalf("expected at least two danger items, got %d", len(dangerIndexes))
	}
	m.cursor = dangerIndexes[1]

	view := m.viewMenu()
	deleteTitle := m.menu[dangerIndexes[0]].title
	wipeTitle := m.menu[dangerIndexes[1]].title
	selectedSeq := lipgloss.NewStyle().Foreground(yellow).Bold(true).Render("› ") + lipgloss.NewStyle().Foreground(yellow).Bold(true).Render(wipeTitle)
	deleteRed := lipgloss.NewStyle().Foreground(red).Bold(true).Render("  " + deleteTitle)

	if !strings.Contains(view, selectedSeq) {
		t.Fatalf("expected selected danger item to use yellow selected style")
	}
	if !strings.Contains(view, deleteRed) {
		t.Fatalf("expected unselected danger item to stay red")
	}
}

func TestEnrollProgressIsClampedInStatus(t *testing.T) {
	m := New("test", "en")
	m.screen = screenRunning
	m.pending = actionEnroll
	m.enrollCurrent = 8
	m.enrollTotal = 6
	m.enrollStatus = enrollProgressText(m.t.T("enroll.progress"), min(m.enrollCurrent, m.enrollTotal), m.enrollTotal)

	view := m.viewEnrolling()
	if strings.Contains(view, "8/6") {
		t.Fatalf("expected progress text to be clamped, got %q", view)
	}
	if !strings.Contains(view, "6/6") {
		t.Fatalf("expected clamped progress text, got %q", view)
	}
}

func TestRunningEnrollEscapeCancelsToFingerPicker(t *testing.T) {
	cancelled := false
	m := New("test", "en")
	m.screen = screenRunning
	m.pending = actionEnroll
	m.enrollTotal = 6
	m.enrollCancel = func() { cancelled = true }
	m.enrollCh = make(chan enrollProgressMsg)

	next, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(Model)
	if cmd != nil {
		t.Fatal("expected cancel without command")
	}
	if !cancelled {
		t.Fatal("expected enroll cancel func to be called")
	}
	if m.screen != screenFinger {
		t.Fatalf("expected screenFinger, got screen=%v", m.screen)
	}
	if m.enrollCancel != nil || m.enrollCh != nil {
		t.Fatal("expected enroll process handles to be cleared")
	}
}
