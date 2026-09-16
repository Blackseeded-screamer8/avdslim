//go:build !windows

package host

import (
	"os/exec"
	"syscall"
)

// SetDetached configures the command to run in its own process group so that
// the emulator process continues running independently after avdslim exits.
func SetDetached(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}
