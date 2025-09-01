package server

import (
	"github.com/NextMind-AI/chatbot-go/processor"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
)

func (s *Server) inboundMessageHandler(c fiber.Ctx) error {
	log.Info().Msg("Received inbound message request")

	var message processor.InboundMessage
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

	go s.messageProcessor.ProcessMessage(message)

	return c.SendStatus(fiber.StatusOK)
}
