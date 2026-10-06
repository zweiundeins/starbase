package app_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

// TestVirtualScrollFocus checks in Chrome that sb-virtual-scroll keeps the focus on its item across
// windows. Tab is focus() on the next focusable element in flat-tree order, as the browser does it.
//
// STARBASE_DATASTAR_BUNDLE=<file> makes the pages load that bundle in place of the vendored build,
// such as the official release: https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.4/bundles/datastar-rocket.js
func TestVirtualScrollFocus(t *testing.T) {
	var bundle []byte
	if p := os.Getenv("STARBASE_DATASTAR_BUNDLE"); p != "" {
		var err error
		if bundle, err = os.ReadFile(p); err != nil {
			t.Fatal(err)
		}
	}
	links := vscrollItems{links: 2}
	cases := []struct {
		name, path, script string
		items              vscrollItems
		stars              bool // the page shows the demo stars
	}{
		{"tab from item 0", "/", vscrollFocusJS + vscrollTab0JS, links, false},
		{"tab from item 5000", "/", vscrollFocusJS + vscrollTab5000JS, links, false},
		{"outside", "/", vscrollFocusJS + vscrollOutsideJS, links, false},
		{"header", "/", vscrollFocusJS + vscrollHeaderJS, links, false},
		{"data-table", "/components/data-table", vscrollHelpersJS + vscrollTableJS, links, true},
		{"shrink", "/", vscrollFocusJS + vscrollShrinkJS, links, false},
		{"focusable items", "/", vscrollFocusJS + vscrollFocusableJS, vscrollItems{links: 2, focusable: true}, false},
		{"component in an item", "/", vscrollFocusJS + vscrollComponentJS, vscrollItems{links: 2, b: "checkbox"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			script := strings.Replace(c.script, "HOST_HTML", jsonString(t, vscrollHost(0, 160, vscrollTotal, c.items)), 1)
			var served atomic.Int32
			handlers := func(app http.Handler) map[string]http.HandlerFunc {
				if c.stars {
					waitForStars(t, app)
				}
				m := map[string]http.HandlerFunc{"/__probe/window": vscrollServer(c.items)}
				if bundle != nil {
					m[datastarPath(t, app)] = func(w http.ResponseWriter, r *http.Request) {
						served.Add(1)
						w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
						w.Write(bundle)
					}
				}
				return m
			}
			_, body := probeWith(t, c.path, script, handlers)
			if bundle != nil && served.Load() == 0 {
				t.Error("the page did not load STARBASE_DATASTAR_BUNDLE")
			}
			var rows []struct{ Case, Step, Got, Want, Error string }
			if err := json.Unmarshal(body, &rows); err != nil {
				t.Fatalf("%v: %s", err, body)
			}
			if len(rows) == 0 {
				t.Fatal("no rows")
			}
			for _, r := range rows {
				switch {
				case r.Error != "":
					t.Errorf("%s: %s", r.Case, r.Error)
				case r.Got != r.Want:
					t.Errorf("%s, %s: got %s, want %s", r.Case, r.Step, r.Got, r.Want)
				}
			}
		})
	}
}

const vscrollTotal = 10000

// vscrollItems is what each item of the test list holds.
type vscrollItems struct {
	links     int    // up to two, a and b
	b         string // b as a link (""), a disabled button ("disabled") or an sb-checkbox ("checkbox")
	focusable bool   // the item takes the focus itself (tabindex="-1"), like a grid's rows
	dead      string // the index of an item that holds only disabled buttons ("": none)
}

