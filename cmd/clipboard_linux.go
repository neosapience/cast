//go:build linux

package cmd

import (
	"os/exec"
	"strings"
)

func copyToClipboard(text string) error {
	for _, bin := range []string{"xclip", "xsel"} {
		path, err := exec.LookPath(bin)
		if err != nil {
			continue
		}
		var cmd *exec.Cmd
		if bin == "xclip" {
			cmd = exec.Command(path, "-selection", "clipboard")
		} else {
			cmd = exec.Command(path, "--clipboard", "--input")
		}
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	return exec.ErrNotFound
}
