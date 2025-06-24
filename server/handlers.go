package server

import (
	"chatbot/openai"
	"chatbot/processor"
	"encoding/json"
	"net/http"
	"time"

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

func (s *Server) healthCheckHandler(c fiber.Ctx) error {
	status := s.messageProcessor.GetProcessorStatus()
	return c.JSON(status)
}

func (s *Server) handleCacheStatus(w http.ResponseWriter, r *http.Request) {
	stats := openai.GetCacheStatistics()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"cache_statistics": stats,
		"timestamp":        time.Now().Unix(),
	})
}
