package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	VonageJWT                 string
	OpenAIKey                 string
	Port                      string
	GeospecificMessagesAPIURL string
	MessagesAPIURL            string
	RedisAddr                 string
	RedisPassword             string
	RedisDB                   int
}

func Load() *Config {
	godotenv.Load()

	cfg := &Config{
		VonageJWT:                 getEnv("VONAGE_JWT", ""),
		OpenAIKey:                 getEnv("OPENAI_API_KEY", ""),
		Port:                      getEnv("PORT", "8080"),
		GeospecificMessagesAPIURL: getEnv("GEOSPECIFIC_MESSAGES_API_URL", "https://api-us.nexmo.com/v1/messages"),
		MessagesAPIURL:            getEnv("MESSAGES_API_URL", "https://api.nexmo.com/v1/messages"),
		RedisAddr:                 getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:             getEnv("REDIS_PASSWORD", ""),
		RedisDB:                   getEnvInt("REDIS_DB", 0),
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

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