// vscrollHost is the test list of total items holding offset to offset+count: a header of 8
// buttons, and items without ids, like /demo/data/list.
func vscrollHost(offset, count, total int, items vscrollItems) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<sb-virtual-scroll id="vsf" offset="%d" total="%d" data-preserve-attr="role aria-label item-size buffer data-on:sb-window">`, offset, total)
	b.WriteString(`<div slot="header">`)
	for i := 1; i <= 8; i++ {
		fmt.Fprintf(&b, `<button type="button">H%d</button>`, i)
	}
	b.WriteString(`</div>`)
	row := ""
	if items.focusable {
		row = ` tabindex="-1"`
	}
	for i := offset; i < offset+count; i++ {
		fmt.Fprintf(&b, `<div role="row" aria-rowindex="%d"%s>`, i+2, row)
		if strconv.Itoa(i) == items.dead {
			fmt.Fprintf(&b, `<button type="button" disabled>%d a</button> <button type="button" disabled>%d b</button></div>`, i, i)
			continue
		}
		if items.links > 0 {
			fmt.Fprintf(&b, `<a href="#r%da">%d a</a>`, i, i)
		}
		if items.links > 1 {
			switch items.b {
			case "disabled":
				fmt.Fprintf(&b, ` <button type="button" disabled>%d b</button>`, i)
			case "checkbox":
				fmt.Fprintf(&b, ` <sb-checkbox label="%d b" data-k="#r%db"></sb-checkbox>`, i, i)
			default:
				fmt.Fprintf(&b, ` <a href="#r%db">%d b</a>`, i, i)
			}
		}
		if items.links == 0 {
			fmt.Fprintf(&b, "%d", i)
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`</sb-virtual-scroll>`)
	return b.String()
}

// vscrollServer answers the test list's sb-window like /demo/data/list: the host with ?count=
// items from ?offset=. A request with ?total= (and ?links=, ?b=) is a new list, as a filter sends.
func vscrollServer(items vscrollItems) http.HandlerFunc {
	var mu sync.Mutex
	total := vscrollTotal
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		mu.Lock()
		if s := q.Get("total"); s != "" {
			total, _ = strconv.Atoi(s)
			items.links, _ = strconv.Atoi(q.Get("links"))
			items.b = q.Get("b")
			items.dead = q.Get("dead")
		}
		n, it := total, items
		mu.Unlock()
		offset, _ := strconv.Atoi(q.Get("offset"))
		count, _ := strconv.Atoi(q.Get("count"))
		offset = min(max(offset, 0), n)
		count = min(max(count, 0), n-offset)
		datastar.NewSSE(w, r).PatchElements(vscrollHost(offset, count, n, it))
	}
}

// datastarPath is the path the app's pages load Datastar from (its import map).
func datastarPath(t *testing.T, app http.Handler) string {
	t.Helper()
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	m := regexp.MustCompile(`"datastar":"([^"]+)"`).FindStringSubmatch(rec.Body.String())
	if m == nil {
		t.Fatal("no Datastar in the import map of /")
	}
	u, err := url.Parse(m[1])
	if err != nil {
		t.Fatal(err)
	}
	return u.Path
}

// waitForStars returns once SeedStars, which the app runs in the background at
// startup, has stored the stars (slow under -race), so Chrome starts after it.
func waitForStars(t *testing.T, app http.Handler) {
	t.Helper()
	for deadline := time.Now().Add(3 * time.Minute); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		rec, req := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/demo/data/rows?offset=0&count=1", nil)
		req.Header.Set("Accept", "application/json")
		app.ServeHTTP(rec, req)
		var got struct{ Total int }
		if json.Unmarshal(rec.Body.Bytes(), &got) == nil && got.Total > 0 {
			return
		}
	}
	t.Fatal("no demo stars after 3 minutes")
}

