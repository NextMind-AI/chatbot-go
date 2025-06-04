package elevenlabs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
)

func (c *Client) ConvertTextToSpeech(voiceID string, text string, modelID string) ([]byte, error) {
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
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	apiURL := fmt.Sprintf("%s%s/%s", BaseURL, TextToSpeechPath, voiceID)

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		apiErr.StatusCode = resp.StatusCode
		if err := json.Unmarshal(respBody, &apiErr); err != nil {
			apiErr.Message = string(respBody)
		}
		return nil, apiErr
	}

	log.Info().
		Str("voice_id", voiceID).
		Int("audio_size_bytes", len(respBody)).
		Msg("Text to speech conversion completed")

	return respBody, nil
}
