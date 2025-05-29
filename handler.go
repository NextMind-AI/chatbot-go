package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
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

func inboundMessage(c fiber.Ctx) error {
	log.Println("Received inbound message request")

	var message InboundMessage
	if err := c.Bind().JSON(&message); err != nil {
		log.Printf("Error parsing JSON: %v\n", err)
		return c.Status(fiber.StatusBadRequest).SendString("Error parsing JSON")
	}

	log.Printf("Parsed inbound message: %+v\n", message)
	processMessage(message)

	return c.SendStatus(fiber.StatusOK)
}
