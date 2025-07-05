package processor

import (
	"context"
	"io"

	"github.com/NextMind-AI/chatbot-go/elevenlabs"
	"github.com/NextMind-AI/chatbot-go/openai"
	"github.com/NextMind-AI/chatbot-go/redis"
	"github.com/NextMind-AI/chatbot-go/vonage"
)

// RedisClientWrapper adapta o cliente Redis real para a interface
type RedisClientWrapper struct {
	client *redis.Client
}

func NewRedisClientWrapper(client *redis.Client) *RedisClientWrapper {
	return &RedisClientWrapper{client: client}
}

func (w *RedisClientWrapper) AddUserMessage(userID, message, messageUUID string) error {
	return w.client.AddUserMessage(userID, message, messageUUID)
}

func (w *RedisClientWrapper) AddBotMessage(userID, message string) error {
	return w.client.AddBotMessage(userID, message)
}

func (w *RedisClientWrapper) GetChatHistory(userID string) ([]redis.ChatMessage, error) {
	return w.client.GetChatHistory(userID)
}

func (w *RedisClientWrapper) ClearChatHistory(userID string) error {
	return w.client.ClearChatHistory(userID)
}

func (w *RedisClientWrapper) GetAllConversationSummaries() ([]redis.ConversationSummary, error) {
	return w.client.GetAllConversationSummaries()
}

func (w *RedisClientWrapper) GetChatHistoryPaginated(userID string, page, pageSize int) (redis.PaginatedMessages, error) {
	return w.client.GetChatHistoryPaginated(userID, page, pageSize)
}

// VonageClientWrapper adapta o cliente Vonage real para a interface
type VonageClientWrapper struct {
	client *vonage.Client
}

func NewVonageClientWrapper(client *vonage.Client) *VonageClientWrapper {
	return &VonageClientWrapper{client: client}
}

func (w *VonageClientWrapper) SendWhatsAppTextMessage(to, text string) (*vonage.MessageResponse, error) {
	return w.client.SendWhatsAppTextMessage(to, text)
}

func (w *VonageClientWrapper) SendWhatsAppAudioMessage(to, audioURL string) (*vonage.MessageResponse, error) {
	return w.client.SendWhatsAppAudioMessage(to, audioURL)
}

func (w *VonageClientWrapper) MarkMessageAsRead(messageUUID string) error {
	return w.client.MarkMessageAsRead(messageUUID)
}

func (w *VonageClientWrapper) SendWhatsAppReplyMessage(to, text, messageUUID string) (*vonage.MessageResponse, error) {
	return w.client.SendWhatsAppReplyMessage(to, text, messageUUID)
}

// OpenAIClientWrapper adapta o cliente OpenAI real para a interface
type OpenAIClientWrapper struct {
	client           *openai.Client
	vonageClient     *vonage.Client
	redisClient      *redis.Client
	elevenLabsClient *elevenlabs.Client
}

func NewOpenAIClientWrapper(client *openai.Client, vonageClient *vonage.Client, redisClient *redis.Client, elevenLabsClient *elevenlabs.Client) *OpenAIClientWrapper {
	return &OpenAIClientWrapper{
		client:           client,
		vonageClient:     vonageClient,
		redisClient:      redisClient,
		elevenLabsClient: elevenLabsClient,
	}
}

func (w *OpenAIClientWrapper) ProcessChatStreamingWithTools(
	ctx context.Context,
	userID string,
	userName string,
	chatHistory []redis.ChatMessage,
	vonageClient VonageClientInterface,
	redisClient RedisClientInterface,
	elevenLabsClient ElevenLabsClientInterface,
	toNumber string,
) error {
	// Usa os clientes concretos armazenados no wrapper
	return w.client.ProcessChatStreamingWithTools(
		ctx,
		userID,
		userName,
		chatHistory,
		w.vonageClient,
		w.redisClient,
		w.elevenLabsClient,
		toNumber,
	)
}

func (w *OpenAIClientWrapper) ProcessChatWithTools(ctx context.Context, userID, userName string, chatHistory []redis.ChatMessage) (string, error) {
	return w.client.ProcessChatWithTools(ctx, userID, userName, chatHistory)
}

// ElevenLabsClientWrapper adapta o cliente ElevenLabs real para a interface
type ElevenLabsClientWrapper struct {
	client *elevenlabs.Client
}

func NewElevenLabsClientWrapper(client *elevenlabs.Client) *ElevenLabsClientWrapper {
	return &ElevenLabsClientWrapper{client: client}
}

func (w *ElevenLabsClientWrapper) ConvertTextToSpeechDefault(text string) (string, error) {
	return w.client.ConvertTextToSpeechDefault(text)
}

func (w *ElevenLabsClientWrapper) ConvertTextToSpeech(voiceID, text, modelID string) (string, error) {
	return w.client.ConvertTextToSpeech(voiceID, text, modelID)
}

func (w *ElevenLabsClientWrapper) TranscribeAudioFile(file io.Reader, fileName string) (string, error) {
	return w.client.TranscribeAudioFile(file, fileName)
}

func (w *ElevenLabsClientWrapper) TranscribeAudio(audioURL string) (string, error) {
	return w.client.TranscribeAudio(audioURL)
}