func jsonString(t *testing.T, s string) string {
	t.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// vscrollHelpersJS: frames, the deep focus, Tab in flat-tree order and the report.
const vscrollHelpersJS = `
const rows = []
const check = (c, step, got, want) => rows.push({ case: c, step: String(step), got: String(got), want: String(want) })
const frame = () => new Promise((r) => requestAnimationFrame(() => r()))
const frames = async (n) => { for (let i = 0; i < n; i++) await frame() }
const deep = () => {
	let a = document.activeElement
	while (a?.shadowRoot?.activeElement) a = a.shadowRoot.activeElement
	return a
}
// An element in an sb-checkbox's shadow root names the checkbox by its data-k.
const key = (el) => el?.getAttribute('href') ?? el?.getRootNode().host?.dataset?.k
const describe = (el) => {
	if (!el) return 'nothing'
	const part = el.getAttribute('part'), k = key(el)
	return el.localName + (part ? '::part(' + part + ')' + (k ? ' of ' + k : '') : k ? ' ' + k : '')
}
const TABBABLE = 'a[href], button, input, select, textarea, summary, [tabindex], [contenteditable]'
const kids = (n) => {
	if (n.localName === 'slot') {
		const a = n.assignedNodes()
		return a.length ? a : n.childNodes
	}
	return (n.shadowRoot ?? n).childNodes
}
const tabbables = (n, out) => {
	for (const c of kids(n)) {
		if (c.nodeType !== 1) continue
		if (c.matches(TABBABLE) && c.tabIndex >= 0 && !c.disabled && c.checkVisibility()) out.push(c)
		tabbables(c, out)
	}
	return out
}
// Tab (Shift+Tab with back): the next focusable element in flat-tree order.
const tab = (back) => {
	const all = tabbables(document.body, [])
	const next = all[all.indexOf(deep()) + (back ? -1 : 1)]
	next?.focus()
	return next
}
// Two frames; after a window asked for since the last settle, until the lists ask for nothing
// and two frames more, since a list restores the focus a frame after a window.
let asked = 0, seen = 0
document.addEventListener('sb-window', () => asked++, true)
const settle = async (...lists) => {
	const loading = () => lists.some((l) => l.matches(':state(loading)'))
	await frames(2)
	for (let i = 0; i < 1000 && (asked !== seen || loading()); i++) {
		seen = asked
		for (let j = 0; j < 1000 && loading(); j++) await frames(1)
		await frames(2)
	}
}
const report = () => fetch('/__probe/result', { method: 'POST', body: JSON.stringify(rows) })
`

// vscrollFocusJS builds the test list (10,000 items of 30 px, a 600 px buffer, the first 160
// items inline) after a button outside it.
const vscrollFocusJS = vscrollHelpersJS + `
await customElements.whenDefined('sb-virtual-scroll')
const css = document.createElement('style')
css.textContent = '#vsf { block-size: 320px } #vsf > div { display: flex; gap: 8px; align-items: center } #vsf > [slot=header] { block-size: 30px; background: Canvas }'
document.head.append(css)
const outside = document.createElement('button')
outside.textContent = 'Outside'
const tpl = document.createElement('template')
tpl.innerHTML = HOST_HTML
const vs = tpl.content.firstElementChild
vs.setAttribute('role', 'table')
vs.setAttribute('aria-label', 'Rows')
vs.setAttribute('item-size', '30')
vs.setAttribute('buffer', '600')
vs.setAttribute('data-on:sb-window', "@get('/__probe/window?offset=' + evt.detail.offset + '&count=' + evt.detail.count)")
document.body.append(outside, vs)
await frames(3)
await settle(vs)
const scroller = vs.shadowRoot.querySelector('[part=scroller]')
const link = (i, l = 'a') => vs.querySelector('a[href="#r' + i + l + '"]')
const inList = (el) => {
	for (; el; el = el.parentNode ?? el.host) if (el === vs) return true
	return false
}
let windows = 0
vs.addEventListener('sb-window', () => windows++)
// n steps of Tab (or Shift+Tab) from link k (item k >> 1, a or b): each one
// lands on the next link, inside the list.
const walk = async (c, k, n, back) => {
	for (let s = 1; s <= n; s++) {
		tab(back)
		await settle(vs)
		k += back ? -1 : 1
		const want = '#r' + (k >> 1) + (k & 1 ? 'b' : 'a'), a = deep()
		if (!inList(a)) return void check(c, 'step ' + s, describe(a), want + ' (in the list)')
		if (key(a) !== want) return void check(c, 'step ' + s, describe(a), want)
	}
	rows.push({ case: c, step: n + ' steps', got: 'ok', want: 'ok' })
	return k
}
`

const vscrollTab0JS = `
try {
	link(0).focus()
	await settle(vs)
	await walk('1: Tab from item 0', 0, 400)
	check('1: Tab from item 0', 'windows on the way', windows > 0, true)
} catch (e) {
	rows.push({ case: '1', error: String(e?.stack || e) })
}
await report()
`

const vscrollTab5000JS = `
try {
	vs.scrollToIndex(5000)
	await settle(vs)
	link(5000).focus()
	await settle(vs)
	let before = windows
	const k = await walk('2: Tab from item 5000', 10000, 400)
	check('2: Tab from item 5000', 'windows on the way', windows > before, true)
	if (k) {
		before = windows
		await walk('2: Shift+Tab back', k, 100, true)
		check('2: Shift+Tab back', 'windows on the way', windows > before, true)
	}
} catch (e) {
	rows.push({ case: '2', error: String(e?.stack || e) })
}
await report()
`

const vscrollOutsideJS = `
try {
	let c = '3: focus outside'
	vs.scrollToIndex(300)
	await settle(vs)
	link(305).focus()
	await settle(vs)
	outside.focus()
	let before = windows
	scroller.scrollTop += 3000
	await settle(vs)
	check(c, 'a window landed', windows > before, true)
	check(c, 'after a window', document.activeElement === outside ? 'the outside button' : describe(deep()), 'the outside button')

	c = '3: focus on the body'
	link(400).focus()
	await settle(vs)
	link(400).blur()
	before = windows
	scroller.scrollTop += 3000
	await settle(vs)
	check(c, 'a window landed', windows > before, true)
	check(c, 'after a window', describe(document.activeElement), 'body')

	c = '4: an item scrolled out of the window'
	vs.scrollToIndex(100)
	await settle(vs)
	link(100).focus()
	await settle(vs)
	vs.scrollToIndex(5000)
	await settle(vs)
	check(c, 'scrollToIndex(5000): the focus', describe(deep()), 'div::part(scroller)')
	check(c, 'scrollToIndex(5000): the scroller\'s tabindex', scroller.getAttribute('tabindex'), '-1')
	vs.scrollToIndex(90)
	await settle(vs)
	check(c, 'scrollToIndex(90): the focus', describe(deep()), 'a #r100a')
	check(c, 'scrollToIndex(90): scrollTop', scroller.scrollTop, 2700)
	check(c, 'scrollToIndex(90): the scroller\'s tabindex', scroller.getAttribute('tabindex'), 'null')
} catch (e) {
	rows.push({ case: '3 and 4', error: String(e?.stack || e) })
}
await report()
`

const vscrollHeaderJS = `
try {
	let c = '5: focus in the header'
	vs.scrollToIndex(5000)
	await settle(vs)
	const top = scroller.scrollTop
	const buttons = [...vs.querySelectorAll('[slot=header] button')]
	for (const [j, b] of buttons.entries()) {
		b.focus()
		await settle(vs)
		check(c, 'focus() on H' + (j + 1) + ': scrollTop', scroller.scrollTop, top)
	}
	outside.focus()
	for (const [j, b] of buttons.entries()) {
		const to = tab()
		await settle(vs)
		check(c, 'Tab to H' + (j + 1) + ': the focus', describe(to), describe(b))
		check(c, 'Tab to H' + (j + 1) + ': scrollTop', scroller.scrollTop, top)
	}

	// The header takes 30 px of the scroller: 20 px more puts item 5000 under it.
	c = '5: an item under the header'
	for (const [from, el] of [['outside', outside], ['H1', buttons[0]]]) {
		el.focus()
		vs.scrollToIndex(5000)
		await settle(vs)
		scroller.scrollTop += 20
		await settle(vs)
		link(5000).focus()
		await settle(vs)
		const head = vs.querySelector('[slot=header]').getBoundingClientRect().bottom
		check(c, 'from ' + from + ': item 5000 below the header', link(5000).getBoundingClientRect().top >= head - 0.5, true)
	}
} catch (e) {
	rows.push({ case: '5', error: String(e?.stack || e) })
}
await report()
`

// vscrollTableJS: on the data-table page's live example, 60 ArrowDown keys from 40 rows past a
// jump, so a window lands on the way, move the focused cell one row each.
const vscrollTableJS = `
const c = '6: sb-data-table, ArrowDown'
try {
	await customElements.whenDefined('sb-data-table')
	const dt = document.querySelector('sb-data-table[selection="multiple"]')
	const grid = dt.shadowRoot.querySelector('sb-virtual-scroll')
	for (let i = 0; i < 1500 && !grid.querySelector('.row[data-r]'); i++) await frame()
	grid.scrollToIndex(5000)
	await settle(grid)
	grid.shadowRoot.querySelector('[part=scroller]').scrollTop += 40 * 36
	await settle(grid)
	const view = grid.getBoundingClientRect()
	const visible = [...grid.querySelectorAll('.row[data-r]')].filter((r) => r.getBoundingClientRect().bottom <= view.bottom)
	const r = +visible.at(-1).dataset.r
	visible.at(-1).querySelector('[data-c]').focus()
	await settle(grid)
	let windows = 0
	dt.addEventListener('sb-window', () => windows++)
	let s = 1
	for (; s <= 60; s++) {
		deep().dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true, composed: true, cancelable: true }))
		await settle(grid)
		// A window that re-renders the rows moves the focus back a frame later.
		let got = deep()?.closest?.('[data-r]')?.dataset.r
		for (let f = 0; f < 10 && got !== String(r + s); f++) (await frame()), (got = deep()?.closest?.('[data-r]')?.dataset.r)
		if (got !== String(r + s)) {
			const row0 = dt.shadowRoot.querySelector('.grid [tabindex="0"]')?.closest('[data-r]')?.dataset.r
			check(c, 'key ' + s + ' (the tab stop is on row ' + row0 + ', ' + windows + ' windows)', got ?? describe(deep()), r + s)
			break
		}
	}
	if (s > 60) rows.push({ case: c, step: '60 keys', got: 'ok', want: 'ok' })
	check(c, 'windows on the way', windows > 0, true)
} catch (e) {
	rows.push({ case: c, error: String(e?.stack || e) })
}
await report()
`

// vscrollShrinkJS: the server sends a new list (a filter) that ends before the focused item, or
// whose item there has fewer links, or none, or a disabled button in place of the focused link.
const vscrollShrinkJS = `
// The new list from offset 0: total items of links links each (b as the server's ?b= says),
// count of them in the window.
const list = async (total, links, count, b = '', offset = 0, dead = '') => {
	const btn = document.createElement('button')
	btn.hidden = true
	btn.setAttribute('data-on:click', "@get('/__probe/window?offset=" + offset + '&count=' + count + '&total=' + total + '&links=' + links + '&b=' + b + '&dead=' + dead + "')")
	document.body.append(btn)
	await frames(2)
	const done = new Promise((resolve, reject) => {
		const timer = setTimeout(() => reject(new Error('no answer for a list of ' + total)), 10000)
		document.addEventListener('datastar-fetch', function seen(e) {
			if (e.detail.el !== btn || e.detail.type !== 'finished') return
			document.removeEventListener('datastar-fetch', seen)
			clearTimeout(timer)
			resolve()
		})
	})
	btn.click()
	await done
	btn.remove()
	await settle(vs)
}
try {
	let c = '7: a disabled button in place of the focused link'
	link(150, 'b').focus()
	await settle(vs)
	await list(10000, 2, 160, 'disabled')
	check(c, 'the focus', describe(deep()), 'a #r150a')

	c = '7: a list that ends before the item, of one link each'
	await list(10000, 2, 160)
	link(150, 'b').focus()
	await settle(vs)
	await list(100, 1, 100)
	check(c, 'the focus', describe(deep()), 'a #r99a')
	check(c, 'the scroller\'s tabindex', scroller.getAttribute('tabindex'), 'null')

	c = '7: an item with nothing to focus'
	await list(100, 0, 100)
	check(c, 'the focus', describe(deep()), 'div::part(scroller)')
	check(c, 'the parked scroller\'s tabindex', scroller.getAttribute('tabindex'), '-1')
	scroller.dispatchEvent(new FocusEvent('focusout', { bubbles: true, composed: true }))
	await frames(2)
	check(c, 'a focusout that leaves it the focus (a window switch) keeps its tabindex', scroller.getAttribute('tabindex'), '-1')

	c = '7: Tab right after a window, before its frame, starts from the same item'
	await list(10000, 2, 160)
	link(150).focus()
	await settle(vs)
	const seen = []
	// Registered after the component's own observer, so it runs right after it, in the same microtask checkpoint.
	const spy = new MutationObserver(() => {
		if (seen.length) return
		deep().dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, composed: true }))
		seen.push(describe(deep()))
	})
	spy.observe(vs, { attributeFilter: ['offset'] })
	await list(10000, 2, 160, '', 10)
	spy.disconnect()
	check(c, 'the focus Tab moves on from', seen[0], 'a #r150a')

	c = '7: the item at the index holds only disabled controls, and its link now shows another item'
	await list(10000, 2, 160)
	link(150).focus()
	await settle(vs)
	await list(10000, 2, 160, '', 10, '150')
	check(c, 'the focus', describe(deep()), 'div::part(scroller)')

	c = '7: a list that ends before the item, outside its first window'
	await list(10000, 2, 160)
	vs.scrollToIndex(5000)
	await settle(vs)
	link(5000).focus()
	await settle(vs)
	await list(3000, 2, 60)
	check(c, 'the focus', describe(deep()), 'a #r2999a')
	check(c, 'the scroller\'s tabindex', scroller.getAttribute('tabindex'), 'null')

	c = '7: an empty list'
	await list(0, 2, 0)
	check(c, 'the focus', describe(deep()), 'div::part(scroller)')
} catch (e) {
	rows.push({ case: '7', error: String(e?.stack || e) })
}
await report()
`

// vscrollFocusableJS: items that take the focus themselves (tabindex="-1"), so the item and its
// link both contain the focused element.
const vscrollFocusableJS = `
const c = '8: items that take the focus themselves'
try {
	let before = windows
	link(150, 'b').focus()
	await settle(vs)
	check(c, 'a window landed', windows > before, true)
	check(c, 'the focus', describe(deep()), 'a #r150b')
	before = windows
	const k = await walk(c, 301, 200)
	check(c, 'windows on the way', windows > before, true)
	if (k) {
		before = windows
		await walk(c + ', Shift+Tab', k, 100, true)
		check(c + ', Shift+Tab', 'windows on the way', windows > before, true)
	}
} catch (e) {
	rows.push({ case: c, error: String(e?.stack || e) })
}
await report()
`

// vscrollComponentJS: item b is an sb-checkbox, whose focusable box is in its shadow root.
const vscrollComponentJS = `
try {
	await customElements.whenDefined('sb-checkbox')
	const box = (i) => vs.querySelector('sb-checkbox[data-k="#r' + i + 'b"]').shadowRoot.querySelector('[part=base]')
	let c = '9: a control inside a component'
	let before = windows
	box(150).focus()
	await settle(vs)
	check(c, 'a window landed', windows > before, true)
	check(c, 'the focus', describe(deep()), 'div::part(base) of #r150b')
	before = windows
	await walk(c, 301, 100)
	check(c, 'windows on the way', windows > before, true)

	c = '9: a control inside a component, scrolled out of the window'
	vs.scrollToIndex(140)
	await settle(vs)
	box(150).focus()
	await settle(vs)
	vs.scrollToIndex(5000)
	await settle(vs)
	check(c, 'scrollToIndex(5000): the focus', describe(deep()), 'div::part(scroller)')
	vs.scrollToIndex(140)
	await settle(vs)
	check(c, 'scrollToIndex(140): the focus', describe(deep()), 'div::part(base) of #r150b')
} catch (e) {
	rows.push({ case: '9', error: String(e?.stack || e) })
}
await report()
`
