package openai

import (
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

// handleToolCalls processes all tool calls from the AI's response.
// It iterates through the tool calls, executes them, and appends the results to the message history.
// Currently supports the sleep tool, but can be extended to handle other tools.
func (c *Client) handleToolCalls(
	ctx context.Context,
	userID string,
	messages []openai.ChatCompletionMessageParamUnion,
	toolCalls []openai.ChatCompletionMessageToolCall,
) ([]openai.ChatCompletionMessageParamUnion, error) {
	for _, toolCall := range toolCalls {
		switch toolCall.Function.Name {
		case "sleep":
			toolMessage, success := c.processSleepTool(ctx, userID, toolCall)
			if !success {
				continue
			}
			messages = append(messages, toolMessage)
		}
	}
	return messages, nil
}
