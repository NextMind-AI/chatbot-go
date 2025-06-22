package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Config struct {
	VonageJWT                 string
	OpenAIKey                 string
	ElevenLabsAPIKey          string
	ElevenLabsVoiceID         string
	ElevenLabsModelID         string
	Port                      string
	GeospecificMessagesAPIURL string
	MessagesAPIURL            string
	RedisAddr                 string
	RedisPassword             string
	RedisDB                   int
	PhoneNumber               string
	S3Bucket                  string
	S3Region                  string
	AWSAccessKeyID            string
	AWSSecretAccessKey        string
	// Trinks API configuration
	TrinksAPIKey            string
	TrinksEstabelecimentoID string
	TrinksBaseURL           string
}

func Load() *Config {
	godotenv.Load()

	cfg := &Config{
		VonageJWT:                 mustGetEnv("VONAGE_JWT"),
		OpenAIKey:                 mustGetEnv("OPENAI_API_KEY"),
		ElevenLabsAPIKey:          mustGetEnv("ELEVENLABS_API_KEY"),
		ElevenLabsVoiceID:         mustGetEnvWithDefault("ELEVENLABS_VOICE_ID", "JNI7HKGyqNaHqfihNoCi"),
		ElevenLabsModelID:         mustGetEnvWithDefault("ELEVENLABS_MODEL_ID", "eleven_multilingual_v2"),
		Port:                      getEnv("PORT", "8080"),
		GeospecificMessagesAPIURL: getEnv("GEOSPECIFIC_MESSAGES_API_URL", "https://api-us.nexmo.com/v1/messages"),
		MessagesAPIURL:            getEnv("MESSAGES_API_URL", "https://api.nexmo.com/v1/messages"),
		RedisAddr:                 getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:             getEnv("REDIS_PASSWORD", ""),
		RedisDB:                   getEnvInt("REDIS_DB", 0),
		PhoneNumber:               mustGetEnv("PHONE_NUMBER"),
		S3Bucket:                  mustGetEnv("AWS_S3_BUCKET"),
		S3Region:                  getEnv("AWS_REGION", "us-east-2"),
		AWSAccessKeyID:            mustGetEnv("AWS_ACCESS_KEY_ID"),
		AWSSecretAccessKey:        mustGetEnv("AWS_SECRET_ACCESS_KEY"),
		// Trinks API configuration with defaults
		TrinksAPIKey:            getEnv("TRINKS_API_KEY", "aYUuejFVLk32PLEV14kAw9mX8U7BxBwtnWS43Tdb"),
		TrinksEstabelecimentoID: getEnv("TRINKS_ESTABELECIMENTO_ID", "222326"),
		TrinksBaseURL:           getEnv("TRINKS_BASE_URL", "https://api.trinks.com/v1"),
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

func mustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatal().Msgf("%s environment variable is required", key)
	}
	return value
}

func mustGetEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		if defaultValue == "" {
			log.Fatal().Msgf("%s environment variable is required", key)
		}
		return defaultValue
	}
	return value
}
