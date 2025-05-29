package main

import (
	"context"
	"log"

	"github.com/openai/openai-go"
)

func processMessage(message InboundMessage) {
	log.Printf("Processing message with UUID: %s\n", message.MessageUUID)

	VonageClient.MarkMessageAsRead(message.MessageUUID)

	log.Println("Using OpenAI client")

	chatCompletion, err := OpenAIClient.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(message.Text),
		},
		Model: openai.ChatModelGPT4_1Mini,
	})

	if err != nil {
		log.Printf("Error creating chat completion: %v\n", err)
		return
	}

	log.Printf("Received chat completion: %s\n", chatCompletion.Choices[0].Message.Content)
	VonageClient.SendWhatsAppTextMessage(message.From, "5563936180023", chatCompletion.Choices[0].Message.Content)
	log.Println("Sent WhatsApp message")
}
