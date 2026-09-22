package web

import (
	"net/http/httptest"
	"testing"
)

func TestClientIP(t *testing.T) {
	for _, tc := range []struct{ remote, real, want string }{
		{"203.0.113.7:5000", "", "203.0.113.7"},
		{"203.0.113.7:5000", "198.51.100.1", "203.0.113.7"},  // not from the proxy: header ignored
		{"127.0.0.1:5000", "198.51.100.1", "198.51.100.1"},   // from the local proxy
		{"@", "198.51.100.1", "198.51.100.1"},                // Unix socket
		{"[2001:db8:1:2:3:4:5:6]:443", "", "2001:db8:1:2::"}, // IPv6 /64
		{"127.0.0.1:5000", "not an ip", "127.0.0.1"},
	} {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = tc.remote
		if tc.real != "" {
			r.Header.Set("X-Real-IP", tc.real)
		}
		if got := clientIP(r); got != tc.want {
			t.Errorf("remote %s, X-Real-IP %q: got %s, want %s", tc.remote, tc.real, got, tc.want)
		}
	}
}
