// Package elevenlabs provides a client for ElevenLabs speech-to-text and text-to-speech APIs.
//
// This package offers simple and clean integration with ElevenLabs services, including:
//   - Speech-to-text transcription from audio files or URLs
//   - Text-to-speech conversion with automatic S3 upload and public URL generation
//   - Support for multiple audio formats (MP3, WAV, OGG, AAC, FLAC, M4A, WebM)
//   - Built-in error handling and logging
//
// Basic usage:
//
//	// Create AWS client
//	awsClient, err := aws.NewClient("us-east-2", "my-bucket")
//
//	// Initialize client
//	client := elevenlabs.NewClient(apiKey, &http.Client{}, awsClient)
//
//	// Transcribe audio
//	text, err := client.TranscribeAudio("https://example.com/audio.mp3")
//
//	// Convert text to speech
//	audioURL, err := client.ConvertTextToSpeech(voiceID, "Hello world", modelID)
package elevenlabs

import (
	"net/http"
)

const (
	BaseURL      = "https://api.elevenlabs.io/v1"
	DefaultModel = "scribe_v1"
	VoiceID      = "JNI7HKGyqNaHqfihNoCi"
	ModelID      = "eleven_multilingual_v2"
)

type AWSClient interface {
	UploadAudio(audioData []byte, voiceID string) (string, error)
}

type Client struct {
	APIKey       string
	LanguageCode string
	HTTPClient   *http.Client
	AWSClient    AWSClient
}

// NewClient creates a new ElevenLabs client with AWS integration for audio storage.
// The client supports both speech-to-text and text-to-speech operations.
//
// Parameters:
//   - apiKey: Your ElevenLabs API key for authentication
//   - httpClient: HTTP client for making requests to ElevenLabs API
//   - awsClient: AWS client configured for S3 operations
//
// Returns a configured Client ready for use with ElevenLabs APIs.
func NewClient(apiKey string, httpClient *http.Client, awsClient AWSClient) Client {
	return Client{
		APIKey:       apiKey,
		LanguageCode: "pt",
		HTTPClient:   httpClient,
		AWSClient:    awsClient,
	}
}
