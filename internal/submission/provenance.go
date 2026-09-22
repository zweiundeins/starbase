package submission

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strings"
)

// Vendored third-party code can't be reviewed line by line (a minified
// library is megabytes of noise), so it has to be proven instead:
// vendor.json, next to the component, says which npm package each
// vendored file is from, and the bot checks the file is byte-for-byte the
// one in that package, as published (the tarball must match the registry's
// integrity hash).
//
//	{"vendor/echarts.esm.min.js": {"npm": "echarts@5.5.1", "file": "dist/echarts.esm.min.js"}}

const VendorManifest = "vendor.json"

type VendorEntry struct {
	NPM  string `json:"npm"`  // package@version, exact
	File string `json:"file"` // path inside the package
}

// VendorCheck is the verdict on one vendored file.
type VendorCheck struct {
	Path     string
	Verified bool
	Package  string // package@version, when verified
	File     string
	License  string // from the package, when verified
	Minified bool
	Problem  string // why it is not verified
}

var (
	npmNameRe    = regexp.MustCompile(`^(@[a-z0-9][a-z0-9._-]*/)?[a-z0-9][a-z0-9._-]*$`)
	npmVersionRe = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$`)
)

const maxPackage = 64 << 20 // compressed npm tarball

// ParseVendorManifest reads vendor.json. Its keys must be vendored files.
func ParseVendorManifest(b []byte, vendored map[string]string) (map[string]VendorEntry, error) {
	var m map[string]VendorEntry
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("vendor.json is not valid JSON: %v", err)
	}
	for p, e := range m {
		if _, ok := vendored[p]; !ok {
			return nil, fmt.Errorf("vendor.json lists `%s`, which the component doesn't import", p)
		}
		name, version, ok := splitPackage(e.NPM)
		if !ok || !npmNameRe.MatchString(name) || !npmVersionRe.MatchString(version) {
			return nil, fmt.Errorf("vendor.json: `%s` needs an exact npm package, like \"echarts@5.5.1\"", p)
		}
		if e.File == "" || !isCleanRel(e.File) {
			return nil, fmt.Errorf("vendor.json: `%s` needs the file's path inside the package, like \"dist/lib.esm.min.js\"", p)
		}
	}
	return m, nil
}

// VerifyVendor checks every vendored file against vendor.json and the npm
// registry (e.g. https://registry.npmjs.org). A file that isn't verified
// and looks minified is an error: nobody can review it.
func VerifyVendor(ctx context.Context, client *http.Client, registry string, vendored map[string]string, manifest map[string]VendorEntry) ([]VendorCheck, error) {
	var checks []VendorCheck
	var problems []string
	packages := map[string]*npmPackage{} // package@version → fetched once
	for _, p := range slices.Sorted(maps.Keys(vendored)) {
		code := vendored[p]
		c := VendorCheck{Path: p, Minified: looksMinified(code)}
		e, listed := manifest[p]
		switch {
		case !listed:
			c.Problem = "not listed in vendor.json"
		default:
			pkg, ok := packages[e.NPM]
			if !ok {
				var err error
				pkg, err = fetchPackage(ctx, client, registry, e.NPM)
				if err != nil {
					return nil, err
				}
				packages[e.NPM] = pkg
			}
			want, found := pkg.files[e.File]
			switch {
			case !found:
				c.Problem = fmt.Sprintf("%s has no file %s", e.NPM, e.File)
			case want != code:
				c.Problem = fmt.Sprintf("differs from %s in %s", e.File, e.NPM)
			default:
				c.Verified, c.Package, c.File, c.License = true, e.NPM, e.File, pkg.license
			}
		}
		if !c.Verified && c.Minified {
			problems = append(problems, fmt.Sprintf("`%s` looks minified and is %s. Vendored libraries must be unmodified files from an npm package, listed in `vendor.json` next to the component, e.g. `{\"%s\": {\"npm\": \"name@1.2.3\", \"file\": \"dist/%s\"}}`.", p, c.Problem, p, path.Base(p)))
		}
		checks = append(checks, c)
	}
	if len(problems) > 0 {
		return checks, errors.New(strings.Join(problems, "\n- "))
	}
	return checks, nil
}

