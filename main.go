package main

import (
	"chatbot/config"
	"chatbot/redis"
	"chatbot/vonage"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/rs/zerolog/log"
)

var VonageClient vonage.Client
var OpenAIClient openai.Client
var RedisClient redis.Client

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
		option.WithAPIKey(appConfig.OpenAIKey),
		option.WithHTTPClient(&httpClient),
	)

	RedisClient = redis.NewClient(
		appConfig.RedisAddr,
		appConfig.RedisPassword,
		appConfig.RedisDB,
	)

	app := fiber.New()

	app.Post("/webhooks/inbound-message", inboundMessage)
	app.Post("/webhooks/status", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	log.Info().Str("port", appConfig.Port).Msg("Starting chatbot server")

	err := app.Listen(":"+appConfig.Port, fiber.ListenConfig{
		DisableStartupMessage: true,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
