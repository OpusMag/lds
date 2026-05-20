//go:build unix

package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"lds/fsinfo"
)

type metaKey struct {
	Path    string
	ModTime time.Time
}

type metaVal struct {
	Mount   string
	SELinux string
	Git     string
}

var metaCache sync.Map // metaKey -> metaVal

// EnrichFileInfo populates MountPoint, SELinuxContext, and GitRepoStatus
// on fi by invoking the corresponding subprocesses. Results are cached by
// (absolute path, mtime) for the lifetime of the process. No-op if fi is
// nil or its name cannot be resolved.
func EnrichFileInfo(fi *fsinfo.FileInfo) {
	if fi == nil {
		return
	}
	// Validate: filename must be the literal os.DirEntry name from the
	// initial scan. Reject NUL.
	if fi.Name == "" || strings.ContainsRune(fi.Name, 0x00) {
		return
	}
	abs, err := filepath.Abs(fi.Name)
	if err != nil {
		return
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return
	}
	// Skip enrichment for symlinks; resolving them invites TOCTOU and
	// the metadata wouldn't describe the symlink itself anyway.
	if info.Mode()&os.ModeSymlink != 0 {
		return
	}
	key := metaKey{Path: abs, ModTime: info.ModTime()}
	if cached, ok := metaCache.Load(key); ok {
		v := cached.(metaVal)
		fi.MountPoint = v.Mount
		fi.SELinuxContext = v.SELinux
		fi.GitRepoStatus = v.Git
		return
	}
	v := metaVal{
		Mount:   getMountPoint(abs),
		SELinux: getSELinuxContext(abs),
		Git:     getGitRepoStatus(abs, info.IsDir()),
	}
	actual, _ := metaCache.LoadOrStore(key, v)
	v = actual.(metaVal)
	fi.MountPoint = v.Mount
	fi.SELinuxContext = v.SELinux
	fi.GitRepoStatus = v.Git
}

// getMountPoint resolves the mount point of the given path. The "--"
// separator prevents names beginning with "-" from being parsed as
// options. The name passed in is the literal directory entry name from
// os.ReadDir; it is not subject to shell expansion because exec.Command
// uses an argv slice rather than sh -c.
func getMountPoint(name string) string {
	cmd := exec.Command("findmnt", "-n", "-o", "TARGET", "--target", "--", name)
	output, err := cmd.Output()
	if err != nil {
		return "N/A"
	}
	return strings.TrimSpace(string(output))
}

// getSELinuxContext returns the SELinux context for the given path. "--"
// terminates option parsing so leading-dash names cannot be mistaken for
// flags.
func getSELinuxContext(name string) string {
	cmd := exec.Command("ls", "-Z", "--", name)
	output, err := cmd.Output()
	if err != nil {
		return "N/A"
	}
	parts := strings.Fields(string(output))
	if len(parts) > 3 {
		return parts[3]
	}
	return "N/A"
}

// getGitRepoStatus reports the git-repo status for the given path.
// For directories, it checks for a .git child. For files, it runs
// `git status --porcelain -- <name>`; "--" terminates options before the
// pathspec.
func getGitRepoStatus(name string, isDir bool) string {
	if isDir {
		gitDir := filepath.Join(name, ".git")
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			return "Not a git repository"
		}
		return "Git repository"
	}
	cmd := exec.Command("git", "status", "--porcelain", "--", name)
	output, err := cmd.Output()
	if err != nil {
		return "Not a git repository"
	}
	if len(output) == 0 {
		return "Clean"
	}
	return "Modified"
}
