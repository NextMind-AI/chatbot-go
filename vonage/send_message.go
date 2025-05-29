package vonage

import (
	"github.com/rs/zerolog/log"
)

func (c *Client) SendWhatsAppTextMessage(toNumber, senderID, text string) (*MessageResponse, error) {
	log.Debug().
		Str("to", toNumber).
		Str("from", senderID).
		Str("text", text).
		Msg("Preparing to send WhatsApp message")

	message := c.createWhatsAppMessage(toNumber, senderID, text, nil)
	return c.sendMessageRequest("POST", c.config.MessagesAPIURL, message)
}

func (c *Client) SendWhatsAppReplyMessage(toNumber, senderID, text, messageUUID string) (*MessageResponse, error) {
	log.Debug().
		Str("to", toNumber).
		Str("from", senderID).
		Str("text", text).
		Str("message_uuid", messageUUID).
		Msg("Preparing to send WhatsApp reply message")

	context := &Context{MessageUUID: messageUUID}
	message := c.createWhatsAppMessage(toNumber, senderID, text, context)
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
