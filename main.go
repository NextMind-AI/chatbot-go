package main

import (
	"fmt"
	"log"
	"net/http"
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

func main() {
	LoadConfig()

	http.HandleFunc("POST /webhooks/inbound-message", inboundMessage)
	http.HandleFunc("POST /webhooks/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	fmt.Printf("Server starting on :%s\n", AppConfig.Port)
	log.Fatal(http.ListenAndServe(":"+AppConfig.Port, nil))
}
