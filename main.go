package main

import (
	"chatbot/config"
	"chatbot/vonage"
	"fmt"
	"log"
	"net/http"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type Profile struct {
	Name string `json:"name"`
}

type InboundMessage struct {
	Channel       string  `json:"channel"`
	ContextStatus string  `json:"context_status"`
	From          string  `json:"from"`
	MessageType   string  `json:"message_type"`
	MessageUUID   string  `json:"message_uuid"`
	Profile       Profile `json:"profile"`
	Text          string  `json:"text"`
	Timestamp     string  `json:"timestamp"`
	To            string  `json:"to"`
}

var VonageClient vonage.Client
var OpenAIClient openai.Client

func main() {
	var appConfig = config.Load()

	vonageConfig := &vonage.Config{
		VonageJWT:                 appConfig.VonageJWT,
		GeospecificMessagesAPIURL: appConfig.GeospecificMessagesAPIURL,
		MessagesAPIURL:            appConfig.MessagesAPIURL,
	}
	VonageClient = vonage.NewClient(vonageConfig)

	OpenAIClient = openai.NewClient(
		option.WithAPIKey(appConfig.OpenAIKey),
	)

	http.HandleFunc("POST /webhooks/inbound-message", inboundMessage)
	http.HandleFunc("POST /webhooks/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	fmt.Printf("Server starting on :%s\n", appConfig.Port)
	log.Fatal(http.ListenAndServe(":"+appConfig.Port, nil))
}
