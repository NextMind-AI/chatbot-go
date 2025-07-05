package chatbot

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/NextMind-AI/chatbot-go/aws"
	"github.com/NextMind-AI/chatbot-go/config"
	"github.com/NextMind-AI/chatbot-go/elevenlabs"
	"github.com/NextMind-AI/chatbot-go/execution"
	"github.com/NextMind-AI/chatbot-go/openai"
	"github.com/NextMind-AI/chatbot-go/processor"
	"github.com/NextMind-AI/chatbot-go/redis"
	"github.com/NextMind-AI/chatbot-go/server"
	"github.com/NextMind-AI/chatbot-go/vonage"

	"github.com/rs/zerolog/log"

	openaiapi "github.com/openai/openai-go"
)

// Tool represents a custom tool that can be called by the AI (using the openai package type)
type Tool = openai.Tool

// ToolHandler represents a function that handles a tool call and returns the result
type ToolHandler = openai.ToolHandler

// PromptGenerator is a function that generates the system prompt based on user context
type PromptGenerator = openai.PromptGenerator

// Config holds the configuration for the chatbot
type Config struct {
	PromptGenerator PromptGenerator
	Tools           []Tool
	Model           string // OpenAI model to use
}

// Chatbot represents the main chatbot instance
type Chatbot struct {
	config           Config
	messageProcessor *processor.MessageProcessor
	server           *server.Server
}

// New creates a new chatbot instance with the given configuration
func New(cfg Config) *Chatbot {
	appConfig := config.Load()
	httpClient := http.Client{}

	var vonageClient processor.VonageClientInterface
	var redisClient processor.RedisClientInterface
	var elevenLabsClient processor.ElevenLabsClientInterface
	var openAIClient processor.OpenAIClientInterface

	if appConfig.LocalMode {
		log.Info().Msg("🧪 Starting in LOCAL MODE for testing")

		// Usa mocks em modo local
		vonageClient = &processor.MockVonageClient{}
		redisClient = processor.NewMockRedisClient()
		elevenLabsClient = &processor.MockElevenLabsClient{}

		// Cria o cliente OpenAI real primeiro
		realOpenAIClient := openai.NewClient(
			appConfig.OpenAIKey,
			httpClient,
			cfg.PromptGenerator,
			cfg.Tools,
			cfg.Model,
		)

		// Envolve com o mock para capturar respostas
		openAIClient = processor.NewMockOpenAIStreamingClient(&realOpenAIClient)

	} else {
		log.Info().Msg("🚀 Starting in PRODUCTION MODE")

		// Usa serviços reais em produção
		awsClient := aws.NewClient(appConfig.S3Region, appConfig.S3Bucket)

		// Cria os clientes reais
		vonageClientReal := vonage.NewClient(
			appConfig.VonageJWT,
			appConfig.GeospecificMessagesAPIURL,
			appConfig.MessagesAPIURL,
			appConfig.PhoneNumber,
			httpClient,
		)

		openAIClientReal := openai.NewClient(
			appConfig.OpenAIKey,
			httpClient,
			cfg.PromptGenerator,
			cfg.Tools,
			cfg.Model,
		)

		redisClientReal := redis.NewClient(
			appConfig.RedisAddr,
			appConfig.RedisPassword,
			appConfig.RedisDB,
		)

		elevenLabsClientReal := elevenlabs.NewClient(
			appConfig.ElevenLabsAPIKey,
			httpClient,
			awsClient,
		)

		// Envolve os clientes reais com os wrappers para implementar as interfaces
		vonageClient = processor.NewVonageClientWrapper(&vonageClientReal)
		redisClient = processor.NewRedisClientWrapper(&redisClientReal)
		elevenLabsClient = processor.NewElevenLabsClientWrapper(&elevenLabsClientReal)
		openAIClient = processor.NewOpenAIClientWrapper(
			&openAIClientReal,
			&vonageClientReal,
			&redisClientReal,
			&elevenLabsClientReal,
		)
	}

	executionManager := execution.NewManager()

	messageProcessor := processor.NewMessageProcessor(
		vonageClient,
		redisClient,
		openAIClient,
		elevenLabsClient,
		executionManager,
	)

	srv := server.New(messageProcessor)

	return &Chatbot{
		config:           cfg,
		messageProcessor: messageProcessor,
		server:           srv,
	}
}

