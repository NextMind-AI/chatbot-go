package elevenlabs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/rs/zerolog/log"
)

func (c *Client) TranscribeAudio(url string) (string, error) {
	log.Info().Str("url", url).Msg("Downloading and transcribing audio from URL")

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download audio: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download audio: HTTP %d", resp.StatusCode)
	}

	return c.transcribeAudioFile(resp.Body, "audio.mp3")
}

func (c *Client) transcribeAudioFile(file io.Reader, fileName string) (string, error) {
	log.Info().Str("file_name", fileName).Msg("Transcribing audio file")

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("model_id", DefaultModel); err != nil {
		return "", fmt.Errorf("failed to write model_id field: %w", err)
	}

	if c.LanguageCode != "" {
		if err := writer.WriteField("language_code", c.LanguageCode); err != nil {
			return "", fmt.Errorf("failed to write language_code field: %w", err)
		}
	}

	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("failed to copy file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	url := BaseURL + SpeechToTextPath
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
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

	var result SpeechToTextResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response JSON: %w", err)
	}

	log.Info().
		Str("transcribed_text", result.Text).
		Str("detected_language", result.LanguageCode).
		Float64("confidence", result.LanguageProbability).
		Msg("Audio transcription completed")

	return result.Text, nil
}
