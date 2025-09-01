package openai

import (
	"context"
	"net/http"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// ToolHandler represents a function that handles a tool call and returns the result
type ToolHandler func(ctx context.Context, args map[string]any) (string, error)

// Tool represents a custom tool that can be called by the AI
type Tool struct {
	Definition openai.ChatCompletionToolParam
	Handler    ToolHandler
}

// PromptGenerator is a function that generates the system prompt based on user context
type PromptGenerator func(userName, userPhone string) string

// Client wraps the OpenAI client with additional functionality for chat processing.
// It provides methods for both simple chat completion and tool-enabled conversations.
type Client struct {
	client          *openai.Client
	promptGenerator PromptGenerator
	tools           []Tool
	model           string
}

// NewClient creates a new OpenAI client wrapper with the specified API key and HTTP client.
// The HTTP client allows for custom configuration such as timeouts and proxy settings.
func NewClient(apiKey string, httpClient http.Client, promptGenerator PromptGenerator, tools []Tool, model string) Client {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithHTTPClient(&httpClient),
	)

	// Use default prompt generator if none provided
	if promptGenerator == nil {
		promptGenerator = func(userName, userPhone string) string {
			var userContext string
			if userName != "" && userPhone != "" {
				userContext = "Você está conversando com " + userName + " (telefone: " + userPhone + ").\n\n"
			} else if userName != "" {
				userContext = "Você está conversando com " + userName + ".\n\n"
			} else if userPhone != "" {
				userContext = "Você está conversando com o usuário do telefone " + userPhone + ".\n\n"
			}
			return userContext + systemPrompt
		}
	}

	// Use default model if none provided
	if model == "" {
		model = openai.ChatModelGPT4_1Mini
	}

	openaiClient := Client{
		client:          &client,
		promptGenerator: promptGenerator,
		tools:           tools,
		model:           model,
	}

	return openaiClient
}
