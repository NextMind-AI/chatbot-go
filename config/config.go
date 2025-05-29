package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	VonageJWT                 string
	OpenAIKey                 string
	Port                      string
	GeospecificMessagesAPIURL string
	MessagesAPIURL            string
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	cfg := &Config{
		VonageJWT:                 getEnv("VONAGE_JWT", ""),
		OpenAIKey:                 getEnv("OPENAI_API_KEY", ""),
		Port:                      getEnv("PORT", "8080"),
		GeospecificMessagesAPIURL: getEnv("GEOSPECIFIC_MESSAGES_API_URL", "https://api-us.nexmo.com/v1/messages"),
		MessagesAPIURL:            getEnv("MESSAGES_API_URL", "https://api.nexmo.com/v1/messages"),
	}

	if cfg.VonageJWT == "" {
		log.Fatal("VONAGE_JWT environment variable is required")
	}

	if cfg.OpenAIKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
