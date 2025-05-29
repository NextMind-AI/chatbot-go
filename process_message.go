package main

import (
	"chatbot/redis"
	"context"
	"log"

	"github.com/openai/openai-go"
)

func processMessage(message InboundMessage) {
	log.Printf("Processing message with UUID: %s\n", message.MessageUUID)

	VonageClient.MarkMessageAsRead(message.MessageUUID)

	userID := message.From

	if err := RedisClient.AddUserMessage(userID, message.Text); err != nil {
		log.Printf("Error storing user message in Redis: %v", err)
	}

	chatHistory, err := RedisClient.GetChatHistory(userID)
	if err != nil {
		log.Printf("Error retrieving chat history from Redis: %v", err)
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

	log.Printf("Using OpenAI client with %d historical messages", len(messages))

	chatCompletion, err := OpenAIClient.Chat.Completions.New(
		context.TODO(),
		openai.ChatCompletionNewParams{
			Messages: messages,
			Model:    openai.ChatModelGPT4_1Mini,
		},
	)

	if err != nil {
		log.Printf("Error creating chat completion: %v\n", err)
		return
	}

	botResponse := chatCompletion.Choices[0].Message.Content
	log.Printf("Received chat completion: %s\n", botResponse)

	if err := RedisClient.AddBotMessage(userID, botResponse); err != nil {
		log.Printf("Error storing bot message in Redis: %v", err)
	}

	VonageClient.SendWhatsAppTextMessage(message.From, "5563936180023", botResponse)
	log.Println("Sent WhatsApp message")
}
