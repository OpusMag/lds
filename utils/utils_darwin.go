//go:build darwin

package utils

import (
	"syscall"
	"time"
)

func getTimeInfo(stat *syscall.Stat_t) (lastAccess, creation time.Time) {
	lastAccess = time.Unix(stat.Atimespec.Sec, stat.Atimespec.Nsec)
	creation = time.Unix(stat.Ctimespec.Sec, stat.Ctimespec.Nsec)
	return
}

func getHardLinksCount(stat *syscall.Stat_t) uint64 {
	return uint64(stat.Nlink)
}
