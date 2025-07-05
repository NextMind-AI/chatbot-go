package processor

import (
	"context"
	"io"

	"github.com/NextMind-AI/chatbot-go/redis"
	"github.com/NextMind-AI/chatbot-go/vonage"
)

// VonageClientInterface define os métodos necessários do cliente Vonage
type VonageClientInterface interface {
    SendWhatsAppTextMessage(to, text string) (*vonage.MessageResponse, error)
    SendWhatsAppAudioMessage(to, audioURL string) (*vonage.MessageResponse, error)
    MarkMessageAsRead(messageUUID string) error
    SendWhatsAppReplyMessage(to, text, messageUUID string) (*vonage.MessageResponse, error)
}

// RedisClientInterface define os métodos necessários do cliente Redis
type RedisClientInterface interface {
    AddUserMessage(userID, message, messageUUID string) error
    AddBotMessage(userID, message string) error
    GetChatHistory(userID string) ([]redis.ChatMessage, error)
    ClearChatHistory(userID string) error
    GetAllConversationSummaries() ([]redis.ConversationSummary, error)
    GetChatHistoryPaginated(userID string, page, pageSize int) (redis.PaginatedMessages, error)
}

// OpenAIClientInterface define os métodos necessários do cliente OpenAI
type OpenAIClientInterface interface {
    ProcessChatStreamingWithTools(
        ctx context.Context,
        userID string,
        userName string,
        chatHistory []redis.ChatMessage,
        vonageClient VonageClientInterface,
        redisClient RedisClientInterface,
        elevenLabsClient ElevenLabsClientInterface,
        toNumber string,
    ) error
    ProcessChatWithTools(ctx context.Context, userID, userName string, chatHistory []redis.ChatMessage) (string, error)
}

// ElevenLabsClientInterface define os métodos necessários do cliente ElevenLabs
type ElevenLabsClientInterface interface {
    ConvertTextToSpeechDefault(text string) (string, error)
    ConvertTextToSpeech(voiceID, text, modelID string) (string, error)
    TranscribeAudioFile(file io.Reader, fileName string) (string, error)
    TranscribeAudio(audioURL string) (string, error)
}