package openai

import (
	"chatbot/vonage"
	"context"
	"encoding/json"
	"time"

	"github.com/openai/openai-go"
	"github.com/rs/zerolog/log"
)

// sleepTool defines the sleep tool that allows the AI to pause conversation for a specified duration.
// This tool can be used when the AI needs to simulate waiting or processing time.
var sleepTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "sleep",
		Description: openai.String("Wait for a specified number of seconds before continuing the conversation"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"seconds": map[string]string{
					"type":        "integer",
					"description": "Number of seconds to wait",
				},
			},
			"required": []string{"seconds"},
		},
	},
}

// sendWhatsAppMessageTool defines the tool for sending WhatsApp messages to a specific number.
// This tool is called when the AI has collected all necessary information and the conversation is complete.
var sendWhatsAppMessageTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "send_whatsapp_message",
		Description: openai.String("Send a WhatsApp template message when the conversation is complete and all necessary data has been collected"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"to_number": map[string]string{
					"type":        "string",
					"description": "The WhatsApp number to send the message to (format: 5511976847785)",
				},
				"customer_name": map[string]string{
					"type":        "string",
					"description": "The customer/supplier name (parameter 1)",
				},
				"product_info": map[string]string{
					"type":        "string",
					"description": "Complete product information including type, brand, model, year and size (parameter 2)",
				},
				"item_conditions": map[string]string{
					"type":        "string",
					"description": "Detailed condition of the items including repairs, micro holes, fabric rating, etc (parameter 3)",
				},
				"item_location": map[string]string{
					"type":        "string",
					"description": "Location of the items/supplier (parameter 4)",
				},
				"availability": map[string]string{
					"type":        "string",
					"description": "Availability for inspection and pickup (parameter 5)",
				},
			},
			"required": []string{"to_number", "customer_name", "product_info", "item_conditions", "item_location", "availability"},
		},
	},
}

// processSleepTool processes a sleep tool call from the AI.
// It parses the arguments, executes the sleep operation, and returns the result.
// Returns a tool message and a success flag indicating whether the operation completed successfully.
func (c *Client) processSleepTool(
	ctx context.Context,
	userID string,
	toolCall openai.ChatCompletionMessageToolCall,
) (openai.ChatCompletionMessageParamUnion, bool) {
	var args map[string]any
	err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error parsing sleep function arguments")
		return openai.ToolMessage("", ""), false
	}

	seconds, ok := args["seconds"].(float64)
	if !ok {
		log.Error().
			Str("user_id", userID).
			Msg("Invalid seconds parameter for sleep function")
		return openai.ToolMessage("", ""), false
	}

	log.Info().
		Str("user_id", userID).
		Float64("seconds", seconds).
		Msg("Sleeping before continuing conversation")

	sleepDuration := time.Duration(seconds) * time.Second
	select {
	case <-time.After(sleepDuration):
	case <-ctx.Done():
		log.Info().
			Str("user_id", userID).
			Msg("Sleep cancelled due to context cancellation")
		return openai.ToolMessage("", ""), false
	}

	return openai.ToolMessage("Sleep completed", toolCall.ID), true
}

// processSendWhatsAppMessageTool processes a WhatsApp message tool call from the AI.
// It parses the arguments, sends a WhatsApp template message, and returns the result.
func (c *Client) processSendWhatsAppMessageTool(
	ctx context.Context,
	userID string,
	toolCall openai.ChatCompletionMessageToolCall,
	vonageClient *vonage.Client,
) (openai.ChatCompletionMessageParamUnion, bool) {
	var args map[string]any
	err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error parsing send_whatsapp_message function arguments")
		return openai.ToolMessage("Error parsing arguments", toolCall.ID), false
	}

	// Extract all required parameters
	toNumber, ok := args["to_number"].(string)
	if !ok {
		log.Error().
			Str("user_id", userID).
			Msg("Invalid to_number parameter")
		return openai.ToolMessage("Invalid to_number parameter", toolCall.ID), false
	}

	customerName, ok := args["customer_name"].(string)
	if !ok {
		log.Error().
			Str("user_id", userID).
			Msg("Invalid customer_name parameter")
		return openai.ToolMessage("Invalid customer_name parameter", toolCall.ID), false
	}

	productInfo, ok := args["product_info"].(string)
	if !ok {
		log.Error().
			Str("user_id", userID).
			Msg("Invalid product_info parameter")
		return openai.ToolMessage("Invalid product_info parameter", toolCall.ID), false
	}

	itemConditions, ok := args["item_conditions"].(string)
	if !ok {
		log.Error().
			Str("user_id", userID).
			Msg("Invalid item_conditions parameter")
		return openai.ToolMessage("Invalid item_conditions parameter", toolCall.ID), false
	}

	itemLocation, ok := args["item_location"].(string)
	if !ok {
		log.Error().
			Str("user_id", userID).
			Msg("Invalid item_location parameter")
		return openai.ToolMessage("Invalid item_location parameter", toolCall.ID), false
	}

	availability, ok := args["availability"].(string)
	if !ok {
		log.Error().
			Str("user_id", userID).
			Msg("Invalid availability parameter")
		return openai.ToolMessage("Invalid availability parameter", toolCall.ID), false
	}

	log.Info().
		Str("user_id", userID).
		Str("to_number", toNumber).
		Str("customer_name", customerName).
		Str("product_info", productInfo).
		Str("item_conditions", itemConditions).
		Str("item_location", itemLocation).
		Str("availability", availability).
		Msg("Sending WhatsApp template message")

	// Send the WhatsApp template message
	err = vonageClient.SendWhatsAppTemplateMessage(
		toNumber,
		customerName,
		productInfo,
		itemConditions,
		itemLocation,
		availability,
	)

	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error sending WhatsApp template message")
		return openai.ToolMessage("Error sending WhatsApp message", toolCall.ID), false
	}

	successMsg := "WhatsApp message sent successfully to " + toNumber
	return openai.ToolMessage(successMsg, toolCall.ID), true
}

// handleToolCalls processes all tool calls from the AI's response.
// It iterates through the tool calls, executes them, and appends the results to the message history.
// Currently supports the sleep tool, but can be extended to handle other tools.
func (c *Client) handleToolCalls(
	ctx context.Context,
	userID string,
	messages []openai.ChatCompletionMessageParamUnion,
	toolCalls []openai.ChatCompletionMessageToolCall,
	vonageClient *vonage.Client,
) ([]openai.ChatCompletionMessageParamUnion, error) {
	for _, toolCall := range toolCalls {
		switch toolCall.Function.Name {
		case "sleep":
			toolMessage, success := c.processSleepTool(ctx, userID, toolCall)
			if !success {
				continue
			}
			messages = append(messages, toolMessage)
		case "send_whatsapp_message":
			toolMessage, success := c.processSendWhatsAppMessageTool(ctx, userID, toolCall, vonageClient)
			if !success {
				continue
			}
			messages = append(messages, toolMessage)
		}
	}
	return messages, nil
}
