package fsinfo

// FileInfo describes a directory entry shown in the TUI. Filesystem domain
// model consumed by utils, ui, events, and app.
type FileInfo struct {
	Name           string
	Permissions    string
	Owner          string
	IsExecutable   bool
	IsSymlink      bool
	SymlinkTarget  string
	MountPoint     string
	SELinuxContext string
	GitRepoStatus  string
	LastAccessTime string
	CreationTime   string
	Size           int64
	FileType       string
	Inode          uint64
	HardLinksCount uint64
}
