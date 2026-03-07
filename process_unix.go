//go:build !windows
package main

import (
	"os/exec"
	"syscall"
)

func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		pgid, err := syscall.Getpgid(cmd.Process.Pid)
		if err == nil {
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		}
	}
}