package openai

import (
	"chatbot/redis"

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
