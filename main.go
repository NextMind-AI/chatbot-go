package main

import (
	"chatbot/config"
	"chatbot/elevenlabs"
	"chatbot/openai"
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

func main() {
	var appConfig = config.Load()

	var httpClient = http.Client{}

	VonageClient = vonage.NewClient(
		appConfig.VonageJWT,
		appConfig.GeospecificMessagesAPIURL,
		appConfig.MessagesAPIURL,
		appConfig.SenderID,
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
