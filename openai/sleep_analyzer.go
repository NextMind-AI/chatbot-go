package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/NextMind-AI/chatbot-go/redis"

	"github.com/openai/openai-go"
	"github.com/rs/zerolog/log"
)

var sleepAnalyzerPrompt = `Você é um analisador de conversas. Sua ÚNICA função é determinar quantos segundos esperar (entre 5 e 15) antes de responder.

Regras resumidas:
- 5–7s: perguntas claras e diretas (ex.: "O que é NextMind?", "Como funciona?")
- 7s: cumprimentos simples / primeiro contato ("Oi", "Olá", "Bom dia")
- 11–13s: mensagens completas que podem continuar (declarações, "Queria perguntar sobre seus serviços")
- 14–15s: mensagens claramente incompletas ou que sugerem continuação ("Deixa eu te falar...", "Eu estava pensando...")

Contexto:
- Início da conversa → favoreça valores maiores na faixa.
- Conversa em andamento ou pergunta específica → favoreça valores menores.

Exemplos:
"O que é NextMind?" → 5
"Oi" → 9
"Queria perguntar sobre seus serviços" → 12
"Eu estava pensando..." → 15

Ação obrigatória: chame sleep(<segundos>) com o inteiro escolhido.`

// quickSleepHeuristic tenta determinar rapidamente o tempo de espera sem chamar o modelo.
// Retorna (seconds, true) se conseguiu decidir de forma confiável, ou (0, false) se não aplicável.
func quickSleepHeuristic(msg string, conversationLength int) (int, bool) {
	last := strings.ToLower(strings.TrimSpace(msg))
	if last == "" {
		return 0, false
	}

	// Regex para detectar horas e datas simples
	timeOrDayRe := regexp.MustCompile(`\b\d{1,2}h\b|\b\d{1,2}:\d{2}\b|\b\d{1,2}[\/\-]\d{1,2}\b|\bhoje\b|\bamanhã\b|\bamanha\b|\bsegunda\b|\bterça\b|\bterca\b|\bquarta\b|\bquinta\b|\bsexta\b|\bsábado\b|\bsabado\b|\bdomingo\b`)
	hasTimeOrDay := timeOrDayRe.MatchString(last)

	// Heurísticas simples
	// 1) Ellipsis / terminações que sugerem continuação
	if strings.Contains(last, "...") || strings.HasSuffix(last, ",") || strings.HasSuffix(last, "-") {
		return 15, true
	}

	// 2) Pergunta explícita curta (usa '?') -> resposta rápida
	if strings.Contains(last, "?") {
		// se for longa, ainda pode ser específica mas pedimos resposta rápida
		if len(last) <= 120 {
			return 5, true
		}
		// mensagem muito longa com '?', usa modelo (não decidir aqui)
		return 0, false
	}

	// 3) Cumprimentos simples / primeiras mensagens
	greetings := []string{"oi", "olá", "ola", "bom dia", "boa tarde", "boa noite", "e aí", "e ai", "fala"}
	for _, g := range greetings {
		if last == g || strings.HasPrefix(last, g+" ") {
			// se pouca conversa, seja mais conservador
			if conversationLength < 4 {
				return 10, true
			}
			return 8, true
		}
	}

	// 4) Frases que começam com padrões de continuação / incompletas
	incompleteStarters := []string{"deixa eu", "vou te", "vou te falar", "deixa eu te", "estou pensando", "tô pensando", "então", "entao", "sobre aquela", "queria dizer", "deixa", "espere", "espera", "eu estava"}
	for _, s := range incompleteStarters {
		if strings.HasPrefix(last, s) || strings.Contains(last, s) {
			return 15, true
		}
	}

	// 5) Intenções vagas que costumam necessitar de follow-up
	vagueIntents := []string{"quero", "gostaria", "preciso", "tenho uma dúvida", "perguntar", "queria"}
	for _, v := range vagueIntents {
		if strings.Contains(last, v) {
			return 12, true
		}
	}

	// 6) Se contém horário/data explícita -> provavelmente terminou
	if hasTimeOrDay {
		return 6, true
	}

	// 7) Mensagens muito curtas (1-2 palavras) e não saudação -> médio conservador
	words := strings.Fields(last)
	if len(words) <= 2 {
		return 10, true
	}

	// 8) Mensagens muito longas -> provavelmente completas
	if len(last) > 120 || len(words) > 20 {
		return 12, true
	}

	// Não conseguiu decidir de forma confiável
	return 0, false
}

// DetermineSleepTime analyzes the full conversation context and determines how long to wait.
// It will first try a fast local heuristic; if that doesn't decide, it falls back to the model flow.
func (c *Client) DetermineSleepTime(
	ctx context.Context,
	userID string,
	userName string,
	chatHistory []redis.ChatMessage,
) (int, error) {
	// Get last user message for heuristic and logging
	var lastUserMessage string
	for i := len(chatHistory) - 1; i >= 0; i-- {
		if chatHistory[i].Role == "user" {
			lastUserMessage = chatHistory[i].Content
			break
		}
	}

	// Try fast heuristic first
	if secs, ok := quickSleepHeuristic(lastUserMessage, len(chatHistory)); ok {
		// ensure bounds 5-15
		if secs < 5 {
			secs = 5
		} else if secs > 15 {
			secs = 15
		}
		log.Info().
			Str("user_id", userID).
			Str("last_user_message", lastUserMessage).
			Int("sleep_seconds", secs).
			Msg("Sleep heurística rápida aplicada (sem chamar modelo)")
		return secs, nil
	}

	// Heurística não decidiu; usar o fluxo original com o modelo
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

	log.Info().
		Str("user_id", userID).
		Str("last_user_message", lastUserMessage).
		Int("conversation_length", len(chatHistory)).
		Msg("Nenhum match na heurística, analisando com o modelo para determinar sleep")

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
			// support also integer-coded JSON numbers
			if sInt, ok2 := args["seconds"].(int); ok2 {
				seconds = float64(sInt)
			} else {
				log.Error().
					Str("user_id", userID).
					Msg("Invalid seconds parameter from sleep analyzer")
				return 10, fmt.Errorf("invalid seconds parameter")
			}
		}

		// Ensure the value is within bounds (5-15 seconds)
		sleepSeconds := int(seconds)
		if sleepSeconds < 5 {
			sleepSeconds = 5
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

// ExecuteSleepAndRespond remains unchanged and will call DetermineSleepTime as before.
func (c *Client) ExecuteSleepAndRespond(
	ctx context.Context,
	config streamingConfig,
) error {
	// Step 1: Determine sleep time using the sleep analyzer with full conversation context (or heuristic)
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
	log.Info().
		Str("user_id", config.userID).
		Msg("Starting response generation with streaming")

	err = c.streamResponseWithoutTools(ctx, config, messages)
	if err != nil {
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
