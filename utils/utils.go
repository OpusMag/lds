package utils

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"lds/config"
	"lds/logging"

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
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe", "/C", fmt.Sprintf("cd %s && lds", directory))
	} else if !up {
		cmd = exec.Command("sh", "-c", fmt.Sprintf("cd %s && lds", directory))
	} else if up {
		cmd = exec.Command("sh", "-c", fmt.Sprintf("cd .. %s && lds", directory))
	}
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

		// Get file permissions and owner
		var permissions, owner string
		var isExecutable, isSymlink bool
		var symlinkTarget, mountPoint, selinuxContext, gitRepoStatus, lastAccessTime, creationTime string
		var size int64
		var fileType string
		var inode, hardLinksCount uint64

		if runtime.GOOS != "windows" {
			stat := info.Sys().(*syscall.Stat_t)
			uid := stat.Uid
			gid := stat.Gid
			usr, _ := user.LookupId(fmt.Sprint(uid))
			grp, _ := user.LookupGroupId(fmt.Sprint(gid))
			permissions = info.Mode().String()
			owner = fmt.Sprintf("%s:%s", usr.Username, grp.Name)
			isExecutable = info.Mode()&0111 != 0
			isSymlink, symlinkTarget = getSymlinkStatus(file)
			mountPoint = getMountPoint(info)
			selinuxContext = getSELinuxContext(info)
			gitRepoStatus = getGitRepoStatus(file)
			lastAccessTime = GetLastModified(time.Unix(stat.Atim.Sec, stat.Atim.Nsec))
			creationTime = GetLastModified(time.Unix(stat.Ctim.Sec, stat.Ctim.Nsec))
			size = info.Size()
			fileType = GetFileType(info)
			inode = stat.Ino
			hardLinksCount = stat.Nlink
		} else {
			permissions = "N/A"
			owner = "N/A"
			isExecutable = false
			isSymlink = false
			symlinkTarget = "N/A"
			mountPoint = "N/A"
			selinuxContext = "N/A"
			gitRepoStatus = "N/A"
			lastAccessTime = GetLastModified(info.ModTime())
			creationTime = GetLastModified(info.ModTime())
			size = info.Size()
			fileType = GetFileType(info)
			inode = 0
			hardLinksCount = 0
		}

		fileInfo := config.FileInfo{
			Name:           info.Name(),
			Permissions:    permissions,
			Owner:          owner,
			IsExecutable:   isExecutable,
			IsSymlink:      isSymlink,
			SymlinkTarget:  symlinkTarget,
			MountPoint:     mountPoint,
			SELinuxContext: selinuxContext,
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

	// Filter file lists based on user input
	filteredDirectories := FilterFiles(directories, query)
	filteredFiles := FilterFiles(regularFiles, query)

	// Find the best match
	bestMatch := FindBestMatch(filteredDirectories, filteredFiles, nil, query)

	return filteredDirectories, filteredFiles, nil, bestMatch
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

// Check if the file is a symlink and get the target
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
	if runtime.GOOS == "windows" {
		return "N/A" // Mount points are not applicable on Windows in the same way
	}
	// Implement logic to get mount point details for Unix-like systems
	cmd := exec.Command("findmnt", "-n", "-o", "TARGET", "--target", info.Name())
	output, err := cmd.Output()
	if err != nil {
		return "N/A"
	}
	return strings.TrimSpace(string(output))
}

// Get the SELinux context of the file
func getSELinuxContext(info os.FileInfo) string {
	if runtime.GOOS != "linux" {
		return "N/A" // SELinux is specific to Linux
	}
	// Implement logic to get SELinux context for Linux
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

// Check if the file or directory is part of a Git repository
func getGitRepoStatus(file os.DirEntry) string {
	if file.IsDir() {
		// Check if the directory contains a .git directory
		gitDir := fmt.Sprintf("%s/.git", file.Name())
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			return "Not a git repository"
		}
		return "Git repository"
	} else {
		// Check if the file is part of a Git repository
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

// How long since the file was last modified
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

// Function to find the best match based on user input
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
