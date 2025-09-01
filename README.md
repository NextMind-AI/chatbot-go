# Chatbot Package

A powerful and flexible Go package for creating WhatsApp chatbots with configurable system prompts and custom tools. This package provides an easy-to-use interface for building intelligent chatbots that can handle text and audio messages, execute custom functions, and maintain conversation context.

## Features

- **Automatic Type Inference**: Just write normal Go functions - parameter types are automatically detected
- **Dynamic Prompt Generation**: Create personalized prompts based on user context (name and phone)
- **Configurable AI Models**: Choose from any OpenAI model
- **Custom Tools**: Define functions that the AI can call to extend its capabilities
- **WhatsApp Integration**: Built-in support for WhatsApp via Vonage API
- **Audio Support**: Text-to-speech and speech-to-text via ElevenLabs
- **Streaming Responses**: Real-time message delivery for better user experience
- **Redis Integration**: Persistent conversation history
- **AWS S3 Integration**: Audio file storage and serving
- **Smart Sleep System**: Intelligent timing for natural conversation flow

## Quick Start

### Installation

```bash
go get github.com/NextMind-AI/chatbot-go
```

#### If you see authentication errors, set the following environment variable:

```bash
go env -w GOPRIVATE=github.com/NextMind-AI
```

### Environment Variables

Create a `.env` file with the following configuration:

```env
# Required: WhatsApp/Vonage Configuration
VONAGE_JWT=your_vonage_jwt_token
PHONE_NUMBER=your_whatsapp_business_number

# Required: AI Services
OPENAI_API_KEY=your_openai_api_key
ELEVENLABS_API_KEY=your_elevenlabs_api_key

# Required: AWS Configuration
AWS_S3_BUCKET=your-s3-bucket-name
AWS_REGION=us-east-2
AWS_ACCESS_KEY_ID=your_aws_access_key_id
AWS_SECRET_ACCESS_KEY=your_aws_secret_access_key

# Optional: Redis Configuration (defaults to localhost)
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Optional: Server Configuration
PORT=8080

# Optional: ElevenLabs Configuration
ELEVENLABS_VOICE_ID=JNI7HKGyqNaHqfihNoCi
ELEVENLABS_MODEL_ID=eleven_multilingual_v2

# Optional: Vonage API URLs (use defaults for most cases)
GEOSPECIFIC_MESSAGES_API_URL=https://api-us.nexmo.com/v1/messages
MESSAGES_API_URL=https://api.nexmo.com/v1/messages
```

### Basic Usage

```go
package main

import (
    "context"
    "log"

    "github.com/NextMind-AI/chatbot-go"
)

// Define your tool functions with proper types
func getWeather(ctx context.Context, location string) string {
    // In a real implementation, this would call a weather API
    return fmt.Sprintf("Sunny, 25°C in %s", location)
}

func main() {
    // Define your custom prompt generator that personalizes based on user context
    promptGenerator := func(userName, userPhone string) string {
        basePrompt := `You are a helpful assistant specialized in weather information.

**MESSAGE FORMATTING:**

You should divide your responses into multiple messages when appropriate. Follow these guidelines:

1. **Split long messages into smaller parts:**
   - Each message should have at most 1 paragraph or 200 characters
   - Use natural content divisions (by topic, by point, etc.)
   - Each message should be complete and make sense on its own

2. **Format for multiple messages:**
   - Return your messages in JSON format with an array of messages
   - Each message should have "content" (the text) and "type" ("text" for normal messages or "audio" for audio messages)
   - Example for text: {"messages": [{"content": "First part...", "type": "text"}, {"content": "Second part...", "type": "text"}]}

3. **When to use tools:**
   - Use the get_weather tool when asked about weather
   - Always include the obtained information in your responses

