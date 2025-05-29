package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/openai/openai-go"
)

func inboundMessage(w http.ResponseWriter, r *http.Request) {
	log.Println("Received inbound message request")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var message InboundMessage
	if err := json.Unmarshal(body, &message); err != nil {
		log.Printf("Error parsing JSON: %v\n", err)
		http.Error(w, "Error parsing JSON", http.StatusBadRequest)
		return
	}

	log.Printf("Parsed inbound message: %+v\n", message)
	processMessage(message)
	w.WriteHeader(http.StatusOK)
}

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
