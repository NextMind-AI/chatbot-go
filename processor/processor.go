package processor

import (
	"chatbot/elevenlabs"
	"chatbot/execution"
	"chatbot/openai"
	"chatbot/redis"
	"chatbot/vonage"
	"context"

	"github.com/rs/zerolog/log"
)

type MessageProcessor struct {
	vonageClient     vonage.Client
	redisClient      redis.Client
	openaiClient     openai.Client
	elevenLabsClient elevenlabs.Client
	executionManager *execution.Manager
}

func NewMessageProcessor(vonageClient vonage.Client, redisClient redis.Client, openaiClient openai.Client, elevenLabsClient elevenlabs.Client, execManager *execution.Manager) *MessageProcessor {
	return &MessageProcessor{
		vonageClient:     vonageClient,
		redisClient:      redisClient,
		openaiClient:     openaiClient,
		elevenLabsClient: elevenLabsClient,
		executionManager: execManager,
	}
}

func (mp *MessageProcessor) ProcessMessage(message InboundMessage) {
	log.Info().Str("message_uuid", message.MessageUUID).Msg("Processing message")

	userID := message.From
	ctx := mp.executionManager.Start(userID)
	defer mp.executionManager.Cleanup(userID, ctx)

	if err := mp.markMessageAsRead(message.MessageUUID); err != nil {
		log.Error().
			Err(err).
			Str("message_uuid", message.MessageUUID).
			Msg("Error marking message as read")
	}

	processedMsg, err := mp.extractMessageContent(message)
	if err != nil {
		log.Error().
			Err(err).
			Str("message_uuid", message.MessageUUID).
			Msg("Error processing message content")
		return
	}

	if mp.cancelled(ctx, userID, "after content extraction") {
		return
	}

	if err := mp.storeUserMessage(userID, processedMsg); err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error storing user message")
	}

	chatHistory, err := mp.getChatHistory(userID)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error retrieving chat history")
		chatHistory = []redis.ChatMessage{}
	}

	if mp.cancelled(ctx, userID, "before OpenAI call") {
		return
	}

	if err := mp.processWithAI(ctx, userID, chatHistory); err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error processing with AI")
		return
	}

	if mp.cancelled(ctx, userID, "after OpenAI call") {
		return
	}

	log.Info().Str("user_id", userID).Msg("Completed message processing")
}

func (mp *MessageProcessor) cancelled(ctx context.Context, userID, stage string) bool {
	if ctx.Err() != nil {
		log.Info().
			Str("user_id", userID).
			Msg("Message processing cancelled " + stage)
		return true
	}
	return false
}

func ProcessMessage(message InboundMessage, vonageClient vonage.Client, redisClient redis.Client, openaiClient openai.Client, elevenLabsClient elevenlabs.Client, execManager *execution.Manager) {
	processor := NewMessageProcessor(vonageClient, redisClient, openaiClient, elevenLabsClient, execManager)
	processor.ProcessMessage(message)
}
