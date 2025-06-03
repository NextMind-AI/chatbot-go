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

	// Process audio transcription if present
	messageText := message.Text
	transcribedText, err := processAudioMessage(message)
	if err != nil {
		log.Error().
			Err(err).
			Str("message_uuid", message.MessageUUID).
			Msg("Error transcribing audio message")
		// Continue processing even if transcription fails
	} else if transcribedText != "" {
		log.Info().
			Str("message_uuid", message.MessageUUID).
			Str("transcribed_text", transcribedText).
			Msg("Audio message transcribed successfully")

		// Combine original text with transcribed text
		if messageText != "" {
			messageText = messageText + "\n\n[Audio transcription]: " + transcribedText
		} else {
			messageText = "[Audio transcription]: " + transcribedText
		}
	}

	// Use the combined text (original + transcribed) for processing
	finalMessageText := strings.TrimSpace(messageText)
	if finalMessageText == "" {
		log.Warn().
			Str("message_uuid", message.MessageUUID).
			Msg("No text content found in message (including transcription)")
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
		message.From,
		"5585936181311",
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
