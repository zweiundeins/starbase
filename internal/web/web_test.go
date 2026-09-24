package web_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"github.com/andybalholm/brotli"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"starbase/components"
	"starbase/content"
	"starbase/internal/catalog"
	"starbase/internal/commands"
	"starbase/internal/config"
	"starbase/internal/cqrs"
	"starbase/internal/db"
	"starbase/internal/queries"
	"starbase/internal/web"
	"starbase/static"
)

func newServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	ts, c, _, _ := newServerBus(t)
	return ts, c
}

// newServerBus also returns the bus and catalog, for tests that send commands.
func newServerBus(t *testing.T) (*httptest.Server, *http.Client, *cqrs.Bus, *catalog.Catalog) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := db.Open(ctx, filepath.Join(t.TempDir(), "web.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	cat, err := catalog.Load(components.FS)
	if err != nil {
		t.Fatal(err)
	}
	hub := cqrs.NewHub()
	bus := cqrs.NewBus(d.W, hub, log)
	go bus.Run(ctx)
	if err := bus.Exec(ctx, commands.SyncCatalog{Catalog: cat}); err != nil {
		t.Fatal(err)
	}
	if err := bus.Exec(ctx, commands.SeedDemo{}); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewUnstartedServer(nil)
	ts.Start()
	cfg := config.Config{BaseURL: ts.URL, RepoURL: "https://example.com/repo"}
	srv := web.New(ctx, web.Deps{Config: cfg, Log: log, Bus: bus, Hub: hub, Queries: queries.New(d.R),
		Catalog: cat, StaticFS: static.FS, Content: content.FS})
	ts.Config.Handler = srv.Handler()
	t.Cleanup(func() { cancel(); ts.Close() })
	jar, _ := cookiejar.New(nil)
	return ts, &http.Client{Jar: jar}, bus, cat
}

func get(t *testing.T, c *http.Client, url string) (*http.Response, string) {
	t.Helper()
	res, err := c.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res, string(b)
}

func post(t *testing.T, c *http.Client, url, body, site string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Datastar-Request", "true")
	if site != "" {
		req.Header.Set("Sec-Fetch-Site", site)
	}
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestPages(t *testing.T) {
	ts, c := newServer(t)
	for path, want := range map[string]int{
		"/":                              200,
		"/?cat=forms&q=but":              200,
		"/components/button":             200,
		"/components/missing":            404,
		"/themes":                        200,
		"/contribute":                    200,
		"/about":                         200,
		"/showcase":                      200,
		"/does/not/exist":                404,
		"/c/index.js":                    200,
		"/c/button/button.js":            200,
		"/c/button/README.md":            404, // only a component's .js files are public
		"/c/code-editor/vendor/prism.js": 200,
		"/c/code-editor/manifest.json":   404,
		"/art/hero.svg":                  200,
	} {
		res, _ := get(t, c, ts.URL+path)
		if res.StatusCode != want {
			t.Errorf("GET %s = %d, want %d", path, res.StatusCode, want)
		}
	}
}

func TestDocumentHasNonceAndStream(t *testing.T) {
	ts, c := newServer(t)
	res, body := get(t, c, ts.URL+"/")
	csp := res.Header.Get("Content-Security-Policy")
	m := regexp.MustCompile(`'nonce-([^']+)'`).FindStringSubmatch(csp)
	if m == nil {
		t.Fatalf("CSP without nonce: %q", csp)
	}
	if !strings.Contains(body, `<script type="importmap" nonce="`+m[1]+`">`) {
		t.Error("import map does not carry the CSP nonce")
	}
	if !strings.Contains(body, `data-init="@post(location.pathname + location.search`) {
		t.Error("page does not open its render stream")
	}
	if strings.Contains(body, "/dev/reload") {
		t.Error("dev reload leaked into a non-dev build")
	}
}

func TestCommandsRequireSameOrigin(t *testing.T) {
	ts, c := newServer(t)
	get(t, c, ts.URL+"/") // obtain a session cookie
	body := `{"tabid":"tab12345","q":"modal"}`
	if res := post(t, c, ts.URL+"/cmd/browse", body, "cross-site"); res.StatusCode != http.StatusForbidden {
		t.Errorf("cross-site = %d, want 403", res.StatusCode)
	}
	if res := post(t, c, ts.URL+"/cmd/browse", body, "same-origin"); res.StatusCode != http.StatusNoContent {
		t.Errorf("same-origin = %d, want 204", res.StatusCode)
	}
	if res := post(t, c, ts.URL+"/cmd/browse", `{"tabid":"x"}`, "same-origin"); res.StatusCode != http.StatusBadRequest {
		t.Errorf("bad tab = %d, want 400", res.StatusCode)
	}
}

// TestRenderStream drives the full CQRS loop: open a tab's stream, send a
// command, and receive a new frame that reflects it.
func TestRenderStream(t *testing.T) {
	ts, c := newServer(t)
	get(t, c, ts.URL+"/")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/", strings.NewReader(`{"tabid":"tab12345"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Accept-Encoding", "identity")
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("content type = %q", ct)
	}
	frames := make(chan string)
	go func() {
		sc := bufio.NewScanner(res.Body)
		sc.Buffer(make([]byte, 1<<20), 8<<20)
		var frame strings.Builder
		for sc.Scan() {
			if sc.Text() == "" && frame.Len() > 0 {
				frames <- frame.String()
				frame.Reset()
				continue
			}
			frame.WriteString(sc.Text() + "\n")
		}
		close(frames)
	}()
	next := func() string {
		select {
		case f, ok := <-frames:
			if !ok {
				t.Fatal("stream closed")
			}
			return f
		case <-ctx.Done():
			t.Fatal("timed out waiting for a frame")
		}
		return ""
	}

	cat, _ := catalog.Load(components.FS)
	listed := 0
	for _, c := range cat.Components {
		if !c.Unlisted {
			listed++
		}
	}
	first := next()
	if n := strings.Count(first, `class="card"`); !strings.Contains(first, "event: datastar-patch-elements") || n != listed {
		t.Fatalf("first frame should list all %d cards, got %d", listed, n)
	}
	if r := post(t, c, ts.URL+"/cmd/browse", `{"tabid":"tab12345","cat":"feedback","sort":"name"}`, "same-origin"); r.StatusCode != 204 {
		t.Fatalf("command = %d", r.StatusCode)
	}
	second := next()
	feedback := 0
	for _, c := range cat.Components {
		if c.Category == "feedback" {
			feedback++
		}
	}
	if n := strings.Count(second, `class="card"`); n != feedback {
		t.Fatalf("filtered frame has %d cards, want %d", n, feedback)
	}
	if !regexp.MustCompile(`replaceState\(null, &#39;&#39;, &#34;/\?cat=feedback\\u0026sort=name&#34;\)`).MatchString(second) {
		i := strings.Index(second, "replaceState")
		t.Errorf("frame does not sync the URL: %s", second[i:min(len(second), i+80)])
	}
}

