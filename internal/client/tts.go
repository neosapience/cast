package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type TTSRequest struct {
	VoiceID  string     `json:"voice_id"`
	Text     string     `json:"text"`
	Model    string     `json:"model"`
	Language string     `json:"language,omitempty"`
	Prompt   *TTSPrompt `json:"prompt,omitempty"`
	Output   *TTSOutput `json:"output,omitempty"`
	Seed     *int       `json:"seed,omitempty"`
}

type TTSPrompt struct {
	EmotionType      string   `json:"emotion_type,omitempty"`
	EmotionPreset    string   `json:"emotion_preset,omitempty"`
	EmotionIntensity *float64 `json:"emotion_intensity,omitempty"`
	PreviousText     string   `json:"previous_text,omitempty"`
	NextText         string   `json:"next_text,omitempty"`
}

type TTSOutput struct {
	Volume      *int     `json:"volume,omitempty"`
	AudioPitch  *int     `json:"audio_pitch,omitempty"`
	AudioTempo  *float64 `json:"audio_tempo,omitempty"`
	AudioFormat string   `json:"audio_format,omitempty"`
}

func (c *Client) TextToSpeech(req TTSRequest) ([]byte, error) {
	return c.post("/v1/text-to-speech", req)
}

func (c *Client) TextToSpeechStream(req TTSRequest, onChunk func([]byte) error) error {
	if onChunk == nil {
		return fmt.Errorf("onChunk callback must not be nil")
	}

	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/v1/text-to-speech/stream", bytes.NewReader(b))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-KEY", c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return fmt.Errorf("authentication failed: check your API key with 'cast login'")
		case http.StatusForbidden:
			return fmt.Errorf("access forbidden: your API key does not have permission")
		case http.StatusNotFound:
			return fmt.Errorf("not found: %s", extractErrorMessage(data))
		case http.StatusBadRequest:
			return fmt.Errorf("invalid request: %s", extractErrorMessage(data))
		case http.StatusTooManyRequests:
			return fmt.Errorf("rate limit exceeded: please try again later")
		default:
			if resp.StatusCode >= 500 {
				return fmt.Errorf("server error (%d): please try again later", resp.StatusCode)
			}
			return fmt.Errorf("API error %d: %s", resp.StatusCode, extractErrorMessage(data))
		}
	}

	buf := make([]byte, 8192)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			if err := onChunk(chunk); err != nil {
				return err
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}
