# OpenAI Package

This package provides a structured interface for interacting with OpenAI's API, specifically designed for a WhatsApp chatbot integration. It supports both standard chat completions and streaming responses with tool capabilities.

## Package Structure

### Core Files

#### `client.go`
- **Purpose**: Provides the main Client wrapper around OpenAI's official Go client
- **Key Components**:
  - `Client` struct: Wraps the OpenAI client with additional functionality
  - `NewClient()`: Creates a new client instance with custom HTTP configuration

#### `types.go`
- **Purpose**: Defines shared data structures and schema generation
- **Key Components**:
  - `Message`: Represents individual chat messages (text or audio)
  - `MessageList`: Container for multiple messages
  - `GenerateSchema()`: Generic function for JSON schema generation
  - `createSchemaParam()`: Creates OpenAI schema parameters for structured output

#### `parser.go`
- **Purpose**: Handles incremental parsing of streaming JSON responses
- **Key Components**:
  - `StreamingJSONParser`: Parses partial JSON chunks to extract complete messages
  - Sophisticated parsing logic that handles incomplete JSON objects
  - Maintains state across multiple chunks

#### `streaming.go`
- **Purpose**: Implements streaming chat completions with real-time message delivery
- **Key Components**:
  - `ProcessChatStreaming()`: Streams responses without tool support
  - `ProcessChatStreamingWithTools()`: Streams responses with tool capabilities
  - Unified internal architecture to avoid code duplication
  - Real-time WhatsApp message delivery as responses are generated

#### `chat.go`
- **Purpose**: Provides non-streaming chat completion methods
- **Key Components**:
  - `ProcessChat()`: Standard chat completion without tools
  - `ProcessChatWithTools()`: Chat completion with tool support

#### `tools.go`
- **Purpose**: Defines and handles AI tool capabilities
- **Key Components**:
  - `sleepTool`: Allows AI to pause conversation strategically
  - Tool processing and execution logic

#### `helpers.go`
- **Purpose**: Utility functions for message conversion and chat completion
- **Key Components**:
  - `convertChatHistory()`: Converts Redis format to OpenAI format
  - `createChatCompletionWithTools()`: Creates tool-enabled completions

#### `system_prompt.go`
- **Purpose**: Contains the system prompt that defines the AI's behavior
- **Key Components**:
  - Detailed instructions for message formatting
  - Guidelines for using the sleep tool
  - WhatsApp-specific conversation patterns

## Key Features

### 1. Streaming Response Architecture
The streaming implementation uses a sophisticated parser that can extract complete JSON messages from partial chunks. This enables:
- Real-time message delivery to users
- Reduced latency in conversation flow
- Better user experience with immediate feedback

### 2. Tool Integration
The package supports OpenAI's function calling (tools) feature, currently implementing:
- **Sleep Tool**: Allows the AI to pause when it detects the user might not have finished their thought

### 3. Structured Output
Uses OpenAI's JSON schema validation to ensure responses are properly formatted as message lists, preventing parsing errors and ensuring consistent output.

### 4. Audio Message Support
The package now supports generating audio messages through ElevenLabs integration:
- **Audio Message Type**: Messages can be marked as type "audio" in the response
- **Text-to-Speech Conversion**: Automatically converts message content to speech using ElevenLabs API
- **WhatsApp Audio Delivery**: Sends audio messages via WhatsApp using the Vonage API
- **Configurable Voice**: Uses configurable voice ID and model ID for speech generation

### 5. Error Handling
Comprehensive error logging using zerolog, with contextual information including:
- User IDs for tracking
- Error details for debugging
- Message content for analysis

## Usage Example

```go
// Create a new OpenAI client
httpClient := &http.Client{Timeout: 30 * time.Second}
openaiClient := openai.NewClient(apiKey, httpClient)

// Process a streaming chat with tools and audio support
err := openaiClient.ProcessChatStreamingWithTools(
    ctx,
    userID,
    chatHistory,
    vonageClient,
    redisClient,
    elevenLabsClient,
    voiceID,
    modelID,
    toNumber,
)
```

## Design Decisions

### 1. Separation of Concerns
- Types, parsing logic, and streaming logic are separated into different files
- This improves maintainability and testability

### 2. Unified Streaming Architecture
- Both tool and non-tool streaming methods share the same core logic
- Reduces code duplication and ensures consistency

### 3. Real-time Parsing
- The StreamingJSONParser can extract messages as soon as they're complete
- Doesn't wait for the entire response to finish

### 4. Flexible Tool System
- Easy to add new tools by following the existing pattern
- Tools are processed in a separate step before streaming

## Dependencies

- `github.com/openai/openai-go`: Official OpenAI Go client
- `github.com/invopop/jsonschema`: JSON schema generation
- `github.com/rs/zerolog`: Structured logging
- Internal packages: `redis`, `vonage`, `elevenlabs` for integration and audio support

## Future Enhancements

1. **Additional Tools**: Easy to add more AI capabilities
2. **Response Caching**: Could cache common responses
3. **Rate Limiting**: Add rate limiting for API calls
4. **Metrics**: Add performance monitoring
5. **Testing**: Comprehensive unit tests for parser and streaming logic 