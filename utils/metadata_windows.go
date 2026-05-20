//go:build windows

package utils

import "lds/fsinfo"

// EnrichFileInfo is a no-op on Windows; the subprocess-based metadata
// helpers are POSIX-only.
func EnrichFileInfo(_ *fsinfo.FileInfo) {}
