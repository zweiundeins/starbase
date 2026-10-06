package tscheck

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf16"
)

//go:embed dist
var dist embed.FS

//go:embed types
var types embed.FS

// Limits of one compiler run. A check of a large component takes about
// 50 ms and 60 MB.
const (
	timeout    = 10 * time.Second
	maxOutput  = 1 << 20
	maxResults = 100
)

// maxData is the compiler's RLIMIT_DATA, where the platform has one.
var maxData uint64 = 192 << 20

// ErrRefused is the answer to code that makes the compiler read a file
// other than its own, the lib files and the Datastar declarations.
var ErrRefused = errors.New("tscheck: the code reaches outside its folder")

// Checker runs the embedded compiler, one check at a time.
type Checker struct {
	dataDir  string
	platform string
	once     sync.Once
	dir      string // the unpacked compiler, with the declarations in types/
	err      error
	slot     chan struct{}
}

// New returns a Checker that unpacks the embedded compiler under dataDir,
// or nil when the binary embeds none for this platform.
func New(dataDir string) *Checker {
	p := Platform(runtime.GOOS, runtime.GOARCH)
	if _, err := fs.Stat(dist, "dist/"+Tarball(p)); p == "" || err != nil {
		return nil
	}
	return &Checker{dataDir: dataDir, platform: p, slot: make(chan struct{}, 1)}
}

// Warm unpacks the compiler now, so the first check doesn't wait for it.
func (c *Checker) Warm() error {
	if c == nil {
		return nil
	}
	c.once.Do(func() { c.dir, c.err = c.unpack() })
	return c.err
}

// unpack puts the compiler and the declarations into a folder named by
// their hashes, unless an earlier start did, and removes older ones.
func (c *Checker) unpack() (string, error) {
	h := sha256.New()
	io.WriteString(h, integrity[c.platform])
	err := fs.WalkDir(types, "types", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := types.ReadFile(name)
		fmt.Fprintf(h, "\x00%s\x00%d\x00", name, len(b))
		h.Write(b)
		return err
	})
	if err != nil {
		return "", err
	}
	base, err := filepath.Abs(c.dataDir)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "typescript-"+Version+"-"+hex.EncodeToString(h.Sum(nil))[:12])
	if _, err := os.Stat(dir); err == nil {
		return filepath.EvalSymlinks(dir)
	}
	tmp, err := os.MkdirTemp(base, ".typescript-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	f, err := dist.Open("dist/" + Tarball(c.platform))
	if err != nil {
		return "", err
	}
	defer f.Close()
	sum := sha512.New()
	tgz := io.TeeReader(f, sum)
	if err := Unpack(tgz, tmp); err != nil {
		return "", err
	}
	io.Copy(io.Discard, tgz)
	if err := verify(c.platform, sum.Sum(nil)); err != nil {
		return "", err
	}
	sub, _ := fs.Sub(types, "types")
	if err := os.CopyFS(filepath.Join(tmp, "types"), sub); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, dir); err != nil {
		return "", err
	}
	old, _ := filepath.Glob(filepath.Join(base, "typescript-*"))
	crashed, _ := filepath.Glob(filepath.Join(base, ".typescript-*"))
	for _, o := range append(old, crashed...) {
		if o != dir {
			os.RemoveAll(o)
		}
	}
	return filepath.EvalSymlinks(dir)
}

// File is a module of the code to check, named relative to its folder.
type File struct {
	Name   string
	Source string
}

// Diagnostic is an error the compiler reports in the checked module.
type Diagnostic struct {
	Line    int    `json:"line"`   // 1-based
	Col     int    `json:"col"`    // 1-based, in UTF-16 code units like a JS string index
	Length  int    `json:"length"` // of Text, at least 1
	Text    string `json:"text"`   // the word at Line:Col, so an editor can tell when it moved
	Code    int    `json:"code"`
	Message string `json:"message"`
	File    string `json:"file,omitempty"` // set by Compile, which reports every file's
}

