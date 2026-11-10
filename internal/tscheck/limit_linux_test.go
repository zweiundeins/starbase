package tscheck

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// The compiler is the OOM killer's first choice, and its data is capped.
func TestLimitMemory(t *testing.T) {
	cmd := exec.Command("sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Skip("no sleep:", err)
	}
	defer cmd.Process.Kill()
	limitMemory(cmd.Process.Pid)
	b, err := os.ReadFile("/proc/" + strconv.Itoa(cmd.Process.Pid) + "/oom_score_adj")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(b)); got != "1000" {
		t.Errorf("oom_score_adj = %s, want 1000", got)
	}
	limits, err := os.ReadFile("/proc/" + strconv.Itoa(cmd.Process.Pid) + "/limits")
	if err != nil {
		t.Fatal(err)
	}
	if want := strconv.FormatUint(maxData, 10); !strings.Contains(string(limits), want) {
		t.Errorf("RLIMIT_DATA %s not set:\n%s", want, limits)
	}
}
