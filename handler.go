package main

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
)

type Profile struct {
	Name string `json:"name"`
}

type Audio struct {
	URL string `json:"url"`
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
	Audio         *Audio  `json:"audio,omitempty"`
}

func inboundMessageHandler(c fiber.Ctx) error {
	log.Info().Msg("Received inbound message request")

	var message InboundMessage
	if err := c.Bind().JSON(&message); err != nil {
		log.Error().Err(err).Msg("Error parsing JSON")
		return c.Status(fiber.StatusBadRequest).SendString("Error parsing JSON")
	}

	log.Info().
		Str("message_uuid", message.MessageUUID).
		Str("message_type", message.MessageType).
		Str("from", message.From).
		Str("text", message.Text).
		Bool("has_audio", message.Audio != nil).
		Msg("Processing inbound message")

	go processMessage(message)

	return c.SendStatus(fiber.StatusOK)
}
