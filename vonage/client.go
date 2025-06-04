package vonage

import (
	"net/http"
)

type Client struct {
	config     Config
	httpClient *http.Client
}

func NewClient(vonageJWT, geospecificMessagesAPIURL, messagesAPIURL, phoneNumberID string, httpClient http.Client) Client {
	client := Client{
		config: Config{
			VonageJWT:                 vonageJWT,
			GeospecificMessagesAPIURL: geospecificMessagesAPIURL,
			MessagesAPIURL:            messagesAPIURL,
			PhoneNumberID:             phoneNumberID,
		},
		httpClient: &httpClient,
	}

	return client
}
