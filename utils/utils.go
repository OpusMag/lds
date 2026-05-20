package utils

import (
	"fmt"
	"os"
	"strings"
	"time"

	"lds/fsinfo"
	"lds/pathx"
)

// ExpandPath is retained as a thin re-export of pathx.ExpandPath for one
// release. Prefer importing pathx directly.
func ExpandPath(path string) (string, error) {
	return pathx.ExpandPath(path)
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

func FilterFiles(files []fsinfo.FileInfo, query string) []fsinfo.FileInfo {
	if query == "" {
		return files
	}
	var filtered []fsinfo.FileInfo
	for _, file := range files {
		if strings.Contains(strings.ToLower(file.Name), strings.ToLower(query)) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

func FindBestMatch(directories, files, hiddenFiles []fsinfo.FileInfo, query string) *fsinfo.FileInfo {
	allFiles := append(directories, append(files, hiddenFiles...)...)
	for i, file := range allFiles {
		if strings.Contains(strings.ToLower(file.Name), strings.ToLower(query)) {
			return &allFiles[i]
		}
	}
	return nil
}
