package client

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
	TargetLUFS  *float64 `json:"target_lufs,omitempty"`
}

func (c *Client) TextToSpeech(req TTSRequest) ([]byte, error) {
	return c.post("/v1/text-to-speech", req)
}
