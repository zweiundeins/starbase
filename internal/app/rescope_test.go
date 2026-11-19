package app_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"starbase/static"
)

// TestRescopeOncePerFrame: a server frame that patches 300 elements by id inside
// five sb-details, a nested sb-details and a light sb-drag-group makes each
// host re-scope its children once (patches/rocket/0013), and the patched
// children work: their data-text and data-on react, a `$$` in them is scoped,
// an sb-card inside one renders the frame's heading, and the next frame
// re-scopes again. The official build re-scopes once per patched element;
// on it the test checks that count instead. With
// STARBASE_DATASTAR_BUNDLE=<file>, on that Datastar build.
func TestRescopeOncePerFrame(t *testing.T) {
	rescope(t)
}

// TestRescopeOncePerFrameOnOfficialBundle is TestRescopeOncePerFrame on the
// official Datastar + Rocket release.
func TestRescopeOncePerFrameOnOfficialBundle(t *testing.T) {
	if os.Getenv("STARBASE_DATASTAR_BUNDLE") == "" {
		t.Setenv("STARBASE_DATASTAR_BUNDLE", officialBundle)
	}
	rescope(t)
}

func rescope(t *testing.T) {
	patched := true
	if p := os.Getenv("STARBASE_DATASTAR_BUNDLE"); p != "" {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		patched = bytes.Equal(b, static.Datastar())
	}
	script := strings.Replace(rescopeJS, "%PATCHED%", strconv.FormatBool(patched), 1)
	copyRows(t, probeBundle(t, "/", rescopeJS0+script, map[string]http.HandlerFunc{"/__test/rescope": rescopeHandler}))
}

