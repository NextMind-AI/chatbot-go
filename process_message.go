package main

import (
	"chatbot/execution"
	"chatbot/redis"
	"context"
	"encoding/json"
	"time"

	"github.com/openai/openai-go"
	"github.com/rs/zerolog/log"
)

var executionManager = execution.NewManager()

func processMessage(message InboundMessage) {
	log.Info().Str("message_uuid", message.MessageUUID).Msg("Processing message")

	userID := message.From

	ctx := executionManager.Start(userID)
	defer executionManager.Cleanup(userID, ctx)

	if err := VonageClient.MarkMessageAsRead(message.MessageUUID); err != nil {
		log.Error().
			Err(err).
			Str("message_uuid", message.MessageUUID).
			Msg("Error marking message as read")
	}

	if err := RedisClient.AddUserMessage(
		userID,
		message.Text,
		message.MessageUUID,
	); err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error storing user message in Redis")
	}

	chatHistory, err := RedisClient.GetChatHistory(userID)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error retrieving chat history from Redis")
		chatHistory = []redis.ChatMessage{}
	}

	messages := []openai.ChatCompletionMessageParamUnion{}
	for _, msg := range chatHistory {
		switch msg.Role {
		case "user":
			messages = append(messages, openai.UserMessage(msg.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(msg.Content))
		}
	}

	if cancelled(ctx, userID, "before OpenAI call") {
		return
	}

	chatCompletion, err := OpenAIClient.Chat.Completions.New(
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
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error creating chat completion")
		return
	}

	if cancelled(ctx, userID, "after OpenAI call") {
		return
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
					return
				}

				messages = append(messages, openai.ToolMessage("Sleep completed", toolCall.ID))
			}
		}

		chatCompletion, err = OpenAIClient.Chat.Completions.New(
			ctx,
			openai.ChatCompletionNewParams{
				Messages: messages,
				Model:    openai.ChatModelGPT4_1Mini,
			},
		)
		if err != nil {
			log.Error().
				Err(err).
				Str("user_id", userID).
				Msg("Error creating follow-up chat completion")
			return
		}

		if cancelled(ctx, userID, "after follow-up OpenAI call") {
			return
		}
	}

	botResponse := chatCompletion.Choices[0].Message.Content

	if err := RedisClient.AddBotMessage(userID, botResponse); err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error storing bot message in Redis")
	}

	_, err = VonageClient.SendWhatsAppTextMessage(message.From, "5563936180023", botResponse)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Str("to", message.From).
			Msg("Error sending WhatsApp message")
		return
	}

	log.Info().Str("user_id", userID).Msg("Sent WhatsApp message")
}

func cancelled(ctx context.Context, userID, stage string) bool {
	if ctx.Err() != nil {
		log.Info().
			Str("user_id", userID).
			Msg("Message processing cancelled " + stage)
		return true
	}
	return false
}
