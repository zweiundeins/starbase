// Package precompress keeps brotli and gzip versions of static bytes: the
// module files, the bundles and the static assets never change for a given
// URL, so they are compressed once at the best levels (brotli -11, gzip -9)
// and served as they are, instead of being compressed on every request at a
// level that is fast enough to do per request.
package precompress

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"math/bits"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/andybalholm/brotli"
)

// MinSize is the smallest body worth compressing: below it the headers cost
// more than they save.
const MinSize = 512

type entry struct {
	once   sync.Once
	br, gz []byte
}

var cache sync.Map // sha256 of the body → *entry

// Get returns the brotli and gzip versions of b, compressed once per content
// (concurrent callers for the same bytes wait for the one compression).
func Get(b []byte) (br, gz []byte) {
	sum := sha256.Sum256(b)
	e, _ := cache.LoadOrStore(sum, &entry{})
	en := e.(*entry)
	en.once.Do(func() { en.br, en.gz = compress(b) })
	return en.br, en.gz
}

func compress(b []byte) (br, gz []byte) {
	var gb bytes.Buffer
	gw, _ := gzip.NewWriterLevel(&gb, gzip.BestCompression)
	gw.Write(b)
	gw.Close()
	return brotliBytes(b), gb.Bytes()
}

// BrotliLen is the size of b under brotli -11, like Get's, but measured
// fresh and not kept: for bytes that are only measured, never served (the
// playground's live size line).
func BrotliLen(b []byte) int {
	return len(brotliBytes(b))
}

func brotliBytes(b []byte) []byte {
	var bb bytes.Buffer
	// A window just larger than the input compresses the same and keeps the
	// encoder's memory small (the default window is 4 MB).
	win := max(10, min(24, bits.Len(uint(len(b)))+1))
	bw := brotli.NewWriterOptions(&bb, brotli.WriterOptions{Quality: brotli.BestCompression, LGWin: win})
	bw.Write(b)
	bw.Close()
	return bb.Bytes()
}

// Write sends body in the best encoding the request accepts (brotli, then
// gzip), or as it is. The Content-Type and caching headers are the caller's.
// A response with Content-Encoding set passes the compression middleware
// untouched.
func Write(w http.ResponseWriter, r *http.Request, body []byte) {
	h := w.Header()
	h.Add("Vary", "Accept-Encoding")
	if len(body) >= MinSize {
		accept := r.Header.Get("Accept-Encoding")
		br, gz := Get(body)
		switch {
		case accepts(accept, "br") && len(br) < len(body):
			body = br
			h.Set("Content-Encoding", "br")
		case accepts(accept, "gzip") && len(gz) < len(body):
			body = gz
			h.Set("Content-Encoding", "gzip")
		}
	}
	h.Set("Content-Length", strconv.Itoa(len(body)))
	w.Write(body)
}

// accepts reports whether an Accept-Encoding header allows coding (a listed
// coding with q=0 is refused).
func accepts(header, coding string) bool {
	for _, part := range strings.Split(header, ",") {
		name, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		if !strings.EqualFold(strings.TrimSpace(name), coding) {
			continue
		}
		q := strings.ReplaceAll(strings.TrimSpace(params), " ", "")
		return q != "q=0" && q != "q=0.0" && q != "q=0.00" && q != "q=0.000"
	}
	return false
}
