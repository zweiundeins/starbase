//go:build !linux

package tscheck

// limitMemory relies on GOMEMLIMIT and the timeout alone.
func limitMemory(int) {}
