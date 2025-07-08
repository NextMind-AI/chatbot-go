package server

import (
	"chatbot/processor"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/rs/zerolog/log"
)

type Server struct {
	app              *fiber.App
	messageProcessor *processor.MessageProcessor
	wsManager        *WebSocketManager
}

func New(messageProcessor *processor.MessageProcessor) *Server {
	app := fiber.New(fiber.Config{
		// Configure error handling to always return JSON
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			message := "Internal server error"

			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
				message = e.Message
			}

			log.Error().
				Err(err).
				Int("status", code).
				Str("method", c.Method()).
				Str("path", c.Path()).
				Msg("Request error")

			return c.Status(code).JSON(fiber.Map{
				"error":  message,
				"status": code,
				"path":   c.Path(),
				"method": c.Method(),
			})
		},
	})

	// Add recovery middleware to handle panics
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	// Add logger middleware
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} ${latency}\n",
	}))

	// Configure CORS middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"https://nextmind.pro",
			"http://localhost:3000",
			"http://localhost:8080",
			"https://chatbot-go-production-d129.up.railway.app",
			"https://*.railway.app",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		AllowCredentials: true,
	}))

	// Create WebSocket manager
	wsManager := NewWebSocketManager()
	wsManager.Start()

	server := &Server{
		app:              app,
		messageProcessor: messageProcessor,
		wsManager:        wsManager,
	}

	// Configure WebSocket callback in message processor
	messageProcessor.SetWebSocketCallback(server.BroadcastMessage)

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