// Start starts the chatbot server
func (c *Chatbot) Run() {
	appConfig := config.Load()
	c.server.Start(appConfig.Port)
}
// ToolFunc represents a tool function with parameter metadata
type ToolFunc struct {
	Fn             any
	ParameterNames []string
	ParameterDescs []string
}

// CreateTool creates a tool from a function with automatic type inference
// You can provide just a function, or use WithParams to add parameter names and descriptions
func CreateTool(name, description string, fn any) (Tool, error) {
	var toolFunc ToolFunc

	switch v := fn.(type) {
	case ToolFunc:
		toolFunc = v
	default:
		toolFunc = ToolFunc{Fn: fn}
	}

	fnValue := reflect.ValueOf(toolFunc.Fn)
	fnType := fnValue.Type()

	if fnType.Kind() != reflect.Func {
		return Tool{}, fmt.Errorf("provided value is not a function")
	}

	// Validate function signature
	if fnType.NumIn() < 1 || fnType.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() {
		return Tool{}, fmt.Errorf("function must have context.Context as first parameter")
	}

	if fnType.NumOut() < 1 || fnType.NumOut() > 2 {
		return Tool{}, fmt.Errorf("function must return (string) or (string, error)")
	}

	if fnType.Out(0).Kind() != reflect.String {
		return Tool{}, fmt.Errorf("function must return string as first return value")
	}

	if fnType.NumOut() == 2 && !fnType.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return Tool{}, fmt.Errorf("function second return value must be error")
	}

	// Generate parameters schema from function signature
	parameters := generateParametersSchema(fnType, toolFunc.ParameterNames, toolFunc.ParameterDescs)

	// Create handler that converts map arguments to function parameters
	handler := createHandler(fnValue, fnType, toolFunc.ParameterNames)

	return Tool{
		Definition: openaiapi.ChatCompletionToolParam{
			Function: openaiapi.FunctionDefinitionParam{
				Name:        name,
				Description: openaiapi.String(description),
				Parameters:  parameters,
			},
		},
		Handler: handler,
	}, nil
}

// WithParams wraps a function with parameter metadata
func WithParams(fn any, names []string, descriptions []string) ToolFunc {
	return ToolFunc{
		Fn:             fn,
		ParameterNames: names,
		ParameterDescs: descriptions,
	}
}

// generateParametersSchema creates OpenAI function parameters from a Go function signature
func generateParametersSchema(fnType reflect.Type, paramNames []string, paramDescs []string) map[string]any {
	properties := make(map[string]any)
	required := []string{}

	// Start from 1 to skip context parameter
	for i := 1; i < fnType.NumIn(); i++ {
		paramType := fnType.In(i)
		paramIndex := i - 1 // Adjust for context parameter

		// Use provided name or generate default
		paramName := fmt.Sprintf("param%d", i)
		if paramIndex < len(paramNames) && paramNames[paramIndex] != "" {
			paramName = paramNames[paramIndex]
		}

		// Generate parameter schema based on type
		paramSchema := getTypeSchema(paramType)

		// Add description if provided
		if paramIndex < len(paramDescs) && paramDescs[paramIndex] != "" {
			paramSchema["description"] = paramDescs[paramIndex]
		}

		properties[paramName] = paramSchema
		required = append(required, paramName)
	}

	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

// getTypeSchema returns the JSON schema for a Go type
func getTypeSchema(t reflect.Type) map[string]any {
	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Slice:
		return map[string]any{
			"type":  "array",
			"items": getTypeSchema(t.Elem()),
		}
	case reflect.Struct:
		return getStructSchema(t)
	case reflect.Ptr:
		// Handle pointers to structs
		if t.Elem().Kind() == reflect.Struct {
			return getStructSchema(t.Elem())
		}
		return getTypeSchema(t.Elem())
	default:
		// Default to string for other complex types
		return map[string]any{"type": "string"}
	}
}

