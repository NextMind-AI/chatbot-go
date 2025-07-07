package processor

import (
	"chatbot/elevenlabs"
	"chatbot/execution"
	"chatbot/openai"
	"chatbot/redis"
	"chatbot/vonage"
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

// WebSocketCallback is a function type for WebSocket message broadcasting
type WebSocketCallback func(msgType, userID, message, role string)

type MessageProcessor struct {
	vonageClient     vonage.Client
	redisClient      redis.Client
	openaiClient     openai.Client
	elevenLabsClient elevenlabs.Client
	executionManager *execution.Manager
	debounceManager  *DebounceManager
	wsCallback       WebSocketCallback
}

func NewMessageProcessor(vonageClient vonage.Client, redisClient redis.Client, openaiClient openai.Client, elevenLabsClient elevenlabs.Client, execManager *execution.Manager) *MessageProcessor {
	return &MessageProcessor{
		vonageClient:     vonageClient,
		redisClient:      redisClient,
		openaiClient:     openaiClient,
		elevenLabsClient: elevenLabsClient,
		executionManager: execManager,
		debounceManager:  NewDebounceManager(),
		wsCallback:       nil, // Will be set by the server
	}
}

// SetWebSocketCallback sets the callback function for WebSocket notifications
func (mp *MessageProcessor) SetWebSocketCallback(callback WebSocketCallback) {
	mp.wsCallback = callback
}

// notifyWebSocket sends a notification via WebSocket if callback is set
func (mp *MessageProcessor) notifyWebSocket(msgType, userID, message, role string) {
	if mp.wsCallback != nil {
		mp.wsCallback(msgType, userID, message, role)
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

	// Add a timeout to prevent indefinite hanging
	ctx, timeoutCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer timeoutCancel()

	// Add a recovery mechanism to ensure we always respond to the user
	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Str("user_id", userID).
				Interface("panic", r).
				Msg("Panic occurred during AI processing - sending fallback response")
			mp.sendFallbackResponse(userID)
		}
	}()

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
			Msg("Error processing with AI - sending fallback response")

		// Instead of just returning, send a fallback response to the user
		mp.sendFallbackResponse(userID)
		return
	}

	if mp.cancelled(ctx, userID, "after OpenAI call") {
		return
	}

	log.Info().Str("user_id", userID).Msg("Completed debounced message processing")
}

// sendFallbackResponse sends a fallback message to the user when AI processing fails
func (mp *MessageProcessor) sendFallbackResponse(userID string) {
	fallbackMessage := "I'm sorry, I'm experiencing some technical difficulties. Please try sending your message again."

	// Try to send the fallback message
	if _, err := mp.vonageClient.SendWhatsAppTextMessage(userID, fallbackMessage); err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Failed to send fallback response")
	} else {
		log.Info().
			Str("user_id", userID).
			Msg("Sent fallback response to user")

		// Store the fallback message in chat history
		if err := mp.storeBotMessage(userID, fallbackMessage); err != nil {
			log.Error().
				Err(err).
				Str("user_id", userID).
				Msg("Error storing fallback message in Redis")
		}
	}
}

// GetProcessorStatus returns the current status of the processor for monitoring
func (mp *MessageProcessor) GetProcessorStatus() map[string]interface{} {
	debounceStats := mp.debounceManager.GetStatistics()
	return map[string]interface{}{
		"debounce_stats": debounceStats,
		"timestamp":      time.Now().Unix(),
	}
}

// GetRedisClient returns the Redis client for CRM operations
func (mp *MessageProcessor) GetRedisClient() *redis.Client {
	return &mp.redisClient
}

func (mp *MessageProcessor) cancelled(ctx context.Context, userID, stage string) bool {
	if ctx.Err() != nil {
		if ctx.Err() == context.Canceled {
			log.Info().
				Str("user_id", userID).
				Str("stage", stage).
				Msg("Message processing cancelled due to context cancellation")
		} else if ctx.Err() == context.DeadlineExceeded {
			log.Warn().
				Str("user_id", userID).
				Str("stage", stage).
				Msg("Message processing cancelled due to timeout")
			// Send a timeout message to the user
			mp.sendTimeoutResponse(userID)
		}
		return true
	}
	return false
}

// sendTimeoutResponse sends a timeout message to the user
func (mp *MessageProcessor) sendTimeoutResponse(userID string) {
	timeoutMessage := "I'm taking longer than expected to process your message. Please try again in a moment."

	if _, err := mp.vonageClient.SendWhatsAppTextMessage(userID, timeoutMessage); err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Failed to send timeout response")
	} else {
		log.Info().
			Str("user_id", userID).
			Msg("Sent timeout response to user")

		if err := mp.storeBotMessage(userID, timeoutMessage); err != nil {
			log.Error().
				Err(err).
				Str("user_id", userID).
				Msg("Error storing timeout message in Redis")
		}
	}
}
