package app

import (
	"fmt"
	"log"
	"os"
	"strings"
)

const logPath = "/tmp/fprint-menu.log"

var appLog *log.Logger
var debugLog *log.Logger

func ConfigureLogging(level string) error {
	normalized := strings.ToLower(strings.TrimSpace(level))
	switch normalized {
	case "", "off", "none", "disabled":
		appLog = nil
		debugLog = nil
		return nil
	case "error", "info":
		logger, err := openLogger()
		if err != nil {
			return err
		}
		appLog = logger
		debugLog = nil
		appLog.Printf("logging enabled level=%s path=%s", normalized, logPath)
		return nil
	case "debug":
		logger, err := openLogger()
		if err != nil {
			return err
		}
		appLog = logger
		debugLog = logger
		debugLog.Printf("logging enabled level=debug path=%s", logPath)
		return nil
	default:
		return fmt.Errorf("unsupported log level %q (use off, error, info, or debug)", level)
	}
}

func openLogger() (*log.Logger, error) {
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	return log.New(file, "", log.LstdFlags|log.Lmicroseconds), nil
}
