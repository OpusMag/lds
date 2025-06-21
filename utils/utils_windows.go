//go:build windows

package utils

import (
	"fmt"
	"lds/config"
	"lds/logging"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
)

func ChangeDirectoryAndRerun(directory string, up bool) {
	var targetDir string

	if up {
		targetDir = ".."
	} else {
		targetDir = filepath.Clean(directory)
	}

	cmdStr := fmt.Sprintf(`cd /d "%s" && lds`, targetDir)
	cmd := exec.Command("cmd.exe", "/C", cmdStr)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to change directory: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func ReadDirectoryAndUpdateBestMatch(screen tcell.Screen, query string) ([]config.FileInfo, []config.FileInfo, []config.FileInfo, *config.FileInfo) {
	files, err := os.ReadDir(".")
	if err != nil {
		logging.LogErrorAndExit("Error reading directory", err)
	}

	var directories, regularFiles []config.FileInfo

	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}

		var isExecutable, isSymlink bool
		var symlinkTarget, gitRepoStatus, lastAccessTime, creationTime string
		var size int64
		var fileType string

		isExecutable = false
		isSymlink = false
		symlinkTarget = "N/A"
		gitRepoStatus = "N/A"
		lastAccessTime = GetLastModified(info.ModTime())
		creationTime = GetLastModified(info.ModTime())
		size = info.Size()
		fileType = GetFileType(info)

		fileInfo := config.FileInfo{
			Name:           info.Name(),
			IsExecutable:   isExecutable,
			IsSymlink:      isSymlink,
			SymlinkTarget:  symlinkTarget,
			GitRepoStatus:  gitRepoStatus,
			LastAccessTime: lastAccessTime,
			CreationTime:   creationTime,
			Size:           size,
			FileType:       fileType,
		}

		if info.IsDir() {
			directories = append(directories, fileInfo)
		} else {
			regularFiles = append(regularFiles, fileInfo)
		}
	}

	filteredDirectories := FilterFiles(directories, query)
	filteredFiles := FilterFiles(regularFiles, query)

	bestMatch := FindBestMatch(filteredDirectories, filteredFiles, nil, query)

	return filteredDirectories, filteredFiles, nil, bestMatch
}
