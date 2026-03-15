package cmd

import (
	"fmt"
	"os"
)

func playAudio(audio []byte, format string) error {
	if format == "" {
		format = "wav"
	}
	tmp, err := os.CreateTemp("", "cast-*."+format)
	if err != nil {
		return err
	}

	if _, err := tmp.Write(audio); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	tmp.Close()

	err = playFile(tmp.Name(), format)
	os.Remove(tmp.Name())
	return err
}

func saveAudio(audio []byte, path string) error {
	if err := os.WriteFile(path, audio, 0644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "saved to %s\n", path)
	return nil
}
