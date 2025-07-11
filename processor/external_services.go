package processor

import (
	"context"

	"github.com/NextMind-AI/chatbot-go/redis"
)

func (mp *MessageProcessor) markMessageAsRead(messageUUID string) error {
	return mp.vonageClient.MarkMessageAsRead(messageUUID)
}

func (mp *MessageProcessor) processWithAI(ctx context.Context, userID string, userName string, chatHistory []redis.ChatMessage) error {
	return mp.openaiClient.ProcessChatStreamingWithTools(
		ctx,
		userID,
		userName,
		chatHistory,
		&mp.vonageClient,
		&mp.redisClient,
		&mp.elevenLabsClient,
		userID,
	)
}
