package tscheck

import "golang.org/x/sys/unix"

// limitMemory caps what the compiler may allocate: past it, its runtime
// stops with "out of memory" instead of growing into the server's share.
func limitMemory(pid int) {
	unix.Prlimit(pid, unix.RLIMIT_DATA, &unix.Rlimit{Cur: maxData, Max: maxData}, nil)
}
