//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

// detachProcess puts the backend in its own session, so closing the browser (and the host) does not stop it.
func detachProcess(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }
