package openai

import (
	"context"
	"time"

	"github.com/NextMind-AI/chatbot-go/redis"

	"github.com/openai/openai-go"
)

// ProcessChat processes a chat conversation without tools using OpenAI's API.
// It takes a context and a slice of messages and returns the AI's response.
func (c *Client) ProcessChat(
	ctx context.Context,
	messages []openai.ChatCompletionMessageParamUnion,
) (string, error) {
	chatCompletion, err := c.client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Messages: messages,
			Model:    openai.ChatModelGPT4_1Mini,
		},
	)
	if err != nil {
		return "", err
	}

	return chatCompletion.Choices[0].Message.Content, nil
}

// ProcessChatWithTools processes a chat conversation with the new two-step approach:
// First, it uses a sleep analyzer to determine wait time, then generates the actual response.
func (c *Client) ProcessChatWithTools(
	ctx context.Context,
	userID string,
	userName string,
	chatHistory []redis.ChatMessage,
) (string, error) {
	// Step 1: Determine sleep time using the sleep analyzer with full conversation context
	sleepSeconds, err := c.DetermineSleepTime(ctx, userID, userName, chatHistory)
	if err != nil {
		// Log warning but continue without sleep
		sleepSeconds = 0
	}

	// Step 2: Execute the sleep if needed
	if sleepSeconds > 0 {
		time.Sleep(time.Duration(sleepSeconds) * time.Second)
	}

	// Step 3: Generate the actual response (without tools)
	messages := c.convertChatHistoryWithUserName(chatHistory, userName, userID)
	return c.ProcessChat(ctx, messages)
}
