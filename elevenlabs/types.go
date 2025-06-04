package elevenlabs

import "fmt"

type SpeechToTextResponse struct {
	LanguageCode        string  `json:"language_code"`
	LanguageProbability float64 `json:"language_probability"`
	Text                string  `json:"text"`
}

type TextToSpeechRequest struct {
	Text    string `json:"text"`
	ModelID string `json:"model_id,omitempty"`
}

type APIError struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
	Detail     string `json:"detail,omitempty"`
}

func (e APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("ElevenLabs API error (status %d): %s - %s", e.StatusCode, e.Message, e.Detail)
	}
	return fmt.Sprintf("ElevenLabs API error (status %d): %s", e.StatusCode, e.Message)
}
