package web_test

import (
	"bufio"
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zweiundeins/lith/live"
	"github.com/zweiundeins/lith/writer"

	"starbase/components"
	"starbase/content"
	"starbase/internal/app"
	"starbase/internal/catalog"
	"starbase/internal/commands"
	"starbase/internal/config"
	"starbase/internal/db"
	"starbase/internal/queries"
	"starbase/internal/tscheck"
	"starbase/internal/web"
	"starbase/static"
)

// openStream opens a tab's render stream, uncompressed, and returns a
// function that waits for its next event, and one that closes it.
func openStream(t *testing.T, c *http.Client, url, tabID string) (next func() string, stop func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(`{"tabid":"`+tabID+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Accept-Encoding", "identity")
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	events := make(chan string)
	go func() {
		defer res.Body.Close()
		defer close(events)
		sc := bufio.NewScanner(res.Body)
		sc.Buffer(make([]byte, 1<<20), 8<<20)
		var e strings.Builder
		for sc.Scan() {
			if sc.Text() != "" || e.Len() == 0 {
				e.WriteString(sc.Text() + "\n")
				continue
			}
			select {
			case events <- e.String():
			case <-ctx.Done():
				return
			}
			e.Reset()
		}
	}()
	return func() string {
		t.Helper()
		select {
		case e, ok := <-events:
			if !ok {
				t.Fatal("the stream ended")
			}
			return e
		case <-time.After(5 * time.Second):
			t.Fatal("no event for 5 s")
		}
		return ""
	}, cancel
}

// until returns the first of the stream's next events that contains want.
func until(t *testing.T, next func() string, want string) string {
	t.Helper()
	for range 10 {
		if e := next(); strings.Contains(e, want) {
			return e
		}
	}
	t.Fatalf("no event with %q in ten", want)
	return ""
}

// The Showcase's "N watching" is part of the board's view, which its
// viewers share: a join or a leave sends the others the board with the
// count, not their whole page again.
func TestShowcaseViewers(t *testing.T) {
	ts, c := newServer(t)
	if _, page := get(t, c, ts.URL+"/showcase"); !strings.Contains(page, `<span id="board-viewers">1</span> watching`) {
		t.Fatal("the document lacks the viewer count")
	}
	a, _ := openStream(t, c, ts.URL+"/showcase", "taba0000")
	until(t, a, `id="app"`)
	b, leave := openStream(t, c, ts.URL+"/showcase", "tabb0000")
	until(t, b, `<span id="board-viewers">2</span>`)
	if e := until(t, a, `<span id="board-viewers">2</span>`); strings.Contains(e, `id="app"`) {
		t.Error("a join sent the other viewer its whole page")
	}
	leave()
	if e := until(t, a, `<span id="board-viewers">1</span>`); strings.Contains(e, `id="app"`) {
		t.Error("a leave sent the other viewer its whole page")
	}
}

// stable waits until the hub has rendered nothing for 300 ms.
func stable(t *testing.T, hub *live.Hub) {
	t.Helper()
	for last, still, end := hub.Stats().Renders, 0, time.Now().Add(5*time.Second); still < 6; time.Sleep(50 * time.Millisecond) {
		if time.Now().After(end) {
			t.Fatal("the hub kept rendering for 5 s")
		}
		if now := hub.Stats().Renders; now != last {
			last, still = now, 0
		} else {
			still++
		}
	}
}

func paint(t *testing.T, c *http.Client, url string, color, cell int) {
	t.Helper()
	res := post(t, c, url+"/cmd/paint", fmt.Sprintf(`{"color":%d,"cells":[%d]}`, color, cell), "same-origin")
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("paint: %s", res.Status)
	}
}

