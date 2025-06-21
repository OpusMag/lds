//go:build unix

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"syscall"
	"time"

	"lds/config"
	"lds/logging"

	"github.com/gdamore/tcell/v2"
)

func ChangeDirectoryAndRerunUnix(directory string, up bool) {
	var cmd *exec.Cmd
	if !up {
		cmd = exec.Command("sh", "-c", fmt.Sprintf("cd %s && lds", directory))
	} else if up {
		cmd = exec.Command("sh", "-c", fmt.Sprintf("cd .. %s && lds", directory))
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
	os.Exit(0)
}

func extractFileInfo(info os.FileInfo) (lastAccessTime, creationTime string, size int64, fileType string, inode uint64, hardLinksCount uint64) {
	stat := info.Sys().(*syscall.Stat_t)

	lastAccess, creation := getTimeInfo(stat)

	lastAccessTime = GetLastModified(lastAccess)
	creationTime = GetLastModified(creation)
	size = info.Size()
	fileType = GetFileType(info)
	inode = stat.Ino
	hardLinksCount = getHardLinksCount(stat)

	return
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

		// Extract Unix-specific file information using helper functions
		lastAccessTime, creationTime, size, fileType, inode, hardLinksCount := extractFileInfo(info)

		// Get ownership information
		stat := info.Sys().(*syscall.Stat_t)
		owner := getOwnerInfo(stat)

		// Get symlink status
		isSymlink, symlinkTarget := getSymlinkStatus(file)

		// Check if file is executable
		isExecutable := info.Mode()&0111 != 0

		fileInfo := config.FileInfo{
			Name:           info.Name(),
			Permissions:    info.Mode().String(),
			Owner:          owner,
			IsExecutable:   isExecutable,
			IsSymlink:      isSymlink,
			SymlinkTarget:  symlinkTarget,
			MountPoint:     getMountPoint(info),
			SELinuxContext: getSELinuxContext(info),
			GitRepoStatus:  getGitRepoStatus(file),
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

	// Use common filtering functions
	filteredDirectories := FilterFiles(directories, query)
	filteredFiles := FilterFiles(regularFiles, query)
	bestMatch := FindBestMatch(filteredDirectories, filteredFiles, nil, query)

	return filteredDirectories, filteredFiles, nil, bestMatch
}

func getOwnerInfo(stat *syscall.Stat_t) string {
	uid := stat.Uid
	gid := stat.Gid

	usr, _ := user.LookupId(fmt.Sprint(uid))
	grp, _ := user.LookupGroupId(fmt.Sprint(gid))

	var username, groupname string
	if usr != nil {
		username = usr.Username
	} else {
		username = fmt.Sprint(uid)
	}
	if grp != nil {
		groupname = grp.Name
	} else {
		groupname = fmt.Sprint(gid)
	}

	return fmt.Sprintf("%s:%s", username, groupname)
}

func getSymlinkStatus(file os.DirEntry) (bool, string) {
	if file.Type()&os.ModeSymlink != 0 {
		target, err := os.Readlink(file.Name())
		if err != nil {
			return true, "unknown"
		}
		return true, target
	}
	return false, ""
}

// Get the mount point of the file
func getMountPoint(info os.FileInfo) string {
	cmd := exec.Command("findmnt", "-n", "-o", "TARGET", "--target", info.Name())
	output, err := cmd.Output()
	if err != nil {
		return "N/A"
	}
	return strings.TrimSpace(string(output))
}

func getSELinuxContext(info os.FileInfo) string {
	cmd := exec.Command("ls", "-Z", info.Name())
	output, err := cmd.Output()
	if err != nil {
		return "N/A"
	}
	parts := strings.Fields(string(output))
	if len(parts) > 3 {
		return parts[3] // SELinux context is usually the 4th field
	}
	return "N/A"
}

func getGitRepoStatus(file os.DirEntry) string {
	if file.IsDir() {
		gitDir := fmt.Sprintf("%s/.git", file.Name())
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			return "Not a git repository"
		}
		return "Git repository"
	} else {
		cmd := exec.Command("git", "status", "--porcelain", file.Name())
		output, err := cmd.Output()
		if err != nil {
			return "Not a git repository"
		}
		if len(output) == 0 {
			return "Clean"
		}
		return "Modified"
	}
}

func getLastModified(modTime time.Time) string {
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
