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
//	// Create AWS session
//	sess, err := session.NewSession(&aws.Config{
//		Region: aws.String("us-east-2"),
//		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
//	})
//
//	// Initialize client
//	client := elevenlabs.NewClient(apiKey, http.Client{}, sess, bucket, region)
//
//	// Transcribe audio
//	text, err := client.TranscribeAudio("https://example.com/audio.mp3")
//
//	// Convert text to speech
//	audioURL, err := client.ConvertTextToSpeech(voiceID, "Hello world", modelID)
package elevenlabs

import (
	"net/http"

	"github.com/aws/aws-sdk-go/aws/session"
)

const (
	BaseURL      = "https://api.elevenlabs.io/v1"
	DefaultModel = "scribe_v1"
)

type Client struct {
	APIKey       string
	LanguageCode string
	HTTPClient   *http.Client
	S3Session    *session.Session
	S3Bucket     string
	S3Region     string
}

// NewClient creates a new ElevenLabs client with S3 integration for audio storage.
// The client supports both speech-to-text and text-to-speech operations.
//
// Parameters:
//   - apiKey: Your ElevenLabs API key for authentication
//   - httpClient: HTTP client for making requests to ElevenLabs API
//   - s3Session: AWS session configured with credentials for S3 operations
//   - s3Bucket: Name of the S3 bucket where audio files will be stored
//   - s3Region: AWS region where the S3 bucket is located
//
// Returns a configured Client ready for use with ElevenLabs APIs.
func NewClient(apiKey string, httpClient http.Client, s3Session *session.Session, s3Bucket string, s3Region string) Client {
	return Client{
		APIKey:       apiKey,
		LanguageCode: "pt",
		HTTPClient:   &httpClient,
		S3Session:    s3Session,
		S3Bucket:     s3Bucket,
		S3Region:     s3Region,
	}
}