// getStructSchema generates JSON schema for a struct type
func getStructSchema(t reflect.Type) map[string]any {
	properties := make(map[string]any)
	required := []string{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get field name from json tag or use field name
		fieldName := field.Name
		if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
			if commaIdx := strings.Index(tag, ","); commaIdx != -1 {
				fieldName = tag[:commaIdx]
			} else {
				fieldName = tag
			}
		}

		// Skip if field name is empty or "-"
		if fieldName == "" || fieldName == "-" {
			continue
		}

		// Generate schema for field type
		fieldSchema := getTypeSchema(field.Type)

		// Add description from tag if available
		if desc := field.Tag.Get("description"); desc != "" {
			fieldSchema["description"] = desc
		}

		properties[fieldName] = fieldSchema

		// Check if field is required (not a pointer and no omitempty tag)
		jsonTag := field.Tag.Get("json")
		if field.Type.Kind() != reflect.Ptr && !strings.Contains(jsonTag, "omitempty") {
			required = append(required, fieldName)
		}
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

// createHandler creates a tool handler from a function value
func createHandler(fnValue reflect.Value, fnType reflect.Type, paramNames []string) ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		// Prepare function arguments
		fnArgs := []reflect.Value{reflect.ValueOf(ctx)}

		// Convert map arguments to function parameters
		for i := 1; i < fnType.NumIn(); i++ {
			paramIndex := i - 1 // Adjust for context parameter

			// Use provided name or default
			paramName := fmt.Sprintf("param%d", i)
			if paramIndex < len(paramNames) && paramNames[paramIndex] != "" {
				paramName = paramNames[paramIndex]
			}

			argValue, ok := args[paramName]
			if !ok {
				return "", fmt.Errorf("missing required parameter: %s", paramName)
			}

			// Convert argument to the expected type
			convertedValue, err := convertToType(argValue, fnType.In(i))
			if err != nil {
				return "", fmt.Errorf("failed to convert parameter %s: %w", paramName, err)
			}
			fnArgs = append(fnArgs, convertedValue)
		}

		// Call the function
		results := fnValue.Call(fnArgs)

		// Handle return values
		result := results[0].String()
		if fnType.NumOut() == 2 && !results[1].IsNil() {
			err := results[1].Interface().(error)
			return result, err
		}

		return result, nil
	}
}

// convertToType converts a value to the specified type
func convertToType(value any, targetType reflect.Type) (reflect.Value, error) {
	// Direct conversion for basic types
	switch targetType.Kind() {
	case reflect.String:
		if s, ok := value.(string); ok {
			return reflect.ValueOf(s), nil
		}
		return reflect.ValueOf(fmt.Sprintf("%v", value)), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if n, ok := value.(float64); ok {
			return reflect.ValueOf(int(n)).Convert(targetType), nil
		}
		return reflect.Zero(targetType), fmt.Errorf("cannot convert %T to %s", value, targetType)
	case reflect.Float32, reflect.Float64:
		if n, ok := value.(float64); ok {
			return reflect.ValueOf(n).Convert(targetType), nil
		}
		return reflect.Zero(targetType), fmt.Errorf("cannot convert %T to %s", value, targetType)
	case reflect.Bool:
		if b, ok := value.(bool); ok {
			return reflect.ValueOf(b), nil
		}
		return reflect.Zero(targetType), fmt.Errorf("cannot convert %T to bool", value)
	case reflect.Slice:
		return convertToSlice(value, targetType)
	case reflect.Struct:
		return convertToStruct(value, targetType)
	case reflect.Ptr:
		if targetType.Elem().Kind() == reflect.Struct {
			structValue, err := convertToStruct(value, targetType.Elem())
			if err != nil {
				return reflect.Zero(targetType), err
			}
			// Create pointer to struct
			ptrValue := reflect.New(targetType.Elem())
			ptrValue.Elem().Set(structValue)
			return ptrValue, nil
		}
		// For other pointer types, convert the element and create pointer
		elemValue, err := convertToType(value, targetType.Elem())
		if err != nil {
			return reflect.Zero(targetType), err
		}
		ptrValue := reflect.New(targetType.Elem())
		ptrValue.Elem().Set(elemValue)
		return ptrValue, nil
	default:
		return reflect.Zero(targetType), fmt.Errorf("unsupported type: %s", targetType)
	}
}

// convertToSlice converts a value to a slice type
func convertToSlice(value any, targetType reflect.Type) (reflect.Value, error) {
	valueSlice, ok := value.([]any)
	if !ok {
		return reflect.Zero(targetType), fmt.Errorf("cannot convert %T to slice", value)
	}

	elemType := targetType.Elem()
	result := reflect.MakeSlice(targetType, len(valueSlice), len(valueSlice))

	for i, item := range valueSlice {
		convertedItem, err := convertToType(item, elemType)
		if err != nil {
			return reflect.Zero(targetType), fmt.Errorf("failed to convert slice element %d: %w", i, err)
		}
		result.Index(i).Set(convertedItem)
	}

	return result, nil
}

