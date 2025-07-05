package server

import (
	"github.com/NextMind-AI/chatbot-go/processor"
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
)

type Server struct {
	app              *fiber.App
	messageProcessor *processor.MessageProcessor
}

func New(messageProcessor *processor.MessageProcessor) *Server {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			return c.Status(500).JSON(map[string]string{
				"error": err.Error(),
			})
		},
	})

	server := &Server{
		app:              app,
		messageProcessor: messageProcessor,
	}

	server.setupRoutes()
	return server
}

func (s *Server) Start(port string) {
	log.Info().Str("port", port).Msg("Starting chatbot server")

	err := s.app.Listen(":"+port, fiber.ListenConfig{
		DisableStartupMessage: true,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
