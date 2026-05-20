package logging

import (
	"fmt"
	"log"
	"os"

	"lds/pathx"
)

func SetupLogging(logFile string) error {
	logFilePath, err := pathx.ExpandPath(logFile)
	if err != nil {
		return fmt.Errorf("failed to expand log file path: %w", err)
	}
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	log.SetOutput(file)
	return nil
}

func LogErrorAndExit(message string, err error) {
	log.Printf("%s: %v\n", message, err)
	os.Exit(1)
}
