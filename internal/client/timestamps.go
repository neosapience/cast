package client

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"unicode/utf8"
)

// AlignmentSegmentWord represents a word-level alignment segment.
type AlignmentSegmentWord struct {
	Text  string  `json:"text"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// AlignmentSegmentCharacter represents a character-level alignment segment.
type AlignmentSegmentCharacter struct {
	Text  string  `json:"text"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// TTSRequestWithTimestamps mirrors TTSRequest. The optional granularity query
// parameter is passed as a method argument, not a body field.
type TTSRequestWithTimestamps struct {
	VoiceID  string     `json:"voice_id"`
	Text     string     `json:"text"`
	Model    string     `json:"model"`
	Language string     `json:"language,omitempty"`
	Prompt   *TTSPrompt `json:"prompt,omitempty"`
	Output   *TTSOutput `json:"output,omitempty"`
	Seed     *int       `json:"seed,omitempty"`
}

// TTSWithTimestampsResponse is the response payload from
// POST /v1/text-to-speech/with-timestamps.
type TTSWithTimestampsResponse struct {
	Audio         string                      `json:"audio"`
	AudioFormat   string                      `json:"audio_format"`
	AudioDuration float64                     `json:"audio_duration"`
	Words         []AlignmentSegmentWord      `json:"words"`
	Characters    []AlignmentSegmentCharacter `json:"characters"`
}

// TextToSpeechWithTimestamps calls POST /v1/text-to-speech/with-timestamps.
// granularity must be one of "", "word", "char", "both" (empty falls back to
// the server default which is "word"). For non-whitespace languages such as
// jpn or zho, prefer "char" or "both" — the server collapses an entire
// sentence into a single word segment otherwise.
func (c *Client) TextToSpeechWithTimestamps(req TTSRequestWithTimestamps, granularity string) (*TTSWithTimestampsResponse, error) {
	if req.VoiceID == "" {
		return nil, errors.New("voice_id is required")
	}
	if req.Text == "" {
		return nil, errors.New("text is required")
	}
	if req.Model == "" {
		return nil, errors.New("model is required")
	}

	path := "/v1/text-to-speech/with-timestamps"
	if granularity != "" {
		q := url.Values{}
		q.Set("granularity", granularity)
		path = path + "?" + q.Encode()
	}

	data, err := c.post(path, req)
	if err != nil {
		return nil, err
	}

	var resp TTSWithTimestampsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// AudioBytes decodes the base64-encoded audio field.
func (r *TTSWithTimestampsResponse) AudioBytes() ([]byte, error) {
	return base64.StdEncoding.DecodeString(r.Audio)
}

// SaveAudio writes the decoded audio bytes to path.
func (r *TTSWithTimestampsResponse) SaveAudio(path string) error {
	b, err := r.AudioBytes()
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

// ToSRT returns SRT-formatted captions. Returns an error if both word and
// character arrays are missing or if the joined text yields no non-empty cues.
func (r *TTSWithTimestampsResponse) ToSRT() (string, error) {
	cues, err := r.cues()
	if err != nil {
		return "", err
	}
	if len(cues) == 0 {
		return "", errors.New("no alignment segments to caption from")
	}
	var sb strings.Builder
	for i, c := range cues {
		fmt.Fprintf(&sb, "%d\n", i+1)
		fmt.Fprintf(&sb, "%s --> %s\n", formatSRTTime(c.start), formatSRTTime(c.end))
		sb.WriteString(c.text)
		sb.WriteString("\n\n")
	}
	return sb.String(), nil
}

// ToVTT returns WebVTT-formatted captions.
func (r *TTSWithTimestampsResponse) ToVTT() (string, error) {
	cues, err := r.cues()
	if err != nil {
		return "", err
	}
	if len(cues) == 0 {
		return "", errors.New("no alignment segments to caption from")
	}
	var sb strings.Builder
	sb.WriteString("WEBVTT\n\n")
	for _, c := range cues {
		fmt.Fprintf(&sb, "%s --> %s\n", formatVTTTime(c.start), formatVTTTime(c.end))
		sb.WriteString(c.text)
		sb.WriteString("\n\n")
	}
	return sb.String(), nil
}

// --- caption helpers (must match Python/JS/Go SDK byte-for-byte) ---

const (
	maxCaptionSeconds = 7.0
	maxCaptionChars   = 42
)

var sentenceTerminators = []string{".", "?", "!", "。", "？", "！"}

type captionCue struct {
	text  string
	start float64
	end   float64
}

type captionSegment struct {
	text  string
	start float64
	end   float64
}

func (r *TTSWithTimestampsResponse) pickSegments() ([]captionSegment, bool, error) {
	if len(r.Words) >= 2 {
		segs := make([]captionSegment, len(r.Words))
		for i, w := range r.Words {
			segs[i] = captionSegment{w.Text, w.Start, w.End}
		}
		return segs, true, nil
	}
	if len(r.Characters) >= 1 {
		segs := make([]captionSegment, len(r.Characters))
		for i, ch := range r.Characters {
			segs[i] = captionSegment{ch.Text, ch.Start, ch.End}
		}
		return segs, false, nil
	}
	if len(r.Words) == 1 {
		segs := []captionSegment{{r.Words[0].Text, r.Words[0].Start, r.Words[0].End}}
		return segs, true, nil
	}
	return nil, false, errors.New("no alignment segments to caption from")
}

func endsInSentence(s string) bool {
	trimmed := strings.TrimRight(s, " \t\n\r")
	for _, t := range sentenceTerminators {
		if strings.HasSuffix(trimmed, t) {
			return true
		}
	}
	return false
}

func joinParts(parts []string, wordMode bool) string {
	var sep string
	if wordMode {
		sep = " "
	}
	return strings.TrimSpace(strings.Join(parts, sep))
}

func groupIntoCues(segs []captionSegment, wordMode bool) []captionCue {
	cues := []captionCue{}
	parts := []string{}
	var curStart float64
	curStartSet := false
	var lastEnd float64
	lastEndSet := false

	flush := func(endTime float64) {
		text := joinParts(parts, wordMode)
		if text != "" && curStartSet {
			cues = append(cues, captionCue{text, curStart, endTime})
		}
	}

	for _, seg := range segs {
		if len(parts) > 0 && curStartSet && lastEndSet {
			tentative := append([]string{}, parts...)
			tentative = append(tentative, seg.text)
			wouldBeText := joinParts(tentative, wordMode)
			wouldExceedSeconds := (seg.end - curStart) > maxCaptionSeconds
			wouldExceedChars := utf8.RuneCountInString(wouldBeText) > maxCaptionChars
			if wouldExceedSeconds || wouldExceedChars {
				flush(lastEnd)
				parts = parts[:0]
				curStartSet = false
			}
		}

		if !curStartSet {
			curStart = seg.start
			curStartSet = true
		}
		parts = append(parts, seg.text)
		lastEnd = seg.end
		lastEndSet = true

		if endsInSentence(seg.text) {
			flush(seg.end)
			parts = parts[:0]
			curStartSet = false
		}
	}
	if len(parts) > 0 && lastEndSet {
		flush(lastEnd)
	}
	return cues
}

func (r *TTSWithTimestampsResponse) cues() ([]captionCue, error) {
	segs, wordMode, err := r.pickSegments()
	if err != nil {
		return nil, err
	}
	return groupIntoCues(segs, wordMode), nil
}

func formatSRTTime(seconds float64) string {
	totalMs := int64(seconds*1000 + 0.5) // round to nearest ms
	ms := totalMs % 1000
	totalSec := totalMs / 1000
	ss := totalSec % 60
	totalMin := totalSec / 60
	mm := totalMin % 60
	hh := totalMin / 60
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hh, mm, ss, ms)
}

func formatVTTTime(seconds float64) string {
	return strings.Replace(formatSRTTime(seconds), ",", ".", 1)
}
