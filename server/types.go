package server

import (
	"chatbot/redis"
	"time"
)

// ConversationSummary represents a summary of a conversation for CRM listing
type ConversationSummary struct {
	UserID          string    `json:"user_id"`
	LastMessage     string    `json:"last_message"`
	LastMessageTime time.Time `json:"last_message_time"`
	MessageCount    int       `json:"message_count"`
	LastRole        string    `json:"last_role"` // "user" or "assistant"
}

// ConversationResponse represents the full conversation history response
type ConversationResponse struct {
	UserID      string              `json:"user_id"`
	Messages    []redis.ChatMessage `json:"messages"`
	TotalCount  int                 `json:"total_count"`
	Page        int                 `json:"page"`
	PageSize    int                 `json:"page_size"`
	TotalPages  int                 `json:"total_pages"`
	HasNext     bool                `json:"has_next"`
	HasPrevious bool                `json:"has_previous"`
}

// PaginationQuery represents query parameters for pagination
type PaginationQuery struct {
	Page      int    `query:"page"`
	PageSize  int    `query:"page_size"`
	StartDate string `query:"start_date"`
	EndDate   string `query:"end_date"`
}
