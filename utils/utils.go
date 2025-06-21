package utils

import (
	"fmt"
	"lds/config"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
)

func ExpandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		usr, err := user.Current()
		if err != nil {
			return "", err
		}
		return filepath.Join(usr.HomeDir, path[1:], "/"), nil
	}
	return path, nil
}

func GetFileType(info os.FileInfo) string {
	switch mode := info.Mode(); {
	case mode.IsRegular():
		return "Regular File"
	case mode.IsDir():
		return "Directory"
	case mode&os.ModeSymlink != 0:
		return "Symlink"
	case mode&os.ModeNamedPipe != 0:
		return "Named Pipe"
	case mode&os.ModeSocket != 0:
		return "Socket"
	case mode&os.ModeDevice != 0:
		return "Device"
	default:
		return "Unknown"
	}
}

func GetLastModified(modTime time.Time) string {
	duration := time.Since(modTime)
	if duration.Hours() < 24 {
		return fmt.Sprintf("%d hours ago", int(duration.Hours()))
	} else if duration.Hours() < 24*30 {
		return fmt.Sprintf("%d days ago", int(duration.Hours()/24))
	} else if duration.Hours() < 24*365 {
		return fmt.Sprintf("%d months ago", int(duration.Hours()/(24*30)))
	} else {
		return fmt.Sprintf("%d years ago", int(duration.Hours()/(24*365)))
	}
}

func ChangeDirectoryAndRerun(directory string, up bool) {
	var cmd *exec.Cmd = exec.Command("cmd.exe", "/C", fmt.Sprintf("cd %s && lds", directory))

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
	os.Exit(0)
}

type FileSystemProvider interface {
	ReadDirectoryAndUpdateBestMatch(screen tcell.Screen, query string) ([]config.FileInfo, []config.FileInfo, []config.FileInfo, *config.FileInfo)
	ChangeDirectoryAndRerun(directory string, up bool)
	GetFileType(info os.FileInfo) string
	GetLastModified(modTime time.Time) string
	ExpandPath(path string) (string, error)
}

func FilterFiles(files []config.FileInfo, query string) []config.FileInfo {
	if query == "" {
		return files
	}
	var filtered []config.FileInfo
	for _, file := range files {
		if strings.Contains(strings.ToLower(file.Name), strings.ToLower(query)) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

func FindBestMatch(directories, files, hiddenFiles []config.FileInfo, query string) *config.FileInfo {
	allFiles := append(directories, append(files, hiddenFiles...)...)
	var bestMatch *config.FileInfo
	for _, file := range allFiles {
		if strings.Contains(strings.ToLower(file.Name), strings.ToLower(query)) {
			bestMatch = &file
			break
		}
	}
	return bestMatch
}
