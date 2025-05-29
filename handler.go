package main

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
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
	log.Info().Msg("Received inbound message request")

	var message InboundMessage
	if err := c.Bind().JSON(&message); err != nil {
		log.Error().Err(err).Msg("Error parsing JSON")
		return c.Status(fiber.StatusBadRequest).SendString("Error parsing JSON")
	}

	log.Debug().
		Str("message_uuid", message.MessageUUID).
		Str("from", message.From).
		Str("text", message.Text).
		Msg("Parsed inbound message")

	processMessage(message)

	return c.SendStatus(fiber.StatusOK)
}
