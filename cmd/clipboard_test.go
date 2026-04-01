package cmd

import "testing"

func TestCopyToClipboard(t *testing.T) {
	err := copyToClipboard("test-voice-id")
	if err != nil {
		t.Skipf("clipboard not available: %v", err)
	}
}
