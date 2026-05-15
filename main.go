package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/erikfrish/fprint-menu/internal/app"
	"github.com/erikfrish/fprint-menu/internal/help"
)

const version = "0.1.0"

func main() {
	lang := flag.String("lang", "", "interface language for debugging: en, ru, or zh")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.BoolFunc("v", "print version and exit", func(string) error {
		*showVersion = true
		return nil
	})
	flag.Usage = func() { help.Print(version) }
	flag.Parse()

	if *showVersion {
		fmt.Printf("fprint-menu %s\n", version)
		return
	}

	program := tea.NewProgram(app.New(version, *lang), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "fprint-menu: %v\n", err)
		os.Exit(1)
	}
}