Always respond helpfully and directly to user questions.`

        // Personalize greeting based on user context
        var greeting string
        if userName != "" && userPhone != "" {
            greeting = fmt.Sprintf("Hello %s! I see you're contacting from %s. ", userName, userPhone)
        } else if userName != "" {
            greeting = fmt.Sprintf("Hello %s! ", userName)
        } else if userPhone != "" {
            greeting = fmt.Sprintf("Hello! I see you're contacting from %s. ", userPhone)
        } else {
            greeting = "Hello! "
        }

        return greeting + "How can I help you today?\n\n" + basePrompt
    }

    // Create tools with automatic type inference
    weatherTool, err := chatbot.CreateTool(
        "get_weather",
        "Get weather at the given location",
        chatbot.WithParams(
            getWeather,
            []string{"location"},
            []string{"The location to get weather for"},
        ),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create chatbot configuration
    config := chatbot.Config{
        PromptGenerator: promptGenerator,
        Tools:           []chatbot.Tool{weatherTool},
    }

    // Create and start the chatbot
    bot := chatbot.New(config)
    log.Println("Starting chatbot server on port 8080...")
    bot.Start("8080")
}
```

## Custom Tools

Tools allow your chatbot to call external functions and APIs. The package uses reflection to automatically infer parameter types from your function signatures.

### Simple Tool Example

```go
// Define your function with proper types
func getWeather(ctx context.Context, location string) string {
    // In a real implementation, this would call a weather API
    return fmt.Sprintf("Sunny, 25°C in %s", location)
}

// Create the tool - types are automatically inferred!
weatherTool, err := chatbot.CreateTool(
    "get_weather",
    "Get weather at the given location",
    getWeather,
)
```

### Tool with Parameter Names and Descriptions

```go
// For better AI understanding, provide parameter names and descriptions
weatherTool, err := chatbot.CreateTool(
    "get_weather",
    "Get current weather for a specific location",
    chatbot.WithParams(
        getWeather,
        []string{"location"},                              // Parameter names
        []string{"The city and country, e.g. San Francisco, CA"}, // Descriptions
    ),
)
```

### Function with Error Handling

```go
// Functions can return (string) or (string, error)
func getUserInfo(ctx context.Context, userID string) (string, error) {
    user, err := db.GetUser(userID)
    if err != nil {
        return "", fmt.Errorf("user not found: %w", err)
    }
    return fmt.Sprintf("User: %s, Email: %s", user.Name, user.Email), nil
}

userInfoTool, err := chatbot.CreateTool(
    "get_user_info",
    "Get user information from database",
    chatbot.WithParams(
        getUserInfo,
        []string{"userID"},
        []string{"The user ID to look up"},
    ),
)
```

### Multiple Parameters Example

```go
func translateText(ctx context.Context, text string, targetLanguage string) (string, error) {
    // Call translation API
    translated, err := translationAPI.Translate(text, targetLanguage)
    if err != nil {
        return "", err
    }
    return translated, nil
}

translateTool, err := chatbot.CreateTool(
    "translate_text",
    "Translate text to another language",
    chatbot.WithParams(
        translateText,
        []string{"text", "targetLanguage"},
        []string{
            "The text to translate",
            "The target language code (e.g., 'es' for Spanish)",
        },
    ),
)
```

### No Parameters Example

```go
func getCurrentTime(ctx context.Context) string {
    return time.Now().Format("2006-01-02 15:04:05")
}

timeTool, err := chatbot.CreateTool(
    "get_current_time",
    "Get the current date and time",
    getCurrentTime, // No parameters needed!
)
```

### Struct Parameters

The package now supports struct parameters with automatic JSON schema generation:

```go
// Define your struct with JSON tags and descriptions
type UserProfile struct {
    Name     string  `json:"name" description:"User's full name"`
    Email    string  `json:"email" description:"User's email address"`
    Age      int     `json:"age" description:"User's age in years"`
    Premium  bool    `json:"premium,omitempty" description:"Whether user has premium subscription"`
    Settings *Config `json:"settings,omitempty" description:"User's configuration settings"`
}

type Config struct {
    Theme    string `json:"theme" description:"UI theme preference (light/dark)"`
    Language string `json:"language" description:"Preferred language code"`
}

// Use the struct in your function
func createUserProfile(ctx context.Context, profile UserProfile) (string, error) {
    if profile.Name == "" || profile.Email == "" {
        return "", fmt.Errorf("name and email are required")
    }
    
    result := fmt.Sprintf("Created profile for %s (%s), age %d", 
        profile.Name, profile.Email, profile.Age)
    
    if profile.Premium {
        result += " [Premium User]"
    }
    
    if profile.Settings != nil {
        result += fmt.Sprintf(" - Theme: %s, Language: %s", 
            profile.Settings.Theme, profile.Settings.Language)
    }
    
    return result, nil
}

// Create the tool - struct schema is automatically generated!
profileTool, err := chatbot.CreateTool(
    "create_user_profile",
    "Create a user profile with the provided information",
    createUserProfile,
)
```

