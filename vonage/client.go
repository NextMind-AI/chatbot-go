package vonage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
)

type Config struct {
	VonageJWT                 string
	GeospecificMessagesAPIURL string
	MessagesAPIURL            string
}

type Client struct {
	config Config
}

type WhatsAppMessage struct {
	To          string `json:"to"`
	From        string `json:"from"`
	Channel     string `json:"channel"`
	MessageType string `json:"message_type"`
	Text        string `json:"text"`
}

type MarkAsReadPayload struct {
	Status string `json:"status"`
}

func NewClient(vonageJWT, geospecificMessagesAPIURL, messagesAPIURL string) Client {
	config := Config{
		VonageJWT:                 vonageJWT,
		GeospecificMessagesAPIURL: geospecificMessagesAPIURL,
		MessagesAPIURL:            messagesAPIURL,
	}

	client := Client{
		config: config,
	}

	log.Info().
		Str("messages_api_url", messagesAPIURL).
		Str("geospecific_messages_api_url", geospecificMessagesAPIURL).
		Msg("Vonage client initialized successfully")

	return client
}

func (c *Client) SendWhatsAppTextMessage(toNumber, senderID, text string) error {
	log.Debug().
		Str("to", toNumber).
		Str("from", senderID).
		Str("text", text).
		Msg("Preparing to send WhatsApp message")

	message := WhatsAppMessage{
		To:          toNumber,
		From:        senderID,
		Channel:     "whatsapp",
		MessageType: "text",
		Text:        text,
	}
	return c.sendRequest("POST", c.config.MessagesAPIURL, message)
}

func (c *Client) MarkMessageAsRead(messageID string) error {
	log.Debug().Str("message_id", messageID).Msg("Marking message as read")

	payload := MarkAsReadPayload{
		Status: "read",
	}
	url := fmt.Sprintf("%s/%s", c.config.GeospecificMessagesAPIURL, messageID)
	return c.sendRequest("PATCH", url, payload)
}

func (c *Client) sendRequest(method, url string, body any) error {
	log.Debug().
		Str("method", method).
		Str("url", url).
		Interface("body", body).
		Msg("Sending HTTP request")

	payload, err := json.Marshal(body)
	if err != nil {
		log.Error().Err(err).Str("method", method).Str("url", url).Msg("Error marshaling payload")
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(payload))
	if err != nil {
		log.Error().Err(err).Str("method", method).Str("url", url).Msg("Error creating HTTP request")
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.config.VonageJWT)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Str("method", method).Str("url", url).Msg("Error sending HTTP request")
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	log.Debug().
		Str("method", method).
		Str("url", url).
		Int("status_code", resp.StatusCode).
		Msg("Received HTTP response")

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		log.Warn().
			Str("method", method).
			Str("url", url).
			Int("status_code", resp.StatusCode).
			Msg("Unexpected HTTP status code")
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
