package app_test

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"starbase/components"
	"starbase/internal/catalog"
)

// textProp tells a prop that holds text a user sees or hears by its name.
var textProp = regexp.MustCompile(`^(label|heading|summary)$|Label$|Hint$`)

// textFixtures put components where their texts show (open, loading, with
// chips, a problem or a number with decimals), under lang="de". want are
// strings that must show: markers filled in, numbers in German.
var textFixtures = []struct {
	tag, html, run string
	want           []string
}{
	{tag: "sb-toast", html: `<sb-toast placement="inline" duration="0" toasts='[{"id":"a","text":"Hallo"}]'></sb-toast>`, want: []string{"Ü·DISMISSLABEL Hallo"}},
	{tag: "sb-sparkline", html: `<sb-sparkline values="[1.5,2.5]" decimals="1" show-value></sb-sparkline>`, want: []string{"Ü·SUMMARYLABEL 2 2,5 1,5 2,5"}},
	{tag: "sb-sparkline", html: `<sb-sparkline></sb-sparkline>`, want: []string{"Ü·EMPTYLABEL"}},
	{tag: "sb-select", html: `<sb-select multiple actions clearable max-chips="5" options='[{"value":"a","label":"Alpha"}]' value="a"></sb-select>`, run: `el.shadowRoot.querySelector('.control').click()`, want: []string{"Ü·REMOVELABEL Alpha", "Ü·SELECTALLLABEL"}},
	{tag: "sb-select", html: `<sb-select searchable></sb-select>`, run: `el.shadowRoot.querySelector('.control').click()`, want: []string{"Ü·EMPTYLABEL"}},
	{tag: "sb-select", html: `<sb-select remote min-chars="2"></sb-select>`, run: `el.shadowRoot.querySelector('.control').click()`, want: []string{"Ü·HINTLABEL"}},
	{tag: "sb-date-picker", html: `<sb-date-picker inline month="2026-10" time></sb-date-picker>`, want: []string{"Oktober", "Ü·PREVIOUSYEARLABEL", "Ü·NEXTMONTHLABEL", "Ü·APPLYLABEL"}},
	{tag: "sb-date-picker", html: `<sb-date-picker value="2026-10-05"></sb-date-picker>`, want: []string{"Ü·CHOOSELABEL"}},
	{tag: "sb-modal", html: `<sb-modal open></sb-modal>`, want: []string{"Ü·CLOSELABEL"}},
	{tag: "sb-drawer", html: `<sb-drawer open modal="false"></sb-drawer>`, want: []string{"Ü·CLOSELABEL"}},
	{tag: "sb-alert", html: `<sb-alert closable>Achtung</sb-alert>`, want: []string{"Ü·DISMISSLABEL"}},
	{tag: "sb-input", html: `<sb-input action></sb-input>`, want: []string{"Ü·SUBMITLABEL"}},
	{tag: "sb-rating", html: `<sb-rating value="2.5" precision="0.5"></sb-rating>`, want: []string{"Ü·VALUELABEL 2,5 5"}},
	{tag: "sb-range", html: `<sb-range value='{"start":2.5,"end":7.5}' min="0.5" max="10.5" step="0.5" ticks></sb-range>`, want: []string{"Ü·STARTLABEL Ü·LABEL", "Ü·ENDLABEL Ü·LABEL", "2,5", "7,5", "10,5"}},
	{tag: "sb-slider", html: `<sb-slider value="2.5" min="0.5" max="10.5" step="0.5" ticks></sb-slider>`, want: []string{"2,5", "10,5"}},
	{tag: "sb-gauge", html: `<sb-gauge value="42.5" decimals="1"></sb-gauge>`, want: []string{"42,5"}},
	{tag: "sb-meter", html: `<sb-meter value="42.5" decimals="1"></sb-meter>`, want: []string{"42,5"}},
	{tag: "sb-busy", html: `<sb-busy busy variant="bar" value="33.5"></sb-busy>`, want: []string{"33,5"}},
	{tag: "sb-tree", html: `<sb-tree items='[{"id":"a","label":"Alpha","lazy":true}]' expanded="a"></sb-tree>`, want: []string{"Ü·LOADINGLABEL"}},
	{tag: "sb-dropdown", html: `<sb-dropdown items='[{"label":"Mehr","children":[]}]'></sb-dropdown>`, run: `el.show(); await new Promise((r) => setTimeout(r, 100)); el.shadowRoot.querySelector('[role^=menuitem]')?.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true, composed: true }))`, want: []string{"Ü·EMPTYLABEL"}},
	{tag: "sb-pixel-board", html: `<sb-pixel-board size="8"></sb-pixel-board>`, want: []string{"Ü·LABEL", "Ü·PALETTELABEL", "Ü·SWATCHLABEL 1"}},
	{tag: "sb-voxel", html: `<sb-voxel></sb-voxel>`, want: []string{"Ü·LABEL rocket 30 15"}},
	{tag: "sb-image-compare", html: `<sb-image-compare position="25" expandable></sb-image-compare>`, want: []string{"Ü·VALUELABEL Ü·BEFORELABEL 25 % Ü·AFTERLABEL 75 %", "Ü·EXPANDLABEL"}},
	{tag: "sb-code-editor", html: `<sb-code-editor value="x" diagnostics='[{"line":1,"col":1,"length":1,"message":"Fehler"}]'></sb-code-editor>`, want: []string{"Ü·PROBLEMLABEL 1 Fehler"}},
	{tag: "sb-code-playground", html: `<sb-code-playground auto-run="false"></sb-code-playground>`, want: []string{"Ü·FILESLABEL", "Ü·RUNLABEL", "Ü·RUNHINT", "Ü·CONSOLELABEL"}},
	{tag: "sb-data-table", html: `<sb-data-table column-picker resizable columns='[{"key":"n","label":"N"}]' rows='[{"id":1,"n":4242.5}]' total="1"></sb-data-table>`, want: []string{"4.242,5", "Ü·COLUMNSLABEL"}},
	{tag: "sb-copy-button", html: `<sb-copy-button value="x"></sb-copy-button>`, want: []string{"Ü·LABEL"}},
}