#### Struct Features

- **Automatic Schema Generation**: JSON schema is generated from struct fields and tags
- **Field Descriptions**: Use `description:"..."` tags to provide field descriptions to the AI
- **JSON Tag Support**: Respects `json:"fieldname,omitempty"` tags for field naming and optional fields
- **Nested Structs**: Supports structs within structs with full type inference
- **Pointer Support**: Optional fields can be pointers (will be nil if not provided)
- **Required vs Optional**: Non-pointer fields without `omitempty` are required

## Dynamic Prompt Generation

The package now supports dynamic prompt generation based on user context. Instead of a static prompt, you can provide a function that generates personalized prompts based on the user's name and phone number.

### Basic Prompt Generator

```go
// Create a custom prompt generator
promptGenerator := func(userName, userPhone string) string {
    basePrompt := `You are a helpful customer service assistant.
    
**INSTRUCTIONS:**
- Always be polite and professional
- Use the customer's name when available
- Provide clear and helpful responses`

    // Personalize based on user context
    var greeting string
    if userName != "" && userPhone != "" {
        greeting = fmt.Sprintf("Hello %s! I see you're contacting from %s. ", userName, userPhone)
    } else if userName != "" {
        greeting = fmt.Sprintf("Hello %s! ", userName)
    } else if userPhone != "" {
        greeting = fmt.Sprintf("Hello! I see you're contacting from %s. ", userPhone)
    } else {
        greeting = "Hello! "
    }

    return greeting + "How can I help you today?\n\n" + basePrompt
}

// Use in configuration
config := chatbot.Config{
    PromptGenerator: promptGenerator,
    Tools:           tools,
}
```

### Convenience Functions

The package provides helper functions for common prompt patterns:

```go
// Simple static prompt with user context
config := chatbot.Config{
    PromptGenerator: chatbot.SimplePromptGenerator(`You are a helpful assistant...`),
    Tools:           tools,
}

// Use the default prompt
config := chatbot.Config{
    PromptGenerator: chatbot.DefaultPromptGenerator(),
    Tools:           tools,
}

// Or set to nil to use default
config := chatbot.Config{
    PromptGenerator: nil, // Will use default
    Tools:           tools,
}
```

### Advanced Prompt Generation

You can create sophisticated prompt generators that adapt based on user information:

```go
promptGenerator := func(userName, userPhone string) string {
    basePrompt := `You are AcmeCorp's AI assistant...`
    
    // Check if it's a known premium customer
    if isPremiumCustomer(userPhone) {
        return fmt.Sprintf("Welcome back, %s! As a premium customer, you have priority support. %s", 
                          userName, basePrompt)
    }
    
    // Check business hours
    if isBusinessHours() {
        return fmt.Sprintf("Hello %s! Our full support team is available. %s", userName, basePrompt)
    } else {
        return fmt.Sprintf("Hello %s! You're contacting us outside business hours, but I'm here to help. %s", 
                          userName, basePrompt)
    }
}
```

## System Prompt Guidelines

The system prompt defines your chatbot's personality and behavior. Here are some best practices:

### Message Formatting

Always include message formatting instructions in your system prompt:

```
**MESSAGE FORMATTING:**

You should divide your responses into multiple messages when appropriate. Follow these guidelines:

1. **Split long messages into smaller parts:**
   - Each message should have at most 1 paragraph or 200 characters
   - Use natural content divisions (by topic, by point, etc.)
   - Each message should be complete and make sense on its own

2. **Format for multiple messages:**
   - Return your messages in JSON format with an array of messages
   - Each message should have "content" (the text) and "type" ("text" for normal messages or "audio" for audio messages)
   - Example: {"messages": [{"content": "First part...", "type": "text"}, {"content": "Second part...", "type": "text"}]}

3. **When to use audio messages:**
   - Only send messages with "type": "audio" when the user explicitly requests audio.
   - Otherwise, always send "text" type messages.
