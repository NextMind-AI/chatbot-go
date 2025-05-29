package main

import (
	"chatbot/config"
	"chatbot/redis"
	"chatbot/vonage"
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

var VonageClient vonage.Client
var OpenAIClient openai.Client
var RedisClient redis.Client

func main() {
	var appConfig = config.Load()

	VonageClient = vonage.NewClient(vonage.Config{
		VonageJWT:                 appConfig.VonageJWT,
		GeospecificMessagesAPIURL: appConfig.GeospecificMessagesAPIURL,
		MessagesAPIURL:            appConfig.MessagesAPIURL,
	})

	OpenAIClient = openai.NewClient(
		option.WithAPIKey(appConfig.OpenAIKey),
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

	log.Printf("Server starting on :%s\n", appConfig.Port)
	log.Fatal(app.Listen(":" + appConfig.Port))
}
