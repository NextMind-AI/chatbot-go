package elevenlabs

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	BaseURL          = "https://api.elevenlabs.io"
	SpeechToTextPath = "/v1/speech-to-text"
	DefaultModel     = "scribe_v1"
	DefaultTimeout   = 30 * time.Second
)

// Client represents the ElevenLabs API client
type Client struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new ElevenLabs client
func NewClient(apiKey string, httpClient *http.Client) Client {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: DefaultTimeout,
		}
	}

	return Client{
		APIKey:     apiKey,
		BaseURL:    BaseURL,
		HTTPClient: httpClient,
	}
}

// SpeechToTextRequest represents the request parameters for speech-to-text conversion
type SpeechToTextRequest struct {
	ModelID               string             `json:"model_id"`
	File                  io.Reader          `json:"-"`
	FileName              string             `json:"-"`
	LanguageCode          string             `json:"language_code,omitempty"`
	TagAudioEvents        *bool              `json:"tag_audio_events,omitempty"`
	NumSpeakers           *int               `json:"num_speakers,omitempty"`
	TimestampsGranularity string             `json:"timestamps_granularity,omitempty"`
	Diarize               *bool              `json:"diarize,omitempty"`
	AdditionalFormats     []AdditionalFormat `json:"additional_formats,omitempty"`
	FileFormat            string             `json:"file_format,omitempty"`
	CloudStorageURL       string             `json:"cloud_storage_url,omitempty"`
	Webhook               *bool              `json:"webhook,omitempty"`
	EnableLogging         *bool              `json:"enable_logging,omitempty"`
}

// AdditionalFormat represents additional export formats
type AdditionalFormat struct {
	Format string `json:"format"`
}

// Word represents a word in the transcription with timing information
type Word struct {
	Text      string  `json:"text"`
	Type      string  `json:"type"`
	LogProb   float64 `json:"logprob"`
	Start     float64 `json:"start"`
	End       float64 `json:"end"`
	SpeakerID string  `json:"speaker_id"`
}

// SpeechToTextResponse represents the response from the speech-to-text API
type SpeechToTextResponse struct {
	LanguageCode        string               `json:"language_code"`
	LanguageProbability float64              `json:"language_probability"`
	Text                string               `json:"text"`
	Words               []Word               `json:"words"`
	AdditionalFormats   []AdditionalResponse `json:"additional_formats,omitempty"`
}

// AdditionalResponse represents additional format responses
type AdditionalResponse struct {
	Format   string `json:"format"`
	Content  string `json:"content"`
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
	Size     int    `json:"size"`
}

// APIError represents an error response from the ElevenLabs API
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
