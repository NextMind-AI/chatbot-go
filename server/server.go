package server

import (
	"chatbot/processor"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/rs/zerolog/log"
)

type Server struct {
	app              *fiber.App
	messageProcessor *processor.MessageProcessor
}

func New(messageProcessor *processor.MessageProcessor) *Server {
	app := fiber.New()

	// Configure CORS middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://nextmind.pro", "http://localhost:3000", "http://localhost:8080"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		AllowCredentials: true,
	}))

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
