package config

import (
	"os"
	"strconv"
	"strings"

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
	}

	// Validate JWT format
	validateJWT(cfg.VonageJWT)

	return cfg
}

func validateJWT(jwt string) {
	if jwt == "" {
		log.Fatal().Msg("VONAGE_JWT is empty")
	}

	// Basic JWT format validation (should have 3 parts separated by dots)
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		log.Fatal().Msg("VONAGE_JWT appears to be malformed - should have 3 parts separated by dots")
	}

	// Check if JWT starts with "Bearer " (common mistake)
	if strings.HasPrefix(jwt, "Bearer ") {
		log.Fatal().Msg("VONAGE_JWT should not include 'Bearer ' prefix - remove it from the environment variable")
	}

	log.Info().
		Str("jwt_prefix", jwt[:min(10, len(jwt))]+"...").
		Msg("VONAGE_JWT loaded successfully")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
