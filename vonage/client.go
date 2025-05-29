package vonage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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

func NewClient(config Config) Client {
	return Client{
		config: config,
	}
}

func (c *Client) SendWhatsAppTextMessage(toNumber, senderID, text string) error {
	fmt.Printf("Preparing to send WhatsApp message to %s from %s: %s\n", toNumber, senderID, text)
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
	fmt.Printf("Marking message %s as read\n", messageID)
	payload := MarkAsReadPayload{
		Status: "read",
	}
	url := fmt.Sprintf("%s/%s", c.config.GeospecificMessagesAPIURL, messageID)
	return c.sendRequest("PATCH", url, payload)
}

func (c *Client) sendRequest(method, url string, body any) error {
	fmt.Printf("Sending %s request to %s with body: %+v\n", method, url, body)
	payload, err := json.Marshal(body)
	if err != nil {
		fmt.Printf("Error marshaling payload: %v\n", err)
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(payload))
	if err != nil {
		fmt.Printf("Error creating HTTP request: %v\n", err)
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.config.VonageJWT)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending HTTP request: %v\n", err)
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Received response with status code: %d\n", resp.StatusCode)
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		fmt.Printf("Unexpected status code: %d\n", resp.StatusCode)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
