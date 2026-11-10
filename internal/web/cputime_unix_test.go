//go:build unix

package web_test

import (
	"syscall"
	"time"
)

// cpuTime is the CPU time the process has used, user and system.
func cpuTime() time.Duration {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return 0
	}
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano())
}
