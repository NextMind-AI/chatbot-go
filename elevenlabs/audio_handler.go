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

// AudioMessage represents an audio message that needs to be transcribed
type AudioMessage struct {
	URL         string
	ContentType string
	FileName    string
	Language    string
}

// AudioHandler handles audio message processing and transcription
type AudioHandler struct {
	Client     *Client
	HTTPClient *http.Client
}

// NewAudioHandler creates a new audio handler
func NewAudioHandler(client *Client, httpClient *http.Client) *AudioHandler {
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	return &AudioHandler{
		Client:     client,
		HTTPClient: httpClient,
	}
}

// ProcessAudioMessage downloads and transcribes an audio message
func (h *AudioHandler) ProcessAudioMessage(audioMsg AudioMessage) (*SpeechToTextResponse, error) {
	log.Info().
		Str("url", audioMsg.URL).
		Str("content_type", audioMsg.ContentType).
		Str("language", audioMsg.Language).
		Msg("Processing audio message")

	audioData, err := h.downloadAudioFile(audioMsg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to download audio file: %w", err)
	}

	fileName := audioMsg.FileName
	if fileName == "" {
		fileName = h.generateFileName(audioMsg.URL, audioMsg.ContentType)
	}

	reader := bytes.NewReader(audioData)
	response, err := h.Client.TranscribeAudioFile(reader, fileName, audioMsg.Language)
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

// ProcessAudioURL transcribes audio from a publicly accessible URL
func (h *AudioHandler) ProcessAudioURL(url string, language string) (*SpeechToTextResponse, error) {
	log.Info().
		Str("url", url).
		Str("language", language).
		Msg("Processing audio from URL")

	response, err := h.Client.TranscribeAudioURL(url, language)
	if err != nil {
		return nil, fmt.Errorf("failed to transcribe audio from URL: %w", err)
	}

	log.Info().
		Str("transcribed_text", response.Text).
		Str("detected_language", response.LanguageCode).
		Float64("language_confidence", response.LanguageProbability).
		Msg("Audio transcription from URL completed")

	return response, nil
}

// downloadAudioFile downloads an audio file from a URL
func (h *AudioHandler) downloadAudioFile(url string) ([]byte, error) {
	log.Debug().Str("url", url).Msg("Downloading audio file")

	resp, err := h.HTTPClient.Get(url)
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

// generateFileName generates a filename based on URL and content type
func (h *AudioHandler) generateFileName(url string, contentType string) string {
	if filename := filepath.Base(url); filename != "." && filename != "/" {
		if idx := strings.Index(filename, "?"); idx != -1 {
			filename = filename[:idx]
		}
		if filename != "" && strings.Contains(filename, ".") {
			return filename
		}
	}

	extension := h.getFileExtensionFromContentType(contentType)
	return fmt.Sprintf("audio_message%s", extension)
}

// getFileExtensionFromContentType returns file extension based on content type
func (h *AudioHandler) getFileExtensionFromContentType(contentType string) string {
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
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "video/quicktime":
		return ".mov"
	default:
		return ".audio"
	}
}

// IsAudioContent checks if the content type represents audio or video
func IsAudioContent(contentType string) bool {
	contentType = strings.ToLower(contentType)
	return strings.HasPrefix(contentType, "audio/") ||
		strings.HasPrefix(contentType, "video/")
}

// ExtractTextFromTranscription extracts clean text from transcription response
func ExtractTextFromTranscription(response *SpeechToTextResponse) string {
	if response == nil {
		return ""
	}
	return strings.TrimSpace(response.Text)
}
