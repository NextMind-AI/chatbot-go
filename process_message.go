package main

import (
	"chatbot/redis"
	"context"
	"sync"

	"github.com/openai/openai-go"
	"github.com/rs/zerolog/log"
)

type UserExecution struct {
	ctx    context.Context
	cancel context.CancelFunc
}

var (
	userExecutions = make(map[string]*UserExecution)
	executionMutex sync.RWMutex
)

func processMessage(message InboundMessage) {
	log.Info().Str("message_uuid", message.MessageUUID).Msg("Processing message")

	userID := message.From

	executionMutex.Lock()
	if existingExecution, exists := userExecutions[userID]; exists {
		log.Info().Str("user_id", userID).Msg("Cancelling previous execution for user")
		existingExecution.cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	userExecutions[userID] = &UserExecution{
		ctx:    ctx,
		cancel: cancel,
	}
	executionMutex.Unlock()

	defer func() {
		executionMutex.Lock()
		if execution, exists := userExecutions[userID]; exists && execution.ctx == ctx {
			delete(userExecutions, userID)
		}
		executionMutex.Unlock()
	}()

	log.Debug().Str("message_uuid", message.MessageUUID).Msg("Marking message as read")
	err := VonageClient.MarkMessageAsRead(message.MessageUUID)
	if err != nil {
		log.Error().
			Err(err).
			Str("message_uuid", message.MessageUUID).
			Msg("Error marking message as read")
	}

	if err := RedisClient.AddUserMessage(
		userID,
		message.Text,
		message.MessageUUID,
	); err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error storing user message in Redis")
	}

	select {
	case <-ctx.Done():
		log.Info().
			Str("user_id", userID).
			Msg("Message processing cancelled after storing user message")
		return
	default:
	}

	chatHistory, err := RedisClient.GetChatHistory(userID)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error retrieving chat history from Redis")
		chatHistory = []redis.ChatMessage{}
	}

	messages := []openai.ChatCompletionMessageParamUnion{}

	for _, msg := range chatHistory {
		if msg.Role == "user" {
			messages = append(messages, openai.UserMessage(msg.Content))
		} else if msg.Role == "assistant" {
			messages = append(messages, openai.AssistantMessage(msg.Content))
		}
	}

	select {
	case <-ctx.Done():
		log.Info().Str("user_id", userID).Msg("Message processing cancelled before OpenAI call")
		return
	default:
	}

	log.Debug().
		Int("message_count", len(messages)).
		Str("user_id", userID).
		Msg("Using OpenAI client with historical messages")

	chatCompletion, err := OpenAIClient.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Messages: messages,
			Model:    openai.ChatModelGPT4_1Mini,
		},
	)

	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error creating chat completion")
		return
	}

	select {
	case <-ctx.Done():
		log.Info().Str("user_id", userID).Msg("Message processing cancelled after OpenAI call")
		return
	default:
	}

	botResponse := chatCompletion.Choices[0].Message.Content
	log.Debug().
		Str("user_id", userID).
		Str("response", botResponse).
		Msg("Received chat completion")

	if err := RedisClient.AddBotMessage(userID, botResponse); err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error storing bot message in Redis")
	}

	log.Debug().
		Str("to", message.From).
		Str("from", "5563936180023").
		Str("text", botResponse).
		Msg("Sending WhatsApp text message")

	_, err = VonageClient.SendWhatsAppTextMessage(message.From, "5563936180023", botResponse)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Str("to", message.From).
			Msg("Error sending WhatsApp message")
		return
	}

	log.Info().Str("user_id", userID).Msg("Sent WhatsApp message")
}
