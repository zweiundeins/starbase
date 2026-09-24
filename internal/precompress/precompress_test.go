package precompress

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
)

func TestAccepts(t *testing.T) {
	for _, c := range []struct {
		header, coding string
		want           bool
	}{
		{"gzip, deflate, br, zstd", "br", true},
		{"gzip;q=1.0, br;q=0.5", "br", true},
		{"br;q=0", "br", false},
		{"gzip, br; q=0", "br", false},
		{"gzip", "br", false},
		{"", "gzip", false},
		{"GZIP", "gzip", true},
	} {
		if got := accepts(c.header, c.coding); got != c.want {
			t.Errorf("accepts(%q, %q) = %v", c.header, c.coding, got)
		}
	}
}

func TestWrite(t *testing.T) {
	body := []byte(strings.Repeat("rocket('sb-x', { render: () => html`<p>hi</p>` })\n", 100))
	get := func(accept string, b []byte) (*httptest.ResponseRecorder, []byte) {
		r := httptest.NewRequest("GET", "/x.js", nil)
		r.Header.Set("Accept-Encoding", accept)
		w := httptest.NewRecorder()
		Write(w, r, b)
		var out []byte
		switch w.Header().Get("Content-Encoding") {
		case "br":
			out, _ = io.ReadAll(brotli.NewReader(w.Body))
		case "gzip":
			zr, _ := gzip.NewReader(w.Body)
			out, _ = io.ReadAll(zr)
		default:
			out = w.Body.Bytes()
		}
		return w, out
	}
	for accept, enc := range map[string]string{"gzip, br": "br", "gzip": "gzip", "": "", "br;q=0, gzip": "gzip"} {
		w, out := get(accept, body)
		if w.Header().Get("Content-Encoding") != enc || !bytes.Equal(out, body) || w.Header().Get("Vary") != "Accept-Encoding" {
			t.Errorf("Accept-Encoding %q: got %q, round trip ok=%v, Vary %q", accept, w.Header().Get("Content-Encoding"), bytes.Equal(out, body), w.Header().Get("Vary"))
		}
	}
	// Tiny bodies go out as they are.
	if w, _ := get("br", []byte("tiny")); w.Header().Get("Content-Encoding") != "" {
		t.Error("a tiny body should not be compressed")
	}
	// The same bytes are compressed once.
	a, _ := Get(body)
	b, _ := Get(body)
	if &a[0] != &b[0] {
		t.Error("Get should return the cached bytes")
	}
}
