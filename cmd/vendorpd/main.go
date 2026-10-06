// Command vendorpd brings PD rockets (https://github.com/derekr/pd-rockets,
// by derekr, Beer-Ware licence) into components/: the release with the
// patches in patches/pd-rockets applied (each an upstream candidate), then
// for each surface its TypeScript and the core modules it imports, with the
// pd- names renamed to sb-, the runtime import pointed at 'datastar' and the
// relative imports at the folder's own .ts files. The README, manifest and
// anything else of Starbase's in those folders stay. Run `go tool task ts`
// after it.
//
//	go run ./cmd/vendorpd [-tag v2026-09-28-2] [-only sortable-list,drag-group]
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

const repo = "https://github.com/derekr/pd-rockets"

// surfaces maps each Starbase component to its PD rockets folder under rocket/.
var surfaces = []struct{ slug, dir string }{
	{"kanban-board", "kanban"},
	{"sortable-list", "sortable-list"},
	{"drag-group", "drag-group"},
	{"bento-workspace", "bento"},
	{"sortable-tree", "sortable-tree"},
	{"context-menu", "context-menu"},
	{"inline-edit", "inline-edit"},
}

var (
	specRe   = regexp.MustCompile(`((?:from|import)\s*)"([^"]+)"`)
	renameRe = regexp.MustCompile(`\bpd-`)
	// rocket(sortableListContract.tag, …): the catalog, the size line and the
	// playground read the tag from a literal rocket('sb-…') call.
	defineRe = regexp.MustCompile(`rocket\((\w+)\.tag,`)
	tagRe    = regexp.MustCompile(`\btag: "(sb-[a-z-]+)"`)
)

func main() {
	tag := flag.String("tag", "v2026-09-28-2", "the PD rockets release to vendor")
	only := flag.String("only", "", "vendor only these components (slugs, separated by commas)")
	flag.Parse()
	if err := run(*tag, strings.Split(*only, ",")); err != nil {
		fmt.Fprintln(os.Stderr, "vendorpd:", err)
		os.Exit(1)
	}
}

func run(tag string, only []string) error {
	tmp, err := os.MkdirTemp("", "vendorpd-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if out, err := exec.Command("git", "clone", "-q", "--depth", "1", "--branch", tag, repo+".git", tmp).CombinedOutput(); err != nil {
		return fmt.Errorf("clone %s: %v: %s", tag, err, out)
	}
	patches, _ := filepath.Glob(filepath.Join("patches", "pd-rockets", "*.patch"))
	if len(patches) > 0 {
		args := append([]string{"-C", tmp, "-c", "user.name=vendor", "-c", "user.email=vendor@localhost", "am", "-q"}, abs(patches)...)
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("apply patches/pd-rockets: %v: %s", err, out)
		}
	}
	sha, err := exec.Command("git", "-C", tmp, "rev-parse", tag+"^{commit}").Output()
	if err != nil {
		return err
	}
	commit := strings.TrimSpace(string(sha))
	license, err := os.ReadFile(filepath.Join(tmp, "LICENSE"))
	if err != nil {
		return err
	}
	header := fmt.Sprintf("// From PD rockets by derekr (%s, %s), under the Beer-Ware licence\n// in LICENSE-pd-rockets.txt. Vendored by `go run ./cmd/vendorpd` with patches/pd-rockets applied,\n// pd- names renamed to sb-.\n", repo, tag)
	for _, s := range surfaces {
		if only[0] != "" && !slices.Contains(only, s.slug) {
			continue
		}
		files, err := closure(tmp, "rocket/"+s.dir+"/client.ts")
		if err != nil {
			return fmt.Errorf("%s: %w", s.slug, err)
		}
		dir := filepath.Join("components", s.slug)
		if err := clear(dir); err != nil {
			return err
		}
		for _, f := range files {
			src, err := os.ReadFile(filepath.Join(tmp, f))
			if err != nil {
				return err
			}
			to := dest(s, f)
			code, err := rewrite(s, f, to, string(src))
			if err != nil {
				return fmt.Errorf("%s: %w", f, err)
			}
			if to == s.slug+".ts" {
				if code, err = literalTag(s.slug, code); err != nil {
					return fmt.Errorf("%s: %w", f, err)
				}
			} else if m := tagRe.FindStringSubmatch(code); m != nil && m[1] != "sb-"+s.slug {
				return fmt.Errorf("%s: its tag is %s, but the component is sb-%s", f, m[1], s.slug)
			}
			out := filepath.Join(dir, filepath.FromSlash(to))
			if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(out, []byte(header+code), 0o644); err != nil {
				return err
			}
		}
		if err := os.WriteFile(filepath.Join(dir, "LICENSE-pd-rockets.txt"), license, 0o644); err != nil {
			return err
		}
		fmt.Printf("%s: %d files from %s\n", dir, len(files), "rocket/"+s.dir)
	}
	fmt.Printf("vendored %s (%s) with %d patches; now run go tool task ts\n", tag, commit[:12], len(patches))
	return nil
}

