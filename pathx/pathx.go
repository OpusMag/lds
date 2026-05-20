package pathx

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// ExpandPath expands a leading "~" to the current user's home directory.
func ExpandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	trimmed := strings.TrimPrefix(path, "~")
	if trimmed == "" {
		return usr.HomeDir, nil
	}
	return filepath.Join(usr.HomeDir, strings.TrimPrefix(trimmed, string(os.PathSeparator))), nil
}
