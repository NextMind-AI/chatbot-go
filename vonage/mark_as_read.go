package vonage

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

func (c *Client) MarkMessageAsRead(messageID string) error {
	log.Debug().Str("message_id", messageID).Msg("Marking message as read")

	payload := MarkAsReadPayload{Status: "read"}
	url := fmt.Sprintf("%s/%s", c.config.GeospecificMessagesAPIURL, messageID)

	_, err := c.sendRequest("PATCH", url, payload)
	return err
}
