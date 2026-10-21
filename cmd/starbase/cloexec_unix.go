//go:build unix

package main

import "syscall"

func closeOnExec(fd int) { syscall.CloseOnExec(fd) }
