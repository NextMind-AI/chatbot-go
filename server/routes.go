package server

func (s *Server) setupRoutes() {
	s.app.Post("/webhooks/inbound-message", s.inboundMessageHandler)

	// CRM API endpoints
	s.app.Get("/crm/conversations", s.crmConversationsHandler)
	s.app.Get("/crm/conversations/:userId", s.crmConversationMessagesHandler)
}
