package openai

import (
	"chatbot/redis"
	"context"

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

// ProcessChatWithTools processes a chat conversation with tool capabilities using OpenAI's API.
// Handles tool calls and manages the conversation flow accordingly.
// Returns the final AI response after processing any tool calls.
func (c *Client) ProcessChatWithTools(
    ctx context.Context,
    userID string,
    chatHistory []redis.ChatMessage,
) (string, error) {
    messages := convertChatHistory(ctx, userID, chatHistory) // Passar ctx e userID

    chatCompletion, err := c.createChatCompletionWithTools(ctx, messages)
    if err != nil {
        return "", err
    }

    toolCalls := chatCompletion.Choices[0].Message.ToolCalls

    if len(toolCalls) > 0 {
        messages = append(messages, chatCompletion.Choices[0].Message.ToParam())

        toolResponses, err := c.handleToolCalls(ctx, userID, toolCalls)
        if err != nil {
            return "", err
        }

        messages = append(messages, toolResponses...)

        return c.ProcessChat(ctx, messages)
    }

    return chatCompletion.Choices[0].Message.Content, nil
}
