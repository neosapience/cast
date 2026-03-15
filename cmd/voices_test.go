package cmd

import (
	"testing"

	"github.com/neosapience/cast/internal/client"
)

var testVoices = []client.Voice{
	{
		VoiceID:   "v1",
		VoiceName: "건석",
		Models: []client.VoiceModel{
			{Version: "ssfm-v21", Emotions: []string{"normal", "happy"}},
		},
	},
	{
		VoiceID:   "v2",
		VoiceName: "Alice",
		Models: []client.VoiceModel{
			{Version: "ssfm-v30", Emotions: []string{"normal", "sad"}},
		},
	},
	{
		VoiceID:   "v3",
		VoiceName: "건우",
		Models: []client.VoiceModel{
			{Version: "ssfm-v30", Emotions: []string{"angry"}},
		},
	},
}

func TestFilterVoicesByName(t *testing.T) {
	cases := []struct {
		name   string
		filter string
		want   []string
	}{
		{"exact match", "건석", []string{"v1"}},
		{"partial match", "건", []string{"v1", "v3"}},
		{"case insensitive", "alice", []string{"v2"}},
		{"no match", "없음", []string{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := filterVoicesByName(append([]client.Voice{}, testVoices...), tc.filter)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d voices, want %d", len(got), len(tc.want))
			}
			for i, v := range got {
				if v.VoiceID != tc.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, v.VoiceID, tc.want[i])
				}
			}
		})
	}
}

func TestFilterVoicesByEmotion(t *testing.T) {
	cases := []struct {
		name    string
		emotion string
		want    []string
	}{
		{"happy", "happy", []string{"v1"}},
		{"normal", "normal", []string{"v1", "v2"}},
		{"angry", "angry", []string{"v3"}},
		{"no match", "whisper", []string{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := filterVoicesByEmotion(append([]client.Voice{}, testVoices...), tc.emotion)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d voices, want %d", len(got), len(tc.want))
			}
			for i, v := range got {
				if v.VoiceID != tc.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, v.VoiceID, tc.want[i])
				}
			}
		})
	}
}
