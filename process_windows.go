//go:build windows

package main

import "os/exec"

func setProcessGroup(cmd *exec.Cmd) {
	// Windows doesn't use PGID the same way Unix does
}

func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
