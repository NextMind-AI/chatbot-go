package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openai/openai-go"
	"github.com/rs/zerolog/log"
)

var sleepAnalyzerPrompt = `You are a conversation analyzer. Your ONLY job is to determine how long to wait before responding to a user message.

You MUST analyze the user's last message and determine a wait time between 8 and 25 seconds based on how likely it is that the user has finished expressing their complete thought.

Guidelines for determining wait time:
- 8-12 seconds: User has clearly finished their thought (complete questions, full statements, clear endings)
- 13-18 seconds: Message seems complete but might lead to follow-up (general statements, open topics)
- 19-25 seconds: User likely has more to say (incomplete thoughts, trailing phrases, conversation starters)

Examples:
- "What is NextMind?" → 8 seconds (complete question)
- "How does it work?" → 8 seconds (complete question)
- "I wanted to ask about your services" → 10 seconds (complete but might have specifics)
- "I was thinking..." → 22 seconds (clearly incomplete)
- "About that thing" → 20 seconds (vague, likely more coming)
- "Hey" → 18 seconds (greeting that often leads to more)
- "Okay so" → 23 seconds (conversation starter)

You MUST call the sleep function with your determined number of seconds.`

// DetermineSleepTime analyzes the user's message and determines how long to wait.
// It forces the AI to call the sleep tool with an appropriate duration between 8-25 seconds.
func (c *Client) DetermineSleepTime(
	ctx context.Context,
	userID string,
	userMessage string,
) (int, error) {
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(sleepAnalyzerPrompt),
		openai.UserMessage(userMessage),
	}

	log.Info().
		Str("user_id", userID).
		Str("user_message", userMessage).
		Msg("Analyzing message to determine sleep time")

	chatCompletion, err := c.client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Messages: messages,
			Model:    openai.ChatModelGPT4_1Mini,
			Tools:    []openai.ChatCompletionToolParam{sleepTool},
			ToolChoice: openai.ChatCompletionToolChoiceOptionUnionParam{
				OfChatCompletionNamedToolChoice: &openai.ChatCompletionNamedToolChoiceParam{
					Function: openai.ChatCompletionNamedToolChoiceFunctionParam{
						Name: sleepTool.Function.Name,
					},
				},
			},
		},
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error calling sleep analyzer")
		return 15, err
	}

	// Extract sleep duration from the tool call
	if len(chatCompletion.Choices) > 0 && len(chatCompletion.Choices[0].Message.ToolCalls) > 0 {
		toolCall := chatCompletion.Choices[0].Message.ToolCalls[0]

		var args map[string]any
		err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
		if err != nil {
			log.Error().
				Err(err).
				Str("user_id", userID).
				Msg("Error parsing sleep analyzer response")
			return 15, err
		}

		seconds, ok := args["seconds"].(float64)
		if !ok {
			log.Error().
				Str("user_id", userID).
				Msg("Invalid seconds parameter from sleep analyzer")
			return 15, fmt.Errorf("invalid seconds parameter")
		}

		// Ensure the value is within bounds
		sleepSeconds := int(seconds)
		if sleepSeconds < 8 {
			sleepSeconds = 8
		} else if sleepSeconds > 25 {
			sleepSeconds = 25
		}

		log.Info().
			Str("user_id", userID).
			Int("sleep_seconds", sleepSeconds).
			Msg("Sleep analyzer determined wait time")

		return sleepSeconds, nil
	}

	log.Warn().
		Str("user_id", userID).
		Msg("Sleep analyzer didn't return a tool call, using default")
	return 15, nil
}

// ExecuteSleepAndRespond first determines sleep time, executes the sleep, then generates the response.
// This replaces the previous flow where the main AI decided whether and how long to sleep.
func (c *Client) ExecuteSleepAndRespond(
	ctx context.Context,
	config streamingConfig,
) error {
	// Get the last user message
	var lastUserMessage string
	for i := len(config.chatHistory) - 1; i >= 0; i-- {
		if config.chatHistory[i].Role == "user" {
			lastUserMessage = config.chatHistory[i].Content
			break
		}
	}

	// Step 1: Determine sleep time using the sleep analyzer
	sleepSeconds, err := c.DetermineSleepTime(ctx, config.userID, lastUserMessage)
	if err != nil {
		log.Warn().
			Err(err).
			Str("user_id", config.userID).
			Msg("Error determining sleep time, continuing without sleep")
	} else {
		// Step 2: Execute the sleep
		log.Info().
			Str("user_id", config.userID).
			Int("seconds", sleepSeconds).
			Msg("Executing sleep before generating response")

		sleepDuration := time.Duration(sleepSeconds) * time.Second
		select {
		case <-time.After(sleepDuration):
			log.Info().
				Str("user_id", config.userID).
				Msg("Sleep completed, generating response")
		case <-ctx.Done():
			log.Info().
				Str("user_id", config.userID).
				Msg("Sleep cancelled due to context cancellation")
			return ctx.Err()
		}
	}

	// Step 3: Generate the actual response using the main chatbot (without sleep tool)
	messages := convertChatHistory(config.chatHistory)
	return c.streamResponse(ctx, config, messages)
}