func TestPlaygroundRendered(t *testing.T) {
	ts, c := newServer(t)
	_, body := get(t, c, ts.URL+"/components/slider")
	for _, want := range []string{
		`class="playground" data-ignore-morph data-signals:_pg=`,
		`data-bind:_pg.value__prop.value`,
		`data-attr:value="$_pg.value"`,
		`data-attr:show-value="$_pg.showValue ? null : 'false'"`,
		`<code data-text="`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("slider page lacks %q", want)
		}
	}
}

func TestTelemetryStream(t *testing.T) {
	ts, c := newServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/demo/telemetry", nil)
	req.Header.Set("Accept-Encoding", "identity")
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	sc := bufio.NewScanner(res.Body)
	events := 0
	for sc.Scan() && events < 2 {
		line := sc.Text()
		if line == "event: datastar-patch-signals" {
			events++
		}
		if strings.HasPrefix(line, "data: signals") && !strings.Contains(line, `"_tm":{"t":`) {
			t.Fatalf("unexpected signals line %q", line)
		}
	}
	if events < 2 {
		t.Fatalf("got %d telemetry events in 2s", events)
	}
}

func TestTelemetryIsAFlight(t *testing.T) {
	base := time.UnixMilli(0)
	pad := web.TelemetryAt(base.Add(2 * time.Second))
	climb := web.TelemetryAt(base.Add(40 * time.Second))
	orbit := web.TelemetryAt(base.Add(80 * time.Second))
	if pad.Stage != "PAD" || climb.Stage != "ASCENT" || orbit.Stage != "ORBIT" {
		t.Fatalf("stages: %s %s %s", pad.Stage, climb.Stage, orbit.Stage)
	}
	if !(pad.Alt == 0 && climb.Alt > 0 && orbit.Alt > climb.Alt && climb.Fuel < pad.Fuel) {
		t.Fatalf("not a flight: %+v %+v %+v", pad, climb, orbit)
	}
}

