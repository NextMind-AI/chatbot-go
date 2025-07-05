package processor

import (
	"context"
	"io"
	"log"

	"github.com/NextMind-AI/chatbot-go/openai"
	"github.com/NextMind-AI/chatbot-go/redis"
	"github.com/NextMind-AI/chatbot-go/vonage"
)

// MockVonageClient implementa VonageClientInterface para testes locais
type MockVonageClient struct{}

func (m *MockVonageClient) SendWhatsAppTextMessage(to, text string) (*vonage.MessageResponse, error) {
    log.Printf("🚀 MOCK: Enviando mensagem de texto para %s: %s", to, text)
    return &vonage.MessageResponse{
        MessageUUID: "mock-message-uuid",
    }, nil
}

func (m *MockVonageClient) SendWhatsAppAudioMessage(to, audioURL string) (*vonage.MessageResponse, error) {
    log.Printf("🎵 MOCK: Enviando mensagem de áudio para %s: %s", to, audioURL)
    return &vonage.MessageResponse{
        MessageUUID: "mock-audio-message-uuid",
    }, nil
}

func (m *MockVonageClient) MarkMessageAsRead(messageUUID string) error {
    log.Printf("✅ MOCK: Marcando mensagem como lida: %s", messageUUID)
    return nil
}

func (m *MockVonageClient) SendWhatsAppReplyMessage(to, text, messageUUID string) (*vonage.MessageResponse, error) {
    log.Printf("🔄 MOCK: Enviando resposta para %s: %s (reply to %s)", to, text, messageUUID)
    return &vonage.MessageResponse{
        MessageUUID: "mock-reply-message-uuid",
    }, nil
}

// MockElevenLabsClient implementa ElevenLabsClientInterface para testes locais
type MockElevenLabsClient struct{}

func (m *MockElevenLabsClient) ConvertTextToSpeechDefault(text string) (string, error) {
    log.Printf("🎙️ MOCK: Convertendo texto para áudio: %s", text)
    return "https://mock-audio-url.com/audio.mp3", nil
}

func (m *MockElevenLabsClient) ConvertTextToSpeech(voiceID, text, modelID string) (string, error) {
    log.Printf("🎙️ MOCK: Convertendo texto para áudio com voz %s: %s", voiceID, text)
    return "https://mock-audio-url.com/audio.mp3", nil
}

func (m *MockElevenLabsClient) TranscribeAudioFile(file io.Reader, fileName string) (string, error) {
    log.Printf("🎧 MOCK: Transcrevendo áudio: %s", fileName)
    return "Mock transcription result", nil
}

func (m *MockElevenLabsClient) TranscribeAudio(audioURL string) (string, error) {
    log.Printf("🎧 MOCK: Transcrevendo áudio da URL: %s", audioURL)
    return "Mock transcription result", nil
}

// MockRedisClient implementa RedisClientInterface para testes locais
type MockRedisClient struct {
    conversations map[string][]redis.ChatMessage
}

func NewMockRedisClient() *MockRedisClient {
    return &MockRedisClient{
        conversations: make(map[string][]redis.ChatMessage),
    }
}

func (m *MockRedisClient) AddUserMessage(userID, message, messageUUID string) error {
    log.Printf("💾 MOCK: Adicionando mensagem do usuário para %s: %s", userID, message)
    if m.conversations[userID] == nil {
        m.conversations[userID] = []redis.ChatMessage{}
    }
    m.conversations[userID] = append(m.conversations[userID], redis.ChatMessage{
        Role:    "user",
        Content: message,
    })
    return nil
}

func (m *MockRedisClient) AddBotMessage(userID, message string) error {
    log.Printf("💾 MOCK: Adicionando mensagem do bot para %s: %s", userID, message)
    if m.conversations[userID] == nil {
        m.conversations[userID] = []redis.ChatMessage{}
    }
    m.conversations[userID] = append(m.conversations[userID], redis.ChatMessage{
        Role:    "assistant",
        Content: message,
    })
    return nil
}

func (m *MockRedisClient) GetChatHistory(userID string) ([]redis.ChatMessage, error) {
    log.Printf("📜 MOCK: Obtendo histórico de chat para %s", userID)
    if history, exists := m.conversations[userID]; exists {
        return history, nil
    }
    return []redis.ChatMessage{}, nil
}

func (m *MockRedisClient) ClearChatHistory(userID string) error {
    log.Printf("🗑️ MOCK: Limpando histórico de chat para %s", userID)
    delete(m.conversations, userID)
    return nil
}

func (m *MockRedisClient) GetAllConversationSummaries() ([]redis.ConversationSummary, error) {
    log.Printf("📊 MOCK: Obtendo sumários de todas as conversas")
    return []redis.ConversationSummary{}, nil
}

func (m *MockRedisClient) GetChatHistoryPaginated(userID string, page, pageSize int) (redis.PaginatedMessages, error) {
    log.Printf("📄 MOCK: Obtendo histórico paginado para %s (página %d, tamanho %d)", userID, page, pageSize)
    history, _ := m.GetChatHistory(userID)
    return redis.PaginatedMessages{
        Messages: history,
        Page:     page,
    }, nil
}

// MockOpenAIStreamingClient implementa OpenAIClientInterface para testes locais
type MockOpenAIStreamingClient struct {
    realClient   *openai.Client  // Mudança para ponteiro
    lastResponse string
}

func NewMockOpenAIStreamingClient(realClient *openai.Client) *MockOpenAIStreamingClient {
    return &MockOpenAIStreamingClient{
        realClient: realClient,
    }
}

func (m *MockOpenAIStreamingClient) ProcessChatStreamingWithTools(
    ctx context.Context,
    userID string,
    userName string,
    chatHistory []redis.ChatMessage,
    vonageClient VonageClientInterface,
    redisClient RedisClientInterface,
    elevenLabsClient ElevenLabsClientInterface,
    toNumber string,
) error {
    log.Printf("🤖 MOCK: Processando chat com ferramentas para %s", userID)
    
    // Usa o cliente real do OpenAI para gerar a resposta
    response, err := m.realClient.ProcessChatWithTools(ctx, userID, userName, chatHistory)
    if err != nil {
        return err
    }
    
    // Armazena a resposta para recuperação posterior
    m.lastResponse = response
    
    // Adiciona a resposta ao Redis
    if redisClient != nil {
        redisClient.AddBotMessage(userID, response)
    }
    
    log.Printf("✅ MOCK: Resposta gerada: %s", response)
    return nil
}

func (m *MockOpenAIStreamingClient) ProcessChatWithTools(ctx context.Context, userID, userName string, chatHistory []redis.ChatMessage) (string, error) {
    return m.realClient.ProcessChatWithTools(ctx, userID, userName, chatHistory)
}

func (m *MockOpenAIStreamingClient) GetLastResponse() string {
    return m.lastResponse
}