// fallbacks are English names components use when a page gives none: with
// every text prop and the host's aria-label set, none may show.
var fallbacks = []string{"Rating", "Range", "Gauge", "Meter", "Value", "Select", "Options", "Date", "Before", "After", "Code", "Toggle", "Comparison", "Pixel board", "October", "Monday", "Mon", "Close", "Dismiss", "Submit", "Running"}

// numbers in English format, which no fixture may show under lang="de".
var englishNumbers = []string{"2.5", "7.5", "0.5", "10.5", "42.5", "1.5", "33.5", "4,242.5", "25%"}

// TestTextProps renders every component under lang="de" with all its text
// props (by name: label, heading, summary, *Label, *Hint) and its host's
// aria-label set to markers: first its gallery preview, then the fixtures
// above, opened where their texts show. Neither the shadow DOM's text nor
// the names, descriptions, titles and placeholders in it may contain an
// English default (the literal parts of each text prop's default), and the
// fixtures may contain no English fallback name or number format.
func TestTextProps(t *testing.T) {
	cat, err := catalog.Load(components.FS)
	if err != nil {
		t.Fatal(err)
	}
	type item struct {
		Tag    string            `json:"tag"`
		HTML   string            `json:"html"`
		Run    string            `json:"run"`
		Attrs  map[string]string `json:"attrs"`
		Want   []string          `json:"want"`
		Strict bool              `json:"strict"`
	}
	attrs := map[string]map[string]string{}
	english := map[string][]string{} // by tag
	n := 0
	placeholder := regexp.MustCompile(`\{\w+\}`)
	letters := regexp.MustCompile(`^[^\p{L}]+|[^\p{L}]+$`)
	for _, c := range cat.Components {
		a := map[string]string{"aria-label": "Ü·ARIA"}
		for _, p := range c.Manifest.Props {
			if p.Type != "string" || !textProp.MatchString(p.Name) {
				continue
			}
			var def string
			json.Unmarshal(p.Default, &def)
			marker := "Ü·" + strings.ToUpper(p.Name)
			for _, ph := range placeholder.FindAllString(def, -1) {
				marker += " " + ph
			}
			if strings.Contains(def, "|") {
				marker += "|" + marker
			}
			a[p.Attribute] = marker
			for _, part := range placeholder.Split(strings.ReplaceAll(def, "|", "{}"), -1) {
				if part = letters.ReplaceAllString(part, ""); len([]rune(part)) >= 4 {
					english[c.Tag] = append(english[c.Tag], part)
					n++
				}
			}
		}
		attrs[c.Tag] = a
	}
	var items []item
	for _, c := range cat.Components {
		if strings.TrimSpace(c.Preview) != "" {
			items = append(items, item{Tag: c.Tag, HTML: c.Preview, Attrs: attrs[c.Tag]})
		}
	}
	for _, f := range textFixtures {
		items = append(items, item{Tag: f.tag, HTML: f.html, Run: f.run, Attrs: attrs[f.tag], Want: f.want, Strict: true})
	}
	in, _ := json.Marshal(map[string]any{"items": items, "english": english, "strict": append(fallbacks, englishNumbers...)})
	body := probeBundle(t, "/", strings.Replace(textPropsJS, "%IN%", string(in), 1), nil)
	var got struct {
		Error    string   `json:"error"`
		Problems []string `json:"problems"`
		Checked  int      `json:"checked"`
	}
	if err := json.Unmarshal(body, &got); err != nil || got.Error != "" {
		t.Fatalf("%v: %s", err, body)
	}
	for _, p := range got.Problems {
		t.Error(p)
	}
	t.Logf("%d renders checked against %d English fragments", got.Checked, n)
}