func TestPlaygroundRunner(t *testing.T) {
	ts, c := newServer(t)
	res, body := get(t, c, ts.URL+"/playground/run")
	csp := res.Header.Get("Content-Security-Policy")
	for _, want := range []string{"script-src " + ts.URL + " 'unsafe-inline' 'unsafe-eval' blob:", "frame-ancestors " + ts.URL, "connect-src " + ts.URL} {
		if !strings.Contains(csp, want) {
			t.Errorf("runner CSP lacks %q: %s", want, csp)
		}
	}
	if res.Header.Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Error("runner must be frameable by the site")
	}
	if !strings.Contains(body, `<script type="importmap">{"imports":{"datastar":"`+ts.URL+`/static/vendor/datastar-rocket-`) {
		t.Error("runner lacks the absolute import map")
	}
	// Assets must be loadable from the sandbox's opaque origin.
	for _, p := range []string{"/c/index.js", "/c/button/button.js", "/static/vendor/datastar-rocket.js"} {
		res, _ := get(t, c, ts.URL+p)
		if res.Header.Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("%s lacks Access-Control-Allow-Origin: *", p)
		}
	}
}

func TestCodePlaygroundPage(t *testing.T) {
	ts, c := newServer(t)
	_, body := get(t, c, ts.URL+"/playground?component=voxel")
	for _, want := range []string{`<sb-code-playground initial="`, `rocket(&#39;sb-voxel&#39;`, `deps="{`, `data-ignore-morph`, `aria-current="page">Playground`} {
		if !strings.Contains(body, want) {
			t.Errorf("playground page lacks %q", want)
		}
	}
	if res, _ := get(t, c, ts.URL+"/playground?component=nope"); res.StatusCode != 404 {
		t.Errorf("unknown component = %d, want 404", res.StatusCode)
	}
	_, body = get(t, c, ts.URL+"/components/button")
	if !strings.Contains(body, `href="/playground?component=button"`) {
		t.Error("component page lacks Open in playground")
	}
}

