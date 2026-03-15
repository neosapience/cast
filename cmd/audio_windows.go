//go:build windows

package cmd

import (
	"fmt"
	"os/exec"
)

func playFile(path, format string) error {
	switch format {
	case "wav", "":
		// Media.SoundPlayer only supports WAV
		script := fmt.Sprintf(`(New-Object Media.SoundPlayer '%s').PlaySync()`, path)
		return exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Run()
	default:
		// For mp3 and other formats, open with the default associated app and wait
		return exec.Command("cmd", "/C", "start", "/wait", "", path).Run()
	}
}
