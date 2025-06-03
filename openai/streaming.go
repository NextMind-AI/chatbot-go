package openai

import (
	"chatbot/redis"
	"chatbot/vonage"
	"context"
	"encoding/json"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go"
	"github.com/rs/zerolog/log"
)

type Message struct {
	Content string `json:"content" jsonschema_description:"The content of the message"`
	Type    string `json:"type" jsonschema:"enum=text,enum=audio" jsonschema_description:"The type of the message: text or audio"`
}

type MessageList struct {
	Messages []Message `json:"messages" jsonschema_description:"A list of messages"`
}

func GenerateSchema[T any]() any {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

var MessageListResponseSchema = GenerateSchema[MessageList]()

type StreamingJSONParser struct {
	buffer        strings.Builder
	lastParsedPos int
	msgCount      int
	foundMessages bool
}

func NewStreamingJSONParser() *StreamingJSONParser {
	return &StreamingJSONParser{}
}

func (p *StreamingJSONParser) AddChunk(chunk string) []Message {
	p.buffer.WriteString(chunk)
	return p.parseNewMessages()
}

func (p *StreamingJSONParser) parseNewMessages() []Message {
	content := p.buffer.String()
	var parsedMessages []Message

	if !p.foundMessages && strings.Contains(content, `"messages":[`) {
		p.foundMessages = true
	}

	searchContent := content[p.lastParsedPos:]

	for {
		startIdx := strings.Index(searchContent, `{"content":`)
		if startIdx == -1 {
			startIdx = strings.Index(searchContent, `{"type":`)
			if startIdx == -1 {
				break
			}
		}

		fullStartIdx := p.lastParsedPos + startIdx

		endIdx := p.findMessageEnd(content, fullStartIdx)
		if endIdx == -1 {
			break
		}

		messageJSON := content[fullStartIdx : endIdx+1]

		var msg Message
		if err := json.Unmarshal([]byte(messageJSON), &msg); err == nil {
			p.msgCount++
			parsedMessages = append(parsedMessages, msg)
		}

		p.lastParsedPos = endIdx + 1
		searchContent = content[p.lastParsedPos:]
	}

	return parsedMessages
}

func (p *StreamingJSONParser) findMessageEnd(content string, startIdx int) int {
	braceCount := 0
	inString := false
	escaped := false

	for i := startIdx; i < len(content); i++ {
		char := content[i]

		if escaped {
			escaped = false
			continue
		}

		if char == '\\' {
			escaped = true
			continue
		}

		if char == '"' {
			inString = !inString
			continue
		}

		if !inString {
			if char == '{' {
				braceCount++
			} else if char == '}' {
				braceCount--
				if braceCount == 0 {
					return i
				}
			}
		}
	}

	return -1
}

func (c *Client) ProcessChatStreaming(
	ctx context.Context,
	userID string,
	chatHistory []redis.ChatMessage,
	vonageClient *vonage.Client,
	redisClient *redis.Client,
	toNumber string,
	senderID string,
) error {
	messages := convertChatHistory(chatHistory)

	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "message_list",
		Description: openai.String("A list of messages to send to the user"),
		Schema:      MessageListResponseSchema,
		Strict:      openai.Bool(true),
	}

	stream := c.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Messages: messages,
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{JSONSchema: schemaParam},
		},
		Model: openai.ChatModelGPT4_1Mini,
	})

	parser := NewStreamingJSONParser()
	var fullContent strings.Builder
	sentMessages := make(map[int]bool)

	for stream.Next() {
		evt := stream.Current()
		if len(evt.Choices) > 0 {
			content := evt.Choices[0].Delta.Content
			fullContent.WriteString(content)

			newMessages := parser.AddChunk(content)

			for i, msg := range newMessages {
				messageIndex := parser.msgCount - len(newMessages) + i
				if !sentMessages[messageIndex] {
					sentMessages[messageIndex] = true

					log.Info().
						Str("user_id", userID).
						Int("message_index", messageIndex).
						Str("content", msg.Content).
						Msg("Sending streamed message via Vonage")

					_, err := vonageClient.SendWhatsAppTextMessage(toNumber, senderID, msg.Content)
					if err != nil {
						log.Error().
							Err(err).
							Str("user_id", userID).
							Str("to", toNumber).
							Msg("Error sending streamed WhatsApp message")
					}
				}
			}
		}
	}

	if err := stream.Err(); err != nil {
		return err
	}

	var messageList MessageList
	if err := json.Unmarshal([]byte(fullContent.String()), &messageList); err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Str("content", fullContent.String()).
			Msg("Error parsing final JSON response")
		return err
	}

	allMessagesContent := []string{}
	for _, msg := range messageList.Messages {
		allMessagesContent = append(allMessagesContent, msg.Content)
	}
	fullResponse := strings.Join(allMessagesContent, "\n\n")

	if err := redisClient.AddBotMessage(userID, fullResponse); err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error storing bot message in Redis")
	}

	return nil
}

