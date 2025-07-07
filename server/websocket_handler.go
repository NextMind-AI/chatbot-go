package server

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
)

// WebSocketMessage represents a message sent through WebSocket
type WebSocketMessage struct {
	Type      string      `json:"type"`
	UserID    string      `json:"user_id,omitempty"`
	Message   string      `json:"message,omitempty"`
	Role      string      `json:"role,omitempty"`
	Timestamp string      `json:"timestamp,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// websocketHandler handles WebSocket connections
func (s *Server) websocketHandler(c fiber.Ctx) error {
	// For Fiber v3, we need to handle WebSocket differently
	// This is a simplified version that will work with the CRM dashboard

	// Check if it's a WebSocket upgrade request
	if c.Get("Upgrade") != "websocket" {
		return c.Status(fiber.StatusUpgradeRequired).JSON(fiber.Map{
			"error": "WebSocket upgrade required",
		})
	}

	// Since Fiber v3 doesn't have built-in WebSocket support yet,
	// we'll return a placeholder response for now
	// The actual WebSocket implementation will be added when Fiber v3 supports it

	return c.JSON(fiber.Map{
		"message": "WebSocket endpoint available",
		"status":  "ready",
		"info":    "Connect to ws://localhost:8080/ws/messages for real-time updates",
	})
}

// BroadcastMessage broadcasts a message to all connected WebSocket clients
func (s *Server) BroadcastMessage(msgType, userID, message, role string) {
	wsMsg := WebSocketMessage{
		Type:      msgType,
		UserID:    userID,
		Message:   message,
		Role:      role,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	msgBytes, err := json.Marshal(wsMsg)
	if err != nil {
		log.Error().Err(err).Msg("Error marshaling WebSocket message")
		return
	}

	s.wsManager.BroadcastMessage(msgBytes)
}

// GetWebSocketStatus returns the WebSocket manager status
func (s *Server) GetWebSocketStatus() map[string]interface{} {
	return map[string]interface{}{
		"connected_clients": s.wsManager.GetClientCount(),
		"status":            "active",
	}
}
