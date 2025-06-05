package main

import (
	"chatbot/aws"
	"chatbot/config"
	"chatbot/elevenlabs"
	"chatbot/execution"
	"chatbot/openai"
	"chatbot/processor"
	"chatbot/redis"
	"chatbot/vonage"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
)

var VonageClient vonage.Client
var OpenAIClient openai.Client
var RedisClient redis.Client
var ElevenLabsClient elevenlabs.Client
var MessageProcessor *processor.MessageProcessor
var executionManager *execution.Manager

func main() {
	appConfig := config.Load()

	var httpClient = http.Client{}

	awsClient := aws.NewClient(appConfig.S3Region, appConfig.S3Bucket)

	VonageClient = vonage.NewClient(
		appConfig.VonageJWT,
		appConfig.GeospecificMessagesAPIURL,
		appConfig.MessagesAPIURL,
		appConfig.PhoneNumber,
		httpClient,
	)

	OpenAIClient = openai.NewClient(
		appConfig.OpenAIKey,
		httpClient,
	)

	RedisClient = redis.NewClient(
		appConfig.RedisAddr,
		appConfig.RedisPassword,
		appConfig.RedisDB,
	)

	ElevenLabsClient = elevenlabs.NewClient(
		appConfig.ElevenLabsAPIKey,
		httpClient,
		awsClient,
	)

	executionManager = execution.NewManager()

	MessageProcessor = processor.NewMessageProcessor(
		VonageClient,
		RedisClient,
		OpenAIClient,
		ElevenLabsClient,
		executionManager,
	)

	app := fiber.New()

	app.Post("/webhooks/inbound-message", inboundMessageHandler)

	log.Info().Str("port", appConfig.Port).Msg("Starting chatbot server")

	err := app.Listen(":"+appConfig.Port, fiber.ListenConfig{
		DisableStartupMessage: true,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
