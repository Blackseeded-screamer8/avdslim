//go:build windows

package host

import (
	"os/exec"
	"syscall"
)

// SetDetached configures the command to run detached on Windows.
func SetDetached(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
