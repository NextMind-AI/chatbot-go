package main

import (
	"chatbot/execution"
	"chatbot/redis"
	"context"
	"strings"

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

	var messageText string
	switch message.MessageType {
	case "text":
		messageText = message.Text
	case "audio":
		transcribedText, err := ElevenLabsClient.TranscribeAudio(message.Audio.URL)
		if err != nil {
			log.Error().
				Err(err).
				Str("message_uuid", message.MessageUUID).
				Msg("Error transcribing audio message")
			return
		}
		messageText = transcribedText
	default:
		log.Warn().
			Str("message_type", message.MessageType).
			Str("message_uuid", message.MessageUUID).
			Msg("Unsupported message type")

		if _, err := VonageClient.SendWhatsAppReplyMessage(
			userID,
			"I can't process this message type for now",
			message.MessageUUID,
		); err != nil {
			log.Error().
				Err(err).
				Str("user_id", userID).
				Msg("Error sending unsupported message type response")
		}
		return
	}

	finalMessageText := strings.TrimSpace(messageText)
	if finalMessageText == "" {
		log.Error().
			Str("message_uuid", message.MessageUUID).
			Msg("No text content found in message")
		return
	}

	if err := RedisClient.AddUserMessage(
		userID,
		finalMessageText,
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

	if cancelled(ctx, userID, "before OpenAI call") {
		return
	}

	err = OpenAIClient.ProcessChatStreamingWithTools(
		ctx,
		userID,
		chatHistory,
		&VonageClient,
		&RedisClient,
		&ElevenLabsClient,
		AppConfig.ElevenLabsVoiceID,
		AppConfig.ElevenLabsModelID,
		userID,
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error processing streaming chat with OpenAI")
		return
	}

	if cancelled(ctx, userID, "after OpenAI call") {
		return
	}

	log.Info().Str("user_id", userID).Msg("Completed streaming chat processing")
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
