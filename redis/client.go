package redis

import (
	"context"
	"encoding/json"
	"fmt"
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
