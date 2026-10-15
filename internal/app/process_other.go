//go:build !windows

package app

import "os/exec"

func exeExt() string                 { return "" }
func runLimited(cmd *exec.Cmd) error { return cmd.Run() }