```

### Tool Usage Instructions

Specify when and how to use your custom tools:

```
**TOOL USAGE:**

You have access to the following tools:
- get_weather: Use when users ask about weather conditions
- get_user_info: Use when you need user account information
- translate_text: Use when users request translations

Always use these tools when relevant to provide accurate, up-to-date information.
```

### Personality and Behavior

Define your chatbot's personality:

```
You are a friendly and professional customer service assistant for AcmeCorp.

Key traits:
- Always be polite and helpful
- Provide clear, concise answers
- Ask clarifying questions when needed
- Escalate complex issues to human agents when appropriate
- Use the customer's name when known
```

## Advanced Configuration

### Multiple Tools

```go
tools := []chatbot.Tool{
    weatherTool,
    userInfoTool,
    translateTool,
    // Add as many tools as needed
}
```

### Supported Function Types

The package supports various function signatures:

```go
// Basic function - returns string
func simpleFunc(ctx context.Context, param string) string

// Function with error handling - returns (string, error)
func funcWithError(ctx context.Context, param string) (string, error)

// Multiple parameters
func multiParam(ctx context.Context, param1 string, param2 int) string

// No parameters (besides context)
func noParams(ctx context.Context) string
```

### Type Support

The following parameter types are automatically recognized:
- `string`
- `int`, `int8`, `int16`, `int32`, `int64`
- `float32`, `float64`
- `bool`
- Slices of the above types
- **Structs** with automatic JSON schema generation
- Pointers to structs
- Nested structs

### Error Handling in Tool Functions

```go
func riskyOperation(ctx context.Context, requiredParam string, optionalParam int) (string, error) {
    // Handle context cancellation
    select {
    case <-ctx.Done():
        return "", ctx.Err()
    default:
    }
    
    // Validate inputs
    if requiredParam == "" {
        return "", fmt.Errorf("required parameter cannot be empty")
    }
    
    // Perform operation
    result, err := performDatabaseQuery(requiredParam, optionalParam)
    if err != nil {
        log.Printf("Error in risky_operation: %v", err)
        return "", fmt.Errorf("operation failed: %w", err)
    }
    
    return result, nil
}

// Create the tool
riskyTool, err := chatbot.CreateTool(
    "risky_operation",
    "Perform a risky database operation",
    chatbot.WithParams(
        riskyOperation,
        []string{"requiredParam", "optionalParam"},
        []string{"Required parameter description", "Optional parameter (default: 0)"},
    ),
)
```

### Custom Port

```go
bot.Start("9000") // Start on port 9000 instead of 8080
```

## Architecture

The package consists of several integrated components:

- **Main Chatbot Package**: Easy-to-use interface for configuration and startup
- **OpenAI Integration**: Handles AI conversations with custom prompts and tools
- **Vonage Integration**: WhatsApp message sending and receiving
- **ElevenLabs Integration**: Text-to-speech and speech-to-text processing
- **Redis Integration**: Conversation history storage
- **AWS S3 Integration**: Audio file storage and serving
- **Execution Manager**: Handles concurrent user conversations
- **Sleep Analyzer**: Intelligent timing for natural conversation flow

## Tool Execution Flow

1. User sends a message via WhatsApp
2. Message is processed and added to conversation history
3. Sleep analyzer determines appropriate wait time
4. If custom tools are defined:
   - AI is called with tools available
   - If AI decides to use tools, tool handlers are executed
   - Tool results are added to conversation context
5. Final response is generated and streamed to user
6. Response is sent via WhatsApp and stored in history

## Webhook Setup

The package automatically sets up a webhook endpoint at `/webhooks/inbound-message` that handles incoming WhatsApp messages. Configure your Vonage webhook URL to point to:

```
https://your-domain.com/webhooks/inbound-message
```

## Dependencies

- Go 1.21+
- Redis server
- AWS S3 bucket
- Vonage API account
- OpenAI API account
- ElevenLabs API account

## Example Applications

Check the `example/` directory for complete working examples:

- **Weather Bot**: A chatbot that provides weather information
- **Customer Service Bot**: A bot that handles customer inquiries
- **Multi-language Bot**: A bot with translation capabilities

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

This project is licensed under the MIT License - see the LICENSE file for details. 
