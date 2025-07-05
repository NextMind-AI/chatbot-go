package processor

type InboundMessage struct {
	Channel       string  `json:"channel"`
	ContextStatus string  `json:"context_status"`
	From          string  `json:"from"`
	MessageType   string  `json:"message_type"`
	MessageUUID   string  `json:"message_uuid"`
	Profile       Profile `json:"profile"`
	Text          string  `json:"text"`
	Timestamp     string  `json:"timestamp"`
	To            string  `json:"to"`
	Audio         *Audio  `json:"audio,omitempty"`
}

type Profile struct {
	Name string `json:"name"`
}

type Audio struct {
	URL string `json:"url"`
}

type ProcessedMessage struct {
	Text string
	UUID string
}

// LocalTestMessage representa uma mensagem simplificada para teste local
type LocalTestMessage struct {
	Text   string `json:"text"`
	UserID string `json:"user_id,omitempty"`
}

// LocalTestResponse representa a resposta do teste local
type LocalTestResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

// ConvertToInboundMessage converte uma mensagem de teste local para InboundMessage
func (ltm LocalTestMessage) ConvertToInboundMessage() InboundMessage {
	userID := ltm.UserID
	if userID == "" {
		userID = "test-user-123"
	}

	return InboundMessage{
		MessageUUID:   "test-message-" + userID,
		From:          userID,
		Text:          ltm.Text,
		MessageType:   "text",
		Channel:       "whatsapp",
		ContextStatus: "none",
		To:            "chatbot",
		Timestamp:     "",
		Profile: Profile{
			Name: "Test User",
		},
	}
}
