package vonage

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

// SendWhatsAppTemplateMessage sends a WhatsApp template message using the Vonage API
func (c *Client) SendWhatsAppTemplateMessage(
	toNumber string,
	customerName string,
	productInfo string,
	itemConditions string,
	itemLocation string,
	availability string,
) error {
	// Create the request body for template message
	requestBody := map[string]interface{}{
		"from":         c.config.PhoneNumberID,
		"to":           toNumber,
		"message_type": "template",
		"channel":      "whatsapp",
		"whatsapp": map[string]interface{}{
			"policy": "deterministic",
			"locale": "pt_BR",
		},
		"template": map[string]interface{}{
			"name": "new_sale_order",
			"parameters": []string{
				customerName,   // {{1}} - Cliente
				productInfo,    // {{2}} - Produto
				itemConditions, // {{3}} - Condições dos itens
				itemLocation,   // {{4}} - Localização dos itens
				availability,   // {{5}} - Disponibilidade
			},
		},
	}

	// Send the request
	response, err := c.sendMessageRequest(
		"POST",
		c.config.MessagesAPIURL,
		requestBody,
	)

	if err != nil {
		log.Error().
			Err(err).
			Str("to", toNumber).
			Interface("request_body", requestBody).
			Msg("Error sending WhatsApp template message")
		return fmt.Errorf("failed to send WhatsApp template message: %w", err)
	}

	log.Info().
		Str("message_uuid", response.MessageUUID).
		Str("to", toNumber).
		Str("template", "new_sale_order").
		Str("customer", customerName).
		Str("product", productInfo).
		Msg("WhatsApp template message sent successfully")

	return nil
}