// A paint renders the board's view alone, once for all its viewers: no
// tab's view renders again, on the Showcase or elsewhere.
func TestPaintRendersTheBoardAlone(t *testing.T) {
	s := startServer(t)
	get(t, s.c, s.ts.URL+"/showcase")
	a, _ := openStream(t, s.c, s.ts.URL+"/showcase", "painta00")
	b, _ := openStream(t, s.c, s.ts.URL+"/showcase", "paintb00")
	about, _ := openStream(t, s.c, s.ts.URL+"/about", "paintc00")
	for _, next := range []func() string{a, b, about} {
		until(t, next, `id="app"`)
	}
	stable(t, s.hub)
	before := s.hub.Stats().RendersByKey

	paint(t, s.c, s.ts.URL, 7, 0)
	for _, next := range []func() string{a, b} {
		if e := until(t, next, "· 1 pixels painted"); strings.Contains(e, `id="app"`) {
			t.Error("a paint sent a viewer its whole page")
		}
	}
	stable(t, s.hub)
	for key, n := range s.hub.Stats().RendersByKey {
		want := int64(0)
		if key == commands.TopicBoard {
			want = 1
		}
		if got := n - before[key]; got != want {
			t.Errorf("a paint rendered %s %d times, want %d", key, got, want)
		}
	}
}

// A frame of the tab's own view carries the board as the board's view
// last sent it, so the morph of #app leaves the painted board as it is.
func TestShowcaseFrameCarriesTheBoard(t *testing.T) {
	s := startServer(t)
	get(t, s.c, s.ts.URL+"/showcase")
	next, _ := openStream(t, s.c, s.ts.URL+"/showcase", "frame000")
	until(t, next, `id="app"`)
	paint(t, s.c, s.ts.URL, 7, 0)
	until(t, next, "· 1 pixels painted")

	res := post(t, s.c, s.ts.URL+"/cmd/flight", `{"tabid":"frame000","name":"thrust","value":70}`, "same-origin")
	res.Body.Close()
	e := until(t, next, `id="app"`)
	switch {
	case !strings.Contains(e, "<dd>70%</dd>"):
		t.Error("the tab's frame lacks the new thrust")
	case strings.Contains(e, "<!--region:"):
		t.Error("the tab's frame still has the board's hole")
	case strings.Count(e, `id="board-head"`) != 1 || !strings.Contains(e, `cells="7`) || !strings.Contains(e, "· 1 pixels painted"):
		t.Error("the tab's frame lacks the painted board")
	}
}

// join opens a render stream on path with accept as its Accept-Encoding
// and waits for its first bytes, or with identity for its page; then it
// reads the rest away until the stream ends.
func join(b *testing.B, ctx context.Context, client *http.Client, url, path, tabID, accept string, wg *sync.WaitGroup) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url+path, strings.NewReader(`{"tabid":"`+tabID+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Datastar-Request", "true")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Accept-Encoding", accept)
	// Each tab is a client of its own, below the cap per client.
	h := fnv.New32a()
	h.Write([]byte(tabID))
	n := h.Sum32()
	req.Header.Set("X-Forwarded-For", fmt.Sprintf("10.%d.%d.%d", n>>16&255, n>>8&255, n&255))
	res, err := client.Do(req)
	if err != nil {
		b.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		b.Fatalf("%s %s: %s", path, tabID, res.Status)
	}
	br := bufio.NewReaderSize(res.Body, 4096)
	if accept == "identity" {
		for {
			line, err := br.ReadSlice('\n')
			if err != nil && err != bufio.ErrBufferFull {
				b.Fatalf("%s: %v", tabID, err)
			}
			if strings.Contains(string(line), `id="app"`) {
				break
			}
		}
	} else if _, err := br.Peek(1); err != nil {
		b.Fatalf("%s: %v", tabID, err)
	}
	wg.Go(func() {
		io.Copy(io.Discard, br)
		res.Body.Close()
	})
}

// settle waits until n has not changed for a second.
func settle(n func() int64) {
	for last, still := n(), 0; still < 10; time.Sleep(100 * time.Millisecond) {
		if now := n(); now != last {
			last, still = now, 0
		} else {
			still++
		}
	}
}

