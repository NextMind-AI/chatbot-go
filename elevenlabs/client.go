package elevenlabs

import (
	"net/http"
)

const (
	BaseURL      = "https://api.elevenlabs.io/v1"
	DefaultModel = "scribe_v1"
)

type Client struct {
	APIKey       string
	LanguageCode string
	HTTPClient   *http.Client
}

func NewClient(apiKey string, httpClient http.Client) Client {
	return Client{
		APIKey:       apiKey,
		LanguageCode: "pt",
		HTTPClient:   &httpClient,
	}
}