// closure is entry and every module it imports relatively, transitively,
// as paths inside the clone.
func closure(root, entry string) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	var walk func(string) error
	walk = func(f string) error {
		if seen[f] {
			return nil
		}
		seen[f] = true
		out = append(out, f)
		src, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			return err
		}
		for _, m := range specRe.FindAllStringSubmatch(string(src), -1) {
			if strings.HasPrefix(m[2], ".") {
				if err := walk(path.Join(path.Dir(f), m[2]) + ".ts"); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return out, walk(entry)
}

// dest is where a file of the clone goes in the component's folder.
func dest(s struct{ slug, dir string }, f string) string {
	if f == "rocket/"+s.dir+"/client.ts" {
		return s.slug + ".ts"
	}
	if rest, ok := strings.CutPrefix(f, "rocket/"+s.dir+"/"); ok {
		return rest
	}
	return f // core/… and contracts/… keep their place
}

// rewrite points imports at 'datastar' and the folder's own .ts files and
// renames pd- to sb-.
func rewrite(s struct{ slug, dir string }, from, to, src string) (string, error) {
	var lines []string
	for _, l := range strings.Split(src, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "// @ts-ignore") && strings.Contains(l, "import map") {
			continue // the runtime import below is typed in Starbase
		}
		lines = append(lines, l)
	}
	src = strings.Join(lines, "\n")
	var bad error
	src = specRe.ReplaceAllStringFunc(src, func(m string) string {
		p := specRe.FindStringSubmatch(m)
		spec := p[2]
		switch {
		case spec == "pd-rockets/runtime":
			spec = "datastar"
		case strings.HasPrefix(spec, "."):
			target := dest(s, path.Join(path.Dir(from), spec)+".ts")
			rel, err := filepath.Rel(path.Dir(to), target)
			if err != nil {
				bad = err
			}
			spec = filepath.ToSlash(rel)
			if !strings.HasPrefix(spec, ".") {
				spec = "./" + spec
			}
		default:
			bad = fmt.Errorf("imports %q: only the runtime and relative modules can be vendored", spec)
		}
		return p[1] + `"` + spec + `"`
	})
	return renameRe.ReplaceAllString(src, "sb-"), bad
}

// literalTag writes the component's tag into its rocket() call.
func literalTag(slug, code string) (string, error) {
	if len(defineRe.FindAllString(code, -1)) != 1 {
		return "", fmt.Errorf("want one rocket(<contract>.tag, …) call")
	}
	return defineRe.ReplaceAllString(code, `rocket("sb-`+slug+`",`), nil
}

// clear removes what an earlier run vendored into dir: its TypeScript, the
// JavaScript compiled from it, and the core and contracts folders.
func clear(dir string) error {
	for _, sub := range []string{"core", "contracts"} {
		if err := os.RemoveAll(filepath.Join(dir, sub)); err != nil {
			return err
		}
	}
	old, _ := filepath.Glob(filepath.Join(dir, "*.ts"))
	for _, f := range old {
		os.Remove(f)
		os.Remove(strings.TrimSuffix(f, ".ts") + ".js")
	}
	return nil
}

// abs makes paths absolute: git am runs in the clone.
func abs(paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i], _ = filepath.Abs(p)
	}
	return out
}
