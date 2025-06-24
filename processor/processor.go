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

// ProcessMessage agora distingue entre execução imediata (COM tools) e debounce (SEM tools)
func (mp *MessageProcessor) ProcessMessage(message InboundMessage) {
	log.Info().Str("message_uuid", message.MessageUUID).Msg("Received message - processing immediately WITH tools")

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

	// EXECUÇÃO IMEDIATA COM TOOLS
	go func() {
		ctx := mp.executionManager.Start(userID + "_immediate")
		defer mp.executionManager.Cleanup(userID+"_immediate", ctx)

		ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()

		chatHistory, err := mp.getChatHistory(userID)
		if err != nil {
			log.Error().Err(err).Str("user_id", userID).Msg("Error getting chat history for immediate processing")
			chatHistory = []redis.ChatMessage{}
		}

		log.Info().Str("user_id", userID).Msg("Processando mensagem imediatamente COM tools")

		if err := mp.processWithAIWithTools(ctx, userID, chatHistory); err != nil {
			log.Error().
				Err(err).
				Str("user_id", userID).
				Msg("Error in immediate processing with tools")
		}
	}()

	// DEBOUNCE SEM TOOLS (apenas para contexto adicional)
	mp.debounceManager.ProcessMessage(userID, func() {
		mp.processMessageWithAIWithoutTools(userID)
	})
}

// processMessageWithAIWithoutTools - versão do debounce SEM tools
func (mp *MessageProcessor) processMessageWithAIWithoutTools(userID string) {
	log.Info().Str("user_id", userID).Msg("Starting AI processing after debounce period WITHOUT tools")

	ctx := mp.executionManager.Start(userID + "_debounce")
	defer mp.executionManager.Cleanup(userID+"_debounce", ctx)

	ctx, timeoutCancel := context.WithTimeout(ctx, 3*time.Minute)
	defer timeoutCancel()

	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Str("user_id", userID).
				Interface("panic", r).
				Msg("Panic occurred during debounce AI processing")
		}
	}()

	chatHistory, err := mp.getChatHistory(userID)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error retrieving chat history for debounce")
		return
	}

	if mp.cancelled(ctx, userID, "before debounce OpenAI call") {
		return
	}

	// Processar SEM TOOLS para evitar duplicação
	if err := mp.processWithAIWithoutTools(ctx, userID, chatHistory); err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error processing debounce without tools")
		return
	}

	log.Info().Str("user_id", userID).Msg("Completed debounce processing WITHOUT tools")
}

// processWithAIWithTools - versão COM tools (execução imediata)
func (mp *MessageProcessor) processWithAIWithTools(ctx context.Context, userID string, chatHistory []redis.ChatMessage) error {
	log.Info().Str("user_id", userID).Msg("Processing with AI WITH tools")

	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Str("user_id", userID).
				Interface("panic", r).
				Msg("Panic recovered in processWithAIWithTools")
		}
	}()

	toNumber := userID
	return mp.openaiClient.ProcessChatStreamingWithTools(
		ctx,
		userID,
		chatHistory,
		&mp.vonageClient,
		&mp.redisClient,
		&mp.elevenLabsClient,
		toNumber,
	)
}

// processWithAIWithoutTools - versão SEM tools (debounce)
func (mp *MessageProcessor) processWithAIWithoutTools(ctx context.Context, userID string, chatHistory []redis.ChatMessage) error {
	log.Info().Str("user_id", userID).Msg("Processing with AI WITHOUT tools (debounce)")

	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Str("user_id", userID).
				Interface("panic", r).
				Msg("Panic recovered in processWithAIWithoutTools")
		}
	}()

	toNumber := userID
	return mp.openaiClient.ProcessChatStreamingWithoutTools(
		ctx,
		userID,
		chatHistory,
		&mp.vonageClient,
		&mp.redisClient,
		&mp.elevenLabsClient,
		toNumber,
	)
}

// Manter a função original processMessageWithAI como fallback
func (mp *MessageProcessor) processMessageWithAI(userID string) {
	// Redirecionar para a versão sem tools
	mp.processMessageWithAIWithoutTools(userID)
}

// GetProcessorStatus returns the current status of the processor for monitoring
func (mp *MessageProcessor) GetProcessorStatus() map[string]interface{} {
	debounceStats := mp.debounceManager.GetStatistics()
	return map[string]interface{}{
		"debounce_stats": debounceStats,
		"timestamp":      time.Now().Unix(),
	}
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

		if err := mp.redisClient.AddBotMessage(userID, timeoutMessage); err != nil {
			log.Error().
				Err(err).
				Str("user_id", userID).
				Msg("Error storing timeout message in Redis")
		}
	}
}
