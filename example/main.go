package main

import (
	"context"
	"fmt"
	"log"

	"github.com/NextMind-AI/chatbot-go"
)

// Mock function to simulate weather data retrieval
func getWeather(ctx context.Context, location string) string {
	// In a real implementation, this would call a weather API
	return "Sunny, 25°C in " + location
}

// Mock function to simulate time retrieval
func getCurrentTime(ctx context.Context) string {
	return "2024-01-15 14:30:00"
}

// Mock function with error handling
func translateText(ctx context.Context, text string, targetLanguage string) (string, error) {
	// Simulate a translation
	translations := map[string]string{
		"es": "Hola mundo",
		"fr": "Bonjour le monde",
		"pt": "Olá mundo",
	}

	if translation, ok := translations[targetLanguage]; ok && text == "Hello world" {
		return translation, nil
	}

	return "Translation not available", nil
}

// Example with struct parameter
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

func main() {
	// Define custom prompt generator that personalizes based on user context
	promptGenerator := func(userName, userPhone string) string {
		basePrompt := `Você é um assistente virtual especializado em informações sobre clima, horário, traduções e criação de perfis de usuário.

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

3. **Quando usar as ferramentas:**
   - Use a ferramenta get_weather quando perguntarem sobre o clima
   - Use a ferramenta get_time quando perguntarem sobre o horário
   - Use a ferramenta translate_text quando pedirem traduções
   - Use a ferramenta create_user_profile quando pedirem para criar um perfil de usuário
   - Sempre inclua as informações obtidas nas suas respostas

Sempre responda de forma útil e direta às perguntas do usuário.`

		// Personalize based on user context
		var greeting string
		if userName != "" && userPhone != "" {
			greeting = fmt.Sprintf("Olá %s! Vejo que você está entrando em contato pelo número %s. ", userName, userPhone)
		} else if userName != "" {
			greeting = fmt.Sprintf("Olá %s! ", userName)
		} else if userPhone != "" {
			greeting = fmt.Sprintf("Olá! Vejo que você está entrando em contato pelo número %s. ", userPhone)
		} else {
			greeting = "Olá! "
		}

		return greeting + "Como posso ajudar você hoje?\n\n" + basePrompt
	}

	// Define custom tools using the simplified API
	weatherTool, err := chatbot.CreateTool(
		"get_weather",
		"Get weather at the given location",
		chatbot.WithParams(
			getWeather,
			[]string{"location"},
			[]string{"The city and state/country to get weather for"},
		),
	)
	if err != nil {
		log.Fatal("Failed to create weather tool:", err)
	}

	timeTool, err := chatbot.CreateTool(
		"get_time",
		"Get the current time",
		getCurrentTime, // Simple function with no parameters
	)
	if err != nil {
		log.Fatal("Failed to create time tool:", err)
	}

	translateTool, err := chatbot.CreateTool(
		"translate_text",
		"Translate text to another language",
		chatbot.WithParams(
			translateText,
			[]string{"text", "targetLanguage"},
			[]string{"The text to translate", "Target language code (e.g., 'es' for Spanish)"},
		),
	)
	if err != nil {
		log.Fatal("Failed to create translate tool:", err)
	}

	profileTool, err := chatbot.CreateTool(
		"create_user_profile",
		"Create a user profile with the provided information",
		createUserProfile, // Struct parameter - types automatically inferred!
	)
	if err != nil {
		log.Fatal("Failed to create profile tool:", err)
	}

	// Create chatbot configuration
	config := chatbot.Config{
		PromptGenerator: promptGenerator,
		Tools:           []chatbot.Tool{weatherTool, timeTool, translateTool, profileTool},
		Model:           "gpt-4.1-mini",
	}

	log.Println("Starting chatbot with simplified tool API...")
	log.Println("Available tools:")
	log.Println("- get_weather(location): Get weather for a location")
	log.Println("- get_time(): Get current time")
	log.Println("- translate_text(text, targetLanguage): Translate text")
	log.Println("- create_user_profile(profile): Create user profile with struct data")
	log.Println("")
	log.Println("Server starting on port 8080...")

	bot := chatbot.New(config)
	bot.Run()
}
