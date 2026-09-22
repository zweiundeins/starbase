package submission_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"starbase/internal/submission"
)

var minified = "/*! lib v1 | MIT */\n" + strings.Repeat("var a=1;", 100) + "\n"

// fakeNPM serves lib@1.0.0 with dist/lib.min.js = minified.
func fakeNPM(t *testing.T, tamperTarball bool, tarballHost string) *httptest.Server {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: "package/dist/lib.min.js", Mode: 0o644, Size: int64(len(minified)), Typeflag: tar.TypeReg})
	tw.Write([]byte(minified))
	tw.Close()
	gz.Close()
	tgz := buf.Bytes()
	sum := sha512.Sum512(tgz)
	if tamperTarball {
		tgz = append([]byte{}, tgz...)
		tgz[len(tgz)-1] ^= 1
	}
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/lib/1.0.0":
			host := srv.URL
			if tarballHost != "" {
				host = tarballHost
			}
			fmt.Fprintf(w, `{"license": "MIT", "dist": {"tarball": "%s/lib/-/lib-1.0.0.tgz", "integrity": "sha512-%s"}}`, host, base64.StdEncoding.EncodeToString(sum[:]))
		case "/lib/-/lib-1.0.0.tgz":
			w.Write(tgz)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestVerifyVendor(t *testing.T) {
	npm := fakeNPM(t, false, "")
	ctx := context.Background()
	manifest := map[string]submission.VendorEntry{"vendor/lib.js": {NPM: "lib@1.0.0", File: "dist/lib.min.js"}}

	checks, err := submission.VerifyVendor(ctx, npm.Client(), npm.URL, map[string]string{"vendor/lib.js": minified, "helper.js": "export const x = 1\n"}, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if len(checks) != 2 || !checks[1].Verified || checks[1].Package != "lib@1.0.0" || checks[1].License != "MIT" {
		t.Fatalf("checks = %+v", checks)
	}
	if checks[0].Verified || checks[0].Minified { // readable, unlisted: a warning, not an error
		t.Errorf("helper.js = %+v", checks[0])
	}

	// A modified copy of the library is not the library.
	if _, err := submission.VerifyVendor(ctx, npm.Client(), npm.URL, map[string]string{"vendor/lib.js": minified + "fetch('/steal')\n"}, manifest); err == nil || !strings.Contains(err.Error(), "differs from dist/lib.min.js") {
		t.Errorf("tampered file: err = %v", err)
	}
	// Minified and unlisted: rejected.
	if _, err := submission.VerifyVendor(ctx, npm.Client(), npm.URL, map[string]string{"vendor/lib.js": minified}, nil); err == nil || !strings.Contains(err.Error(), "looks minified") {
		t.Errorf("unlisted minified file: err = %v", err)
	}
	// The registry's integrity hash is checked.
	bad := fakeNPM(t, true, "")
	if _, err := submission.VerifyVendor(ctx, bad.Client(), bad.URL, map[string]string{"vendor/lib.js": minified}, manifest); err == nil || !strings.Contains(err.Error(), "integrity") {
		t.Errorf("tampered tarball: err = %v", err)
	}
	// The tarball must come from the registry itself.
	elsewhere := fakeNPM(t, false, "http://evil.example")
	if _, err := submission.VerifyVendor(ctx, elsewhere.Client(), elsewhere.URL, map[string]string{"vendor/lib.js": minified}, manifest); err == nil || !strings.Contains(err.Error(), "not on the registry") {
		t.Errorf("foreign tarball host: err = %v", err)
	}
}

func TestParseVendorManifest(t *testing.T) {
	vendored := map[string]string{"vendor/lib.js": ""}
	for _, tc := range []struct{ json, want string }{
		{`{"vendor/lib.js": {"npm": "lib@1.0.0", "file": "dist/lib.js"}}`, ""},
		{`{"vendor/lib.js": {"npm": "@scope/lib@1.0.0-rc.1", "file": "dist/lib.js"}}`, ""},
		{`{"vendor/other.js": {"npm": "lib@1.0.0", "file": "dist/lib.js"}}`, "doesn't import"},
		{`{"vendor/lib.js": {"npm": "lib@^1.0.0", "file": "dist/lib.js"}}`, "exact npm package"},
		{`{"vendor/lib.js": {"npm": "lib", "file": "dist/lib.js"}}`, "exact npm package"},
		{`{"vendor/lib.js": {"npm": "lib@1.0.0", "file": "../x.js"}}`, "path inside the package"},
		{`not json`, "not valid JSON"},
	} {
		_, err := submission.ParseVendorManifest([]byte(tc.json), vendored)
		if (tc.want == "") != (err == nil) || (err != nil && !strings.Contains(err.Error(), tc.want)) {
			t.Errorf("%s: err = %v, want %q", tc.json, err, tc.want)
		}
	}
}

// Against the real registry: STARBASE_NPM_LIVE=1 go test -run Live ./internal/submission/
func TestVerifyVendorLive(t *testing.T) {
	if os.Getenv("STARBASE_NPM_LIVE") == "" {
		t.Skip("set STARBASE_NPM_LIVE=1")
	}
	const spec, file = "canvas-confetti@1.9.3", "dist/confetti.module.mjs"
	res, err := http.Get("https://cdn.jsdelivr.net/npm/" + spec + "/" + file)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	code, _ := io.ReadAll(res.Body)
	checks, err := submission.VerifyVendor(context.Background(), http.DefaultClient, "https://registry.npmjs.org",
		map[string]string{"vendor/confetti.js": string(code)},
		map[string]submission.VendorEntry{"vendor/confetti.js": {NPM: spec, File: file}})
	if err != nil || !checks[0].Verified {
		t.Fatalf("checks = %+v, err = %v", checks, err)
	}
	t.Logf("%+v", checks[0])
}
