package elevenlabs

import "fmt"

// SpeechToTextResponse represents the response from ElevenLabs speech-to-text API.
// It contains the transcribed text along with language detection information.
type SpeechToTextResponse struct {
	LanguageCode        string  `json:"language_code"`        // Detected language code (e.g., "en", "pt")
	LanguageProbability float64 `json:"language_probability"` // Confidence score for language detection (0.0-1.0)
	Text                string  `json:"text"`                 // The transcribed text content
}

// TextToSpeechRequest represents the request payload for ElevenLabs text-to-speech API.
// It contains the text to be converted and optional model configuration.
type TextToSpeechRequest struct {
	Text    string `json:"text"`               // The text content to convert to speech
	ModelID string `json:"model_id,omitempty"` // Optional model ID for voice synthesis
}

// APIError represents an error response from the ElevenLabs API.
// It provides detailed error information including HTTP status codes and descriptive messages.
type APIError struct {
	StatusCode int    `json:"status_code"`      // HTTP status code from the API response
	Message    string `json:"message"`          // Primary error message
	Detail     string `json:"detail,omitempty"` // Additional error details, if available
}

// Error implements the error interface for APIError.
// It returns a formatted error message that includes the status code and error details.
func (e APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("ElevenLabs API error (status %d): %s - %s", e.StatusCode, e.Message, e.Detail)
	}
	return fmt.Sprintf("ElevenLabs API error (status %d): %s", e.StatusCode, e.Message)
}
