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
// It includes the user's name and phone number in the system prompt to provide context to the AI.
func convertChatHistoryWithUserName(chatHistory []redis.ChatMessage, userName string, userID string) []openai.ChatCompletionMessageParamUnion {
	personalizedPrompt := createPersonalizedSystemPrompt(userName, userID)
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

// createPersonalizedSystemPrompt creates a system prompt that includes the user's name and phone number for context.
func createPersonalizedSystemPrompt(userName string, userID string) string {
	var userContext string

	if userName != "" && userID != "" {
		userContext = "Você está conversando com " + userName + " (telefone: " + userID + ").\n\n"
	} else if userName != "" {
		userContext = "Você está conversando com " + userName + ".\n\n"
	} else if userID != "" {
		userContext = "Você está conversando com o usuário do telefone " + userID + ".\n\n"
	}

	if userContext == "" {
		return systemPrompt
	}

	return userContext + systemPrompt
}
