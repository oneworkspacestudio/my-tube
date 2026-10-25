//go:build !windows

package app

import (
	"os"
	"os/exec"
	"syscall"
)

func runYouTubeLimited(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	if e := cmd.Start(); e != nil {
		return e
	}
	defer syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	return cmd.Wait()
}
