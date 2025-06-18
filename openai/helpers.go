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

// convertChatHistoryWithUserName converts Redis chat messages to OpenAI message format with personalized system prompt.
// It includes the user's name in the system prompt to provide context to the AI.
func convertChatHistoryWithUserName(chatHistory []redis.ChatMessage, userName string) []openai.ChatCompletionMessageParamUnion {
	personalizedPrompt := createPersonalizedSystemPrompt(userName)
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(personalizedPrompt),
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

// createPersonalizedSystemPrompt creates a system prompt that includes the user's name for context.
func createPersonalizedSystemPrompt(userName string) string {
	if userName == "" {
		return systemPrompt
	}

	nameContext := "Você está conversando com " + userName + ".\n\n"
	return nameContext + systemPrompt
}
