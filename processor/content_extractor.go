package processor

import (
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
)

func (mp *MessageProcessor) extractMessageContent(message InboundMessage) (*ProcessedMessage, error) {
	var messageText string
	var err error

	switch message.MessageType {
	case "text":
		messageText = message.Text
	case "audio":
		messageText, err = mp.transcribeAudio(message.Audio.URL)
		if err != nil {
			return nil, err
		}
	default:
		return nil, mp.handleUnsupportedMessageType(message)
	}

	finalMessageText := strings.TrimSpace(messageText)
	if finalMessageText == "" {
		log.Error().
			Str("message_uuid", message.MessageUUID).
			Msg("No text content found in message")
		return nil, errors.New("no text content found in message")
	}

	return &ProcessedMessage{
		Text: finalMessageText,
		UUID: message.MessageUUID,
	}, nil
}

func (mp *MessageProcessor) transcribeAudio(audioURL string) (string, error) {
	return mp.elevenLabsClient.TranscribeAudio(audioURL)
}

func (mp *MessageProcessor) handleUnsupportedMessageType(message InboundMessage) error {
	log.Warn().
		Str("message_type", message.MessageType).
		Str("message_uuid", message.MessageUUID).
		Msg("Unsupported message type")

	_, err := mp.vonageClient.SendWhatsAppReplyMessage(
		message.From,
		"I can't process this message type for now",
		message.MessageUUID,
	)
	return err
}