// benchApp is the app in process for a benchmark, with the board seeded
// and the hub on app.HubOptions.
type benchApp struct {
	ctx     context.Context
	url     string
	hub     *live.Hub
	catalog *catalog.Catalog
	client  *http.Client // for up to streams streams at once
	stop    func(streams *sync.WaitGroup)
}

func startBench(b *testing.B, streams int) *benchApp {
	ctx, cancel := context.WithCancel(context.Background())
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := db.Open(ctx, filepath.Join(b.TempDir(), "bench.db"))
	if err != nil {
		b.Fatal(err)
	}
	cat, err := catalog.Load(components.FS)
	if err != nil {
		b.Fatal(err)
	}
	hub := live.New(app.HubOptions(d.R, log, config.Config{}.StreamHeartbeat()))
	hubDone := make(chan struct{})
	go func() { hub.Run(ctx); close(hubDone) }()
	wr, err := writer.New(d, writer.Options{OnCommit: hub.Changed, Log: log})
	if err != nil {
		b.Fatal(err)
	}
	go wr.Run(ctx)
	if err := wr.Do(ctx, commands.SyncCatalog{Catalog: cat, Datastar: static.Datastar()}); err != nil {
		b.Fatal(err)
	}
	if err := wr.Do(ctx, commands.SeedBoard{}); err != nil {
		b.Fatal(err)
	}
	ts := httptest.NewUnstartedServer(nil)
	ts.Start()
	cfg := config.Config{BaseURL: ts.URL, RepoURL: "https://example.com/repo"}
	srv := web.New(ctx, web.Deps{Config: cfg, Log: log, Writer: wr, Hub: hub, Queries: queries.New(d),
		Catalog: cat, StaticFS: static.FS, Content: content.FS, Checker: tscheck.New(b.TempDir())})
	ts.Config.Handler = srv.Handler()
	client := &http.Client{Transport: &http.Transport{DisableCompression: true, MaxIdleConnsPerHost: 2 * streams}}
	if res, err := client.Get(ts.URL + "/showcase"); err != nil {
		b.Fatal(err)
	} else {
		io.Copy(io.Discard, res.Body)
		res.Body.Close()
	}
	return &benchApp{ctx: ctx, url: ts.URL, hub: hub, catalog: cat, client: client, stop: func(streams *sync.WaitGroup) {
		cancel()
		ts.CloseClientConnections()
		streams.Wait()
		ts.Close()
		<-hubDone
		wr.Close()
		d.Close()
	}}
}

// BenchmarkShowcaseJoins opens 100 render streams on /showcase one after
// another, each once its page has arrived, and reports the renders and read
// transactions that took until they settled. Then it opens 100 more as
// browsers that accept zstd and reports the heap each open stream holds
// with the app's stream options.
//
//	go test ./internal/web -run '^$' -bench ShowcaseJoins -benchtime 1x
func BenchmarkShowcaseJoins(b *testing.B) {
	const streams = 100
	for range b.N {
		a := startBench(b, streams)
		renders := func() int64 { return a.hub.Stats().Renders }

		b.ResetTimer()
		r0, tx0 := renders(), a.hub.Stats().ReadTxs
		joins, stop := context.WithCancel(a.ctx)
		var wg sync.WaitGroup
		for i := range streams {
			join(b, joins, a.client, a.url, "/showcase", fmt.Sprintf("join%05d", i), "identity", &wg)
		}
		settle(renders)
		b.ReportMetric(float64(renders()-r0), "renders")
		b.ReportMetric(float64(a.hub.Stats().ReadTxs-tx0), "readtx")
		stop()
		wg.Wait()
		settle(func() int64 { return int64(a.hub.Stats().Streams) })

		runtime.GC()
		var m0, m1 runtime.MemStats
		runtime.ReadMemStats(&m0)
		for i := range streams {
			join(b, a.ctx, a.client, a.url, "/showcase", fmt.Sprintf("heap%05d", i), "gzip, deflate, br, zstd", &wg)
		}
		settle(renders)
		b.StopTimer()
		runtime.GC()
		runtime.ReadMemStats(&m1)
		b.ReportMetric(float64(int64(m1.HeapAlloc)-int64(m0.HeapAlloc))/streams/(1<<20), "MiB/stream")

		a.stop(&wg)
	}
}

