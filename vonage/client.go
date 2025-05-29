package vonage

import (
	"net/http"

	"github.com/rs/zerolog/log"
)

type Client struct {
	config     Config
	httpClient *http.Client
}

func NewClient(vonageJWT, geospecificMessagesAPIURL, messagesAPIURL string, httpClient http.Client) Client {
	client := Client{
		config: Config{
			VonageJWT:                 vonageJWT,
			GeospecificMessagesAPIURL: geospecificMessagesAPIURL,
			MessagesAPIURL:            messagesAPIURL,
		},
		httpClient: &httpClient,
	}

	log.Info().
		Str("messages_api_url", messagesAPIURL).
		Str("geospecific_messages_api_url", geospecificMessagesAPIURL).
		Msg("Vonage client initialized successfully")

	return client
}
