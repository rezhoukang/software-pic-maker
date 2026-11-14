//go:build !windows

package raster

import "os/exec"

func hideWindow(cmd *exec.Cmd) {}