type npmPackage struct {
	license string
	files   map[string]string // path inside the package → contents (.js and .mjs)
}

func fetchPackage(ctx context.Context, client *http.Client, registry, spec string) (*npmPackage, error) {
	name, version, _ := splitPackage(spec)
	var meta struct {
		License any `json:"license"`
		Dist    struct {
			Tarball   string `json:"tarball"`
			Integrity string `json:"integrity"`
		} `json:"dist"`
	}
	u := strings.TrimSuffix(registry, "/") + "/" + url.PathEscape(name) + "/" + version
	if err := getJSON(ctx, client, u, &meta); err != nil {
		return nil, fmt.Errorf("npm package %s: %w", spec, err)
	}
	algo, sum, ok := strings.Cut(meta.Dist.Integrity, "-")
	if !ok || algo != "sha512" || meta.Dist.Tarball == "" {
		return nil, fmt.Errorf("npm package %s has no sha512 integrity hash", spec)
	}
	want, err := base64.StdEncoding.DecodeString(sum)
	if err != nil {
		return nil, fmt.Errorf("npm package %s: bad integrity hash", spec)
	}
	if t, err := url.Parse(meta.Dist.Tarball); err != nil || !sameHost(t, registry) {
		return nil, fmt.Errorf("npm package %s: the tarball is not on the registry", spec)
	}
	tgz, err := get(ctx, client, meta.Dist.Tarball, maxPackage)
	if err != nil {
		return nil, fmt.Errorf("npm package %s: %w", spec, err)
	}
	if got := sha512.Sum512(tgz); !bytes.Equal(got[:], want) {
		return nil, fmt.Errorf("npm package %s: the tarball doesn't match the registry's integrity hash", spec)
	}
	pkg := &npmPackage{license: licenseString(meta.License), files: map[string]string{}}
	gz, err := gzip.NewReader(bytes.NewReader(tgz))
	if err != nil {
		return nil, fmt.Errorf("npm package %s: %w", spec, err)
	}
	tr := tar.NewReader(gz)
	read := 0
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("npm package %s: %w", spec, err)
		}
		// Entries are "package/<path>" (the top folder's name varies).
		_, rel, _ := strings.Cut(h.Name, "/")
		if h.Typeflag != tar.TypeReg || !(strings.HasSuffix(rel, ".js") || strings.HasSuffix(rel, ".mjs")) || h.Size > maxVendorFile {
			continue
		}
		b, err := io.ReadAll(io.LimitReader(tr, maxVendorFile))
		if err != nil {
			return nil, err
		}
		if read += len(b); read > maxRead {
			return nil, fmt.Errorf("npm package %s is too large to check", spec)
		}
		pkg.files[rel] = string(b)
	}
	return pkg, nil
}

func getJSON(ctx context.Context, client *http.Client, u string, v any) error {
	b, err := get(ctx, client, u, 8<<20)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func get(ctx context.Context, client *http.Client, u string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", u, res.Status)
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if err == nil && int64(len(b)) > limit {
		err = fmt.Errorf("%s is too large", u)
	}
	return b, err
}

// splitPackage splits "name@1.2.3" and "@scope/name@1.2.3".
func splitPackage(spec string) (name, version string, ok bool) {
	i := strings.LastIndex(spec, "@")
	if i <= 0 {
		return "", "", false
	}
	return spec[:i], spec[i+1:], true
}

func licenseString(v any) string {
	switch l := v.(type) {
	case string:
		return l
	case map[string]any:
		if t, ok := l["type"].(string); ok {
			return t
		}
	}
	return "unknown"
}

// looksMinified: long lines, which no one reviews.
func looksMinified(code string) bool {
	for line := range strings.Lines(code) {
		if len(line) > 500 {
			return true
		}
	}
	return false
}

func isCleanRel(p string) bool {
	return p == path.Clean(p) && !path.IsAbs(p) && p != ".." && !strings.HasPrefix(p, "../")
}

func sameHost(u *url.URL, registry string) bool {
	r, err := url.Parse(registry)
	return err == nil && u.Scheme == r.Scheme && strings.EqualFold(u.Host, r.Host)
}
