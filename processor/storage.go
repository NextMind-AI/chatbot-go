package processor

import (
	"chatbot/redis"
)

func (mp *MessageProcessor) storeUserMessage(userID string, processedMsg *ProcessedMessage) error {
	return mp.redisClient.AddUserMessage(userID, processedMsg.Text, processedMsg.UUID)
}

func (mp *MessageProcessor) getChatHistory(userID string) ([]redis.ChatMessage, error) {
	return mp.redisClient.GetChatHistory(userID)
}
