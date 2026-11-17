package app_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/zweiundeins/starbase/templ/sb"

	"starbase/internal/catalog"
	"starbase/internal/templgen/registry"
	"starbase/static"
)

// TestTemplTracked renders a page through sb.Tracked under a nonce CSP and
// checks in Chrome that exactly the components it rendered (and the ones
// those render) are defined, loaded with their integrity.
func TestTemplTracked(t *testing.T) {
	templTracked(t)
}

// TestTemplTrackedOnOfficialBundle is TestTemplTracked on the official
// Datastar + Rocket release.
func TestTemplTrackedOnOfficialBundle(t *testing.T) {
	if os.Getenv("STARBASE_DATASTAR_BUNDLE") == "" {
		t.Setenv("STARBASE_DATASTAR_BUNDLE", officialBundle)
	}
	templTracked(t)
}

func templTracked(t *testing.T) {
	var all []string
	for _, e := range registry.Entries {
		all = append(all, e.Tag)
	}
	want := []string{"sb-data-table", "sb-modal", "sb-slider", "sb-toggle", "sb-virtual-scroll"}
	aj, _ := json.Marshal(all)
	script := `let out
try {
	await Promise.race([Promise.all(` + mustJSON(want) + `.map((t) => customElements.whenDefined(t))), new Promise((_, no) => setTimeout(() => no(new Error('not defined in time')), 20000))])
	await new Promise((r) => setTimeout(r, 300))
	const scripts = [...document.querySelectorAll('script[type=module][src]')].map((s) => ({ src: s.src, integrity: s.integrity, nonce: s.nonce }))
	out = JSON.stringify({ defined: ` + string(aj) + `.filter((t) => customElements.get(t)), scripts, violations: window.__violations })
} catch (e) { out = JSON.stringify({ error: String(e), violations: window.__violations }) }
await fetch('/__probe/result', { method: 'POST', body: out })`
	// The probe adds the script inline before </body>: the CSP allows it by hash.
	sum := sha256.Sum256([]byte("\n" + script + "\n"))
	official := os.Getenv("STARBASE_DATASTAR_BUNDLE") != ""
	page := func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, `<main>`)
		if err == nil {
			err = sb.Modal(sb.ModalProps{Heading: "Abort?"}).Render(templ.WithChildren(ctx, sb.Toggle(sb.ToggleProps{Label: "Sure"})), w)
		}
		for _, c := range []templ.Component{sb.Toggle(sb.ToggleProps{Label: "Shields"}), sb.Slider(sb.SliderProps{Label: "Thrust"}), sb.DataTable(sb.DataTableProps{})} {
			if err == nil {
				err = c.Render(ctx, w)
			}
		}
		if err == nil {
			_, err = io.WriteString(w, `</main>`)
		}
		return err
	}
	handlers := map[string]http.HandlerFunc{"/__templ": func(w http.ResponseWriter, r *http.Request) {
		base := "http://" + r.Host
		p := sb.ImportsProps{Base: base}
		if official {
			p.Datastar = base + "/c/" + catalog.DatastarSlug + "@" + catalog.VersionHash(static.Datastar()) + "/" + catalog.DatastarFile
		}
		var body strings.Builder
		if err := sb.Tracked(p).Render(templ.WithChildren(templ.WithNonce(r.Context(), "n0nce"), templ.ComponentFunc(page)), &body); err != nil {
			t.Error(err)
		}
		csp := fmt.Sprintf("script-src 'nonce-n0nce' 'sha256-%s' %s/c/ 'unsafe-eval'", base64.StdEncoding.EncodeToString(sum[:]), base)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="%s"><title>tracked</title>`+
			`<script nonce="n0nce">window.__violations = []; addEventListener('securitypolicyviolation', (e) => __violations.push(e.violatedDirective + ' ' + e.blockedURI))</script>`+
			`</head><body>%s</body></html>`, csp, body.String())
	}}
	res := probeBundle(t, "/__templ", script, handlers)
	var got struct {
		Defined    []string
		Scripts    []struct{ Src, Integrity, Nonce string }
		Violations []string
		Error      string
	}
	if err := json.Unmarshal(res, &got); err != nil || got.Error != "" {
		t.Fatalf("%v: %s", err, res)
	}
	slices.Sort(got.Defined)
	if !slices.Equal(got.Defined, want) {
		t.Errorf("defined %v, want %v", got.Defined, want)
	}
	// The probe's own script has no src; every component's has its pin.
	if len(got.Scripts) != len(want) {
		t.Errorf("module scripts: %+v", got.Scripts)
	}
	for _, s := range got.Scripts {
		if !strings.HasPrefix(s.Integrity, "sha384-") || s.Nonce != "n0nce" {
			t.Errorf("script %+v", s)
		}
	}
	if len(got.Violations) > 0 {
		t.Errorf("CSP violations: %v", got.Violations)
	}
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}