// TestShareFlow: saving a snippet is a command; the tab's stream then shows
// the share link and moves the address bar to it.
func TestShareFlow(t *testing.T) {
	ts, c := newServer(t)
	get(t, c, ts.URL+"/playground")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/playground", strings.NewReader(`{"tabid":"tab12345"}`))
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Accept-Encoding", "identity")
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	frames := make(chan string, 4)
	go func() {
		sc := bufio.NewScanner(res.Body)
		sc.Buffer(make([]byte, 1<<20), 8<<20)
		var f strings.Builder
		for sc.Scan() {
			if sc.Text() == "" && f.Len() > 0 {
				frames <- f.String()
				f.Reset()
				continue
			}
			f.WriteString(sc.Text() + "\n")
		}
	}()
	<-frames // initial frame
	save := post(t, c, ts.URL+"/cmd/snippet", `{"tabid":"tab12345","component":"button","files":{"component.js":"rocket('sb-shared', {})","index.html":"<sb-shared></sb-shared>"}}`, "same-origin")
	if save.StatusCode != http.StatusNoContent {
		t.Fatalf("save = %d", save.StatusCode)
	}
	var frame string
	select {
	case frame = <-frames:
	case <-ctx.Done():
		t.Fatal("no frame after saving")
	}
	m := regexp.MustCompile(`/playground\?s=([A-Za-z0-9]{8})`).FindStringSubmatch(frame)
	if m == nil {
		t.Fatalf("frame lacks a share link")
	}
	id := m[1]
	if !strings.Contains(frame, `href="/submit?s=`+id+`"`) {
		t.Error("frame lacks Submit as component")
	}
	// The link opens the saved code; the JSON is public; /submit prefills.
	_, body := get(t, c, ts.URL+"/playground?s="+id)
	if !strings.Contains(body, "sb-shared") {
		t.Error("shared link does not load the code")
	}
	r2, js := get(t, c, ts.URL+"/playground/snippet/"+id)
	if r2.Header.Get("Access-Control-Allow-Origin") != "*" || !strings.Contains(js, `"component":"button"`) {
		t.Errorf("snippet JSON = %s", js)
	}
	noRedirect := &http.Client{Jar: c.Jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	r3, err := noRedirect.Get(ts.URL + "/submit?s=" + id)
	if err != nil {
		t.Fatal(err)
	}
	loc := r3.Header.Get("Location")
	if !strings.Contains(loc, "template=new-component.yml") || !strings.Contains(loc, "source="+url.QueryEscape(ts.URL+"/playground?s="+id)) || !strings.Contains(loc, "name=Shared") {
		t.Errorf("submit redirect = %s", loc)
	}
	if r, _ := get(t, c, ts.URL+"/playground?s=Zzzzzzzz"); r.StatusCode != 404 {
		t.Errorf("unknown snippet = %d", r.StatusCode)
	}
}

func TestAutoloader(t *testing.T) {
	ts, c := newServer(t)
	res, js := get(t, c, ts.URL+"/c/autoloader.js")
	if res.Header.Get("Access-Control-Allow-Origin") != "*" || !strings.Contains(res.Header.Get("Content-Type"), "javascript") {
		t.Fatalf("headers: %v", res.Header)
	}
	for _, want := range []string{`"sb-button":"button@`, `"sb-code-playground":["sb-code-editor"]`, "export const discover", "new MutationObserver"} {
		if !strings.Contains(js, want) {
			t.Errorf("autoloader lacks %q", want)
		}
	}
	_, page := get(t, c, ts.URL+"/?load=auto") // the default is the bundle, see TestLoadingStrategy
	if !strings.Contains(page, `src="/c/autoloader.js?v=`) {
		t.Error("?load=auto pages don't use the autoloader")
	}
}

func TestAutoloaderCloak(t *testing.T) {
	ts, c := newServer(t)
	_, js := get(t, c, ts.URL+"/c/autoloader.js")
	for _, want := range []string{"export const ready", "classList.remove('sb-cloak')", "setTimeout(uncloak, 3000)", "customElements.whenDefined(tag)", "reportError"} {
		if !strings.Contains(js, want) {
			t.Errorf("autoloader lacks %q", want)
		}
	}
	_, page := get(t, c, ts.URL+"/about")
	if !strings.Contains(page, `class="sb-cloak"`) {
		t.Error("site pages should start cloaked")
	}
}

func TestSnippetSaveRateLimit(t *testing.T) {
	ts, c := newServer(t)
	c.Get(ts.URL + "/playground") // session cookie
	body := `{"tabid":"tab12345","files":{"component.js":"rocket('sb-x', {})"}}`
	codes := map[int]int{}
	for range 12 {
		res := post(t, c, ts.URL+"/cmd/snippet", body, "same-origin")
		res.Body.Close()
		codes[res.StatusCode]++
	}
	if codes[http.StatusNoContent] != 10 || codes[http.StatusTooManyRequests] != 2 {
		t.Fatalf("status codes = %v, want 10×204 then 429s (burst 10)", codes)
	}
}

func TestSiteThemeFromCookie(t *testing.T) {
	ts, c := newServer(t)
	for _, tc := range []struct{ cookie, html, scheme string }{
		{"", `<html lang="en" class="sb-cloak">`, "dark light"},
		{"auto", `<html lang="en" class="sb-cloak">`, "dark light"},
		{"terminal", `<html lang="en" class="sb-cloak" data-sb-theme="terminal">`, "dark"},
		{"daylight", `<html lang="en" class="sb-cloak" data-sb-theme="daylight">`, "light"},
		{`x"><script>`, `<html lang="en" class="sb-cloak">`, "dark light"},
	} {
		req, _ := http.NewRequest("GET", ts.URL+"/about", nil)
		if tc.cookie != "" {
			req.AddCookie(&http.Cookie{Name: "sb-theme", Value: tc.cookie})
		}
		res, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if !strings.Contains(string(b), tc.html) || !strings.Contains(string(b), `<meta name="color-scheme" content="`+tc.scheme+`">`) {
			t.Errorf("cookie %q: want %s and color-scheme %q", tc.cookie, tc.html, tc.scheme)
		}
	}
	res, err := c.Get(ts.URL + "/theme/auto.css")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.Contains(string(b), "@media (prefers-color-scheme: light)") || !strings.Contains(string(b), ":root:not([data-sb-theme])") || !strings.Contains(string(b), "color-scheme: light") {
		t.Errorf("auto.css:\n%s", b)
	}
}

func TestVersionedURLs(t *testing.T) {
	ts, c, bus, cat := newServerBus(t)
	button, _ := cat.Get("button")

	// The current version of a module: immutable and open to other sites.
	res, js := get(t, c, ts.URL+"/c/"+button.VersionedScript())
	if res.StatusCode != 200 || !strings.Contains(res.Header.Get("Cache-Control"), "immutable") || res.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("versioned module: %d %v", res.StatusCode, res.Header)
	}
	if catalog.SRI([]byte(js)) != button.Integrity {
		t.Error("the module's bytes don't match its integrity")
	}
	for _, bad := range []string{"button@000000000000/button.js", "button@" + button.Hash + "/README.md", "button@nothex/button.js", "@000000000000/autoloader.js"} {
		if r, _ := get(t, c, ts.URL+"/c/"+bad); r.StatusCode != 404 {
			t.Errorf("%s = %d, want 404", bad, r.StatusCode)
		}
	}

	// The snapshot: its autoloader loads versioned modules, and every file
	// in its import map serves exactly the bytes its hash says.
	res, auto := get(t, c, ts.URL+"/c/@"+cat.Hash+"/autoloader.js")
	if res.StatusCode != 200 || !strings.Contains(auto, `"sb-button":"../`+button.VersionedMinScript()+`"`) {
		t.Fatalf("snapshot autoloader: %d (it should load the minified modules)", res.StatusCode)
	}

	// The minified module: served, immutable, smaller, and its relative
	// imports point at the minified siblings (code-editor imports Prism).
	res, minJS := get(t, c, ts.URL+"/c/"+button.VersionedMinScript())
	if res.StatusCode != 200 || !strings.Contains(res.Header.Get("Cache-Control"), "immutable") || len(minJS) >= len(js) {
		t.Fatalf("minified module: %d, %d bytes vs %d readable", res.StatusCode, len(minJS), len(js))
	}
	editor, _ := cat.Get("code-editor")
	if _, ed := get(t, c, ts.URL+"/c/"+editor.VersionedMinScript()); !strings.Contains(ed, `"./vendor/prism.min.js"`) || strings.Contains(ed, `"./vendor/prism.js"`) {
		t.Error("the minified code-editor should import the minified Prism")
	}
	if r, _ := get(t, c, ts.URL+"/c/"+editor.Slug+"@"+editor.Hash+"/vendor/prism.min.js"); r.StatusCode != 200 {
		t.Errorf("minified vendored file: %d", r.StatusCode)
	}
	_, mapJSON := get(t, c, ts.URL+"/c/@"+cat.Hash+"/importmap.json")
	var im struct{ Integrity map[string]string }
	if err := json.Unmarshal([]byte(mapJSON), &im); err != nil || len(im.Integrity) < len(cat.Components)+1 {
		t.Fatalf("import map: %v, %d entries", err, len(im.Integrity))
	}
	for u, sri := range im.Integrity {
		_, body := get(t, c, u)
		if catalog.SRI([]byte(body)) != sri {
			t.Errorf("%s does not match its integrity", u)
		}
	}

	// A version that is no longer in the catalog keeps working (from the database).
	old := fstest.MapFS{
		"gone/README.md": {Data: []byte("---\nname: Gone\ntag: sb-gone\ncategory: forms\nsummary: Removed since.\nauthor: someone\nsince: 2026-01-01\npreview: <sb-gone></sb-gone>\n---\nDocs.\n")},
		"gone/gone.js":   {Data: []byte("rocket('sb-gone', {})")},
	}
	oldCat, err := catalog.Load(old)
	if err != nil {
		t.Fatal(err)
	}
	if err := bus.Exec(context.Background(), commands.SyncCatalog{Catalog: oldCat}); err != nil {
		t.Fatal(err)
	}
	gone, _ := oldCat.Get("gone")
	if r, body := get(t, c, ts.URL+"/c/"+gone.VersionedScript()); r.StatusCode != 200 || body != "rocket('sb-gone', {})" {
		t.Errorf("old version = %d %q", r.StatusCode, body)
	}
	if r, _ := get(t, c, ts.URL+"/c/@"+oldCat.Hash+"/autoloader.js"); r.StatusCode != 200 {
		t.Errorf("old snapshot = %d", r.StatusCode)
	}
}

