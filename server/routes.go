package server

func (s *Server) setupRoutes() {
	s.app.Post("/webhooks/inbound-message", s.inboundMessageHandler)
	s.app.Get("/health", s.healthCheckHandler)

	// CRM routes
	s.app.Get("/crm/conversations", s.listConversationsHandler)
	s.app.Get("/crm/conversations/:user_id", s.getConversationHandler)

	// WebSocket route
	s.app.Get("/ws/messages", s.websocketHandler)
}