// textPropsJS renders each item in its own lang="de" box, attributes set
// before the element upgrades (in a template), runs its script, waits, and
// collects the texts of every shadow root inside, nested ones too.
const textPropsJS = `let out
try {
	const { items, english, strict } = %IN%
	const problems = []
	const ATTRS = ['aria-label', 'aria-description', 'aria-valuetext', 'aria-roledescription', 'title', 'placeholder', 'alt', 'abbr', 'data-label']
	const texts = (root, acc = []) => {
		for (const n of root.querySelectorAll('*')) {
			for (const a of ATTRS) if (n.hasAttribute(a)) acc.push(n.getAttribute(a))
			if (n.shadowRoot) texts(n.shadowRoot, acc)
		}
		const walk = document.createTreeWalker(root, NodeFilter.SHOW_TEXT)
		while (walk.nextNode()) if (!walk.currentNode.parentElement?.closest('style, script')) acc.push(walk.currentNode.data)
		return acc
	}
	const tags = [...new Set(items.map((i) => i.tag))]
	await Promise.all(tags.map((t) => customElements.whenDefined(t)))
	let checked = 0
	for (const it of items) {
		const box = document.createElement('div')
		box.lang = 'de'
		const tpl = document.createElement('template')
		tpl.innerHTML = it.html
		for (const el of tpl.content.querySelectorAll(it.tag)) for (const [k, v] of Object.entries(it.attrs)) el.setAttribute(k, v)
		box.append(tpl.content)
		document.body.append(box)
		const el = box.querySelector(it.tag)
		await new Promise((r) => setTimeout(r, 150))
		if (it.run) await new (Object.getPrototypeOf(async function () {}).constructor)('el', it.run)(el)
		await new Promise((r) => setTimeout(r, 250))
		const all = [...box.querySelectorAll(it.tag)].flatMap((e) => (e.shadowRoot ? texts(e.shadowRoot) : []))
		const seen = all.join('\n')
		const what = it.tag + (it.strict ? ' (' + it.html.slice(0, 80) + ')' : ' (preview)')
		for (const e of english[it.tag] || []) if (seen.includes(e)) problems.push(what + ': shows the English default "' + e + '"')
		if (it.strict) {
			for (const e of strict) if (new RegExp('(^|[^\\p{L}\\p{N}.,])' + e.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + '($|[^\\p{L}\\p{N}])', 'u').test(seen)) problems.push(what + ': shows "' + e + '"')
			for (const w of it.want) if (!seen.replace(/\s+/g, ' ').includes(w)) problems.push(what + ': does not show "' + w + '" in: ' + JSON.stringify(seen.replace(/\s+/g, ' ').slice(0, 400)))
		}
		checked++
		box.remove()
	}
	out = JSON.stringify({ problems, checked })
} catch (e) { out = JSON.stringify({ error: String(e && e.stack || e) }) }
await fetch('/__probe/result', { method: 'POST', body: out })`
