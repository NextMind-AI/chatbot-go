package elevenlabs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
)

const TextToSpeechPath = "/text-to-speech"

// ConvertTextToSpeechDefault converts text to speech using the default hardcoded voice and model,
// then uploads the audio to S3. This is a convenience method that uses predefined constants.
//
// Parameters:
//   - text: The text content to convert to speech
//
// Returns:
//   - string: Public S3 URL where the generated audio file can be accessed
//   - error: Any error that occurred during text-to-speech conversion or S3 upload
func (c *Client) ConvertTextToSpeechDefault(text string) (string, error) {
	return c.ConvertTextToSpeech(VoiceID, text, ModelID)
}

// ConvertTextToSpeech converts text to speech using ElevenLabs API and uploads the audio to S3.
// The function generates audio from the provided text using the specified voice and model,
// then automatically uploads the audio file to the configured S3 bucket with public read access.
//
// Parameters:
//   - voiceID: ElevenLabs voice ID to use for speech generation
//   - text: The text content to convert to speech
//   - modelID: ElevenLabs model ID (e.g., "eleven_multilingual_v2")
//
// Returns:
//   - string: Public S3 URL where the generated audio file can be accessed
//   - error: Any error that occurred during text-to-speech conversion or S3 upload
//
// The audio file is stored in S3 with the path format: "audio/{voiceID}_{timestamp}.mp3"
// and is publicly accessible via the returned URL.
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

	return c.AWSClient.UploadAudio(respBody, voiceID)
}
