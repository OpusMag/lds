//go:build linux

package utils

import (
	"syscall"
	"time"
)

func getTimeInfo(stat *syscall.Stat_t) (lastAccess, creation time.Time) {
	lastAccess = time.Unix(stat.Atim.Sec, stat.Atim.Nsec)
	creation = time.Unix(stat.Ctim.Sec, stat.Ctim.Nsec)
	return
}

func getHardLinksCount(stat *syscall.Stat_t) uint64 {
	return stat.Nlink
}
