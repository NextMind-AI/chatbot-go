package processor

type InboundMessage struct {
	Channel       string  `json:"channel"`
	ContextStatus string  `json:"context_status"`
	From          string  `json:"from"`
	MessageType   string  `json:"message_type"`
	MessageUUID   string  `json:"message_uuid"`
	Profile       Profile `json:"profile"`
	Text          string  `json:"text"`
	Timestamp     string  `json:"timestamp"`
	To            string  `json:"to"`
	Audio         *Audio  `json:"audio,omitempty"`
}

type Profile struct {
	Name string `json:"name"`
}

type Audio struct {
	URL string `json:"url"`
}

type ProcessedMessage struct {
	Text string
	UUID string
}
