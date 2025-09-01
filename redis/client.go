package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type Client struct {
	rdb *redis.Client
	ctx context.Context
}

type ChatMessage struct {
	Role        string    `json:"role"`
	Content     string    `json:"content"`
	Timestamp   time.Time `json:"timestamp"`
	MessageUUID string    `json:"message_uuid,omitempty"`
}

// ConversationSummary represents a conversation summary
type ConversationSummary struct {
	UserID             string
	LastMessageTime    time.Time
	LastMessagePreview string
	MessageCount       int
}

// PaginatedMessages represents paginated chat messages
type PaginatedMessages struct {
	Messages      []ChatMessage
	TotalMessages int
	Page          int
	TotalPages    int
}

func NewClient(addr, password string, db int) Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	client := Client{
		rdb: rdb,
		ctx: context.Background(),
	}

	if err := client.Ping(); err != nil {
		log.Fatal().Err(err).
			Str("addr", addr).
			Int("db", db).
			Msg("Redis connection failed")
	} else {
		log.Info().
			Str("addr", addr).
			Int("db", db).
			Msg("Redis connected successfully")
	}

	return client
}

func (c *Client) Ping() error {
	return c.rdb.Ping(c.ctx).Err()
}

func (c *Client) AddUserMessage(userID, message, messageUUID string) error {
	chatMsg := ChatMessage{
		Role:        "user",
		Content:     message,
		Timestamp:   time.Now(),
		MessageUUID: messageUUID,
	}

	return c.addMessage(userID, chatMsg)
}

func (c *Client) AddBotMessage(userID, message string) error {
	chatMsg := ChatMessage{
		Role:      "assistant",
		Content:   message,
		Timestamp: time.Now(),
	}

	return c.addMessage(userID, chatMsg)
}

func (c *Client) addMessage(userID string, message ChatMessage) error {
	key := fmt.Sprintf("chat_history:%s", userID)

	messageJSON, err := json.Marshal(message)
	if err != nil {
		return err
	}

	_, err = c.rdb.RPush(c.ctx, key, messageJSON).Result()
	if err != nil {
		return err
	}

	c.rdb.Expire(c.ctx, key, 24*time.Hour)

	return nil
}

func (c *Client) GetChatHistory(userID string) ([]ChatMessage, error) {
	key := fmt.Sprintf("chat_history:%s", userID)

	messages, err := c.rdb.LRange(c.ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	var chatHistory []ChatMessage
	for _, message := range messages {
		var msg ChatMessage
		err := json.Unmarshal([]byte(message), &msg)
		if err != nil {
			continue
		}
		chatHistory = append(chatHistory, msg)
	}

	return chatHistory, nil
}

func (c *Client) ClearChatHistory(userID string) error {
	key := fmt.Sprintf("chat_history:%s", userID)
	return c.rdb.Del(c.ctx, key).Err()
}

// GetAllConversationSummaries returns summaries of all conversations
func (c *Client) GetAllConversationSummaries() ([]ConversationSummary, error) {
	keys, err := c.rdb.Keys(c.ctx, "chat_history:*").Result()
	if err != nil {
		return nil, err
	}

	var summaries []ConversationSummary
	for _, key := range keys {
		userID := strings.TrimPrefix(key, "chat_history:")

		// Get message count
		count, err := c.rdb.LLen(c.ctx, key).Result()
		if err != nil {
			log.Error().Err(err).Str("key", key).Msg("Error getting message count")
			continue
		}

		if count == 0 {
			continue
		}

		// Get last message
		lastMessageJSON, err := c.rdb.LIndex(c.ctx, key, -1).Result()
		if err != nil {
			log.Error().Err(err).Str("key", key).Msg("Error getting last message")
			continue
		}

		var lastMessage ChatMessage
		if err := json.Unmarshal([]byte(lastMessageJSON), &lastMessage); err != nil {
			log.Error().Err(err).Str("key", key).Msg("Error unmarshaling last message")
			continue
		}

		// Create preview (first 100 characters)
		preview := lastMessage.Content
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}

		summaries = append(summaries, ConversationSummary{
			UserID:             userID,
			LastMessageTime:    lastMessage.Timestamp,
			LastMessagePreview: preview,
			MessageCount:       int(count),
		})
	}

	return summaries, nil
}

// GetChatHistoryPaginated returns paginated chat history for a user
func (c *Client) GetChatHistoryPaginated(userID string, page, pageSize int) (PaginatedMessages, error) {
	key := fmt.Sprintf("chat_history:%s", userID)

	// Get total count
	totalCount, err := c.rdb.LLen(c.ctx, key).Result()
	if err != nil {
		return PaginatedMessages{}, err
	}

	if totalCount == 0 {
		return PaginatedMessages{
			Messages:      []ChatMessage{},
			TotalMessages: 0,
			Page:          page,
			TotalPages:    0,
		}, nil
	}

	// Calculate pagination
	totalPages := (int(totalCount) + pageSize - 1) / pageSize
	if page > totalPages {
		page = totalPages
	}
	if page < 1 {
		page = 1
	}

	// Calculate Redis list indices (0-based)
	start := (page - 1) * pageSize
	end := start + pageSize - 1
	if end >= int(totalCount) {
		end = int(totalCount) - 1
	}

	// Get messages for the page
	messageStrings, err := c.rdb.LRange(c.ctx, key, int64(start), int64(end)).Result()
	if err != nil {
		return PaginatedMessages{}, err
	}

	var messages []ChatMessage
	for _, messageStr := range messageStrings {
		var msg ChatMessage
		if err := json.Unmarshal([]byte(messageStr), &msg); err != nil {
			log.Error().Err(err).Str("user_id", userID).Msg("Error unmarshaling message")
			continue
		}
		messages = append(messages, msg)
	}

	return PaginatedMessages{
		Messages:      messages,
		TotalMessages: int(totalCount),
		Page:          page,
		TotalPages:    totalPages,
	}, nil
}
