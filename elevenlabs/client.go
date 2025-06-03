package elevenlabs

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

const (
	BaseURL          = "https://api.elevenlabs.io"
	SpeechToTextPath = "/v1/speech-to-text"
	DefaultModel     = "scribe_v1"
)

// Client represents the ElevenLabs API client
type Client struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new ElevenLabs client
func NewClient(apiKey string, httpClient http.Client) Client {
	return Client{
		APIKey:     apiKey,
		BaseURL:    BaseURL,
		HTTPClient: &httpClient,
	}
}

// ProcessAudioFromURL downloads and transcribes audio from a URL
func (c *Client) ProcessAudioFromURL(url string, language string) (*SpeechToTextResponse, error) {
	log.Info().
		Str("url", url).
		Str("language", language).
		Msg("Processing audio from URL")

	audioData, err := c.downloadAudioFile(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download audio file: %w", err)
	}

	fileName := c.generateFileName(url, "audio/mpeg")
	reader := bytes.NewReader(audioData)
	response, err := c.TranscribeAudioFile(reader, fileName, language)
	if err != nil {
		return nil, fmt.Errorf("failed to transcribe audio: %w", err)
	}

	log.Info().
		Str("transcribed_text", response.Text).
		Str("detected_language", response.LanguageCode).
		Float64("language_confidence", response.LanguageProbability).
		Msg("Audio transcription completed")

	return response, nil
}

func (c *Client) downloadAudioFile(url string) ([]byte, error) {
	log.Debug().Str("url", url).Msg("Downloading audio file")

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download file: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}

	log.Debug().
		Int("file_size", len(data)).
		Str("content_type", resp.Header.Get("Content-Type")).
		Msg("Audio file downloaded successfully")

	return data, nil
}

func (c *Client) generateFileName(url string, contentType string) string {
	if filename := filepath.Base(url); filename != "." && filename != "/" {
		if idx := strings.Index(filename, "?"); idx != -1 {
			filename = filename[:idx]
		}
		if filename != "" && strings.Contains(filename, ".") {
			return filename
		}
	}

	extension := c.getFileExtensionFromContentType(contentType)
	return fmt.Sprintf("audio_message%s", extension)
}

func (c *Client) getFileExtensionFromContentType(contentType string) string {
	switch strings.ToLower(contentType) {
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/wav", "audio/wave":
		return ".wav"
	case "audio/ogg":
		return ".ogg"
	case "audio/aac":
		return ".aac"
	case "audio/flac":
		return ".flac"
	case "audio/m4a":
		return ".m4a"
	case "audio/webm":
		return ".webm"
	default:
		return ".audio"
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

// TranscribeAudio transcribes audio from a URL and returns the text
func (c *Client) TranscribeAudio(url string) (string, error) {
	log.Info().Str("url", url).Msg("Transcribing audio from URL")

	response, err := c.ProcessAudioFromURL(url, "")
	if err != nil {
		log.Error().Err(err).Str("url", url).Msg("Failed to transcribe audio")
		return "", err
	}

	transcribedText := ExtractTextFromTranscription(response)

	log.Info().
		Str("transcribed_text", transcribedText).
		Str("detected_language", response.LanguageCode).
		Float64("confidence", response.LanguageProbability).
		Msg("Audio transcription completed")

	return transcribedText, nil
}
