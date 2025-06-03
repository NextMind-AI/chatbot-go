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
var AudioHandler *elevenlabs.AudioHandler

func main() {
	var appConfig = config.Load()

	var httpClient = http.Client{}

	VonageClient = vonage.NewClient(
		appConfig.VonageJWT,
		appConfig.GeospecificMessagesAPIURL,
		appConfig.MessagesAPIURL,
		httpClient,
	)

	OpenAIClient = openai.NewClient(
		appConfig.OpenAIKey,
		&httpClient,
	)

	RedisClient = redis.NewClient(
		appConfig.RedisAddr,
		appConfig.RedisPassword,
		appConfig.RedisDB,
	)

	// Initialize ElevenLabs client if API key is provided
	if appConfig.ElevenLabsAPIKey != "" {
		ElevenLabsClient = elevenlabs.NewClient(
			appConfig.ElevenLabsAPIKey,
			&httpClient,
		)
		AudioHandler = elevenlabs.NewAudioHandler(&ElevenLabsClient, &httpClient)
		log.Info().Msg("ElevenLabs speech-to-text client initialized")
	} else {
		log.Warn().Msg("ElevenLabs API key not provided - audio transcription will be disabled")
	}

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
