package openai

import (
	"chatbot/redis"
	"context"

	"github.com/openai/openai-go"
)

// convertChatHistory converts Redis chat messages to OpenAI message format.
// It transforms the chat history from the Redis format to the format expected by OpenAI's API.
func convertChatHistory(chatHistory []redis.ChatMessage) []openai.ChatCompletionMessageParamUnion {
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
	}
	for _, msg := range chatHistory {
		switch msg.Role {
		case "user":
			messages = append(messages, openai.UserMessage(msg.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(msg.Content))
		}
	}
	return messages
}

// createChatCompletionWithTools creates a chat completion request with tool capabilities.
// It sends a request to OpenAI's API with the specified messages and available tools.
func (c *Client) createChatCompletionWithTools(
	ctx context.Context,
	messages []openai.ChatCompletionMessageParamUnion,
) (*openai.ChatCompletion, error) {
	return c.client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Messages: messages,
			Model:    openai.ChatModelGPT4_1Mini,
			Tools:    []openai.ChatCompletionToolParam{checkServicesTool},
		},
	)
}
