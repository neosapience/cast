package client

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCloneVoice_SendsMultipartRequest(t *testing.T) {
	audioPath := writeTempAudio(t, "sample.wav", []byte("RIFFdata"))

	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/voices/clone" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("X-API-KEY"); got != "test-api-key" {
			t.Errorf("expected API key header, got %q", got)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("expected multipart content type, got %q", r.Header.Get("Content-Type"))
		}

		if err := r.ParseMultipartForm(MaxCloneAudioSize); err != nil {
			t.Fatalf("ParseMultipartForm failed: %v", err)
		}
		if got := r.FormValue("name"); got != "Review Clone" {
			t.Errorf("name: want %q, got %q", "Review Clone", got)
		}
		if got := r.FormValue("model"); got != "ssfm-v30" {
			t.Errorf("model: want %q, got %q", "ssfm-v30", got)
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("missing file field: %v", err)
		}
		defer file.Close()
		if header.Filename != "sample.wav" {
			t.Errorf("filename: want sample.wav, got %q", header.Filename)
		}
		if got := header.Header.Get("Content-Type"); got != "audio/wav" {
			t.Errorf("file content type: want audio/wav, got %q", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"voice_id": "uc_clone_123",
			"name":     "Review Clone",
			"model":    "ssfm-v30",
		})
	})

	voice, err := c.CloneVoice(CloneVoiceRequest{
		Name:          "Review Clone",
		Model:         "ssfm-v30",
		AudioFilePath: audioPath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if voice.VoiceID != "uc_clone_123" || voice.ClonedVoiceID != "uc_clone_123" {
		t.Errorf("unexpected voice IDs: %+v", voice)
	}
	if voice.NextStepVoiceID != "uc_clone_123" || voice.NextStepModel != "ssfm-v30" {
		t.Errorf("unexpected handoff fields: %+v", voice)
	}
	if voice.FileSize != int64(len("RIFFdata")) {
		t.Errorf("file size: want %d, got %d", len("RIFFdata"), voice.FileSize)
	}
}

func TestCloneVoice_ValidatesInputs(t *testing.T) {
	audioPath := writeTempAudio(t, "sample.wav", []byte("RIFFdata"))

	c := NewWithBaseURL("test-api-key", "http://127.0.0.1:1")
	cases := []struct {
		name string
		req  CloneVoiceRequest
		want string
	}{
		{
			name: "empty name",
			req:  CloneVoiceRequest{Name: "", Model: "ssfm-v30", AudioFilePath: audioPath},
			want: "voice name must be between",
		},
		{
			name: "name too long",
			req:  CloneVoiceRequest{Name: strings.Repeat("가", 31), Model: "ssfm-v30", AudioFilePath: audioPath},
			want: "voice name must be between",
		},
		{
			name: "unsupported model",
			req:  CloneVoiceRequest{Name: "Clone", Model: "ssfm-v21", AudioFilePath: audioPath},
			want: "voice cloning model must be ssfm-v30",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := c.CloneVoice(tc.req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("want error containing %q, got %q", tc.want, err.Error())
			}
		})
	}
}

func TestOpenCloneAudioFile_ValidatesFile(t *testing.T) {
	dir := t.TempDir()
	wav := writeTempAudio(t, "sample.wav", []byte("RIFFdata"))
	mp3 := writeTempAudio(t, "sample.mp3", []byte("ID3data"))
	txt := writeTempAudio(t, "sample.txt", []byte("not audio"))

	large := filepath.Join(t.TempDir(), "large.wav")
	f, err := os.Create(large)
	if err != nil {
		t.Fatalf("Create large file: %v", err)
	}
	if err := f.Truncate(MaxCloneAudioSize + 1); err != nil {
		t.Fatalf("Truncate large file: %v", err)
	}
	f.Close()

	cases := []struct {
		name        string
		path        string
		wantContent string
		wantErr     string
	}{
		{name: "wav", path: wav, wantContent: "audio/wav"},
		{name: "mp3", path: mp3, wantContent: "audio/mpeg"},
		{name: "missing", path: filepath.Join(t.TempDir(), "missing.wav"), wantErr: "audio file not found"},
		{name: "directory", path: dir, wantErr: "audio file path is a directory"},
		{name: "unsupported extension", path: txt, wantErr: "audio file must be WAV or MP3"},
		{name: "too large", path: large, wantErr: "audio file must be 25 MB or smaller"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file, contentType, _, err := openCloneAudioFile(tc.path)
			if file != nil {
				file.Close()
			}
			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("want error containing %q, got %q", tc.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if contentType != tc.wantContent {
				t.Errorf("content type: want %q, got %q", tc.wantContent, contentType)
			}
		})
	}
}

func TestParseClonedVoice_ResponseShapes(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{name: "top level", body: `{"voice_id":"uc_top","name":"Top","model":"ssfm-v30"}`, want: "uc_top"},
		{name: "result", body: `{"result":{"voice_id":"uc_result","voice_name":"Result","model":"ssfm-v30"}}`, want: "uc_result"},
		{name: "data", body: `{"data":{"voiceId":"uc_data","name":"Data","model":"ssfm-v30"}}`, want: "uc_data"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			voice, err := parseClonedVoice([]byte(tc.body), "Fallback", "ssfm-v30", 10)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if voice.VoiceID != tc.want || voice.NextStepVoiceID != tc.want {
				t.Errorf("unexpected voice: %+v", voice)
			}
		})
	}
}

func TestParseClonedVoice_RejectsMissingOrNonCloneID(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{name: "missing voice id", body: `{}`, want: "voice_id not found"},
		{name: "non clone id", body: `{"voice_id":"tc_builtin"}`, want: "expected an ID starting with 'uc_'"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseClonedVoice([]byte(tc.body), "Fallback", "ssfm-v30", 10)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("want error containing %q, got %q", tc.want, err.Error())
			}
		})
	}
}

func TestDeleteClonedVoice(t *testing.T) {
	var gotPath string
	var gotMethod string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	})

	if err := c.DeleteClonedVoice("uc_clone_123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method: want DELETE, got %s", gotMethod)
	}
	if gotPath != "/v1/voices/uc_clone_123" {
		t.Errorf("path: want /v1/voices/uc_clone_123, got %s", gotPath)
	}

	if err := c.DeleteClonedVoice("tc_builtin"); err == nil {
		t.Fatal("expected error for non-cloned voice ID")
	}
}

func writeTempAudio(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func multipartFileContentType(header *multipart.FileHeader) string {
	return header.Header.Get("Content-Type")
}
