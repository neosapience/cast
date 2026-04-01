package cmd

import (
	"testing"

	"github.com/neosapience/cast/internal/client"
)

func testPickVoices() []client.Voice {
	return []client.Voice{
		{VoiceID: "v1", VoiceName: "수진", Gender: "female", Age: "young_adult", UseCases: []string{"Conversational", "News"}},
		{VoiceID: "v2", VoiceName: "건석", Gender: "male", Age: "middle_age", UseCases: []string{"Documentary"}},
		{VoiceID: "v3", VoiceName: "수아", Gender: "female", Age: "teenager", UseCases: []string{"Anime"}},
	}
}

func TestPickModelFilter(t *testing.T) {
	voices := testPickVoices()
	m := newPickModel(voices, nil, defaultPreviewText)

	if len(m.filtered) != 3 {
		t.Fatalf("initial filtered = %d, want 3", len(m.filtered))
	}

	m.textInput.SetValue("수")
	m.applyFilter()

	if len(m.filtered) != 2 {
		t.Fatalf("after '수' filtered = %d, want 2", len(m.filtered))
	}
	if m.filtered[0].VoiceID != "v1" || m.filtered[1].VoiceID != "v3" {
		t.Errorf("got %v %v, want v1, v3", m.filtered[0].VoiceID, m.filtered[1].VoiceID)
	}
}

func TestPickModelFilterNoMatch(t *testing.T) {
	voices := testPickVoices()
	m := newPickModel(voices, nil, defaultPreviewText)

	m.textInput.SetValue("없는이름")
	m.applyFilter()

	if len(m.filtered) != 0 {
		t.Fatalf("filtered = %d, want 0", len(m.filtered))
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

func TestPickModelCursorBounds(t *testing.T) {
	voices := testPickVoices()
	m := newPickModel(voices, nil, defaultPreviewText)

	m.cursor = 2
	m.textInput.SetValue("건석")
	m.applyFilter()

	if m.cursor != 0 {
		t.Errorf("cursor after filter = %d, want 0", m.cursor)
	}
}

func TestPickModelClearFilter(t *testing.T) {
	voices := testPickVoices()
	m := newPickModel(voices, nil, defaultPreviewText)

	m.textInput.SetValue("수")
	m.applyFilter()
	if len(m.filtered) != 2 {
		t.Fatalf("filtered = %d, want 2", len(m.filtered))
	}

	m.textInput.SetValue("")
	m.applyFilter()
	if len(m.filtered) != 3 {
		t.Fatalf("after clear filtered = %d, want 3", len(m.filtered))
	}
}

func TestPickModelFilterByUseCase(t *testing.T) {
	voices := testPickVoices()
	m := newPickModel(voices, nil, defaultPreviewText)

	m.textInput.SetValue("anime")
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("filtered = %d, want 1", len(m.filtered))
	}
	if m.filtered[0].VoiceID != "v3" {
		t.Errorf("got %s, want v3", m.filtered[0].VoiceID)
	}
}

func TestPickModelFilterByAge(t *testing.T) {
	voices := testPickVoices()
	m := newPickModel(voices, nil, defaultPreviewText)

	m.textInput.SetValue("teenager")
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("filtered = %d, want 1", len(m.filtered))
	}
	if m.filtered[0].VoiceID != "v3" {
		t.Errorf("got %s, want v3", m.filtered[0].VoiceID)
	}
}

func TestPickModelFilterDoesNotCorruptVoices(t *testing.T) {
	voices := testPickVoices()
	m := newPickModel(voices, nil, defaultPreviewText)

	// Filter to subset.
	m.textInput.SetValue("수")
	m.applyFilter()

	// Clear and refilter — original voices must be intact.
	m.textInput.SetValue("")
	m.applyFilter()

	if len(m.filtered) != 3 {
		t.Fatalf("after clear filtered = %d, want 3", len(m.filtered))
	}
	// Verify original voice names are unchanged.
	expected := []string{"수진", "건석", "수아"}
	for i, v := range m.voices {
		if v.VoiceName != expected[i] {
			t.Errorf("voices[%d].VoiceName = %q, want %q", i, v.VoiceName, expected[i])
		}
	}
}
