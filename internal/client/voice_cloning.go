package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const MaxCloneAudioSize = 25 * 1024 * 1024

type CloneVoiceRequest struct {
	Name          string
	Model         string
	AudioFilePath string
}

type ClonedVoice struct {
	VoiceID         string `json:"voice_id"`
	ClonedVoiceID   string `json:"cloned_voice_id"`
	VoiceName       string `json:"voice_name,omitempty"`
	Name            string `json:"name,omitempty"`
	Model           string `json:"model,omitempty"`
	FileSize        int64  `json:"file_size,omitempty"`
	NextStepVoiceID string `json:"next_step_voice_id"`
	NextStepModel   string `json:"next_step_model"`
}

type cloneVoicePayload struct {
	VoiceID    string `json:"voice_id"`
	VoiceIDAlt string `json:"voiceId"`
	VoiceName  string `json:"voice_name"`
	Name       string `json:"name"`
	Model      string `json:"model"`
}

type cloneVoiceEnvelope struct {
	cloneVoicePayload
	Result *cloneVoicePayload `json:"result"`
	Data   *cloneVoicePayload `json:"data"`
}

func (c *Client) CloneVoice(req CloneVoiceRequest) (*ClonedVoice, error) {
	if n := utf8.RuneCountInString(req.Name); n < 1 || n > 30 {
		return nil, fmt.Errorf("voice name must be between 1 and 30 characters, got %d", n)
	}
	if req.Model == "" {
		req.Model = "ssfm-v30"
	}
	if req.Model != "ssfm-v30" {
		return nil, fmt.Errorf("voice cloning model must be ssfm-v30, got %q", req.Model)
	}

	audioFile, contentType, size, err := openCloneAudioFile(req.AudioFilePath)
	if err != nil {
		return nil, err
	}
	defer audioFile.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("name", req.Name); err != nil {
		return nil, err
	}
	if err := writer.WriteField("model", req.Model); err != nil {
		return nil, err
	}

	part, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": []string{fmt.Sprintf(`form-data; name="file"; filename="%s"`, escapeQuotes(filepath.Base(req.AudioFilePath)))},
		"Content-Type":        []string{contentType},
	})
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, audioFile); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/v1/voices/clone", body)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("X-API-KEY", c.apiKey)

	data, err := c.do(httpReq)
	if err != nil {
		return nil, err
	}

	voice, err := parseClonedVoice(data, req.Name, req.Model, size)
	if err != nil {
		return nil, err
	}
	return voice, nil
}

func (c *Client) DeleteClonedVoice(voiceID string) error {
	if !strings.HasPrefix(voiceID, "uc_") {
		return fmt.Errorf("only cloned voice IDs that start with 'uc_' can be deleted")
	}
	_, err := c.delete("/v1/voices/" + url.PathEscape(voiceID))
	return err
}

func openCloneAudioFile(path string) (*os.File, string, int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, "", 0, fmt.Errorf("audio file not found: %w", err)
	}
	if info.IsDir() {
		return nil, "", 0, fmt.Errorf("audio file path is a directory: %s", path)
	}
	if info.Size() > MaxCloneAudioSize {
		return nil, "", 0, fmt.Errorf("audio file must be 25 MB or smaller, got %d bytes", info.Size())
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to open audio file: %w", err)
	}

	contentType, err := cloneAudioContentType(f, path)
	if err != nil {
		_ = f.Close()
		return nil, "", 0, err
	}

	return f, contentType, info.Size(), nil
}

func cloneAudioContentType(f *os.File, path string) (string, error) {
	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read audio file header: %w", err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to rewind audio file: %w", err)
	}
	header := buf[:n]

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".wav":
		if isWAVHeader(header) {
			return "audio/wav", nil
		}
	case ".mp3":
		if isMP3Header(header) {
			return "audio/mpeg", nil
		}
	default:
		return "", fmt.Errorf("audio file must be WAV or MP3: %s", path)
	}

	return "", fmt.Errorf("audio file content does not match %s format: %s", strings.TrimPrefix(ext, "."), path)
}

func isWAVHeader(header []byte) bool {
	return len(header) >= 12 &&
		string(header[0:4]) == "RIFF" &&
		string(header[8:12]) == "WAVE"
}

func isMP3Header(header []byte) bool {
	if len(header) >= 3 && string(header[0:3]) == "ID3" {
		return true
	}
	return len(header) >= 2 && header[0] == 0xFF && header[1]&0xE0 == 0xE0
}

func escapeQuotes(s string) string {
	return strings.NewReplacer("\\", "\\\\", `"`, "\\\"").Replace(s)
}

func parseClonedVoice(data []byte, fallbackName, fallbackModel string, fileSize int64) (*ClonedVoice, error) {
	var env cloneVoiceEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}

	payload := env.cloneVoicePayload
	if env.Result != nil {
		payload = *env.Result
	} else if env.Data != nil {
		payload = *env.Data
	}

	voiceID := payload.VoiceID
	if voiceID == "" {
		voiceID = payload.VoiceIDAlt
	}
	if voiceID == "" {
		return nil, fmt.Errorf("voice_id not found in clone response")
	}
	if !strings.HasPrefix(voiceID, "uc_") {
		return nil, fmt.Errorf("clone response returned non-cloned voice ID %q; expected an ID starting with 'uc_'", voiceID)
	}

	name := payload.Name
	if name == "" {
		name = payload.VoiceName
	}
	if name == "" {
		name = fallbackName
	}

	model := payload.Model
	if model == "" {
		model = fallbackModel
	}

	return &ClonedVoice{
		VoiceID:         voiceID,
		ClonedVoiceID:   voiceID,
		VoiceName:       name,
		Name:            name,
		Model:           model,
		FileSize:        fileSize,
		NextStepVoiceID: voiceID,
		NextStepModel:   model,
	}, nil
}
