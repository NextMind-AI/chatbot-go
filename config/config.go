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
}

func Load() *Config {
	godotenv.Load()

	cfg := &Config{
		VonageJWT:                 getEnv("VONAGE_JWT", ""),
		OpenAIKey:                 getEnv("OPENAI_API_KEY", ""),
		ElevenLabsAPIKey:          getEnv("ELEVENLABS_API_KEY", ""),
		ElevenLabsVoiceID:         getEnv("ELEVENLABS_VOICE_ID", "JNI7HKGyqNaHqfihNoCi"),
		ElevenLabsModelID:         getEnv("ELEVENLABS_MODEL_ID", "eleven_multilingual_v2"),
		Port:                      getEnv("PORT", "8080"),
		GeospecificMessagesAPIURL: getEnv("GEOSPECIFIC_MESSAGES_API_URL", "https://api-us.nexmo.com/v1/messages"),
		MessagesAPIURL:            getEnv("MESSAGES_API_URL", "https://api.nexmo.com/v1/messages"),
		RedisAddr:                 getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:             getEnv("REDIS_PASSWORD", ""),
		RedisDB:                   getEnvInt("REDIS_DB", 0),
		PhoneNumber:               getEnv("PHONE_NUMBER", ""),
		S3Bucket:                  getEnv("AWS_S3_BUCKET", ""),
		S3Region:                  getEnv("AWS_REGION", "us-east-2"),
		AWSAccessKeyID:            getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey:        getEnv("AWS_SECRET_ACCESS_KEY", ""),
	}

	if cfg.VonageJWT == "" {
		log.Fatal().Msg("VONAGE_JWT environment variable is required")
	}

	if cfg.OpenAIKey == "" {
		log.Fatal().Msg("OPENAI_API_KEY environment variable is required")
	}

	if cfg.ElevenLabsAPIKey == "" {
		log.Fatal().Msg("ELEVENLABS_API_KEY environment variable is required")
	}

	if cfg.ElevenLabsVoiceID == "" {
		log.Fatal().Msg("ELEVENLABS_VOICE_ID environment variable is required")
	}

	if cfg.ElevenLabsModelID == "" {
		log.Fatal().Msg("ELEVENLABS_MODEL_ID environment variable is required")
	}

	if cfg.PhoneNumber == "" {
		log.Fatal().Msg("PHONE_NUMBER environment variable is required")
	}

	if cfg.S3Bucket == "" {
		log.Fatal().Msg("AWS_S3_BUCKET environment variable is required")
	}

	// Comment out these checks if using environment variables for AWS credentials:
	// if cfg.AWSAccessKeyID == "" {
	// 	log.Fatal().Msg("AWS_ACCESS_KEY_ID environment variable is required")
	// }

	// if cfg.AWSSecretAccessKey == "" {
	// 	log.Fatal().Msg("AWS_SECRET_ACCESS_KEY environment variable is required")
	// }

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
