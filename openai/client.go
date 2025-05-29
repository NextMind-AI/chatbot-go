package openai

import (
	"net/http"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type Client struct {
	client *openai.Client
}

func NewClient(apiKey string, httpClient *http.Client) Client {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithHTTPClient(httpClient),
	)

	openaiClient := Client{
		client: &client,
	}

	return openaiClient
}
