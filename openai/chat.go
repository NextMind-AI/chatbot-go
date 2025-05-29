package openai

import (
	"chatbot/redis"
	"context"
	"encoding/json"
	"time"

	"github.com/openai/openai-go"
	"github.com/rs/zerolog/log"
)

func (c *Client) ProcessChatWithTools(ctx context.Context, userID string, chatHistory []redis.ChatMessage) (string, error) {
	messages := []openai.ChatCompletionMessageParamUnion{}
	for _, msg := range chatHistory {
		switch msg.Role {
		case "user":
			messages = append(messages, openai.UserMessage(msg.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(msg.Content))
		}
	}

	chatCompletion, err := c.client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Messages: messages,
			Model:    openai.ChatModelGPT4_1Mini,
			Tools: []openai.ChatCompletionToolParam{
				{
					Function: openai.FunctionDefinitionParam{
						Name:        "sleep",
						Description: openai.String("Wait for a specified number of seconds before continuing the conversation"),
						Parameters: openai.FunctionParameters{
							"type": "object",
							"properties": map[string]any{
								"seconds": map[string]string{
									"type":        "integer",
									"description": "Number of seconds to wait",
								},
							},
							"required": []string{"seconds"},
						},
					},
				},
			},
		},
	)
	if err != nil {
		return "", err
	}

	toolCalls := chatCompletion.Choices[0].Message.ToolCalls

	if len(toolCalls) > 0 {
		messages = append(messages, chatCompletion.Choices[0].Message.ToParam())

		for _, toolCall := range toolCalls {
			if toolCall.Function.Name == "sleep" {
				var args map[string]any
				err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
				if err != nil {
					log.Error().
						Err(err).
						Str("user_id", userID).
						Msg("Error parsing sleep function arguments")
					continue
				}

				seconds, ok := args["seconds"].(float64)
				if !ok {
					log.Error().
						Str("user_id", userID).
						Msg("Invalid seconds parameter for sleep function")
					continue
				}

				log.Info().
					Str("user_id", userID).
					Float64("seconds", seconds).
					Msg("Sleeping before continuing conversation")

				sleepDuration := time.Duration(seconds) * time.Second
				select {
				case <-time.After(sleepDuration):
				case <-ctx.Done():
					log.Info().
						Str("user_id", userID).
						Msg("Sleep cancelled due to context cancellation")
					return "", ctx.Err()
				}

				messages = append(messages, openai.ToolMessage("Sleep completed", toolCall.ID))
			}
		}

		chatCompletion, err = c.client.Chat.Completions.New(
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

	return chatCompletion.Choices[0].Message.Content, nil
}
