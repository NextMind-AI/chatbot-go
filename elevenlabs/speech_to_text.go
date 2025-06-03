package elevenlabs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

// SpeechToText converts audio to text using ElevenLabs API
func (c *Client) SpeechToText(req SpeechToTextRequest) (*SpeechToTextResponse, error) {
	if req.ModelID == "" {
		req.ModelID = DefaultModel
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("model_id", req.ModelID); err != nil {
		return nil, fmt.Errorf("failed to write model_id field: %w", err)
	}

	if req.LanguageCode != "" {
		if err := writer.WriteField("language_code", req.LanguageCode); err != nil {
			return nil, fmt.Errorf("failed to write language_code field: %w", err)
		}
	}

	if req.TagAudioEvents != nil {
		if err := writer.WriteField("tag_audio_events", strconv.FormatBool(*req.TagAudioEvents)); err != nil {
			return nil, fmt.Errorf("failed to write tag_audio_events field: %w", err)
		}
	}

	if req.NumSpeakers != nil {
		if err := writer.WriteField("num_speakers", strconv.Itoa(*req.NumSpeakers)); err != nil {
			return nil, fmt.Errorf("failed to write num_speakers field: %w", err)
		}
	}

	if req.TimestampsGranularity != "" {
		if err := writer.WriteField("timestamps_granularity", req.TimestampsGranularity); err != nil {
			return nil, fmt.Errorf("failed to write timestamps_granularity field: %w", err)
		}
	}

	if req.Diarize != nil {
		if err := writer.WriteField("diarize", strconv.FormatBool(*req.Diarize)); err != nil {
			return nil, fmt.Errorf("failed to write diarize field: %w", err)
		}
	}

	if req.FileFormat != "" {
		if err := writer.WriteField("file_format", req.FileFormat); err != nil {
			return nil, fmt.Errorf("failed to write file_format field: %w", err)
		}
	}

	if req.CloudStorageURL != "" {
		if err := writer.WriteField("cloud_storage_url", req.CloudStorageURL); err != nil {
			return nil, fmt.Errorf("failed to write cloud_storage_url field: %w", err)
		}
	}

	if req.Webhook != nil {
		if err := writer.WriteField("webhook", strconv.FormatBool(*req.Webhook)); err != nil {
			return nil, fmt.Errorf("failed to write webhook field: %w", err)
		}
	}

	if req.EnableLogging != nil {
		if err := writer.WriteField("enable_logging", strconv.FormatBool(*req.EnableLogging)); err != nil {
			return nil, fmt.Errorf("failed to write enable_logging field: %w", err)
		}
	}

	if len(req.AdditionalFormats) > 0 {
		for i, format := range req.AdditionalFormats {
			fieldName := fmt.Sprintf("additional_formats[%d][format]", i)
			if err := writer.WriteField(fieldName, format.Format); err != nil {
				return nil, fmt.Errorf("failed to write additional_formats field: %w", err)
			}
		}
	}

	if req.File != nil {
		fileName := req.FileName
		if fileName == "" {
			fileName = "audio_file"
		}

		part, err := writer.CreateFormFile("file", fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to create form file: %w", err)
		}

		if _, err := io.Copy(part, req.File); err != nil {
			return nil, fmt.Errorf("failed to copy file data: %w", err)
		}
	} else if req.CloudStorageURL == "" {
		return nil, fmt.Errorf("either file or cloud_storage_url must be provided")
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	url := c.BaseURL + SpeechToTextPath
	httpReq, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	if c.APIKey != "" {
		httpReq.Header.Set("xi-api-key", c.APIKey)
	}

	log.Debug().
		Str("url", url).
		Str("content_type", writer.FormDataContentType()).
		Msg("Making speech-to-text request to ElevenLabs")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	log.Debug().
		Int("status_code", resp.StatusCode).
		Str("response_body", string(respBody)).
		Msg("Received response from ElevenLabs")

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		apiErr.StatusCode = resp.StatusCode

		if err := json.Unmarshal(respBody, &apiErr); err != nil {
			apiErr.Message = string(respBody)
		}

		return nil, apiErr
	}

	var result SpeechToTextResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %w", err)
	}

	log.Info().
		Str("language_code", result.LanguageCode).
		Float64("language_probability", result.LanguageProbability).
		Int("word_count", len(result.Words)).
		Str("text_preview", truncateText(result.Text, 100)).
		Msg("Successfully transcribed audio")

	return &result, nil
}

// TranscribeAudioFile is a convenience method for transcribing audio files with common settings
func (c *Client) TranscribeAudioFile(file io.Reader, fileName string, languageCode string) (*SpeechToTextResponse, error) {
	tagAudioEvents := true
	timestampsGranularity := "word"

	req := SpeechToTextRequest{
		ModelID:               DefaultModel,
		File:                  file,
		FileName:              fileName,
		LanguageCode:          languageCode,
		TagAudioEvents:        &tagAudioEvents,
		TimestampsGranularity: timestampsGranularity,
	}

	return c.SpeechToText(req)
}

// TranscribeAudioURL is a convenience method for transcribing audio from cloud storage URLs
func (c *Client) TranscribeAudioURL(cloudStorageURL string, languageCode string) (*SpeechToTextResponse, error) {
	tagAudioEvents := true
	timestampsGranularity := "word"

	req := SpeechToTextRequest{
		ModelID:               DefaultModel,
		CloudStorageURL:       cloudStorageURL,
		LanguageCode:          languageCode,
		TagAudioEvents:        &tagAudioEvents,
		TimestampsGranularity: timestampsGranularity,
	}

	return c.SpeechToText(req)
}

// truncateText truncates text to a specified length for logging
func truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return strings.TrimSpace(text[:maxLength]) + "..."
}
