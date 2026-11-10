package tscheck

import (
	"os"
	"strconv"

	"golang.org/x/sys/unix"
)

// limitMemory caps what the compiler may allocate: past it, its runtime
// stops with "out of memory" instead of growing into the server's share.
// It also makes the compiler the first process the kernel's OOM killer
// picks, so a cgroup at its MemoryMax loses a type check, not the server.
func limitMemory(pid int) {
	unix.Prlimit(pid, unix.RLIMIT_DATA, &unix.Rlimit{Cur: maxData, Max: maxData}, nil)
	os.WriteFile("/proc/"+strconv.Itoa(pid)+"/oom_score_adj", []byte("1000"), 0)
}
