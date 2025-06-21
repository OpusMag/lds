//go:build windows

package utils

import (
	"fmt"
	"lds/config"
	"lds/logging"
	"os"
	"os/exec"

	"github.com/gdamore/tcell/v2"
)

func ChangeDirectoryAndRerunWin(directory string, up bool) {
	var cmd *exec.Cmd = exec.Command("cmd.exe", "/C", fmt.Sprintf("cd %s && lds", directory))

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
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

		var permissions, owner string
		var isExecutable, isSymlink bool
		var symlinkTarget, gitRepoStatus, lastAccessTime, creationTime string
		var size int64
		var fileType string
		var inode, hardLinksCount uint64

		permissions = "N/A"
		owner = "N/A"
		isExecutable = false
		isSymlink = false
		symlinkTarget = "N/A"
		gitRepoStatus = "N/A"
		lastAccessTime = GetLastModified(info.ModTime())
		creationTime = GetLastModified(info.ModTime())
		size = info.Size()
		fileType = GetFileType(info)
		inode = 0
		hardLinksCount = 0

		fileInfo := config.FileInfo{
			Name:           info.Name(),
			Permissions:    permissions,
			Owner:          owner,
			IsExecutable:   isExecutable,
			IsSymlink:      isSymlink,
			SymlinkTarget:  symlinkTarget,
			GitRepoStatus:  gitRepoStatus,
			LastAccessTime: lastAccessTime,
			CreationTime:   creationTime,
			Size:           size,
			FileType:       fileType,
			Inode:          inode,
			HardLinksCount: hardLinksCount,
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