// convertToStruct converts a map to a struct type
func convertToStruct(value any, targetType reflect.Type) (reflect.Value, error) {
	valueMap, ok := value.(map[string]any)
	if !ok {
		return reflect.Zero(targetType), fmt.Errorf("cannot convert %T to struct", value)
	}

	result := reflect.New(targetType).Elem()

	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get field name from json tag or use field name
		fieldName := field.Name
		if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
			if commaIdx := strings.Index(tag, ","); commaIdx != -1 {
				fieldName = tag[:commaIdx]
			} else {
				fieldName = tag
			}
		}

		// Skip if field name is empty or "-"
		if fieldName == "" || fieldName == "-" {
			continue
		}

		// Get value from map
		mapValue, exists := valueMap[fieldName]
		if !exists {
			// Check if field is required
			jsonTag := field.Tag.Get("json")
			if field.Type.Kind() != reflect.Ptr && !strings.Contains(jsonTag, "omitempty") {
				return reflect.Zero(targetType), fmt.Errorf("required field %s is missing", fieldName)
			}
			continue
		}

		// Convert and set field value
		convertedValue, err := convertToType(mapValue, field.Type)
		if err != nil {
			return reflect.Zero(targetType), fmt.Errorf("failed to convert field %s: %w", fieldName, err)
		}

		result.Field(i).Set(convertedValue)
	}

	return result, nil
}

// CreateSimpleTool is a convenience function for the most common case
func CreateSimpleTool(name, description string, fn any) Tool {
	tool, err := CreateTool(name, description, fn)
	if err != nil {
		panic(fmt.Sprintf("Failed to create tool %s: %v", name, err))
	}
	return tool
}

// SimplePromptGenerator creates a basic prompt generator from a static prompt
func SimplePromptGenerator(basePrompt string) PromptGenerator {
	return func(userName, userPhone string) string {
		var userContext string
		if userName != "" && userPhone != "" {
			userContext = fmt.Sprintf("Você está conversando com %s (telefone: %s).\n\n", userName, userPhone)
		} else if userName != "" {
			userContext = fmt.Sprintf("Você está conversando com %s.\n\n", userName)
		} else if userPhone != "" {
			userContext = fmt.Sprintf("Você está conversando com o usuário do telefone %s.\n\n", userPhone)
		}
		return userContext + basePrompt
	}
}

// DefaultPromptGenerator returns the default prompt generator that includes user context
func DefaultPromptGenerator() PromptGenerator {
	return SimplePromptGenerator(`Você é um assistente inteligente da NextMind.

**FORMATAÇÃO DE MENSAGENS:**

Você deve dividir suas respostas em múltiplas mensagens quando apropriado. Siga estas diretrizes:

1. **Divida mensagens longas em partes menores:**
   - Cada mensagem deve ter no máximo 1 parágrafo ou 200 caracteres
   - Use divisões naturais de conteúdo (por tópico, por ponto, etc.)
   - Cada mensagem deve ser completa e fazer sentido por si só

2. **Formato para múltiplas mensagens:**
   - Retorne suas mensagens no formato JSON com um array de mensagens
   - Cada mensagem deve ter "content" (o texto) e "type" ("text" para mensagens normais ou "audio" para mensagens de áudio)
   - Exemplo para texto: {"messages": [{"content": "Primeira parte...", "type": "text"}, {"content": "Segunda parte...", "type": "text"}]}
   - Exemplo para áudio: {"messages": [{"content": "Esta mensagem será falada", "type": "audio"}]}

3. **Quando usar mensagens de áudio:**
   - Só envie mensagens com "type": "audio" quando o usuário pedir explicitamente para mandar um áudio.
   - Caso contrário, sempre envie mensagens do tipo "text".

4. **Quando dividir:**
   - Explicações longas: divida por conceitos ou etapas
   - Listas: considere enviar cada item importante como uma mensagem separada
   - Instruções: divida em passos claros

Sempre responda de forma útil e direta às perguntas do usuário.`)
}