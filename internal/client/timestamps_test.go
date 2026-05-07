package client

import "testing"

func TestFormatSRTTime(t *testing.T) {
	cases := []struct {
		seconds float64
		want    string
	}{
		{0, "00:00:00,000"},
		{1.5, "00:00:01,500"},
		{65.123, "00:01:05,123"},
		{3661.999, "01:01:01,999"},
	}
	for _, tc := range cases {
		if got := formatSRTTime(tc.seconds); got != tc.want {
			t.Errorf("formatSRTTime(%v) = %q, want %q", tc.seconds, got, tc.want)
		}
	}
}

func TestFormatVTTTime(t *testing.T) {
	if got, want := formatVTTTime(1.5), "00:00:01.500"; got != want {
		t.Errorf("formatVTTTime(1.5) = %q, want %q", got, want)
	}
}

func TestEndsInSentence(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"Hello.", true},
		{"What?", true},
		{"Wow!", true},
		{"こんにちは。", true},
		{"これは？", true},
		{"Hello. ", true},
		{"Hello", false},
		{"Hello,", false},
	}
	for _, tc := range cases {
		if got := endsInSentence(tc.s); got != tc.want {
			t.Errorf("endsInSentence(%q) = %v, want %v", tc.s, got, tc.want)
		}
	}
}

func TestPickSegmentsPrefersWords(t *testing.T) {
	resp := &TTSWithTimestampsResponse{
		Words: []AlignmentSegmentWord{
			{Text: "Hello", Start: 0, End: 0.5},
			{Text: "world", Start: 0.5, End: 1.0},
		},
		Characters: []AlignmentSegmentCharacter{
			{Text: "H", Start: 0, End: 0.1},
		},
	}
	segs, wordMode, err := resp.pickSegments()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !wordMode {
		t.Errorf("expected wordMode=true when words has >=2 entries")
	}
	if len(segs) != 2 {
		t.Errorf("expected 2 segments, got %d", len(segs))
	}
}

func TestPickSegmentsFallsBackToCharacters(t *testing.T) {
	resp := &TTSWithTimestampsResponse{
		Characters: []AlignmentSegmentCharacter{
			{Text: "あ", Start: 0, End: 0.1},
			{Text: "い", Start: 0.1, End: 0.2},
		},
	}
	segs, wordMode, err := resp.pickSegments()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wordMode {
		t.Errorf("expected wordMode=false when only characters present")
	}
	if len(segs) != 2 {
		t.Errorf("expected 2 segments, got %d", len(segs))
	}
}

func TestPickSegmentsErrorsWhenEmpty(t *testing.T) {
	resp := &TTSWithTimestampsResponse{}
	if _, _, err := resp.pickSegments(); err == nil {
		t.Errorf("expected error when both arrays empty")
	}
}

func TestGroupIntoCuesSplitsBySentence(t *testing.T) {
	segs := []captionSegment{
		{"Hello.", 0, 0.5},
		{"World.", 0.5, 1.0},
	}
	cues := groupIntoCues(segs, true)
	if len(cues) != 2 {
		t.Fatalf("expected 2 cues split at sentence ends, got %d", len(cues))
	}
	if cues[0].text != "Hello." || cues[1].text != "World." {
		t.Errorf("unexpected cue texts: %v", cues)
	}
}

func TestGroupIntoCuesSplitsOnSecondsLimit(t *testing.T) {
	segs := []captionSegment{
		{"a", 0, 4.0},
		{"b", 4.0, 8.0}, // 8 - 0 = 8s > 7s cap → must flush before adding 'b'
	}
	cues := groupIntoCues(segs, true)
	if len(cues) != 2 {
		t.Errorf("expected 2 cues split on 7s cap, got %d", len(cues))
	}
}

func TestGroupIntoCuesUnderCharLimitStaysSingleCue(t *testing.T) {
	long := "12345678901234567890123456789012345" // 35 chars
	segs := []captionSegment{
		{long, 0, 1.0},
		{"more", 1.0, 1.5}, // 35 + 1 + 4 = 40 chars (under 42 cap) and 1.5s (under 7s)
	}
	cues := groupIntoCues(segs, true)
	if len(cues) != 1 {
		t.Errorf("expected 1 cue under char cap, got %d", len(cues))
	}
}

func TestToSRT(t *testing.T) {
	resp := &TTSWithTimestampsResponse{
		Words: []AlignmentSegmentWord{
			{Text: "Hello.", Start: 0, End: 0.5},
			{Text: "World.", Start: 0.5, End: 1.0},
		},
	}
	got, err := resp.ToSRT()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "1\n00:00:00,000 --> 00:00:00,500\nHello.\n\n2\n00:00:00,500 --> 00:00:01,000\nWorld.\n\n"
	if got != want {
		t.Errorf("ToSRT mismatch:\n got: %q\nwant: %q", got, want)
	}
}

func TestToVTT(t *testing.T) {
	resp := &TTSWithTimestampsResponse{
		Words: []AlignmentSegmentWord{
			{Text: "Hello.", Start: 0, End: 0.5},
			{Text: "World.", Start: 0.5, End: 1.0},
		},
	}
	got, err := resp.ToVTT()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "WEBVTT\n\n00:00:00.000 --> 00:00:00.500\nHello.\n\n00:00:00.500 --> 00:00:01.000\nWorld.\n\n"
	if got != want {
		t.Errorf("ToVTT mismatch:\n got: %q\nwant: %q", got, want)
	}
}

func TestTextToSpeechWithTimestampsValidation(t *testing.T) {
	c := New("test-key")
	if _, err := c.TextToSpeechWithTimestamps(TTSRequestWithTimestamps{}, ""); err == nil {
		t.Errorf("expected error when voice_id is missing")
	}
	if _, err := c.TextToSpeechWithTimestamps(TTSRequestWithTimestamps{VoiceID: "v"}, ""); err == nil {
		t.Errorf("expected error when text is missing")
	}
	if _, err := c.TextToSpeechWithTimestamps(TTSRequestWithTimestamps{VoiceID: "v", Text: "t"}, ""); err == nil {
		t.Errorf("expected error when model is missing")
	}
}
