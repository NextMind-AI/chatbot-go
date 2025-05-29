package main

import (
	"chatbot/redis"
	"context"

	"github.com/openai/openai-go"
	"github.com/rs/zerolog/log"
)

func processMessage(message InboundMessage) {
	log.Info().Str("message_uuid", message.MessageUUID).Msg("Processing message")

	VonageClient.MarkMessageAsRead(message.MessageUUID)

	userID := message.From

	if err := RedisClient.AddUserMessage(userID, message.Text); err != nil {
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
		if msg.Role == "user" {
			messages = append(messages, openai.UserMessage(msg.Content))
		} else if msg.Role == "assistant" {
			messages = append(messages, openai.AssistantMessage(msg.Content))
		}
	}

	log.Debug().
		Int("message_count", len(messages)).
		Str("user_id", userID).
		Msg("Using OpenAI client with historical messages")

	chatCompletion, err := OpenAIClient.Chat.Completions.New(
		context.TODO(),
		openai.ChatCompletionNewParams{
			Messages: messages,
			Model:    openai.ChatModelGPT4_1Mini,
		},
	)

	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error creating chat completion")
		return
	}

	botResponse := chatCompletion.Choices[0].Message.Content
	log.Debug().
		Str("user_id", userID).
		Str("response", botResponse).
		Msg("Received chat completion")

	if err := RedisClient.AddBotMessage(userID, botResponse); err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error storing bot message in Redis")
	}

	VonageClient.SendWhatsAppTextMessage(message.From, "5563936180023", botResponse)
	log.Info().Str("user_id", userID).Msg("Sent WhatsApp message")
}
