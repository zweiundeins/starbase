package web

import (
	"net/http/httptest"
	"testing"

	"github.com/zweiundeins/lith/edge"
)

// The rate limits' client key believes X-Real-IP only from the local proxy
// (Caddy, over a Unix socket in production).
func TestClientKey(t *testing.T) {
	for _, tc := range []struct{ remote, real, xff, want string }{
		{"203.0.113.7:5000", "", "", "203.0.113.7"},
		{"203.0.113.7:5000", "198.51.100.1", "", "203.0.113.7"},     // not from the proxy: header ignored
		{"203.0.113.7:5000", "", "198.51.100.1", "203.0.113.7"},     // nor X-Forwarded-For
		{"127.0.0.1:5000", "198.51.100.1", "", "198.51.100.1"},      // from the local proxy
		{"@", "198.51.100.1", "", "198.51.100.1"},                   // Unix socket
		{"@", "", "", "local"},                                      // Unix socket, no one named
		{"[2001:db8:1:2:3:4:5:6]:443", "", "", "2001:db8:1:2::/64"}, // IPv6 /64
		{"127.0.0.1:5000", "not an ip", "", "127.0.0.1"},
		// The proxy appends the address it saw; what the client sent comes first.
		{"127.0.0.1:5000", "", "198.51.100.1", "198.51.100.1"},
		{"127.0.0.1:5000", "", "192.0.2.66, 198.51.100.1", "198.51.100.1"},
		{"@", "192.0.2.66", "192.0.2.66, 198.51.100.1", "198.51.100.1"}, // a spoofed X-Real-IP passed on
	} {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = tc.remote
		if tc.real != "" {
			r.Header.Set("X-Real-IP", tc.real)
		}
		if tc.xff != "" {
			r.Header.Set("X-Forwarded-For", tc.xff)
		}
		if got := edge.ClientKey(r); got != tc.want {
			t.Errorf("remote %s, X-Real-IP %q, X-Forwarded-For %q: got %s, want %s", tc.remote, tc.real, tc.xff, got, tc.want)
		}
	}
}
