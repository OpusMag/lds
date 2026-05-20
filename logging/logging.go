package logging

import (
	"log"
	"os"

	"lds/pathx"
)

func SetupLogging(logFile string) {
	logFilePath, err := pathx.ExpandPath(logFile)
	if err != nil {
		log.Fatalf("Failed to expand log file path: %v", err)
	}
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	log.SetOutput(file)
}

func LogErrorAndExit(message string, err error) {
	log.Printf("%s: %v\n", message, err)
	os.Exit(1)
}