func TestCompression(t *testing.T) {
	ts, _ := newServer(t)
	c := &http.Client{Transport: &http.Transport{DisableCompression: true}} // see the raw encoding
	for _, tc := range []struct{ path, accept, want string }{
		{"/", "br, gzip", "br"},
		{"/", "gzip", "gzip"},
		{"/", "", ""},
		{"/c/autoloader.js", "br", "br"},
		{"/healthz", "br", ""}, // too small to bother
	} {
		req, _ := http.NewRequest("GET", ts.URL+tc.path, nil)
		if tc.accept != "" {
			req.Header.Set("Accept-Encoding", tc.accept)
		}
		res, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, res.Body)
		res.Body.Close()
		if got := res.Header.Get("Content-Encoding"); got != tc.want {
			t.Errorf("%s with %q: encoding %q, want %q", tc.path, tc.accept, got, tc.want)
		}
		if tc.want != "" && !slices.Contains(res.Header.Values("Vary"), "Accept-Encoding") {
			t.Errorf("%s: Vary = %v", tc.path, res.Header.Values("Vary"))
		}
	}
}

func TestSEO(t *testing.T) {
	ts, c := newServer(t)
	res, robots := get(t, c, ts.URL+"/robots.txt")
	if res.StatusCode != 200 || !strings.Contains(robots, "Sitemap: "+ts.URL+"/sitemap.xml") || !strings.Contains(robots, "Disallow: /cmd/") {
		t.Errorf("robots.txt:\n%s", robots)
	}
	_, sm := get(t, c, ts.URL+"/sitemap.xml")
	if !strings.Contains(sm, "<loc>"+ts.URL+"/components/button</loc>") || !strings.Contains(sm, "<lastmod>") {
		t.Errorf("sitemap lacks components:\n%.400s", sm)
	}
	for _, p := range []string{"/og.png", "/apple-touch-icon.png", "/favicon.ico"} {
		if r, body := get(t, c, ts.URL+p); r.StatusCode != 200 || r.Header.Get("Content-Type") != "image/png" || !strings.HasPrefix(body, "\x89PNG") {
			t.Errorf("%s: %d %s", p, r.StatusCode, r.Header.Get("Content-Type"))
		}
	}
	_, page := get(t, c, ts.URL+"/components/button?x=1")
	for _, want := range []string{
		`<link rel="canonical" href="` + ts.URL + `/components/button">`,
		`<meta property="og:image" content="` + ts.URL + `/og.png">`,
		`<meta name="twitter:card" content="summary_large_image">`,
		`"@type":"SoftwareSourceCode"`, `"@type":"WebSite"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("component page lacks %s", want)
		}
	}
}

func TestFlightCommand(t *testing.T) {
	ts, c := newServer(t)
	get(t, c, ts.URL+"/showcase") // session cookie
	for _, tc := range []struct {
		body string
		want int
	}{
		{`{"tabid":"tab12345","name":"thrust","value":70}`, http.StatusNoContent},
		{`{"tabid":"tab12345","name":"shields","value":false}`, http.StatusNoContent},
		{`{"tabid":"tab12345","name":"thrust","value":95}`, http.StatusBadRequest},
	} {
		res := post(t, c, ts.URL+"/cmd/flight", tc.body, "same-origin")
		res.Body.Close()
		if res.StatusCode != tc.want {
			t.Errorf("%s: %d, want %d", tc.body, res.StatusCode, tc.want)
		}
	}
}

// A component the site itself uses (sb-code-playground) is documented and
// served, but not part of the gallery.
func TestUnlistedComponent(t *testing.T) {
	ts, c := newServer(t)
	_, home := get(t, c, ts.URL+"/")
	if strings.Contains(home, `data-slug="code-playground"`) {
		t.Error("the gallery lists an unlisted component")
	}
	if res, _ := get(t, c, ts.URL+"/components/code-playground"); res.StatusCode != 200 {
		t.Errorf("its page must stay reachable: %d", res.StatusCode)
	}
	if res, _ := get(t, c, ts.URL+"/c/code-playground/code-playground.js"); res.StatusCode != 200 {
		t.Errorf("its module must stay served: %d", res.StatusCode)
	}
	_, sitemap := get(t, c, ts.URL+"/sitemap.xml")
	if strings.Contains(sitemap, "/components/code-playground") {
		t.Error("the sitemap lists an unlisted component")
	}
}

// Immutable assets go out precompressed: brotli when accepted, decoding to
// exactly the original bytes, and never compressed a second time by the
// middleware.
func TestPrecompressedAssets(t *testing.T) {
	ts, _, _, cat := newServerBus(t)
	button, _ := cat.Get("button")
	client := &http.Client{Transport: &http.Transport{DisableCompression: true}} // see the raw encoding
	for _, u := range []string{"/c/" + button.VersionedMinScript(), "/c/" + button.VersionedScript(), "/c/autoloader.js", "/c/bundle.js"} {
		want, _ := func() ([]byte, error) {
			req, _ := http.NewRequest("GET", ts.URL+u, nil)
			res, err := client.Do(req) // identity
			if err != nil {
				return nil, err
			}
			defer res.Body.Close()
			return io.ReadAll(res.Body)
		}()
		req, _ := http.NewRequest("GET", ts.URL+u, nil)
		req.Header.Set("Accept-Encoding", "gzip, br")
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		got, _ := io.ReadAll(brotli.NewReader(res.Body))
		res.Body.Close()
		if res.Header.Get("Content-Encoding") != "br" || !bytes.Equal(got, want) || len(want) == 0 {
			t.Errorf("%s: Content-Encoding %q, round trip ok=%v (%d bytes)", u, res.Header.Get("Content-Encoding"), bytes.Equal(got, want), len(want))
		}
	}
}

// Pages load every component as one file while the bundle fits its budget;
// ?load=auto forces the autoloader (for measuring), and over budget the
// autoloader is the default.
func TestLoadingStrategy(t *testing.T) {
	ts, c := newServer(t)
	_, page := get(t, c, ts.URL+"/about")
	if !strings.Contains(page, `src="/c/bundle.js?v=`) || strings.Contains(page, `src="/c/autoloader.js?v=`) {
		t.Error("a page should load the bundle by default")
	}
	_, page = get(t, c, ts.URL+"/about?load=auto")
	if !strings.Contains(page, `src="/c/autoloader.js?v=`) || strings.Contains(page, `src="/c/bundle.js?v=`) {
		t.Error("?load=auto should load the autoloader")
	}
	if res, body := get(t, c, ts.URL+"/c/bundle.js"); res.StatusCode != 200 || len(body) < 10_000 {
		t.Fatalf("bundle: %d, %d bytes", res.StatusCode, len(body))
	}
}
