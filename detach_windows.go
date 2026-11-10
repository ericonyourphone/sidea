package main

import (
	"os/exec"
	"syscall"
)

// detachedProcess is DETACHED_PROCESS: no console shared with sidea's.
const detachedProcess = 0x00000008

func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: detachedProcess}
}
