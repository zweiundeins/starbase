package app_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/starfederation/datastar-go/datastar"
)

// TestVirtualScrollFeed checks in Chrome sb-virtual-scroll's feed keys, windows rounded to steps,
// a row height from a CSS length, and a list whose total is unknown. STARBASE_DATASTAR_BUNDLE as in
// TestVirtualScrollFocus.
func TestVirtualScrollFeed(t *testing.T) {
	var bundle []byte
	if p := os.Getenv("STARBASE_DATASTAR_BUNDLE"); p != "" {
		var err error
		if bundle, err = os.ReadFile(p); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		name, script string
		feed         vscrollFeed
	}{
		{"page keys", vscrollFeedKeysJS, vscrollFeed{total: 10000}},
		{"steps", vscrollFeedStepJS, vscrollFeed{total: 10000}},
		{"steps with a cap", vscrollFeedStepCapJS, vscrollFeed{total: 10000, cap: 60}},
		{"item size", vscrollFeedSizeJS, vscrollFeed{total: 10000}},
		{"unknown total", vscrollFeedUnknownJS, vscrollFeed{total: 500, unknown: true}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			script := strings.Replace(vscrollFeedJS+c.script, "HOST_HTML", jsonString(t, c.feed.host(0, 30)), 1)
			var served atomic.Int32
			handlers := func(app http.Handler) map[string]http.HandlerFunc {
				m := map[string]http.HandlerFunc{"/__probe/feed": c.feed.serve}
				if bundle != nil {
					m[datastarPath(t, app)] = func(w http.ResponseWriter, r *http.Request) {
						served.Add(1)
						w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
						w.Write(bundle)
					}
				}
				return m
			}
			_, body := probeWith(t, "/", script, handlers)
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

// vscrollFeed is the test feed: total articles, sent at most cap at a time (0: no cap). An unknown
// total is -1 until a window reaches the end.
type vscrollFeed struct {
	total, cap int
	unknown    bool
}

// host is the feed's host with count articles from offset, each a link, keyed by data-key.
func (f vscrollFeed) host(offset, count int) string {
	offset = min(max(offset, 0), f.total)
	count = min(max(count, 0), f.total-offset)
	if f.cap > 0 {
		count = min(count, f.cap)
	}
	total := f.total
	if f.unknown && offset+count < f.total {
		total = -1
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<sb-virtual-scroll id="vsf" role="feed" aria-label="Feed" offset="%d" total="%d" data-preserve-attr="item-size buffer step style data-on:sb-window">`, offset, total)
	for i := offset; i < offset+count; i++ {
		fmt.Fprintf(&b, `<article tabindex="-1" data-key="e%d" aria-posinset="%d" aria-setsize="%d"><a href="#r%da">%d a</a></article>`, i, i+1, total, i, i)
	}
	b.WriteString(`</sb-virtual-scroll>`)
	return b.String()
}

func (f vscrollFeed) serve(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	offset, _ := strconv.Atoi(q.Get("offset"))
	count, _ := strconv.Atoi(q.Get("count"))
	datastar.NewSSE(w, r).PatchElements(f.host(offset, count))
}

// vscrollFeedJS builds the feed between a button before it and one after it: 300 px high, rows of
// 30 px, a 300 px buffer. log holds every sb-window's detail.
const vscrollFeedJS = vscrollHelpersJS + `
await customElements.whenDefined('sb-virtual-scroll')
const css = document.createElement('style')
css.textContent = '#vsf { block-size: 300px } #vsf > article { display: flex; align-items: center }'
document.head.append(css)
const before = document.createElement('button'), after = document.createElement('button')
before.textContent = 'Before'
after.textContent = 'After'
const tpl = document.createElement('template')
tpl.innerHTML = HOST_HTML
const vs = tpl.content.firstElementChild
vs.setAttribute('item-size', '30')
vs.setAttribute('buffer', '300')
vs.setAttribute('data-on:sb-window', "@get('/__probe/feed?offset=' + evt.detail.offset + '&count=' + evt.detail.count)")
const log = []
vs.addEventListener('sb-window', (e) => log.push(e.detail))
document.body.append(before, vs, after)
await frames(3)
await settle(vs)
const scroller = vs.shadowRoot.querySelector('[part=scroller]')
const art = (i) => vs.querySelector(':scope > article[aria-posinset="' + (i + 1) + '"]')
// The focused article's index, or what has the focus.
const pos = () => {
	const a = deep()
	return a?.localName === 'article' ? +a.getAttribute('aria-posinset') - 1 : describe(a)
}
const press = (key, opts = {}) => {
	const e = new KeyboardEvent('keydown', { key, bubbles: true, composed: true, cancelable: true, ...opts })
	deep().dispatchEvent(e)
	return e.defaultPrevented
}
const shown = (el) => {
	const r = el.getBoundingClientRect(), v = scroller.getBoundingClientRect()
	return r.top >= v.top - 0.5 && r.bottom <= v.bottom + 0.5
}
const end = () => +vs.getAttribute('offset') + vs.querySelectorAll(':scope > article').length
`

const vscrollFeedKeysJS = `
let c = '1: Page Down'
try {
	art(0).focus()
	await settle(vs)
	let windows = log.length, ok = true
	for (let s = 1; s <= 80 && ok; s++) {
		press('PageDown')
		await settle(vs)
		if (pos() !== s) ok = false, check(c, 'key ' + s, pos(), s)
		else if (!shown(art(s))) ok = false, check(c, 'key ' + s + ': in view', false, true)
	}
	if (ok) rows.push({ case: c, step: '80 keys', got: 'ok', want: 'ok' })
	check(c, 'windows on the way', log.length > windows, true)

	c = '1: Page Down to an article the window does not have'
	const last = end() - 1
	art(last).focus({ preventScroll: true })
	await settle(vs)
	check(c, 'the next article is not loaded', art(last + 1), null)
	press('PageDown')
	await settle(vs)
	check(c, 'the focus', pos(), last + 1)

	c = '1: five Page Downs before a window lands'
	const from = end() - 2
	art(from).focus({ preventScroll: true })
	await settle(vs)
	for (let s = 0; s < 5; s++) press('PageDown')
	await settle(vs)
	check(c, 'the focus', pos(), from + 5)

	c = '1: Page Up'
	const top = pos()
	windows = log.length
	ok = true
	for (let s = 1; s <= 60 && ok; s++) {
		press('PageUp')
		await settle(vs)
		if (pos() !== top - s) ok = false, check(c, 'key ' + s, pos(), top - s)
	}
	if (ok) rows.push({ case: c, step: '60 keys', got: 'ok', want: 'ok' })
	check(c, 'windows on the way', log.length > windows, true)

	c = '1: Page Up at the first article'
	vs.scrollToIndex(0)
	await settle(vs)
	art(0).focus()
	await settle(vs)
	check(c, 'handled', press('PageUp'), false)
	check(c, 'the focus', pos(), 0)

	c = '1: Ctrl+End and Ctrl+Home'
	art(3).focus()
	check(c, 'Ctrl+End handled', press('End', { ctrlKey: true }), true)
	check(c, 'Ctrl+End: the focus', document.activeElement === after, true)
	art(3).focus()
	check(c, 'Ctrl+Home handled', press('Home', { ctrlKey: true }), true)
	check(c, 'Ctrl+Home: the focus', document.activeElement === before, true)

	c = '1: no feed keys without role="feed"'
	vs.removeAttribute('role')
	await settle(vs)
	art(3).focus()
	check(c, 'Page Down handled', press('PageDown'), false)
	check(c, 'the focus', pos(), 3)
} catch (e) {
	rows.push({ case: c, error: String(e?.stack || e) })
}
await report()
`

// The view is 10 rows of 30 px: a window of 11 rows and 10 rows of buffer on each side, rounded out to 25.
const vscrollFeedStepJS = `
const c = '2: step'
try {
	vs.setAttribute('step', '25')
	await settle(vs)
	log.length = 0
	for (const top of [1234 * 30, 2000 * 30 + 7, 50 * 30, 7777 * 30 + 29, 9990 * 30]) {
		scroller.scrollTop = top
		await settle(vs)
		const at = Math.floor(scroller.scrollTop / 30)
		check(c, 'scrollTop ' + top + ': the view has its items', !!art(at) && !!art(Math.min(at + 10, 9999)), true)
	}
	check(c, 'every window starts and ends at a step', log.filter((d) => d.offset % 25 || (d.offset + d.count) % 25).map((d) => d.offset + '+' + d.count).join(' '), '')
	log.length = 0
	scroller.scrollTop = 0
	await settle(vs)
	scroller.scrollTop = 1234 * 30
	await settle(vs)
	check(c, 'the window for row 1234', JSON.stringify(log.at(-1)), '{"offset":1200,"count":75}')
} catch (e) {
	rows.push({ case: c, error: String(e?.stack || e) })
}
await report()
`

const vscrollFeedStepCapJS = `
const c = '3: step, and a server that sends 60 at most'
try {
	vs.setAttribute('step', '25')
	await settle(vs)
	for (const top of [1234 * 30, 2000 * 30 + 7, 1999 * 30, 50 * 30, 7777 * 30 + 29]) {
		scroller.scrollTop = top
		await settle(vs)
		const at = Math.floor(scroller.scrollTop / 30)
		check(c, 'scrollTop ' + top + ': the view has its items', !!art(at) && !!art(at + 10), true)
	}
	check(c, 'every window starts at a step', log.filter((d) => d.offset % 25).length, 0)
} catch (e) {
	rows.push({ case: c, error: String(e?.stack || e) })
}
await report()
`

const vscrollFeedSizeJS = `
let c = '4: item-size in rem'
try {
	document.documentElement.style.fontSize = '16px'
	vs.setAttribute('item-size', '2rem')
	await settle(vs)
	check(c, 'row height at 16px', art(0).getBoundingClientRect().height, 32)
	vs.scrollToIndex(3000)
	await settle(vs)
	scroller.scrollTop += 8
	await settle(vs)
	const y = () => Math.round(art(3000).getBoundingClientRect().top - scroller.getBoundingClientRect().top)
	check(c, 'item 3000 at the top', y(), -8)
	document.documentElement.style.fontSize = '20px'
	await frames(2)
	await settle(vs)
	check(c, 'row height at 20px', art(3000).getBoundingClientRect().height, 40)
	check(c, 'item 3000 still at the top', y(), -10)
	check(c, 'scrollTop', scroller.scrollTop, 3000 * 40 + 10)
	check(c, 'the view has its items', !!art(3009), true)
	document.documentElement.style.fontSize = ''

	c = '4: item-size from a custom property'
	vs.style.setProperty('--row', '50px')
	vs.setAttribute('item-size', 'var(--row)')
	await frames(2)
	await settle(vs)
	check(c, 'row height', art(3000).getBoundingClientRect().height, 50)
	vs.style.setProperty('--row', '3.5rem')
	await frames(2)
	await settle(vs)
	check(c, 'row height after the property changed', art(3000).getBoundingClientRect().height, 56)
	check(c, 'item 3000 still at the top', y() > -56 && y() <= 0, true)
	vs.setAttribute('item-size', '30')
	await frames(2)
	await settle(vs)
	check(c, 'back to px', art(3000).getBoundingClientRect().height, 30)
} catch (e) {
	rows.push({ case: c, error: String(e?.stack || e) })
}
await report()
`

const vscrollFeedUnknownJS = `
let c = '5: unknown total'
try {
	check(c, 'total at first', vs.getAttribute('total'), '-1')
	// A window of 31 rows: 11 in view, 10 of buffer on each side.
	check(c, 'height at first', scroller.scrollHeight, (31 + 31) * 30)
	let h = scroller.scrollHeight, grew = true, past = false
	for (let i = 0; i < 100 && vs.getAttribute('total') === '-1'; i++) {
		const reach = end()
		scroller.scrollTop = scroller.scrollHeight
		await settle(vs)
		if (log.some((d) => d.offset + d.count > reach)) past = true
		if (vs.getAttribute('total') === '-1' && !(scroller.scrollHeight > h)) grew = false
		h = scroller.scrollHeight
	}
	check(c, 'the height grew with every window', grew, true)
	check(c, 'asked past the known end', past, true)
	check(c, 'total once the server reached the end', vs.getAttribute('total'), '500')
	check(c, 'height at the end', scroller.scrollHeight, 500 * 30)
	scroller.scrollTop = scroller.scrollHeight
	await settle(vs)
	check(c, 'the last article', !!art(499) && shown(art(499)), true)
	const n = log.length
	scroller.scrollTop -= 5
	await settle(vs)
	scroller.scrollTop = scroller.scrollHeight
	await settle(vs)
	check(c, 'no window past the end', log.length, n)

	c = '5: Page Down past the known end'
	// The total unknown again: the server's window at the top says -1.
	vs.setAttribute('total', '-1')
	vs.scrollToIndex(0)
	await settle(vs)
	check(c, 'total', vs.getAttribute('total'), '-1')
	const last = end() - 1
	art(last).focus({ preventScroll: true })
	await settle(vs)
	check(c, 'the next article is not loaded', art(last + 1), null)
	press('PageDown')
	await settle(vs)
	check(c, 'the focus', pos(), last + 1)
} catch (e) {
	rows.push({ case: c, error: String(e?.stack || e) })
}
await report()
`
