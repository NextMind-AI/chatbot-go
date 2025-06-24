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
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &messageResponse, nil
}

func (c *Client) sendRequest(method, url string, body any) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	log.Debug().
		Str("method", method).
		Str("url", url).
		Str("jwt_prefix", c.getJWTPreview()).
		Msg("Sending Vonage API request")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if !c.isSuccessStatusCode(resp.StatusCode) {
		log.Error().
			Int("status_code", resp.StatusCode).
			Str("response_body", string(responseBody)).
			Str("url", url).
			Str("method", method).
			Msg("Vonage API request failed")
		
		if resp.StatusCode == 401 {
			return nil, fmt.Errorf("authentication failed (401): check your VONAGE_JWT token. Response: %s", string(responseBody))
		}
		
		return nil, fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(responseBody))
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

// getJWTPreview returns the first 10 characters of the JWT for debugging
func (c *Client) getJWTPreview() string {
	if len(c.config.VonageJWT) > 10 {
		return c.config.VonageJWT[:10] + "..."
	}
	return c.config.VonageJWT
}
