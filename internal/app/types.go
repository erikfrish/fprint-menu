package app

import (
	"context"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/erikfrish/fprint-menu/internal/fprint"
	"github.com/erikfrish/fprint-menu/internal/i18n"
	"github.com/erikfrish/fprint-menu/internal/sudo"
)

const appName = "FPrint Control"

type screen int

const (
	screenMenu screen = iota
	screenFinger
	screenConfirm
	screenAuth
	screenEnrollSuccess
	screenOutput
	screenRunning
)

type confirmKind int

const (
	confirmDanger confirmKind = iota
)

type action int

const (
	actionEnroll action = iota
	actionVerify
	actionDelete
	actionWipe
	actionDiagnostics
	actionInstallDeps
	actionRestartService
)

type menuItem struct {
	title  string
	desc   string
	act    action
	danger bool
}

type commandDoneMsg struct {
	title  string
	cmd    []string
	output string
	err    error
}

type enrollProgressMsg struct {
	line    string
	current int
	total   int
	done    bool
	err     error
	retry   bool
}

type enrolledMsg struct {
	fingers []string
	err     error
}

type authStartedMsg struct {
	session *sudo.Session
	cancel  context.CancelFunc
}

type authSuccessMsg struct{}

type enrollRetryBlinkMsg struct {
	seq int
}

type Model struct {
	version string
	t       *i18n.Catalog
	user    string
	width   int
	height  int

	screen       screen
	cursor       int
	menuCursor   int
	outputBack   screen
	outputCursor int
	menu         []menuItem
	picker       []string

	pending     action
	confirmMode confirmKind
	finger      string

	authSession      *sudo.Session
	authCancel       context.CancelFunc
	authStatus       string
	authWaitPass     bool
	authPassword     string
	authFailed       bool
	authSavedPwd     string
	authPasswordOnly bool

	spinner        spinner.Model
	running        string
	enrollCurrent  int
	enrollTotal    int
	enrollStatus   string
	enrollWaiting  bool
	enrollCh       chan enrollProgressMsg
	enrollCancel   context.CancelFunc
	enrollCanceled bool
	enrollRetry    bool
	enrollRetries  int
	enrollBlink    bool
	enrollBlinkSeq int
	result         commandDoneMsg
	message        string

	pm          fprint.PackageManager
	pmFound     bool
	missing     []string
	missingCmds []string
	enrolled    []string
	enrolledErr error
}
