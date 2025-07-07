package server

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// WebSocketManager manages WebSocket connections
type WebSocketManager struct {
	clients    map[*websocket.Conn]bool
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	broadcast  chan []byte
	mu         sync.RWMutex
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		clients:    make(map[*websocket.Conn]bool),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		broadcast:  make(chan []byte),
	}
}

// Start starts the WebSocket manager
func (manager *WebSocketManager) Start() {
	go manager.run()
}

// run handles WebSocket connections
func (manager *WebSocketManager) run() {
	for {
		select {
		case conn := <-manager.register:
			manager.mu.Lock()
			manager.clients[conn] = true
			manager.mu.Unlock()
			log.Info().Msg("WebSocket client connected")

		case conn := <-manager.unregister:
			manager.mu.Lock()
			if _, ok := manager.clients[conn]; ok {
				delete(manager.clients, conn)
				conn.Close()
			}
			manager.mu.Unlock()
			log.Info().Msg("WebSocket client disconnected")

		case message := <-manager.broadcast:
			manager.mu.RLock()
			for conn := range manager.clients {
				if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
					log.Error().Err(err).Msg("Error writing to WebSocket")
					conn.Close()
					delete(manager.clients, conn)
				}
			}
			manager.mu.RUnlock()
		}
	}
}

// BroadcastMessage sends a message to all connected clients
func (manager *WebSocketManager) BroadcastMessage(message []byte) {
	select {
	case manager.broadcast <- message:
	default:
		log.Warn().Msg("WebSocket broadcast channel full")
	}
}

// Register registers a new WebSocket connection
func (manager *WebSocketManager) Register(conn *websocket.Conn) {
	manager.register <- conn
}

// Unregister unregisters a WebSocket connection
func (manager *WebSocketManager) Unregister(conn *websocket.Conn) {
	manager.unregister <- conn
}

// GetClientCount returns the number of connected clients
func (manager *WebSocketManager) GetClientCount() int {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	return len(manager.clients)
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for now, you can restrict this as needed
		return true
	},
}
