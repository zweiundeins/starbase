//go:build !unix

package main

// closeOnExec does nothing where systemd can't pass sockets.
func closeOnExec(int) {}
