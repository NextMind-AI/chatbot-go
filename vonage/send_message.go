package vonage

func (c *Client) SendWhatsAppTextMessage(toNumber, text string) (*MessageResponse, error) {
	message := c.createWhatsAppMessage(toNumber, c.config.PhoneNumberID, text, nil)
	return c.sendMessageRequest("POST", c.config.MessagesAPIURL, message)
}

func (c *Client) SendWhatsAppReplyMessage(toNumber, text, messageUUID string) (*MessageResponse, error) {
	context := &Context{MessageUUID: messageUUID}
	message := c.createWhatsAppMessage(toNumber, c.config.PhoneNumberID, text, context)
	return c.sendMessageRequest("POST", c.config.MessagesAPIURL, message)
}

func (c *Client) createWhatsAppMessage(toNumber, senderID, text string, context *Context) WhatsAppMessage {
	return WhatsAppMessage{
		To:          toNumber,
		From:        senderID,
		Channel:     "whatsapp",
		MessageType: "text",
		Text:        text,
		Context:     context,
	}
}
