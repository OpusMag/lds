//go:build freebsd

package utils

import (
	"syscall"
	"time"
)

func getTimeInfo(stat *syscall.Stat_t) (lastAccess, creation time.Time) {
	lastAccess = time.Unix(int64(stat.Atimespec.Sec), int64(stat.Atimespec.Nsec))
	creation = time.Unix(int64(stat.Birthtimespec.Sec), int64(stat.Birthtimespec.Nsec))
	return
}

func getHardLinksCount(stat *syscall.Stat_t) uint64 {
	return uint64(stat.Nlink)
}
