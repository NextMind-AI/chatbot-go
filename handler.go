package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func inboundMessage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received inbound message request")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var message InboundMessage
	if err := json.Unmarshal(body, &message); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		http.Error(w, "Error parsing JSON", http.StatusBadRequest)
		return
	}

	fmt.Printf("Parsed inbound message: %+v\n", message)
	processMessage(message)
	w.WriteHeader(http.StatusOK)
}

func processMessage(message InboundMessage) {
	fmt.Printf("Processing message with UUID: %s\n", message.MessageUUID)
	MarkMessageAsRead(message.MessageUUID)

	client := openai.NewClient(
		option.WithAPIKey(AppConfig.OpenAIKey),
	)
	fmt.Println("Created OpenAI client")

	chatCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(message.Text),
		},
		Model: openai.ChatModelGPT4_1Mini,
	})

	if err != nil {
		fmt.Printf("Error creating chat completion: %v\n", err)
		return
	}

	fmt.Printf("Received chat completion: %s\n", chatCompletion.Choices[0].Message.Content)
	SendWhatsAppTextMessage(message.From, "5563936180023", chatCompletion.Choices[0].Message.Content)
	fmt.Println("Sent WhatsApp message")
}
