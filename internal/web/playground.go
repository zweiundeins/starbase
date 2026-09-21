package web

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"html"
	"io/fs"
	"net/http"
	"net/url"
	"regexp"
	"strings"

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
//	parent → runner  {type: "run", files: {"component.js", "index.html", "style.css"}, deps: [urls], theme}
//	runner → parent  {type: "console", level, args: [string]} | {type: "error", message, line} | {type: "done"}
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
	for _, name := range []string{"css/tokens.css", "css/theme.css", "css/themes/showcase.css", "css/reset.css"} {
		fmt.Fprintf(&css, `<link rel="stylesheet" href="%s">`, html.EscapeString(origin+s.assets.Static(name)))
	}
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
	addEventListener('error', (e) => send({ type: 'error', message: e.message, line: e.lineno }))
	addEventListener('unhandledrejection', (e) => send({ type: 'error', message: String(e.reason && e.reason.message || e.reason) }))
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
			const js = files['component.js'] || ''
			if (js.trim()) await import(URL.createObjectURL(new Blob([js], { type: 'text/javascript' })))
			for (const url of m.deps || []) await import(url)
			await import('datastar')
		} catch (err) {
			send({ type: 'error', message: String(err && err.message || err) })
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
		}
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
	}
	initial, _ := json.Marshal(files)
	v.Initial = string(initial)
	title := "Playground · Starbase"
	if v.ComponentName != "" {
		title = v.ComponentName + " in the playground · Starbase"
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
	var uid int64
	s.q.View(r.Context(), func(rd *queries.Reader) error {
		if u, _ := rd.SessionUser(r.Context(), sessionID(r)); u != nil {
			uid = u.ID
		}
		return nil
	})
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
