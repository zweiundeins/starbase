//go:build !unix

package web_test

import "time"

func cpuTime() time.Duration { return 0 }
