package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"regexp"
	"strings"

	"github.com/NextMind-AI/chatbot-go/redis"

	"github.com/openai/openai-go"
	"github.com/rs/zerolog/log"
)

var sleepAnalyzerPrompt = `Você é um analisador de conversas. Sua ÚNICA função é determinar quanto tempo esperar antes de responder a uma mensagem do usuário.

Você DEVE analisar todo o contexto da conversa e a última mensagem do usuário para determinar um tempo de espera entre 3 e 15 segundos baseado na probabilidade do usuário ter terminado completamente seu pensamento.

Diretrizes para determinar o tempo de espera:
- 3-6 segundos: Perguntas claras e diretas (ex: "O que é NextMind?", "Como funciona?")
- 7-9 segundos: Primeira mensagem tipo cumprimento simples ("Oi", "Olá", "Bom dia")
- 10-12 segundos: Mensagem parece completa mas pode levar a continuação (declarações gerais, tópicos abertos)
- 13-15 segundos: Usuário provavelmente tem mais a dizer (pensamentos incompletos, frases que claramente continuam)

Exemplos:
- "O que é NextMind?" → 3 segundos
- "Olá, tudo bem?" → 8 segundos
- "Queria perguntar sobre seus serviços" → 9 segundos
- "Deixa eu te falar uma coisa" → 14 segundos
- "Eu estava pensando..." → 15 segundos
- "Então né" → 14 segundos

Considere também o contexto da conversa:
- Se for início da conversa, prefira valores ligeiramente maiores para cumprimentos.
- Se a conversa já está em andamento, avalie melhor a completude da mensagem.

Você DEVE chamar a função sleep com o número de segundos determinado.`

// decideSleepHeuristic tenta retornar imediatamente um tempo (3-15).
// Retorna 0 se não conseguiu decidir (nesse caso, chama o analyzer).
func decideSleepHeuristic(lastMessage string, recentMessages []redis.ChatMessage) int {
	if lastMessage == "" {
		return 5 // fallback barato
	}

	msg := strings.TrimSpace(lastMessage)
	msgLower := strings.ToLower(msg)

	// Pergunta clara? termina com ?
	if strings.HasSuffix(msg, "?") {
		return 3
	}

	// Cumprimentos curtos
	if msgLower == "oi" || msgLower == "olá" || msgLower == "ola" || msgLower == "bom dia" || msgLower == "boa tarde" || msgLower == "boa noite" {
		// se conversa já em andamento, ser mais curto
		if len(recentMessages) > 3 {
			return 5
		}
		return 8
	}

	// Frases que normalmente indicam continuação (reticências, vírgula final, "..." ou "etc", "deixa eu")
	if strings.HasSuffix(msg, "...") || strings.HasSuffix(msg, "…") {
		return 15
	}
	if strings.HasSuffix(msg, ",") || strings.HasSuffix(msg, " e") || strings.HasSuffix(msgLower, "então") || strings.HasSuffix(msgLower, "tá") {
		return 10
	}

	// Padrões vagos / início de pensamento
	if regexp.MustCompile(`(?i)\b(queria|vou|quero|estou pensando|deixa eu|sobre aquilo)\b`).MatchString(msg) {
		return 7
	}

	// Pequena mensagem sem pontuação (provável cumprimento curto)
	if len(msg) <= 3 {
		return 6
	}

	// Se não bateu em heurística, devolve 0 para chamar o analisador
	return 0
}

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

		// ... após parse do tool call e extração de 'seconds' como float64
		sleepSeconds := int(seconds)

		// Garantir intervalo 3..15
		if sleepSeconds < 3 {
			sleepSeconds = 3
		} else if sleepSeconds > 15 {
			sleepSeconds = 15
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

func (c *Client) ExecuteSleepAndRespond(
	ctx context.Context,
	config streamingConfig,
) error {
	// --- 0) obter última mensagem do usuário (se houver) ---
	var lastUserMessage string
	for i := len(config.chatHistory) - 1; i >= 0; i-- {
		if config.chatHistory[i].Role == "user" {
			lastUserMessage = config.chatHistory[i].Content
			break
		}
	}

	// --- 1) heurística rápida (evita chamar o modelo quando possível) ---
	sleepSeconds := decideSleepHeuristic(lastUserMessage, config.chatHistory)
	if sleepSeconds > 0 {
		log.Info().
			Str("user_id", config.userID).
			Int("sleep_seconds", sleepSeconds).
			Str("reason", "heuristic").
			Str("last_user_message", lastUserMessage).
			Msg("Determined sleep time via heuristic")
	} else {
		// --- 2) chamar o analyzer com timeout (fallback se algo der errado) ---
		// usamos um timeout curto para não bloquear demais (ex.: 3s).
		analyzerTimeout := 3 * time.Second
		analyzerCtx, cancel := context.WithTimeout(ctx, analyzerTimeout)
		defer cancel()

		var err error
		sleepSeconds, err = c.DetermineSleepTime(analyzerCtx, config.userID, config.userName, config.chatHistory)
		if err != nil {
			log.Warn().
				Err(err).
				Str("user_id", config.userID).
				Str("last_user_message", lastUserMessage).
				Msgf("DetermineSleepTime failed (timeout %s), falling back to default", analyzerTimeout)
			// default curto para manter reatividade
			sleepSeconds = 3
		} else {
			log.Info().
				Str("user_id", config.userID).
				Int("sleep_seconds", sleepSeconds).
				Str("reason", "analyzer").
				Str("last_user_message", lastUserMessage).
				Msg("Determined sleep time via analyzer")
		}
	}

	// --- 3) garantir bounds finais: 3..15 segundos ---
	if sleepSeconds < 3 {
		sleepSeconds = 3
	} else if sleepSeconds > 15 {
		sleepSeconds = 15
	}

	// --- 4) executar o sleep (respeitando cancelamento do contexto) ---
	if sleepSeconds > 0 {
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

	// --- 5) Handle custom tools if any are defined (mantém seu fluxo atual) ---
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

	// --- 6) Generate the actual response using streaming (without tools) ---
	log.Info().
		Str("user_id", config.userID).
		Msg("Starting response generation with streaming")

	if err := c.streamResponseWithoutTools(ctx, config, messages); err != nil {
		log.Error().
			Err(err).
			Str("user_id", config.userID).
			Msg("Error in streaming response generation")
		return err
	}

	log.Info().
		Str("user_id", config.userID).
		Msg("Completed message processing")

	return nil
}
