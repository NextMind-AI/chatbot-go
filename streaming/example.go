package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

func (p *StreamingJSONParser) AddChunk(chunk string) {
	p.buffer.WriteString(chunk)
	p.parseNewMessages()
}

func (p *StreamingJSONParser) parseNewMessages() {
	content := p.buffer.String()

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
			p.printMessage(msg, p.msgCount)
		}

		p.lastParsedPos = endIdx + 1
		searchContent = content[p.lastParsedPos:]
	}
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

func (p *StreamingJSONParser) printMessage(msg Message, count int) {
	typeEmoji := map[string]string{
		"text":  "📝",
		"audio": "🎵",
	}

	emoji := typeEmoji[msg.Type]
	if emoji == "" {
		emoji = "💬"
	}

	fmt.Printf("%s %s #%d:\n%s\n", emoji, cases.Title(language.Und).String(msg.Type), count, msg.Content)
	fmt.Println("--------")
}

func main() {
	client := openai.NewClient()
	ctx := context.Background()

	question := "Generate at least 3 AI assistant messages about explaining artificial intelligence. Mix different message types: some should be 'text' messages with explanations, and some should be 'audio' messages describing what would be spoken aloud."

	print("🚀 Question: ")
	println(question)
	println()

	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "message_list",
		Description: openai.String("A list of AI assistant messages with different types (text or audio)"),
		Schema:      MessageListResponseSchema,
		Strict:      openai.Bool(true),
	}

	stream := client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(question),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{JSONSchema: schemaParam},
		},
		Model: openai.ChatModelGPT4_1Mini,
	})

	parser := NewStreamingJSONParser()
	var fullContent string

	for stream.Next() {
		evt := stream.Current()
		if len(evt.Choices) > 0 {
			content := evt.Choices[0].Delta.Content
			fullContent += content
			parser.AddChunk(content)
		}
	}

	if err := stream.Err(); err != nil {
		panic(err.Error())
	}

	var messageList MessageList
	err := json.Unmarshal([]byte(fullContent), &messageList)
	if err != nil {
		fmt.Printf("❌ Error parsing JSON: %v\n", err)
		return
	}
}
