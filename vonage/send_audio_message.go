package vonage

func (c *Client) SendWhatsAppAudioMessage(toNumber, audioURL string) (*MessageResponse, error) {
	message := c.createWhatsAppAudioMessage(toNumber, c.config.PhoneNumberID, audioURL, nil)
	return c.sendMessageRequest("POST", c.config.MessagesAPIURL, message)
}

func (c *Client) SendWhatsAppReplyAudioMessage(toNumber, audioURL, messageUUID string) (*MessageResponse, error) {
	context := &Context{MessageUUID: messageUUID}
	message := c.createWhatsAppAudioMessage(toNumber, c.config.PhoneNumberID, audioURL, context)
	return c.sendMessageRequest("POST", c.config.MessagesAPIURL, message)
}

func (c *Client) createWhatsAppAudioMessage(toNumber, senderID, audioURL string, context *Context) WhatsAppMessage {
	return WhatsAppMessage{
		To:          toNumber,
		From:        senderID,
		Channel:     "whatsapp",
		MessageType: "audio",
		Audio: &Audio{
			URL: audioURL,
		},
		Context: context,
	}
}
