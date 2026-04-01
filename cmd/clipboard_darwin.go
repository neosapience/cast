//go:build darwin

package cmd

import (
	"os/exec"
	"strings"
)

func copyToClipboard(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