func (c *Client) ProcessChatStreamingWithTools(
	ctx context.Context,
	userID string,
	chatHistory []redis.ChatMessage,
	vonageClient *vonage.Client,
	redisClient *redis.Client,
	toNumber string,
	senderID string,
) error {
	messages := convertChatHistory(chatHistory)

	chatCompletion, err := c.createChatCompletionWithTools(ctx, messages)
	if err != nil {
		return err
	}

	toolCalls := chatCompletion.Choices[0].Message.ToolCalls

	if len(toolCalls) > 0 {
		messages = append(messages, chatCompletion.Choices[0].Message.ToParam())

		messages, err = c.handleToolCalls(ctx, userID, messages, toolCalls)
		if err != nil {
			return err
		}

		schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
			Name:        "message_list",
			Description: openai.String("A list of messages to send to the user"),
			Schema:      MessageListResponseSchema,
			Strict:      openai.Bool(true),
		}

		stream := c.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
			Messages: messages,
			ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
				OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{JSONSchema: schemaParam},
			},
			Model: openai.ChatModelGPT4_1Mini,
		})

		parser := NewStreamingJSONParser()
		var fullContent strings.Builder
		sentMessages := make(map[int]bool)

		for stream.Next() {
			evt := stream.Current()
			if len(evt.Choices) > 0 {
				content := evt.Choices[0].Delta.Content
				fullContent.WriteString(content)

				newMessages := parser.AddChunk(content)

				for i, msg := range newMessages {
					messageIndex := parser.msgCount - len(newMessages) + i
					if !sentMessages[messageIndex] {
						sentMessages[messageIndex] = true

						log.Info().
							Str("user_id", userID).
							Int("message_index", messageIndex).
							Str("content", msg.Content).
							Msg("Sending streamed message via Vonage")

						_, err := vonageClient.SendWhatsAppTextMessage(toNumber, senderID, msg.Content)
						if err != nil {
							log.Error().
								Err(err).
								Str("user_id", userID).
								Str("to", toNumber).
								Msg("Error sending streamed WhatsApp message")
						}
					}
				}
			}
		}

		if err := stream.Err(); err != nil {
			return err
		}

		var messageList MessageList
		if err := json.Unmarshal([]byte(fullContent.String()), &messageList); err != nil {
			log.Error().
				Err(err).
				Str("user_id", userID).
				Str("content", fullContent.String()).
				Msg("Error parsing final JSON response")
			return err
		}

		allMessagesContent := []string{}
		for _, msg := range messageList.Messages {
			allMessagesContent = append(allMessagesContent, msg.Content)
		}
		fullResponse := strings.Join(allMessagesContent, "\n\n")

		if err := redisClient.AddBotMessage(userID, fullResponse); err != nil {
			log.Error().
				Err(err).
				Str("user_id", userID).
				Msg("Error storing bot message in Redis")
		}

		return nil
	}

	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "message_list",
		Description: openai.String("A list of messages to send to the user"),
		Schema:      MessageListResponseSchema,
		Strict:      openai.Bool(true),
	}

	stream := c.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Messages: messages,
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{JSONSchema: schemaParam},
		},
		Model: openai.ChatModelGPT4_1Mini,
	})

	parser := NewStreamingJSONParser()
	var fullContent strings.Builder
	sentMessages := make(map[int]bool)

	for stream.Next() {
		evt := stream.Current()
		if len(evt.Choices) > 0 {
			content := evt.Choices[0].Delta.Content
			fullContent.WriteString(content)

			newMessages := parser.AddChunk(content)

			for i, msg := range newMessages {
				messageIndex := parser.msgCount - len(newMessages) + i
				if !sentMessages[messageIndex] {
					sentMessages[messageIndex] = true

					log.Info().
						Str("user_id", userID).
						Int("message_index", messageIndex).
						Str("content", msg.Content).
						Msg("Sending streamed message via Vonage")

					_, err := vonageClient.SendWhatsAppTextMessage(toNumber, senderID, msg.Content)
					if err != nil {
						log.Error().
							Err(err).
							Str("user_id", userID).
							Str("to", toNumber).
							Msg("Error sending streamed WhatsApp message")
					}
				}
			}
		}
	}

	if err := stream.Err(); err != nil {
		return err
	}

	var messageList MessageList
	if err := json.Unmarshal([]byte(fullContent.String()), &messageList); err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Str("content", fullContent.String()).
			Msg("Error parsing final JSON response")
		return err
	}

	allMessagesContent := []string{}
	for _, msg := range messageList.Messages {
		allMessagesContent = append(allMessagesContent, msg.Content)
	}
	fullResponse := strings.Join(allMessagesContent, "\n\n")

	if err := redisClient.AddBotMessage(userID, fullResponse); err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error storing bot message in Redis")
	}

	return nil
}
