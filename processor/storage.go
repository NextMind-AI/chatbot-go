package processor

import (
	"chatbot/redis"
)

func (mp *MessageProcessor) storeUserMessage(userID string, processedMsg *ProcessedMessage) error {
	err := mp.redisClient.AddUserMessage(userID, processedMsg.Text, processedMsg.UUID)
	if err == nil {
		// Notify WebSocket clients about new user message
		mp.notifyWebSocket("new_message", userID, processedMsg.Text, "user")
	}
	return err
}

func (mp *MessageProcessor) getChatHistory(userID string) ([]redis.ChatMessage, error) {
	return mp.redisClient.GetChatHistory(userID)
}

// storeBotMessage stores a bot message and notifies WebSocket clients
func (mp *MessageProcessor) storeBotMessage(userID, message string) error {
	err := mp.redisClient.AddBotMessage(userID, message)
	if err == nil {
		// Notify WebSocket clients about new bot message
		mp.notifyWebSocket("new_message", userID, message, "assistant")
	}
	return err
}
