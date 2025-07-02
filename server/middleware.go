package server

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

// setupMiddleware configures middleware for the server
func (s *Server) setupMiddleware() {
	// Add recovery middleware
	s.app.Use(recover.New())

	// Add logger middleware
	s.app.Use(logger.New())

	// Add CORS middleware for CRM API access
	s.app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
	}))

	// Add JSON content type for CRM endpoints
	s.app.Use("/crm/*", func(c fiber.Ctx) error {
		c.Set("Content-Type", "application/json")
		return c.Next()
	})
}
