package web

import (
	"cmp"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io/fs"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/starfederation/datastar-go/datastar"

	"starbase/internal/commands"
	"starbase/internal/queries"
	"starbase/internal/ui"
)

// playgroundRun serves the code playground's runner: the page inside the
// preview iframe. It is always embedded with sandbox="allow-scripts" (no
// allow-same-origin), so user code runs in an opaque origin: it cannot read
// the site's cookies, storage or DOM. The page has its own CSP; the site's
// nonce-only policy would forbid running user code at all.
//
// Protocol (postMessage, all messages carry source: "sb-runner"):
//
//	runner → parent  {type: "ready"}
//	parent → runner  {type: "run", files: {"component.js", "index.html", "style.css"}, deps: [urls], theme, base}
//
// base (optional) is the URL relative imports in component.js resolve
// against: the folder the component's other files are served from.
//
//	runner → parent  {type: "console", level, args: [string]} | {type: "error", message, line} | {type: "done"}
//
// The runner stops endless loops in component.js (see guardLoops in the
// script): a hung frame can't be stopped from outside, and in Chrome it also
// hangs every other preview frame of the site.
func (s *Server) playgroundRun(w http.ResponseWriter, r *http.Request) {
	scheme := "http"
	if s.secure {
		scheme = "https"
	}
	origin := scheme + "://" + r.Host
	h := w.Header()
	h.Set("Content-Security-Policy", fmt.Sprintf("default-src 'none'; "+
		"script-src %[1]s 'unsafe-inline' 'unsafe-eval' blob:; "+
		"style-src %[1]s 'unsafe-inline'; font-src %[1]s; "+
		"img-src %[1]s data: blob: https://github.com https://avatars.githubusercontent.com; "+
		"connect-src %[1]s; frame-ancestors %[1]s; base-uri 'none'; form-action 'none'", origin))
	h.Set("X-Frame-Options", "SAMEORIGIN")
	h.Del("Cross-Origin-Opener-Policy")
	h.Set("Cache-Control", "no-store")
	h.Set("Content-Type", "text/html; charset=utf-8")

	imports, _ := json.Marshal(map[string]any{"imports": map[string]string{"datastar": origin + s.assets.Datastar()}})
	var css strings.Builder
	fmt.Fprintf(&css, `<link rel="stylesheet" href="%s">`, html.EscapeString(origin+s.assets.RunnerCSS()))
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en" data-sb-theme="deep-space">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Playground preview</title>
%s
<style>
@layer base {
	html { background: var(--sb-bg); color-scheme: dark; }
	body { margin: 0; padding: 20px; min-block-size: 100dvh; box-sizing: border-box; background: var(--sb-bg); color: var(--sb-text-1); font: 14px/1.55 var(--sb-font-ui); }
	a { color: var(--sb-brand-light); }
}
</style>
<script type="importmap">%s</script>
<script>
(() => {
	const send = (m) => parent.postMessage({ source: 'sb-runner', ...m }, '*')
	const fmt = (a) => {
		if (typeof a === 'string') return a
		if (a instanceof Error) return a.stack || String(a)
		try { return JSON.stringify(a) } catch { return String(a) }
	}
	for (const level of ['log', 'info', 'warn', 'error']) {
		const orig = console[level]
		console[level] = (...args) => { send({ type: 'console', level, args: args.map(fmt) }); orig.apply(console, args) }
	}
	addEventListener('error', (e) => send({ type: 'error', message: e.message, line: e.error && e.error.line || e.lineno }))
	addEventListener('unhandledrejection', (e) => send({ type: 'error', message: String(e.reason && e.reason.message || e.reason) }))

	// Loop guard. An endless loop would hang this frame for good (the host
	// can only replace it). Every while (…) and for (…; …; …) condition in
	// component.js first calls __sbLoop(line), which throws once one task has
	// spent a second in loops; it stays tripped until the task ends, so a
	// catch inside an outer loop can't keep it going. It reads the clock on
	// every 1024th call only: a tight loop must stay tight.
	let loopStart = 0, calls = 0
	self.__sbLoop = (line) => {
		if (++calls & 1023) return true
		const now = performance.now()
		if (!loopStart) loopStart = now, setTimeout(() => (loopStart = 0))
		else if (now - loopStart > 1000) throw Object.assign(new RangeError('Endless loop? Stopped after 1 s'), { line })
		return true
	}
	// The scanner skips strings, comments, regular expressions and template
	// text, and only ever inserts text into a loop header. Where it guesses
	// wrong, the code no longer parses and runs as written instead.
	const BT = '\x60' // a backtick (this script lives in a Go raw string)
	const guardLoops = (src) => {
		const toks = [], tpl = [], n = src.length
		let i = 0, depth = 0
		// Template text from i to the closing backtick, or to a ${ (then code).
		const text = () => {
			for (; i < n; i++) {
				if (src[i] === '\\') i++
				else if (src[i] === BT) return void i++
				else if (src[i] === '$' && src[i + 1] === '{') return void (i += 2, tpl.push(depth++))
			}
		}
		// A slash starts a regular expression unless it follows a value.
		const regexOK = () => {
			const t = toks[toks.length - 1]
			return !t || (t.p ? !')]'.includes(t.v) : /^(return|typeof|instanceof|in|of|new|delete|void|throw|case|do|else|yield|await)$/.test(t.v))
		}
		while (i < n) {
			const c = src[i], d = src[i + 1]
			if (/\s/.test(c)) i++
			else if (c === '/' && d === '/') i = (src.indexOf('\n', i) + 1 || n + 1) - 1
			else if (c === '/' && d === '*') i = (src.indexOf('*/', i + 2) + 1 || n - 1) + 1
			else if (c === '"' || c === "'") {
				let j = i + 1
				while (j < n && src[j] !== c && src[j] !== '\n') j += src[j] === '\\' ? 2 : 1
				i = j + 1
				toks.push({ v: '"' })
			} else if (c === BT) {
				i++
				text()
				toks.push({ v: '"' })
			} else if (c === '/' && regexOK()) {
				let j = i + 1, cls = false
				for (; j < n && src[j] !== '\n'; j++) {
					if (src[j] === '\\') j++
					else if (src[j] === '[') cls = true
					else if (src[j] === ']') cls = false
					else if (src[j] === '/' && !cls) break
				}
				if (src[j] === '/') i = j + 1, toks.push({ v: '"' })
				else toks.push({ v: c, p: 1, at: i++ }) // no closing slash on the line: a division
			} else if (/[\p{L}\p{N}_$\\]/u.test(c)) {
				const w = /^[\p{L}\p{N}_$\\\u200c\u200d]+/u.exec(src.slice(i, i + 256))[0]
				toks.push({ v: w, at: i })
				i += w.length
			} else if (c === '}' && tpl[tpl.length - 1] === depth - 1) {
				depth--, tpl.pop(), i++
				text()
				toks.push({ v: '"' })
			} else {
				if (c === '{') depth++
				else if (c === '}') depth--
				toks.push({ v: c, p: 1, at: i++ })
			}
		}
		const ins = []
		for (let k = 0; k < toks.length; k++) {
			const w = toks[k].v, prev = toks[k - 1] && toks[k - 1].v
			if ((w !== 'while' && w !== 'for') || prev === '.' || prev === '#' || !toks[k + 1] || toks[k + 1].v !== '(') continue
			// The header: up to the matching parenthesis, and its top-level semicolons.
			let open = 0, m = k + 1
			const semis = []
			for (; m < toks.length; m++) {
				const t = toks[m]
				if (!t.p) continue
				if ('([{'.includes(t.v)) open++
				else if (')]}'.includes(t.v) && !--open) break
				else if (t.v === ';' && open === 1) semis.push(t.at)
			}
			if (m === toks.length || (w === 'for' && semis.length !== 2)) continue // for…of, for…in
			let a = w === 'for' ? semis[0] + 1 : toks[k + 1].at + 1, b = w === 'for' ? semis[1] : toks[m].at
			while (a < b && /\s/.test(src[a])) a++
			while (b > a && /\s/.test(src[b - 1])) b--
			const call = '__sbLoop(' + src.slice(0, toks[k].at).split('\n').length + ')'
			if (a < b) ins.push([b, ')'], [a, call + ' && ('])
			else if (w === 'for') ins.push([a, call]) // for (;;)
		}
		let out = src
		for (const [at, str] of ins.sort((x, y) => y[0] - x[0])) out = out.slice(0, at) + str + out.slice(at)
		return out
	}

	let ran = false
	addEventListener('message', async (e) => {
		const m = e.data
		if (e.source !== parent || !m || m.type !== 'run' || ran) return
		ran = true
		const files = m.files || {}
		document.documentElement.dataset.sbTheme = m.theme || 'deep-space'
		if (files['style.css']) {
			const st = document.createElement('style')
			st.textContent = files['style.css']
			document.head.append(st)
		}
		document.body.innerHTML = files['index.html'] || ''
		try {
			// The edited code first: Rocket must never see the site's copy of its tag.
			let js = files['component.js'] || ''
			// A blob has no folder: resolve relative imports (vendored files) against base.
			if (m.base) js = js.replace(/(\bfrom\s*|\bimport\s*\(\s*|\bimport\s+)(['"])(\.{1,2}\/[^'"\n]*)\2/g, (_, pre, q, spec) => pre + q + new URL(spec, m.base).href + q)
			if (js.trim()) {
				const load = (code) => import(URL.createObjectURL(new Blob([code], { type: 'text/javascript' })))
				const guarded = guardLoops(js)
				self.__sbStarted = 0
				try {
					await load(guarded === js ? js : '__sbStarted = 1;' + guarded)
				} catch (err) {
					// A SyntaxError before the code started: the guard broke it, so run it as written.
					if (guarded === js || self.__sbStarted || !(err instanceof SyntaxError)) throw err
					await load(js)
				}
			}
			for (const url of m.deps || []) await import(url)
			await import('datastar')
		} catch (err) {
			send({ type: 'error', message: String(err && err.message || err), line: err && err.line })
		}
		send({ type: 'done' })
	})
	send({ type: 'ready' })
})()
</script>
</head>
<body></body>
</html>
`, css.String(), imports)
}

// Starter files for an empty playground (the same starter as the
// submission form).
const starterJS = `import { rocket } from 'datastar'

const styles = /* css */ ` + "`" + `
:host {
	--_bg: var(--sb-surface-card, #141D32);
	--_border: var(--sb-border, #283552);
	--_text: var(--sb-text-1, #F3F4FA);
	--_brand: var(--sb-brand, #8C6BFF);
	display: inline-block;
}
button {
	all: unset;
	padding: 0.5rem 1rem;
	border: 1px solid var(--_border);
	border-radius: 8px;
	background: var(--_bg);
	color: var(--_text);
	cursor: pointer;
}
button:hover { border-color: var(--_brand); }
` + "`" + `

rocket('sb-your-name', {
	props: ({ string }) => ({
		label: string.trim.default('Launch').docs({ description: 'Text on the button.' }),
	}),
	setup: ({ $$, action, adoptStyles, emit, host }) => {
		adoptStyles(host, styles)
		$$.presses = 0
		action('press', () => {
			$$.presses++
			console.log('pressed', $$.presses)
			emit('sb-press', { presses: $$.presses })
		})
	},
	render: ({ html, props: { label } }) => html` + "`" + `
		<button type="button" part="button" data-on:click="@press()">${label}</button>
	` + "`" + `,
})
`

const starterHTML = `<sb-your-name label="Hello, Starbase"></sb-your-name>
`

// componentDeps maps every catalog tag to its module URL, so previews can
// use any component. Tags defined by the edited code are skipped client-side.
func (s *Server) componentDeps() string {
	m := map[string]string{}
	for _, c := range s.catalog.Components {
		m[c.Tag] = s.assets.ComponentScript(c)
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func (s *Server) codePlaygroundPage(rc *renderCtx) (view, error) {
	files := map[string]string{"component.js": starterJS, "index.html": starterHTML}
	v := ui.CodePlaygroundView{Deps: s.componentDeps(), Share: rc.tab.PlaygroundShare}
	q := rc.req.URL.Query()
	if id := q.Get("s"); id != "" {
		sn, err := rc.r.Snippet(rc.ctx, id)
		if err != nil {
			return view{}, err
		}
		if sn == nil {
			return view{}, errNotFound
		}
		files = sn.Files
		v.Loaded, v.Author = sn.ID, sn.Author
		if c, ok := s.catalog.Get(sn.Component); ok {
			v.Component, v.ComponentName = c.Slug, c.Name
			v.Base = "/c/" + c.Slug + "@" + c.Hash + "/"
		}
	} else if ref := q.Get("preview"); ref != "" {
		p, err := s.previews.get(rc.ctx, ref)
		if errors.Is(err, errNoPreview) {
			return view{}, errNotFound
		} else if err != nil {
			return view{}, err
		}
		files = p.Files
		v.Preview = cmp.Or(p.Name, "This component")
		v.Base = "/playground/preview/" + ref + "/"
	} else if slug := q.Get("component"); slug != "" {
		comp, ok := s.catalog.Get(slug)
		if !ok {
			return view{}, errNotFound
		}
		src, err := fs.ReadFile(s.catalog.FS, comp.Script)
		if err != nil {
			return view{}, err
		}
		files = map[string]string{"component.js": string(src), "index.html": strings.Join(comp.Examples, "\n\n") + "\n"}
		v.Component, v.ComponentName = comp.Slug, comp.Name
		v.Base = "/c/" + comp.Slug + "@" + comp.Hash + "/"
	}
	v.Size = s.measure(v.Component, files["component.js"])
	initial, _ := json.Marshal(files)
	v.Initial = string(initial)
	title := "Playground · Starbase"
	if name := cmp.Or(v.ComponentName, v.Preview); name != "" {
		title = name + " in the playground · Starbase"
	}
	if v.Share == "" {
		v.Share = v.Loaded
	}
	if v.Share != "" {
		v.ShareURL = strings.TrimSuffix(s.cfg.BaseURL, "/") + "/playground?s=" + v.Share
	}
	u := ""
	if rc.tab.PlaygroundShare != "" {
		u = "/playground?s=" + rc.tab.PlaygroundShare // the address bar follows the last save
	}
	return view{
		URL:         u,
		Title:       title,
		Description: "Edit Rocket components live: code, HTML and a sandboxed preview.",
		Nav:         "playground",
		Body:        func(ui.Shell) templ.Component { return ui.CodePlaygroundPage(v) },
	}, nil
}

const idAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"

func snippetID() string {
	b := make([]byte, 8)
	rand.Read(b)
	for i := range b {
		b[i] = idAlphabet[int(b[i])%len(idAlphabet)]
	}
	return string(b)
}

// cmdSnippet saves the playground's files. The payload is sent with
// Datastar's payload option, so the files never ride along as signals.
func (s *Server) cmdSnippet(w http.ResponseWriter, r *http.Request) {
	var p struct {
		TabID     string            `json:"tabid"`
		Files     map[string]string `json:"files"`
		Component string            `json:"component"`
	}
	if err := datastar.ReadSignals(r, &p); err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	now := time.Now()
	if !s.saveLimit.allow(sessionID(r), 1, now) || !s.saveLimitIP.allow(clientIP(r), 1, now) {
		http.Error(w, "saving too often, try again in a minute", http.StatusTooManyRequests)
		return
	}
	var uid int64
	var stored int64
	s.q.View(r.Context(), func(rd *queries.Reader) error {
		if u, _ := rd.SessionUser(r.Context(), sessionID(r)); u != nil {
			uid = u.ID
		}
		stored, _ = rd.SnippetBytes(r.Context())
		return nil
	})
	if stored+commands.MaxSnippetBytes > commands.MaxSnippetStore {
		s.log.Warn("snippet storage is full", "bytes", stored)
		http.Error(w, commands.ErrSnippetStoreFull.Error(), http.StatusInsufficientStorage)
		return
	}
	if _, ok := s.catalog.Get(p.Component); !ok {
		p.Component = ""
	}
	s.send(w, r, commands.SaveSnippet{SID: sessionID(r), TabID: p.TabID, ID: snippetID(), Files: p.Files, Component: p.Component, UserID: uid})
}

// snippetJSON is the public, read-only form of a snippet (used by the
// submission bot to import playground links).
func (s *Server) snippetJSON(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var sn *queries.Snippet
	err := s.q.View(r.Context(), func(rd *queries.Reader) (err error) {
		sn, err = rd.Snippet(r.Context(), id)
		return
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if sn == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable") // snippets never change
	json.NewEncoder(w).Encode(map[string]any{"id": sn.ID, "files": sn.Files, "component": sn.Component, "author": sn.Author})
}

var tagRe = regexp.MustCompile(`rocket\(\s*['"](sb-[a-z0-9-]+)['"]`)

// submit sends people to the "Submit a component" issue form. With ?s=<id>
// it prefills the form with the playground link, which the bot imports, so
// no code has to travel in the URL.
func (s *Server) submit(w http.ResponseWriter, r *http.Request) {
	form := strings.TrimSuffix(s.cfg.RepoURL, "/") + "/issues/new"
	q := url.Values{"template": {"new-component.yml"}}
	if id := r.URL.Query().Get("s"); commands.SnippetIDRe.MatchString(id) {
		var sn *queries.Snippet
		s.q.View(r.Context(), func(rd *queries.Reader) (err error) {
			sn, err = rd.Snippet(r.Context(), id)
			return
		})
		if sn != nil {
			q.Set("source", strings.TrimSuffix(s.cfg.BaseURL, "/")+"/playground?s="+id)
			if m := tagRe.FindStringSubmatch(sn.Files["component.js"]); m != nil {
				name := strings.ReplaceAll(strings.TrimPrefix(m[1], "sb-"), "-", " ")
				name = strings.ToUpper(name[:1]) + name[1:]
				q.Set("name", name)
				q.Set("title", "[Component]: "+name)
			}
		}
	}
	http.Redirect(w, r, form+"?"+q.Encode(), http.StatusSeeOther)
}