// Check type-checks files[0], with the other files as the modules it may
// import relatively and 'datastar' as the patched Datastar build. A .ts
// module is checked in strict mode, a .js one only when it opts in with a
// // @ts-check comment (and not strictly, since JavaScript has no
// annotations to satisfy it). It returns the first 100 diagnostics in
// files[0], in order.
func (c *Checker) Check(ctx context.Context, files []File) ([]Diagnostic, error) {
	if len(files) == 0 {
		return nil, errors.New("tscheck: nothing to check")
	}
	for _, f := range files {
		if !validName(f.Name) {
			return nil, fmt.Errorf("tscheck: bad file name %q", f.Name)
		}
	}
	strict := strings.HasSuffix(files[0].Name, "ts")
	work, done, err := c.workspace(ctx, files, map[string]any{"strict": strict, "noEmit": true}, files[0].Name)
	if err != nil {
		return nil, err
	}
	defer done()

	// The first run only resolves the program: every file the compiler
	// would read must be one of these, or nothing of the check is shown.
	out, exit, err := c.run(ctx, work, "--listFilesOnly")
	if err == nil && exit != 0 {
		err = fmt.Errorf("tscheck: listing exited with %d: %s", exit, firstLine(out))
	}
	if err != nil {
		return nil, err
	}
	for _, f := range strings.Split(strings.TrimSpace(out), "\n") {
		f = filepath.Clean(f)
		if !strings.HasPrefix(f, work+string(filepath.Separator)) && !strings.HasPrefix(f, c.dir+string(filepath.Separator)) {
			return nil, ErrRefused
		}
	}
	if out, exit, err = c.run(ctx, work, "--pretty", "false"); err != nil {
		return nil, err
	}
	// 1 and 2 mean the program has errors, but 2 is also a crash of the
	// compiler's runtime (out of memory, past maxData).
	if exit != 0 && !strings.Contains(out, "error TS") {
		return nil, fmt.Errorf("tscheck: exited with %d: %s", exit, firstLine(out))
	}
	return parse(out, files[:1], false)
}

// Compile type-checks the .ts files among files strictly, with the others as
// the modules they may import (.d.ts files declare types), and returns the
// JavaScript the compiler emits for each .ts, comments kept, by the name of
// its .js. When the code has errors, it returns their diagnostics (with File
// set) instead.
func (c *Checker) Compile(ctx context.Context, files []File) (map[string]string, []Diagnostic, error) {
	var roots, emitted []string
	for _, f := range files {
		if !validName(f.Name) {
			return nil, nil, fmt.Errorf("tscheck: bad file name %q", f.Name)
		}
		if strings.HasSuffix(f.Name, ".ts") {
			roots = append(roots, f.Name)
			if !strings.HasSuffix(f.Name, ".d.ts") {
				emitted = append(emitted, f.Name)
			}
		}
	}
	work, done, err := c.workspace(ctx, files, map[string]any{
		"strict": true, "outDir": "out", "rootDir": ".", "removeComments": false, "newLine": "lf",
		"noEmitOnError": true, "rewriteRelativeImportExtensions": true,
	}, roots...)
	if err != nil {
		return nil, nil, err
	}
	defer done()
	out, exit, err := c.run(ctx, work, "--pretty", "false")
	if err != nil {
		return nil, nil, err
	}
	if exit != 0 {
		ds, err := parse(out, files, true)
		if err == nil && len(ds) == 0 {
			err = fmt.Errorf("tscheck: exited with %d: %s", exit, firstLine(out))
		}
		return nil, ds, err
	}
	js := map[string]string{}
	for _, r := range emitted {
		name := strings.TrimSuffix(r, ".ts") + ".js"
		b, err := os.ReadFile(filepath.Join(work, "out", filepath.FromSlash(name)))
		if err != nil {
			return nil, nil, err
		}
		js[name] = string(b)
	}
	return js, nil, nil
}

// workspace writes files and a tsconfig.json with the options every run
// shares, plus these, into a temporary folder, and holds the one slot
// for runs until done is called.
func (c *Checker) workspace(ctx context.Context, files []File, options map[string]any, roots ...string) (string, func(), error) {
	if err := c.Warm(); err != nil {
		return "", nil, err
	}
	select {
	case c.slot <- struct{}{}:
	case <-ctx.Done():
		return "", nil, ctx.Err()
	}
	work, err := os.MkdirTemp("", "tscheck-")
	done := func() {
		os.RemoveAll(work)
		<-c.slot
	}
	if err == nil {
		work, err = filepath.EvalSymlinks(work)
	}
	for _, f := range files {
		if err != nil {
			break
		}
		name := filepath.Join(work, filepath.FromSlash(f.Name))
		if err = os.MkdirAll(filepath.Dir(name), 0o755); err == nil {
			err = os.WriteFile(name, []byte(f.Source), 0o644)
		}
	}
	opts := map[string]any{
		"target": "esnext", "module": "esnext", "moduleResolution": "bundler",
		"lib": []string{"esnext", "dom", "dom.iterable"}, "types": []string{},
		"allowJs": true, "checkJs": false, "skipLibCheck": true, "allowImportingTsExtensions": true,
		"paths": map[string][]string{"datastar": {filepath.Join(c.dir, "types", "bundles", "datastar-rocket.d.ts")}},
	}
	maps.Copy(opts, options)
	config, _ := json.Marshal(map[string]any{"compilerOptions": opts, "files": roots})
	if err == nil {
		err = os.WriteFile(filepath.Join(work, "tsconfig.json"), config, 0o644)
	}
	if err != nil {
		done()
		return "", nil, err
	}
	return work, done, nil
}

