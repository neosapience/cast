//go:build linux

package cmd

import (
	"fmt"
	"os/exec"
)

// playFile tries common Linux audio players in order of preference.
// Attempts: paplay (PulseAudio), aplay (ALSA), ffplay, mpv
func playFile(path, format string) error {
	type player struct {
		cmd  string
		args []string
	}

	players := []player{
		{"paplay", []string{path}},
		{"aplay", []string{path}},
		{"ffplay", []string{"-nodisp", "-autoexit", path}},
		{"mpv", []string{"--no-video", path}},
		{"vlc", []string{"--intf", "dummy", "--play-and-exit", path}},
		{"play", []string{path}},
	}

	for _, p := range players {
		if _, err := exec.LookPath(p.cmd); err != nil {
			continue
		}
		if err := exec.Command(p.cmd, p.args...).Run(); err == nil {
			return nil
		}
	}

	return fmt.Errorf("no audio player could play the file; try saving with --out and playing manually")
}
