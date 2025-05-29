package vonage

import (
	"fmt"
)

func (c *Client) MarkMessageAsRead(messageID string) error {
	payload := MarkAsReadPayload{Status: "read"}
	url := fmt.Sprintf("%s/%s", c.config.GeospecificMessagesAPIURL, messageID)

	_, err := c.sendRequest("PATCH", url, payload)
	return err
}
