package app

import (
	"os"
	"os/user"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/erikfrish/fprint-menu/internal/fprint"
	"github.com/erikfrish/fprint-menu/internal/i18n"
)

func New(version, lang string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(accent)
	catalog, err := i18n.New(lang)
	if err != nil {
		panic(err)
	}

	m := Model{
		version: version,
		t:       catalog,
		user:    targetUser(),
		width:   96,
		height:  28,
		screen:  screenMenu,
		spinner: s,
		menu: []menuItem{
			{catalog.T("menu.enroll"), catalog.T("menu.enroll.desc"), actionEnroll, false},
			{catalog.T("menu.verify"), catalog.T("menu.verify.desc"), actionVerify, false},
			{catalog.T("menu.delete"), catalog.T("menu.delete.desc"), actionDelete, true},
			{catalog.T("menu.wipe"), catalog.T("menu.wipe.desc"), actionWipe, true},
			{catalog.T("menu.diagnostics"), catalog.T("menu.diagnostics.desc"), actionDiagnostics, false},
			{catalog.T("menu.restart"), catalog.T("menu.restart.desc"), actionRestartService, false},
		},
	}
	m.pm, m.pmFound = fprint.DetectPackageManager()
	m.missingCmds = fprint.MissingCommands()
	if m.pmFound {
		m.missing = fprint.MissingPackages(m.pm)
	}
	if len(m.missing) > 0 {
		m.menu = append([]menuItem{{catalog.T("menu.install"), catalog.T("menu.install.desc"), actionInstallDeps, false}}, m.menu...)
		m.message = "Missing packages: " + strings.Join(m.missing, ", ")
	}
	return m
}

func targetUser() string {
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		return sudoUser
	}
	if envUser := os.Getenv("USER"); envUser != "" {
		return envUser
	}
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	return "unknown"
}
