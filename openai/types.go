package openai

import (
	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go"
)

// Message represents a single message in the chat conversation.
// It can be either a text message or an audio message.
type Message struct {
	// Content is the actual message text
	Content string `json:"content" jsonschema_description:"The content of the message"`
	// Type specifies the message format: "text" or "audio"
	Type string `json:"type" jsonschema:"enum=text,enum=audio" jsonschema_description:"The type of the message: text or audio"`
}

// MessageList represents a collection of messages to be sent to the user.
// This structure is used for JSON schema validation in streaming responses.
type MessageList struct {
	// Messages contains the list of messages to send
	Messages []Message `json:"messages" jsonschema_description:"A list of messages"`
}

// GenerateSchema creates a JSON schema for the given type T.
// It uses reflection to generate a strict schema that disallows additional properties
// and doesn't use references for better compatibility with OpenAI's API.
func GenerateSchema[T any]() any {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

// MessageListResponseSchema is the pre-generated JSON schema for MessageList.
// This schema is used to enforce structured output from OpenAI's API.
var MessageListResponseSchema = GenerateSchema[MessageList]()

// createSchemaParam creates an OpenAI schema parameter for structured output.
// This ensures the AI responds with a properly formatted MessageList.
func createSchemaParam() openai.ResponseFormatJSONSchemaJSONSchemaParam {
	return openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "message_list",
		Description: openai.String("A list of messages to send to the user"),
		Schema:      MessageListResponseSchema,
		Strict:      openai.Bool(true),
	}
}
