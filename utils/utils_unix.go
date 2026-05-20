//go:build unix

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"

	"lds/fsinfo"
)

func ChangeDirectoryAndRerun(directory string, up bool) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get cwd: %v\n", err)
		os.Exit(1)
	}
	var targetDir string
	if up {
		targetDir = filepath.Dir(cwd)
	} else {
		targetDir = filepath.Clean(filepath.Join(cwd, directory))
	}

	info, err := os.Stat(targetDir)
	if err != nil || !info.IsDir() {
		if err == nil {
			err = fmt.Errorf("%s is not a directory", targetDir)
		}
		fmt.Fprintf(os.Stderr, "Failed to change directory: %v\n", err)
		os.Exit(1)
	}

	if err := os.Chdir(targetDir); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to chdir: %v\n", err)
		os.Exit(1)
	}

	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to locate executable: %v\n", err)
		os.Exit(1)
	}

	if err := syscall.Exec(exePath, os.Args, os.Environ()); err != nil {
		cmd := exec.Command(exePath, os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err2 := cmd.Run(); err2 != nil {
			fmt.Fprintf(os.Stderr, "Failed to rerun: %v\n", err2)
			os.Exit(1)
		}
		os.Exit(0)
	}
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

// ReadDirectoryAndUpdateBestMatch scans the current directory and returns
// directories, regular files, and the best match for the supplied query.
// Subprocess-based metadata (mount point, SELinux context, git status) is
// deferred to EnrichFileInfo so it is only computed for the highlighted
// entry.
func ReadDirectoryAndUpdateBestMatch(query string, showHidden bool) (directories, regularFiles []fsinfo.FileInfo, bestMatch *fsinfo.FileInfo, err error) {
	files, readErr := os.ReadDir(".")
	if readErr != nil {
		err = readErr
		return
	}

	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}

		if !showHidden && strings.HasPrefix(info.Name(), ".") {
			continue
		}

		lastAccessTime, creationTime, size, fileType, inode, hardLinksCount := extractFileInfo(info)

		stat := info.Sys().(*syscall.Stat_t)
		owner := getOwnerInfo(stat)

		isSymlink, symlinkTarget := getSymlinkStatus(file)

		isExecutable := info.Mode()&0111 != 0

		fileInfo := fsinfo.FileInfo{
			Name:           info.Name(),
			Permissions:    info.Mode().String(),
			Owner:          owner,
			IsExecutable:   isExecutable,
			IsSymlink:      isSymlink,
			SymlinkTarget:  symlinkTarget,
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
	bestMatch = FindBestMatch(filteredDirectories, filteredFiles, nil, query)

	return directories, regularFiles, bestMatch, nil
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