// BenchmarkBoardPaint opens 150 render streams as browsers that accept
// zstd, 20 on /showcase and the rest spread over the other pages, and has
// one person paint the board for 10 s at the paint limit: a pixel every
// 50 ms. It reports the renders per tick, the renders by view (the tabs on
// /showcase, the tabs elsewhere, and the views the Showcase's viewers
// share), and the cores the process kept busy while the board was painted.
//
//	go test ./internal/web -run '^$' -bench BoardPaint -benchtime 1x
func BenchmarkBoardPaint(b *testing.B) {
	const tabs, onShowcase = 150, 20
	const paintFor, every = 10 * time.Second, 50 * time.Millisecond
	for range b.N {
		a := startBench(b, tabs)
		pages := []string{"/", "/themes", "/about", "/contribute", "/playground"}
		for _, c := range a.catalog.Components {
			if !c.Unlisted {
				pages = append(pages, "/components/"+c.Slug)
			}
		}
		var wg sync.WaitGroup
		for i := range tabs {
			path, tab := pages[i%len(pages)], fmt.Sprintf("page%04d", i)
			if i < onShowcase {
				path, tab = "/showcase", fmt.Sprintf("show%04d", i)
			}
			join(b, a.ctx, a.client, a.url, path, tab, "gzip, deflate, br, zstd", &wg)
		}
		renders := func() int64 { return a.hub.Stats().Renders }
		settle(renders)

		jar, _ := cookiejar.New(nil)
		painter := &http.Client{Jar: jar}
		paint := func(i int) int {
			body := fmt.Sprintf(`{"color":%d,"cells":[%d]}`, 1+i%15, i*7%(commands.BoardSize*commands.BoardSize))
			req, _ := http.NewRequest(http.MethodPost, a.url+"/cmd/paint", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Datastar-Request", "true")
			req.Header.Set("Sec-Fetch-Site", "same-origin")
			res, err := painter.Do(req)
			if err != nil {
				b.Fatal(err)
			}
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
			return res.StatusCode
		}

		b.ResetTimer()
		s0, cpu0, t0 := a.hub.Stats(), cpuTime(), time.Now()
		tick := time.NewTicker(every)
		painted, refused := 0, 0
		for i := 0; time.Since(t0) < paintFor; i++ {
			if paint(i) == http.StatusNoContent {
				painted++
			} else {
				refused++
			}
			<-tick.C
		}
		tick.Stop()
		cpu, wall := cpuTime()-cpu0, time.Since(t0)
		settle(renders)
		b.StopTimer()
		s1 := a.hub.Stats()
		if refused > 0 {
			b.Errorf("%d of %d paints refused", refused, painted+refused)
		}

		groups := map[string]int64{}
		views := map[string]int{}
		for k, n := range s1.RendersByKey {
			g := k
			switch {
			case strings.HasPrefix(k, "tab:show"):
				g = "showcase-tab"
			case strings.HasPrefix(k, "tab:page"):
				g = "other-tab"
			}
			groups[g] += n - s0.RendersByKey[k]
			views[g]++
		}
		ticks := s1.Ticks - s0.Ticks
		b.ReportMetric(float64(s1.Renders-s0.Renders)/float64(max(1, ticks)), "renders/tick")
		b.ReportMetric(float64(ticks), "ticks")
		b.ReportMetric(float64(cpu)/float64(wall), "cores")
		for _, g := range slices.Sorted(maps.Keys(groups)) {
			b.ReportMetric(float64(groups[g]), g+"-renders")
			b.Logf("%-18s %3d views, %6d renders", g, views[g], groups[g])
		}
		b.Logf("%d pixels painted in %v, %d frames sent", painted, wall.Round(time.Millisecond), s1.FramesSent-s0.FramesSent)

		a.stop(&wg)
	}
}
