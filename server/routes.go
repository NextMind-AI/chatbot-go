package server

func (s *Server) setupRoutes() {
	s.app.Post("/webhooks/inbound-message", s.inboundMessageHandler)
	s.app.Get("/health", s.healthCheckHandler)
}
