package openai

import (
	"net/http"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// Client wraps the OpenAI client with additional functionality for chat processing.
// It provides methods for both simple chat completion and tool-enabled conversations.
type Client struct {
	client *openai.Client
}

// NewClient creates a new OpenAI client wrapper with the specified API key and HTTP client.
// The HTTP client allows for custom configuration such as timeouts and proxy settings.
func NewClient(apiKey string, httpClient http.Client) Client {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithHTTPClient(&httpClient),
	)

	openaiClient := Client{
		client: &client,
	}

	return openaiClient
}
