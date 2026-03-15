//go:build darwin

package cmd

import "os/exec"

func playFile(path, format string) error {
	return exec.Command("afplay", path).Run()
}
