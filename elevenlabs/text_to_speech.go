package elevenlabs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/rs/zerolog/log"
)

const TextToSpeechPath = "/text-to-speech"

func (c *Client) ConvertTextToSpeech(voiceID string, text string, modelID string) (string, error) {
	log.Info().
		Str("voice_id", voiceID).
		Str("text", text).
		Str("model_id", modelID).
		Msg("Converting text to speech")

	requestBody := TextToSpeechRequest{
		Text:    text,
		ModelID: modelID,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	apiURL := fmt.Sprintf("%s%s/%s", BaseURL, TextToSpeechPath, voiceID)

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		apiErr.StatusCode = resp.StatusCode
		if err := json.Unmarshal(respBody, &apiErr); err != nil {
			apiErr.Message = string(respBody)
		}
		return "", apiErr
	}

	log.Info().
		Str("voice_id", voiceID).
		Int("audio_size_bytes", len(respBody)).
		Msg("Text to speech conversion completed")

	key := fmt.Sprintf("audio/%s_%d.mp3", voiceID, time.Now().Unix())

	uploader := s3manager.NewUploader(c.S3Session)

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(c.S3Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(respBody),
		ACL:         aws.String("public-read"),
		ContentType: aws.String("audio/mpeg"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload audio to S3: %w", err)
	}

	publicURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", c.S3Bucket, c.S3Region, key)

	log.Info().
		Str("voice_id", voiceID).
		Str("s3_url", publicURL).
		Str("s3_location", result.Location).
		Msg("Audio uploaded to S3 successfully")

	return publicURL, nil
}
