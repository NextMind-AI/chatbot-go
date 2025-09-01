package openai

import (
	"github.com/NextMind-AI/chatbot-go/redis"

	"github.com/openai/openai-go"
)

// convertChatHistoryWithUserName converts Redis chat messages to OpenAI message format with personalized system prompt.
// It includes the user's name and phone number in the system prompt to provide context to the AI.
func (c *Client) convertChatHistoryWithUserName(chatHistory []redis.ChatMessage, userName string, userID string) []openai.ChatCompletionMessageParamUnion {
	personalizedPrompt := c.promptGenerator(userName, userID)
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
