package processor

import (
	"chatbot/redis"
	"context"
)

func (mp *MessageProcessor) markMessageAsRead(messageUUID string) error {
	return mp.vonageClient.MarkMessageAsRead(messageUUID)
}

func (mp *MessageProcessor) processWithAI(ctx context.Context, userID string, chatHistory []redis.ChatMessage) error {
	return mp.openaiClient.ProcessChatStreamingWithTools(
		ctx,
		userID,
		chatHistory,
		&mp.vonageClient,
		&mp.redisClient,
		&mp.elevenLabsClient,
		userID,
	)
}
