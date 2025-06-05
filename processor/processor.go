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
	debounceManager  *DebounceManager
}

func NewMessageProcessor(vonageClient vonage.Client, redisClient redis.Client, openaiClient openai.Client, elevenLabsClient elevenlabs.Client, execManager *execution.Manager) *MessageProcessor {
	return &MessageProcessor{
		vonageClient:     vonageClient,
		redisClient:      redisClient,
		openaiClient:     openaiClient,
		elevenLabsClient: elevenLabsClient,
		executionManager: execManager,
		debounceManager:  NewDebounceManager(),
	}
}

func (mp *MessageProcessor) ProcessMessage(message InboundMessage) {
	log.Info().Str("message_uuid", message.MessageUUID).Msg("Received message - scheduling for debounced processing")

	userID := message.From

	// Mark message as read immediately
	if err := mp.markMessageAsRead(message.MessageUUID); err != nil {
		log.Error().
			Err(err).
			Str("message_uuid", message.MessageUUID).
			Msg("Error marking message as read")
	}

	// Extract and store message content immediately
	processedMsg, err := mp.extractMessageContent(message)
	if err != nil {
		log.Error().
			Err(err).
			Str("message_uuid", message.MessageUUID).
			Msg("Error processing message content")
		return
	}

	if err := mp.storeUserMessage(userID, processedMsg); err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error storing user message")
	}

	// Use debounce manager to schedule AI processing after 15 seconds
	// If another message comes within 15 seconds, this timer will be reset
	mp.debounceManager.ProcessMessage(userID, func() {
		mp.processMessageWithAI(userID)
	})
}

// processMessageWithAI handles the actual AI processing after the debounce period
func (mp *MessageProcessor) processMessageWithAI(userID string) {
	log.Info().Str("user_id", userID).Msg("Starting AI processing after debounce period")

	// Start execution context for this processing
	ctx := mp.executionManager.Start(userID)
	defer mp.executionManager.Cleanup(userID, ctx)

	// Get the latest chat history
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

	log.Info().Str("user_id", userID).Msg("Completed debounced message processing")
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
