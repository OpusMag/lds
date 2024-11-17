package logging

import (
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

func ExpandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		usr, err := user.Current()
		if err != nil {
			return "", err
		}
		return filepath.Join(usr.HomeDir, path[1:]), nil
	}
	return path, nil
}

func SetupLogging(logFile string) {
	logFilePath, err := ExpandPath(logFile)
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
