package main

import (
	"encoding/json"
	"fmt"
	"io"
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

func inboundMessage(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var message InboundMessage
	if err := json.Unmarshal(body, &message); err != nil {
		http.Error(w, "Error parsing JSON", http.StatusBadRequest)
		return
	}

	fmt.Printf("Received message: %+v\n", message)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}

func main() {
	http.HandleFunc("POST /webhooks/inbound-message", inboundMessage)

	fmt.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
