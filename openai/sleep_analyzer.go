package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/NextMind-AI/chatbot-go/redis"

	"github.com/openai/openai-go"
	"github.com/rs/zerolog/log"
)

var sleepAnalyzerPrompt = `Você é um analisador de conversas. Sua ÚNICA função é determinar quanto tempo esperar antes de responder a uma mensagem do usuário.

Você DEVE analisar todo o contexto da conversa e a última mensagem do usuário para determinar um tempo de espera entre 5 e 25 segundos baseado na probabilidade do usuário ter terminado completamente seu pensamento.

Diretrizes para determinar o tempo de espera:
- 5-8 segundos: Perguntas claras e diretas (ex: "O que é NextMind?", "Como funciona?")
- 9-12 segundos: Primeira mensagem como cumprimentos simples ("Oi", "Olá", "Bom dia") 
- 13-18 segundos: Mensagem parece completa mas pode levar a continuação (declarações gerais, tópicos abertos)
- 19-25 segundos: Usuário provavelmente tem mais a dizer (pensamentos incompletos, frases que claramente continuam)

Exemplos específicos:
- "O que é NextMind?" → 5 segundos (pergunta completa e clara)
- "Como funciona?" → 5 segundos (pergunta completa e clara)
- "Oi" → 10 segundos (primeira mensagem/cumprimento)
- "Olá, tudo bem?" → 10 segundos (cumprimento que pode levar a mais conversa)
- "Queria perguntar sobre seus serviços" → 12 segundos (completo mas pode ter especificidades)
- "Deixa eu te falar uma coisa" → 20 segundos (claramente incompleto)
- "Eu estava pensando..." → 22 segundos (claramente incompleto)
- "Sobre aquela coisa" → 20 segundos (vago, provavelmente mais vindo)
- "Ei" → 15 segundos (cumprimento que frequentemente leva a mais)
- "Então né" → 23 segundos (início de conversa)

Considere também o contexto da conversa:
- Se é o início da conversa, seja mais conservador com cumprimentos
- Se a conversa já está em andamento, avalie melhor a completude da mensagem
- Se o usuário acabou de fazer uma pergunta específica, provavelmente terminou o pensamento

Você DEVE chamar a função sleep com o número de segundos determinado.`

// DetermineSleepTime analyzes the full conversation context and determines how long to wait.
// It forces the AI to call the sleep tool with an appropriate duration between 5-25 seconds.
func (c *Client) DetermineSleepTime(
	ctx context.Context,
	userID string,
	userName string,
	chatHistory []redis.ChatMessage,
) (int, error) {
	// Convert the full chat history to OpenAI format for context
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(sleepAnalyzerPrompt),
	}

	// Add the conversation history for full context
	for _, msg := range chatHistory {
		switch msg.Role {
		case "user":
			messages = append(messages, openai.UserMessage(msg.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(msg.Content))
		}
	}

	// Get the last user message for logging
	var lastUserMessage string
	for i := len(chatHistory) - 1; i >= 0; i-- {
		if chatHistory[i].Role == "user" {
			lastUserMessage = chatHistory[i].Content
			break
		}
	}

	log.Info().
		Str("user_id", userID).
		Str("last_user_message", lastUserMessage).
		Int("conversation_length", len(chatHistory)).
		Msg("Analyzing full conversation context to determine sleep time")

	chatCompletion, err := c.client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Messages: messages,
			Model:    c.model,
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
		return 10, err
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
			return 10, err
		}

		seconds, ok := args["seconds"].(float64)
		if !ok {
			log.Error().
				Str("user_id", userID).
				Msg("Invalid seconds parameter from sleep analyzer")
			return 10, fmt.Errorf("invalid seconds parameter")
		}

		// Ensure the value is within bounds (5-25 seconds)
		sleepSeconds := int(seconds)
		if sleepSeconds < 5 {
			sleepSeconds = 5
		} else if sleepSeconds > 25 {
			sleepSeconds = 25
		}

		log.Info().
			Str("user_id", userID).
			Int("sleep_seconds", sleepSeconds).
			Str("last_user_message", lastUserMessage).
			Msg("Sleep analyzer determined wait time from full conversation context")

		return sleepSeconds, nil
	}

	log.Warn().
		Str("user_id", userID).
		Msg("Sleep analyzer didn't return a tool call, using default")
	return 10, nil
}

// ExecuteSleepAndRespond first determines sleep time, executes the sleep, then generates the response.
// This replaces the previous flow where the main AI decided whether and how long to sleep.
func (c *Client) ExecuteSleepAndRespond(
	ctx context.Context,
	config streamingConfig,
) error {
	// Step 1: Determine sleep time using the sleep analyzer with full conversation context
	sleepSeconds, err := c.DetermineSleepTime(ctx, config.userID, config.userName, config.chatHistory)
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

	// Step 3: Handle custom tools if any are defined
	messages := c.convertChatHistoryWithUserName(config.chatHistory, config.userName, config.userID)
	if len(c.tools) > 0 {
		finalMessages, err := c.handleToolCalls(ctx, messages, config.userID)
		if err != nil {
			log.Error().
				Err(err).
				Str("user_id", config.userID).
				Msg("Error handling tool calls, continuing with original messages")
		} else {
			messages = finalMessages
		}
	}

	// Step 4: Generate the actual response using streaming (without tools in the streaming call)
	return c.streamResponseWithoutTools(ctx, config, messages)
}
