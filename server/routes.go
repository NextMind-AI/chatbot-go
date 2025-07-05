package server

import (
	"github.com/NextMind-AI/chatbot-go/processor"
	"github.com/gofiber/fiber/v3"
)

func (s *Server) setupRoutes() {
	s.app.Post("/webhooks/inbound-message", s.inboundMessageHandler)
	    // Nova rota para teste local
    s.app.Post("/test/chat", s.handleLocalTest)
    
    // Rota para limpar histórico em modo de teste
    s.app.Delete("/test/chat/:user_id", s.handleClearHistory)
	// CRM API endpoints
	s.app.Get("/crm/conversations", s.crmConversationsHandler)
	s.app.Get("/crm/conversations/:userId", s.crmConversationMessagesHandler)
}

// handleLocalTest processa mensagens de teste local
func (s *Server) handleLocalTest(c fiber.Ctx) error {
    var testMessage processor.LocalTestMessage
    
    if err := c.Bind().Body(&testMessage); err != nil {
        return c.Status(400).JSON(processor.LocalTestResponse{
            Error: "Invalid request body: " + err.Error(),
        })
    }
    
    if testMessage.Text == "" {
        return c.Status(400).JSON(processor.LocalTestResponse{
            Error: "Text field is required",
        })
    }
    
    // Converte para InboundMessage usando o método público
    inboundMessage := testMessage.ConvertToInboundMessage()
    
    // Processa a mensagem
    response, err := s.messageProcessor.ProcessLocalTestMessage(inboundMessage)
    if err != nil {
        return c.Status(500).JSON(processor.LocalTestResponse{
            Error: "Processing error: " + err.Error(),
        })
    }
    
    return c.JSON(processor.LocalTestResponse{
        Response: response,
    })
}

// handleClearHistory limpa o histórico de chat de um usuário em modo de teste
func (s *Server) handleClearHistory(c fiber.Ctx) error {
    userID := c.Params("user_id")
    if userID == "" {
        return c.Status(400).JSON(map[string]string{
            "error": "user_id is required",
        })
    }
    
    err := s.messageProcessor.ClearTestUserHistory(userID)
    if err != nil {
        return c.Status(500).JSON(map[string]string{
            "error": "Failed to clear history: " + err.Error(),
        })
    }
    
    return c.JSON(map[string]string{
        "message": "History cleared successfully",
    })
}