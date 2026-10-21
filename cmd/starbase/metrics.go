package main

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zweiundeins/lith/live"
	"github.com/zweiundeins/lith/prom"
	"github.com/zweiundeins/lith/writer"
)

// namespace prefixes Starbase's metrics, starbase_writer_*, starbase_live_*
// and starbase_sse_budget_*, which the "lith apps" dashboard and alert rules
// in lith's deploy/observability read.
const namespace = "starbase"

// metricsSocket is the FileDescriptorName of deploy/starbase-metrics.socket.
const metricsSocket = "metrics"

// listenFDsStart is the first descriptor systemd passes (SD_LISTEN_FDS_START).
var listenFDsStart = 3

// metrics serves Starbase's numbers in Prometheus' text format at every path:
// the writer's, the hub's and the streams' budget's (lith/prom), the Go
// runtime's and the process's.
func metrics(w *writer.Writer, hub *live.Hub, log *slog.Logger) http.Handler {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		prom.New(namespace, prom.Sources{Writer: w, Hub: hub}), // the budget is the hub's: app.HubOptions'
	)
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{ErrorLog: slog.NewLogLogger(log.Handler(), slog.LevelWarn)})
}

// metricsListener opens the metrics' listener: the socket systemd passes as
// "metrics" if there is one, else METRICS_ADDR, else none. Either must be on
// loopback, since the metrics are not for the public.
func metricsListener(addr string) (net.Listener, error) {
	ln, err := inheritedListener(metricsSocket)
	if err != nil {
		return nil, fmt.Errorf("metrics: %w", err)
	}
	if ln == nil {
		if addr == "" {
			return nil, nil
		}
		if err := loopbackOnly(addr); err != nil {
			return nil, err
		}
		if ln, err = net.Listen("tcp", addr); err != nil {
			return nil, fmt.Errorf("metrics: %w", err)
		}
		return ln, nil
	}
	if err := loopbackOnly(ln.Addr().String()); err != nil {
		ln.Close()
		return nil, err
	}
	return ln, nil
}

func loopbackOnly(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("METRICS_ADDR: %w", err)
	}
	if ip := net.ParseIP(host); host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return fmt.Errorf("metrics on %s: the metrics listen on loopback only", addr)
	}
	return nil
}

// inheritedListener returns the socket systemd passed under name (socket
// activation: LISTEN_PID, LISTEN_FDS, LISTEN_FDNAMES), or nil. The variables
// are cleared and the descriptors closed on exec, so subprocesses get neither.
func inheritedListener(name string) (net.Listener, error) {
	pid, fds, names := os.Getenv("LISTEN_PID"), os.Getenv("LISTEN_FDS"), os.Getenv("LISTEN_FDNAMES")
	for _, k := range []string{"LISTEN_PID", "LISTEN_FDS", "LISTEN_FDNAMES"} {
		os.Unsetenv(k)
	}
	n, err := strconv.Atoi(fds)
	if pid != strconv.Itoa(os.Getpid()) || err != nil || n <= 0 {
		return nil, nil
	}
	for i := range n {
		closeOnExec(listenFDsStart + i)
	}
	for i, fdName := range strings.Split(names, ":") {
		if i >= n || fdName != name {
			continue
		}
		f := os.NewFile(uintptr(listenFDsStart+i), name)
		defer f.Close() // FileListener has a copy
		return net.FileListener(f)
	}
	return nil, nil
}
