package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/zweiundeins/lith/live"
	"github.com/zweiundeins/lith/writer"

	"starbase/internal/app"
	"starbase/internal/config"
	"starbase/internal/db"
)

func TestMetricsListener(t *testing.T) {
	for _, addr := range []string{"0.0.0.0:0", ":0", "192.0.2.1:9000", "example.com:9000", "8198"} {
		if ln, err := metricsListener(addr); err == nil {
			ln.Close()
			t.Errorf("METRICS_ADDR=%s: accepted", addr)
		}
	}
	if ln, err := metricsListener(""); ln != nil || err != nil {
		t.Errorf("no METRICS_ADDR: %v, %v; want none", ln, err)
	}
	ln, err := metricsListener("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ln.Close()
}

// TestInheritedListener: a socket systemd passes as "metrics" wins over
// METRICS_ADDR, and one off loopback is refused.
func TestInheritedListener(t *testing.T) {
	pass := func(t *testing.T, addr string) {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			t.Skip(err)
		}
		defer ln.Close()
		f, err := ln.(*net.TCPListener).File()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { f.Close() })
		old := listenFDsStart
		listenFDsStart = int(f.Fd())
		t.Cleanup(func() { listenFDsStart = old })
		t.Setenv("LISTEN_PID", strconv.Itoa(os.Getpid()))
		t.Setenv("LISTEN_FDS", "1")
		t.Setenv("LISTEN_FDNAMES", metricsSocket)
	}

	pass(t, "127.0.0.1:0")
	ln, err := metricsListener("127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	if ln.Addr().(*net.TCPAddr).Port == 1 {
		t.Error("listened on METRICS_ADDR, not on the socket systemd passed")
	}
	ln.Close()
	if v := os.Getenv("LISTEN_FDS"); v != "" {
		t.Errorf("LISTEN_FDS=%s stays for subprocesses", v)
	}

	t.Run("off loopback", func(t *testing.T) {
		pass(t, "0.0.0.0:0")
		if ln, err := metricsListener(""); err == nil {
			ln.Close()
			t.Error("a socket on every interface: accepted")
		}
	})
}

// TestDashboard checks that every metric of lith/prom (writer_, live_ and
// sse_budget_), the Go runtime and the process that the "lith apps"
// dashboard and alert rules in lith's deploy/observability read is one
// Starbase exports. Their other rows are an app's own: requests by route
// (http_), statements (db_) and lith/events (events_), which Starbase does
// not export.
func TestDashboard(t *testing.T) {
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/zweiundeins/lith").Output()
	dir := strings.TrimSpace(string(out))
	if err != nil || dir == "" {
		t.Skipf("lith's files are not at hand (go list: %v)", err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "starbase.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	w, err := writer.New(d, writer.Options{Log: log})
	if err != nil {
		t.Fatal(err)
	}
	hub := live.New(app.HubOptions(d.R, log, config.Config{}.StreamHeartbeat()))
	srv := httptest.NewServer(metrics(w, hub, log))
	defer srv.Close()
	res, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()

	exported := map[string]bool{}
	for line := range strings.Lines(string(body)) {
		if strings.HasPrefix(line, "#") {
			continue
		}
		if name, _, ok := strings.Cut(line, "{"); ok {
			exported[name] = true
		} else if name, _, ok := strings.Cut(line, " "); ok {
			exported[name] = true
		}
	}
	if !exported["starbase_sse_budget_limit_bytes"] {
		t.Fatalf("no starbase_sse_budget_limit_bytes:\n%.2000s", body)
	}
	wanted := map[string]string{}
	for _, file := range []string{"deploy/observability/lith-apps.json", "deploy/observability/rules-lith-apps.yaml"} {
		b, err := os.ReadFile(filepath.Join(dir, file))
		if err != nil {
			t.Fatal(err)
		}
		for _, re := range []*regexp.Regexp{namespaced, anyNamespace} {
			for _, m := range re.FindAllStringSubmatch(string(b), -1) {
				for _, family := range []string{"writer_", "live_", "sse_budget_"} {
					if strings.HasPrefix(m[1], family) {
						wanted[namespace+"_"+m[1]] = file
					}
				}
			}
		}
		for _, name := range goProcess.FindAllString(string(b), -1) {
			if strings.HasPrefix(name, "go_") || runtime.GOOS == "linux" { // the process collector reads /proc
				wanted[name] = file
			}
		}
	}
	if len(wanted) < 15 {
		t.Fatalf("found only %d metric names in the dashboard and the rules", len(wanted))
	}
	for name, file := range wanted {
		// The budget counts its streams by setting only while one is open.
		if !exported[name] && name != namespace+"_sse_budget_streams" {
			t.Errorf("%s reads %s, which Starbase does not export", file, name)
		}
	}
}

var (
	namespaced   = regexp.MustCompile(`\$\{ns\}_([a-z0-9_]+)`)
	anyNamespace = regexp.MustCompile(`__name__=~\\?"\.\+_([a-z0-9_]+)\\?"`)
	goProcess    = regexp.MustCompile(`\b(go_[a-z_]+|process_[a-z_]+)\b`)
)

// TestMetricsPort checks that the socket unit, the env and the scrape file
// name the same port, and that the service takes the socket.
func TestMetricsPort(t *testing.T) {
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join("../../deploy", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	socket, env, target, unit := read("starbase-metrics.socket"), read("starbase.env.example"),
		read("observability/starbase.metrics.yml"), read("starbase.service")
	m := regexp.MustCompile(`\nListenStream=127\.0\.0\.1:(\d+)\n`).FindStringSubmatch(socket)
	if m == nil {
		t.Fatal("the socket unit listens on no 127.0.0.1:<port>")
	}
	addr := "127.0.0.1:" + m[1]
	if !strings.Contains(socket, "\nFileDescriptorName="+metricsSocket+"\n") {
		t.Errorf("the socket unit has no FileDescriptorName=%s", metricsSocket)
	}
	if !strings.Contains(env, "\nMETRICS_ADDR="+addr+"\n") {
		t.Errorf("the socket listens on %s, but the env has:\n%s", addr, env)
	}
	if !strings.Contains(target, "- targets: ['"+addr+"']") {
		t.Errorf("the socket listens on %s, but Prometheus scrapes:\n%s", addr, target)
	}
	if !strings.Contains(unit, "\nSockets=starbase-metrics.socket\n") {
		t.Error("the service does not take starbase-metrics.socket")
	}
	if !strings.Contains(unit, "\nIPAddressDeny=localhost\n") {
		t.Error("the service may reach localhost")
	}
}
