//go:build windows

package cmd

import (
	"os/exec"
	"strings"
)

func copyToClipboard(text string) error {
	cmd := exec.Command("clip")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