// run runs the compiler on work/tsconfig.json and returns what it printed
// and its exit code.
func (c *Checker) run(ctx context.Context, work string, args ...string) (string, int, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, filepath.Join(c.dir, "tsc"), append([]string{"-p", "tsconfig.json"}, args...)...)
	if runtime.GOOS == "windows" {
		cmd.Path += ".exe"
	}
	cmd.Dir = work
	cmd.Env = []string{"GOMEMLIMIT=128MiB", "GOMAXPROCS=2", "SYSTEMROOT=" + os.Getenv("SYSTEMROOT")}
	out := &capped{max: maxOutput}
	cmd.Stdout, cmd.Stderr = out, out
	cmd.WaitDelay = time.Second
	if err := cmd.Start(); err != nil {
		return "", 0, err
	}
	limitMemory(cmd.Process.Pid)
	err := cmd.Wait()
	if ctx.Err() == context.DeadlineExceeded {
		return "", 0, fmt.Errorf("tscheck: no answer within %v", timeout)
	}
	if ctx.Err() != nil {
		return "", 0, ctx.Err()
	}
	var exit *exec.ExitError
	if err != nil && (!errors.As(err, &exit) || exit.ExitCode() > 2 || exit.ExitCode() < 0) {
		return "", 0, fmt.Errorf("tscheck: %v: %s", err, firstLine(out.String()))
	}
	return out.String(), cmd.ProcessState.ExitCode(), nil
}

var diagLine = regexp.MustCompile(`^(.+)\((\d+),(\d+)\): error TS(\d+): (.*)$`)

// parse reads the compiler's --pretty false output: a line per diagnostic,
// followed by indented lines that continue its message. It keeps the
// diagnostics in files, with their File set when named is true.
func parse(out string, files []File, named bool) ([]Diagnostic, error) {
	lines := map[string][]string{}
	for _, f := range files {
		lines[f.Name] = strings.Split(f.Source, "\n")
	}
	var ds []Diagnostic
	var last *Diagnostic // the diagnostic continuation lines belong to, if it is kept
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		l := sc.Text()
		if rest, ok := strings.CutPrefix(l, "  "); ok {
			if last != nil {
				last.Message += "\n" + rest
			}
			continue
		}
		last = nil
		m := diagLine.FindStringSubmatch(l)
		if m == nil {
			if strings.Contains(l, "error TS") {
				return nil, fmt.Errorf("tscheck: %s", l)
			}
			continue
		}
		name := filepath.ToSlash(m[1])
		src, ok := lines[name]
		if !ok || len(ds) == maxResults {
			continue
		}
		line, _ := strconv.Atoi(m[2])
		col, _ := strconv.Atoi(m[3])
		code, _ := strconv.Atoi(m[4])
		w := wordAt(src, line, col)
		ds = append(ds, Diagnostic{Line: line, Col: col, Length: max(len(w), 1), Text: string(utf16.Decode(w)), Code: code, Message: m[5]})
		if named {
			ds[len(ds)-1].File = name
		}
		last = &ds[len(ds)-1]
	}
	return ds, nil
}

// wordAt is the identifier or string at line:col (1-based, in UTF-16 code
// units), or the one code unit there when it starts neither.
func wordAt(lines []string, line, col int) []uint16 {
	if line < 1 || line > len(lines) {
		return nil
	}
	u := utf16.Encode([]rune(lines[line-1]))
	if col < 1 || col > len(u) {
		return nil
	}
	if q := u[col-1]; q == '\'' || q == '"' || q == '`' {
		for end := col; end < len(u); end++ {
			if u[end] == q && u[end-1] != '\\' {
				return u[col-1 : end+1]
			}
		}
	}
	end := col - 1
	for end < len(u) {
		if r := rune(u[end]); r != '_' && r != '$' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			break
		}
		end++
	}
	return u[col-1 : max(end, col)]
}

var segment = regexp.MustCompile(`^[A-Za-z0-9_-][A-Za-z0-9._-]*$`)

// validName accepts a slash-separated path inside the folder to a module.
func validName(name string) bool {
	if path.Clean(name) != name || !(strings.HasSuffix(name, ".js") || strings.HasSuffix(name, ".mjs") || strings.HasSuffix(name, ".ts") || strings.HasSuffix(name, ".mts")) {
		return false
	}
	for _, s := range strings.Split(name, "/") {
		if !segment.MatchString(s) {
			return false
		}
	}
	return true
}

// capped keeps the first max bytes written to it.
type capped struct {
	bytes.Buffer
	max int
}

func (c *capped) Write(p []byte) (int, error) {
	if room := c.max - c.Len(); room > 0 {
		c.Buffer.Write(p[:min(len(p), room)])
	}
	return len(p), nil
}

func firstLine(s string) string {
	s, _, _ = strings.Cut(strings.TrimSpace(s), "\n")
	return s
}
