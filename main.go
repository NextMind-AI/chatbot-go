package main

import (
	"chatbot/config"
	"chatbot/elevenlabs"
	"chatbot/openai"
	"chatbot/redis"
	"chatbot/vonage"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
)

var VonageClient vonage.Client
var OpenAIClient openai.Client
var RedisClient redis.Client
var ElevenLabsClient elevenlabs.Client
var AppConfig *config.Config

func main() {
	AppConfig = config.Load()

	var httpClient = http.Client{}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(AppConfig.S3Region),
	})

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create AWS session")
	}

	log.Info().
		Str("bucket", AppConfig.S3Bucket).
		Str("region", AppConfig.S3Region).
		Msg("AWS session created successfully")

	VonageClient = vonage.NewClient(
		AppConfig.VonageJWT,
		AppConfig.GeospecificMessagesAPIURL,
		AppConfig.MessagesAPIURL,
		AppConfig.PhoneNumber,
		httpClient,
	)

	OpenAIClient = openai.NewClient(
		AppConfig.OpenAIKey,
		httpClient,
	)

	RedisClient = redis.NewClient(
		AppConfig.RedisAddr,
		AppConfig.RedisPassword,
		AppConfig.RedisDB,
	)

	ElevenLabsClient = elevenlabs.NewClient(
		AppConfig.ElevenLabsAPIKey,
		httpClient,
		sess,
		AppConfig.S3Bucket,
		AppConfig.S3Region,
	)

	app := fiber.New()

	app.Post("/webhooks/inbound-message", inboundMessageHandler)

	log.Info().Str("port", AppConfig.Port).Msg("Starting chatbot server")

	err = app.Listen(":"+AppConfig.Port, fiber.ListenConfig{
		DisableStartupMessage: true,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