// TestRescopeFrameTiming measures a frame that patches 370 elements in five
// sb-details (the slots page of blind-discussion), at CPU 1x and 4x: the
// frame's wall time and its longest task. Only with STARBASE_RESCOPE_BENCH=1;
// run it once more with STARBASE_DATASTAR_BUNDLE for the official build.
func TestRescopeFrameTiming(t *testing.T) {
	if os.Getenv("STARBASE_RESCOPE_BENCH") != "1" {
		t.Skip("set STARBASE_RESCOPE_BENCH=1")
	}
	body := probeCDP(t, "/", rescopeJS0+rescopeBenchJS, map[string]http.HandlerFunc{"/__test/rescope": rescopeHandler})
	var res []struct {
		CPU      int       `json:"cpu"`
		Wall     []float64 `json:"wall"`
		Longest  []float64 `json:"longest"`
		Rescopes int       `json:"rescopes"`
		Error    string    `json:"error"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		t.Fatalf("%v: %s", err, body)
	}
	for _, r := range res {
		if r.Error != "" {
			t.Fatal(r.Error)
		}
		t.Logf("CPU %dx: frame %v ms, longest task %v ms, %d re-scopes per frame", r.CPU, r.Wall, r.Longest, r.Rescopes)
	}
}

// rescopeHandler answers ?frame=<k> with the rows of every host as one
// element patch (frame 0: the hosts with their rows, as HTML). ?hosts= and
// ?rows= size the sb-details; ?extras=0 leaves out the nested sb-details,
// the sb-drag-group and the sb-card.
func rescopeHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	k, _ := strconv.Atoi(q.Get("frame"))
	hosts, _ := strconv.Atoi(q.Get("hosts"))
	rows, _ := strconv.Atoi(q.Get("rows"))
	extras := q.Get("extras") != "0"
	row := func(id string, inner string) string {
		return fmt.Sprintf(`<div id="%s" data-frame="%d"><span class="t" data-text="'n=' + $rescope.n"></span> <button data-on:click="$rescope.n++">+</button>%s</div>`, id, k, inner)
	}
	rowsOf := func(h string, n int) []string {
		var out []string
		for i := range n {
			inner := ""
			switch {
			case extras && h == "d0" && i == 0:
				inner = fmt.Sprintf(`<sb-card id="rs-card" heading="Card %d"></sb-card>`, k)
			case extras && h == "d2" && i == 0:
				inner = `<i class="scoped" data-attr:title="$$hint"></i>`
			}
			out = append(out, row(fmt.Sprintf("rs-%s-%d", h, i), inner))
		}
		return out
	}
	if k == 0 {
		var b strings.Builder
		for h := range hosts {
			fmt.Fprintf(&b, `<sb-details id="rs-d%d" summary="Week %d" open>%s`, h, h, strings.Join(rowsOf(fmt.Sprint("d", h), rows), ""))
			if extras && h == 1 {
				fmt.Fprintf(&b, `<sb-details id="rs-dn" summary="Nested" open>%s</sb-details>`, strings.Join(rowsOf("dn", 10), ""))
			}
			b.WriteString(`</sb-details>`)
		}
		if extras {
			fmt.Fprintf(&b, `<sb-drag-group id="rs-dg"><section data-drop-list="a">%s</section></sb-drag-group>`, strings.Join(rowsOf("g", 20), ""))
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(b.String()))
		return
	}
	var all []string
	for h := range hosts {
		all = append(all, rowsOf(fmt.Sprint("d", h), rows)...)
		if extras && h == 1 {
			all = append(all, rowsOf("dn", 10)...)
		}
	}
	if extras {
		all = append(all, rowsOf("g", 20)...)
	}
	w.Header().Set("Content-Type", "text/event-stream")
	fmt.Fprint(w, "event: datastar-patch-elements\n")
	for _, l := range all {
		fmt.Fprintf(w, "data: elements %s\n", l)
	}
	fmt.Fprint(w, "\n")
}

// rescopeJS0 builds the hosts from frame 0 of ?q, counts the scope events
// each host gets, and sends frames: frame(k) resolves once Datastar has
// patched frame k and the page has settled.
const rescopeJS0 = `
const settle = (ms = 60) => new Promise((r) => setTimeout(r, ms))
const tags = ['sb-details', 'sb-card', 'sb-drag-group']
await Promise.all(tags.map((t) => customElements.whenDefined(t)))
const box = document.createElement('div')
box.setAttribute('data-signals', '{rescope: {n: 0}}')
document.body.prepend(box)
const counts = {}
const build = async (q) => {
	box.innerHTML = await (await fetch('/__test/rescope?frame=0&' + q)).text()
	await settle(200)
	for (const host of box.querySelectorAll('[data-scope-children]')) {
		counts[host.id] = 0
		host.addEventListener('datastar-scope-children', () => counts[host.id]++)
	}
}
const sender = document.createElement('button')
sender.hidden = true
sender.setAttribute('data-on:click', "@get(el.dataset.url)")
document.body.append(sender)
const frame = async (q, k, after = settle) => {
	const done = new Promise((r) => document.addEventListener('datastar-fetch', function f(e) {
		if (e.detail.el === sender && e.detail.type === 'finished') document.removeEventListener('datastar-fetch', f), r()
	}))
	sender.dataset.url = '/__test/rescope?frame=' + k + '&' + q
	await settle(0)
	sender.click()
	await done
	await after()
}
`

const rescopeJS = `
const PATCHED = %PATCHED%
const rows = []
const check = (step, got, want) => rows.push({ step, got: JSON.stringify(got), want: JSON.stringify(want) })
try {
	const q = 'hosts=5&rows=60'
	await build(q)
	// Patched elements per host: what the official build re-scopes each frame.
	const perElement = { 'rs-d0': 60, 'rs-d1': 60, 'rs-dn': 10, 'rs-d2': 60, 'rs-d3': 60, 'rs-d4': 60, 'rs-dg': 20, 'rs-card': 1 }
	const sorted = (o) => Object.fromEntries(Object.entries(o).sort(([a], [b]) => a.localeCompare(b)))
	const want = (frames) => sorted(Object.fromEntries(Object.entries(perElement).map(([id, n]) => [id, (PATCHED ? 1 : n) * frames])))
	const texts = () => [...new Set([...box.querySelectorAll('.t')].map((s) => s.textContent))]
	const cardHeading = () => document.getElementById('rs-card').shadowRoot?.textContent.includes(` + "`Card ${document.getElementById('rs-card').getAttribute('heading').slice(5)}`" + `)
	check('hosts before a frame', Object.keys(counts).sort(), Object.keys(perElement).sort())

	await frame(q, 1)
	check('re-scopes after frame 1', sorted(counts), want(1))
	check('rows carry frame 1', [...new Set([...box.querySelectorAll('[data-frame]')].map((r) => r.dataset.frame))], ['1'])
	check('data-text in every host after frame 1', texts(), ['n=0'])
	document.querySelector('#rs-d3-7 button').click()
	await settle()
	check('a click in sb-details reaches the page signal once', texts(), ['n=1'])
	document.querySelector('#rs-g-3 button').click()
	await settle()
	check('a click in the light sb-drag-group', texts(), ['n=2'])
	document.querySelector('#rs-dn-4 button').click()
	await settle()
	check('a click in the nested sb-details', texts(), ['n=3'])
	const scoped = document.querySelector('#rs-d2-0 .scoped').getAttribute('data-attr:title')
	check('$$ in a patched child is scoped', !scoped.includes('$$') && scoped.includes('_rocket'), true)
	check('the sb-card inside renders frame 1', document.getElementById('rs-card').getAttribute('heading'), 'Card 1')
	check('its shadow shows the heading', cardHeading(), true)

	await frame(q, 2)
	check('re-scopes after frame 2', sorted(counts), want(2))
	check('rows carry frame 2', [...new Set([...box.querySelectorAll('[data-frame]')].map((r) => r.dataset.frame))], ['2'])
	check('data-text after frame 2', texts(), ['n=3'])
	document.querySelector('#rs-d4-59 button').click()
	await settle()
	check('a click after frame 2 counts once', texts(), ['n=4'])
	check('the sb-card renders frame 2', cardHeading(), true)
	check('$$ still scoped after frame 2', document.querySelector('#rs-d2-0 .scoped').getAttribute('data-attr:title').includes('$$'), false)
} catch (e) {
	rows.push({ step: 'run', error: String(e?.stack || e) })
}
await fetch('/__probe/result', { method: 'POST', body: JSON.stringify(rows) })
`

const rescopeBenchJS = `
const cdp = (cmds) => fetch('/__probe/cdp', { method: 'POST', body: JSON.stringify(cmds) }).then((r) => r.json())
const out = []
try {
	const q = 'hosts=5&rows=74&extras=0'
	await build(q)
	let longest = 0
	new PerformanceObserver((l) => { for (const e of l.getEntries()) longest = Math.max(longest, e.duration) }).observe({ type: 'longtask' })
	const twoFrames = () => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)))
	let k = 0
	for (const cpu of [1, 4]) {
		await cdp([{ method: 'Emulation.setCPUThrottlingRate', params: { rate: cpu } }])
		const res = { cpu, wall: [], longest: [], rescopes: 0 }
		for (let run = 0; run < 6; run++) {
			const before = counts['rs-d0']
			longest = 0
			const t0 = performance.now()
			await frame(q, ++k, twoFrames)
			const wall = performance.now() - t0
			await settle(100)
			if (run === 0) continue // the first frame warms up
			res.wall.push(Math.round(wall))
			res.longest.push(Math.round(longest))
			res.rescopes = (counts['rs-d0'] - before) * 5
		}
		out.push(res)
	}
} catch (e) {
	out.push({ error: String(e?.stack || e) })
}
await fetch('/__probe/result', { method: 'POST', body: JSON.stringify(out) })
`
