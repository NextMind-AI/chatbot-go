package main

import (
	"chatbot/aws"
	"chatbot/config"
	"chatbot/elevenlabs"
	"chatbot/execution"
	"chatbot/openai"
	"chatbot/processor"
	"chatbot/redis"
	"chatbot/server"
	"chatbot/vonage"
	"net/http"
)

func main() {
	appConfig := config.Load()

	var httpClient = http.Client{}

	awsClient := aws.NewClient(appConfig.S3Region, appConfig.S3Bucket)

	vonageClient := vonage.NewClient(
		appConfig.VonageJWT,
		appConfig.GeospecificMessagesAPIURL,
		appConfig.MessagesAPIURL,
		appConfig.PhoneNumber,
		httpClient,
	)

	openAIClient := openai.NewClient(
		appConfig.OpenAIKey,
		httpClient,
	)

	redisClient := redis.NewClient(
		appConfig.RedisAddr,
		appConfig.RedisPassword,
		appConfig.RedisDB,
	)

	elevenLabsClient := elevenlabs.NewClient(
		appConfig.ElevenLabsAPIKey,
		httpClient,
		awsClient,
	)

	executionManager := execution.NewManager()

	messageProcessor := processor.NewMessageProcessor(
		vonageClient,
		redisClient,
		openAIClient,
		elevenLabsClient,
		executionManager,
	)

	srv := server.New(messageProcessor)

	srv.Start(appConfig.Port)
}
