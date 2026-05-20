//go:build windows

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"lds/fsinfo"
	"lds/logging"
)

func ChangeDirectoryAndRerun(directory string, up bool) {
	var targetDir string

	if up {
		targetDir = ".."
	} else {
		targetDir = filepath.Clean(directory)
	}

	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to resolve directory: %v\n", err)
		os.Exit(1)
	}

	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get executable path: %v\n", err)
		os.Exit(1)
	}

	cmd := exec.Command(execPath)
	cmd.Dir = absTargetDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to change directory: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func ReadDirectoryAndUpdateBestMatch(query string, showHidden bool) (directories, regularFiles []fsinfo.FileInfo, bestMatch *fsinfo.FileInfo) {
	files, err := os.ReadDir(".")
	if err != nil {
		logging.LogErrorAndExit("Error reading directory", err)
	}

	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}

		if !showHidden && strings.HasPrefix(info.Name(), ".") {
			continue
		}

		fileInfo := fsinfo.FileInfo{
			Name:           info.Name(),
			IsExecutable:   false,
			IsSymlink:      false,
			SymlinkTarget:  "N/A",
			GitRepoStatus:  "N/A",
			LastAccessTime: GetLastModified(info.ModTime()),
			CreationTime:   GetLastModified(info.ModTime()),
			Size:           info.Size(),
			FileType:       GetFileType(info),
		}

		if info.IsDir() {
			directories = append(directories, fileInfo)
		} else {
			regularFiles = append(regularFiles, fileInfo)
		}
	}

	filteredDirectories := FilterFiles(directories, query)
	filteredFiles := FilterFiles(regularFiles, query)
	bestMatch = FindBestMatch(filteredDirectories, filteredFiles, nil, query)

	return directories, regularFiles, bestMatch
}
