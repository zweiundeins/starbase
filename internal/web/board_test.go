package web

import (
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	l := newLimiter(20, 60)
	now := time.Unix(1000, 0)
	if !l.allow("a", 60, now) {
		t.Fatal("burst should be allowed")
	}
	if l.allow("a", 1, now) {
		t.Fatal("bucket should be empty")
	}
	if !l.allow("b", 10, now) {
		t.Fatal("sessions are independent")
	}
	if !l.allow("a", 10, now.Add(500*time.Millisecond)) { // +10 tokens
		t.Fatal("refill: 20/s")
	}
	if l.allow("a", 1, now.Add(500*time.Millisecond)) {
		t.Fatal("refill must not exceed elapsed time")
	}
	if !l.allow("a", 60, now.Add(time.Hour)) || l.allow("a", 61, now.Add(2*time.Hour)) {
		t.Fatal("tokens cap at the burst size")
	}
}
