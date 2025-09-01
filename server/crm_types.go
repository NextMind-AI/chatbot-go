package server

// ConversationSummary represents a conversation summary for the CRM API
type ConversationSummary struct {
	UserID             string `json:"user_id"`
	LastMessageTime    string `json:"last_message_time"`
	LastMessagePreview string `json:"last_message_preview"`
	MessageCount       int    `json:"message_count"`
}

// ConversationMessage represents a message in a conversation for the CRM API
type ConversationMessage struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Content   string `json:"content"`
	Sender    string `json:"sender"`
}

// ConversationResponse represents the paginated response for conversation messages
type ConversationResponse struct {
	Messages        []ConversationMessage `json:"messages"`
	TotalMessages   int                   `json:"total_messages"`
	Page            int                   `json:"page"`
	TotalPages      int                   `json:"total_pages"`
	HasNextPage     bool                  `json:"has_next_page"`
	HasPreviousPage bool                  `json:"has_previous_page"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error information
type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type AppMessageCountRequest struct {
    ApplicationID string `json:"application_id"`
    Month         string `json:"month"`          // no formato YYYY-MM
    Count         int    `json:"count"`
}
