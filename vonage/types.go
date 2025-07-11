package vonage

type Config struct {
	VonageJWT                 string
	GeospecificMessagesAPIURL string
	MessagesAPIURL            string
	PhoneNumberID             string
}

type Context struct {
	MessageUUID string `json:"message_uuid"`
}

type WhatsAppMessage struct {
	To          string   `json:"to"`
	From        string   `json:"from"`
	Channel     string   `json:"channel"`
	MessageType string   `json:"message_type"`
	Text        string   `json:"text,omitempty"`
	Audio       *Audio   `json:"audio,omitempty"`
	Context     *Context `json:"context,omitempty"`
}

type Audio struct {
	URL string `json:"url"`
}

type MessageResponse struct {
	MessageUUID string `json:"message_uuid"`
}

type MarkAsReadPayload struct {
	Status string `json:"status"`
}
