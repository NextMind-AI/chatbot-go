package main

import (
	"chatbot/elevenlabs"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
)

type Profile struct {
	Name string `json:"name"`
}

// Audio represents WhatsApp audio message structure
type Audio struct {
	URL string `json:"url"`
}

// Video represents WhatsApp video message structure
type Video struct {
	URL string `json:"url"`
}

// Image represents WhatsApp image message structure
type Image struct {
	URL     string `json:"url"`
	Caption string `json:"caption,omitempty"`
}

// Document represents WhatsApp document message structure
type Document struct {
	URL      string `json:"url"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type InboundMessage struct {
	Channel       string    `json:"channel"`
	ContextStatus string    `json:"context_status"`
	From          string    `json:"from"`
	MessageType   string    `json:"message_type"`
	MessageUUID   string    `json:"message_uuid"`
	Profile       Profile   `json:"profile"`
	Text          string    `json:"text"`
	Timestamp     string    `json:"timestamp"`
	To            string    `json:"to"`
	Audio         *Audio    `json:"audio,omitempty"`
	Video         *Video    `json:"video,omitempty"`
	Image         *Image    `json:"image,omitempty"`
	Document      *Document `json:"document,omitempty"`
}

func inboundMessageHandler(c fiber.Ctx) error {
	log.Info().Msg("Received inbound message request")

	var message InboundMessage
	if err := c.Bind().JSON(&message); err != nil {
		log.Error().Err(err).Msg("Error parsing JSON")
		return c.Status(fiber.StatusBadRequest).SendString("Error parsing JSON")
	}

	// Log message details for debugging
	log.Info().
		Str("message_uuid", message.MessageUUID).
		Str("message_type", message.MessageType).
		Str("from", message.From).
		Str("text", message.Text).
		Bool("has_audio", message.Audio != nil).
		Bool("has_video", message.Video != nil).
		Bool("has_image", message.Image != nil).
		Bool("has_document", message.Document != nil).
		Msg("Processing inbound message")

	go processMessage(message)

	return c.SendStatus(fiber.StatusOK)
}

// processAudioMessage handles audio message transcription
func processAudioMessage(message InboundMessage) (string, error) {
	if AudioHandler == nil {
		log.Warn().Msg("Audio handler not initialized - skipping audio transcription")
		return "", nil
	}

	var audioURL string
	var contentType string

	// Check for audio in WhatsApp audio message
	if message.Audio != nil && message.Audio.URL != "" {
		audioURL = message.Audio.URL
		contentType = "audio/mpeg" // Default for WhatsApp audio
		log.Debug().Str("audio_url", audioURL).Msg("Found WhatsApp audio message")
	} else if message.Video != nil && message.Video.URL != "" {
		// Video messages can also contain audio to transcribe
		audioURL = message.Video.URL
		contentType = "video/mp4" // Default for WhatsApp video
		log.Debug().Str("video_url", audioURL).Msg("Found WhatsApp video message")
	}

	if audioURL == "" {
		return "", nil // No audio content found
	}

	log.Info().
		Str("audio_url", audioURL).
		Str("content_type", contentType).
		Str("message_type", message.MessageType).
		Msg("Transcribing audio message")

	audioMsg := elevenlabs.AudioMessage{
		URL:         audioURL,
		ContentType: contentType,
		Language:    "", // Auto-detect language
	}

	response, err := AudioHandler.ProcessAudioMessage(audioMsg)
	if err != nil {
		log.Error().
			Err(err).
			Str("audio_url", audioURL).
			Msg("Failed to transcribe audio message")
		return "", err
	}

	transcribedText := elevenlabs.ExtractTextFromTranscription(response)

	log.Info().
		Str("transcribed_text", transcribedText).
		Str("detected_language", response.LanguageCode).
		Float64("confidence", response.LanguageProbability).
		Msg("Audio transcription completed")

	return transcribedText, nil
}
