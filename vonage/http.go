package vonage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
)

func (c *Client) sendMessageRequest(method, url string, body any) (*MessageResponse, error) {
	respBody, err := c.sendRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	var messageResponse MessageResponse
	if err := json.Unmarshal(respBody, &messageResponse); err != nil {
		log.Error().Err(err).Str("method", method).Str("url", url).Msg("Error unmarshaling response")
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &messageResponse, nil
}

func (c *Client) sendRequest(method, url string, body any) ([]byte, error) {
	log.Debug().
		Str("method", method).
		Str("url", url).
		Interface("body", body).
		Msg("Sending HTTP request")

	payload, err := json.Marshal(body)
	if err != nil {
		log.Error().Err(err).Str("method", method).Str("url", url).Msg("Error marshaling payload")
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(payload))
	if err != nil {
		log.Error().Err(err).Str("method", method).Str("url", url).Msg("Error creating HTTP request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Str("method", method).Str("url", url).Msg("Error sending HTTP request")
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	log.Debug().
		Str("method", method).
		Str("url", url).
		Int("status_code", resp.StatusCode).
		Msg("Received HTTP response")

	if !c.isSuccessStatusCode(resp.StatusCode) {
		log.Warn().
			Str("method", method).
			Str("url", url).
			Int("status_code", resp.StatusCode).
			Msg("Unexpected HTTP status code")
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Str("method", method).Str("url", url).Msg("Error reading response body")
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return responseBody, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.config.VonageJWT)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}

func (c *Client) isSuccessStatusCode(statusCode int) bool {
	return statusCode == http.StatusOK || statusCode == http.StatusAccepted
}
