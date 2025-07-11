package processor

import (
	"context"
	"errors"

	"github.com/NextMind-AI/chatbot-go/elevenlabs"
	"github.com/NextMind-AI/chatbot-go/execution"
	"github.com/NextMind-AI/chatbot-go/openai"
	"github.com/NextMind-AI/chatbot-go/redis"
	"github.com/NextMind-AI/chatbot-go/vonage"

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

	if err := mp.processWithAI(ctx, userID, message.Profile.Name, chatHistory); err != nil {
		if errors.Is(err, context.Canceled) {
			log.Info().
				Str("user_id", userID).
				Msg("AI processing cancelled due to context cancellation")
			return
		}
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

// GetRedisClient returns the Redis client for external access
func (mp *MessageProcessor) GetRedisClient() *redis.Client {
	return &mp.redisClient
}
