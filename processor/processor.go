package processor

import (
	"context"
	"errors"
	"fmt"

	"github.com/NextMind-AI/chatbot-go/execution"
	"github.com/NextMind-AI/chatbot-go/redis"

	"github.com/rs/zerolog/log"
)

type MessageProcessor struct {
	vonageClient     VonageClientInterface
	redisClient      RedisClientInterface
	openaiClient     OpenAIClientInterface
	elevenLabsClient ElevenLabsClientInterface
	executionManager *execution.Manager
}

func NewMessageProcessor(vonageClient VonageClientInterface, redisClient RedisClientInterface, openaiClient OpenAIClientInterface, elevenLabsClient ElevenLabsClientInterface, execManager *execution.Manager) *MessageProcessor {
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
func (mp *MessageProcessor) GetRedisClient() RedisClientInterface {
	return mp.redisClient
}

// ProcessLocalTestMessage processa uma mensagem de teste local e retorna a resposta
func (mp *MessageProcessor) ProcessLocalTestMessage(message InboundMessage) (string, error) {
	log.Info().Str("message_text", message.Text).Msg("Processing local test message")

	userID := message.From
	ctx := context.Background() // Usa context simples para teste local

	// Extrai o conteúdo da mensagem
	processedMsg, err := mp.extractMessageContent(message)
	if err != nil {
		return "", fmt.Errorf("error processing message content: %w", err)
	}

	// Armazena a mensagem do usuário
	if err := mp.storeUserMessage(userID, processedMsg); err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Error storing user message")
		// Não retorna erro, apenas registra
	}

	// Obtém o histórico do chat
	chatHistory, err := mp.getChatHistory(userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Error retrieving chat history")
		chatHistory = []redis.ChatMessage{}
	}

	// Processa com IA (em modo local, usa o wrapper mock)
	if mockClient, ok := mp.openaiClient.(*MockOpenAIStreamingClient); ok {
		err = mockClient.ProcessChatStreamingWithTools(
			ctx,
			userID,
			message.Profile.Name,
			chatHistory,
			nil, // vonageClient não é usado em modo local
			mp.redisClient,
			nil, // elevenLabsClient não é usado em modo local
			userID,
		)
		if err != nil {
			return "", fmt.Errorf("error processing with AI: %w", err)
		}

		// Retorna a última resposta gerada
		return mockClient.GetLastResponse(), nil
	}

	// Fallback para cliente real (não deveria acontecer em modo local)
	return "", fmt.Errorf("local mode not properly configured")
}

// ClearTestUserHistory limpa o histórico de um usuário em modo de teste
func (mp *MessageProcessor) ClearTestUserHistory(userID string) error {
	if mockRedis, ok := mp.redisClient.(*MockRedisClient); ok {
		return mockRedis.ClearChatHistory(userID)
	}
	return fmt.Errorf("not in local mode")
}
