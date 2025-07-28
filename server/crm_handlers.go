package server

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
)

// crmConversationsHandler handles GET /crm/conversations
func (s *Server) crmConversationsHandler(c fiber.Ctx) error {
	log.Info().Msg("Received CRM conversations request")

	// Get all conversation summaries from Redis
	summaries, err := s.messageProcessor.GetRedisClient().GetAllConversationSummaries()
	if err != nil {
		log.Error().Err(err).Msg("Error getting conversation summaries")
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve conversation summaries",
			},
		})
	}

	// Convert to API format and sort by last message time (newest first)
	var apiSummaries []ConversationSummary
	for _, summary := range summaries {
		apiSummaries = append(apiSummaries, ConversationSummary{
			UserID:             summary.UserID,
			LastMessageTime:    summary.LastMessageTime.Format("2006-01-02T15:04:05Z"),
			LastMessagePreview: summary.LastMessagePreview,
			MessageCount:       summary.MessageCount,
		})
	}

	// Sort by last message time (newest first)
	sort.Slice(apiSummaries, func(i, j int) bool {
		return apiSummaries[i].LastMessageTime > apiSummaries[j].LastMessageTime
	})

	return c.JSON(apiSummaries)
}

// crmConversationMessagesHandler handles GET /crm/conversations/{userId}
func (s *Server) crmConversationMessagesHandler(c fiber.Ctx) error {
	userID := c.Params("userId")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_PARAMETER",
				Message: "userId parameter is required",
			},
		})
	}

	log.Info().Str("user_id", userID).Msg("Received CRM conversation messages request")

	// Parse pagination parameters
	page := 1
	pageSize := 10

	if pageParam := c.Query("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeParam := c.Query("page_size"); pageSizeParam != "" {
		if ps, err := strconv.Atoi(pageSizeParam); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// Get paginated messages from Redis
	result, err := s.messageProcessor.GetRedisClient().GetChatHistoryPaginated(userID, page, pageSize)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Error getting paginated chat history")
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to retrieve conversation messages",
			},
		})
	}

	// Convert to API format
	var apiMessages []ConversationMessage
	for i, msg := range result.Messages {
		// Generate message ID (using index + timestamp for uniqueness)
		messageID := fmt.Sprintf("msg_%d_%d", msg.Timestamp.Unix(), i)
		if msg.MessageUUID != "" {
			messageID = msg.MessageUUID
		}

		// Convert role to sender format
		sender := "user"
		if msg.Role == "assistant" {
			sender = "system"
		}

		apiMessages = append(apiMessages, ConversationMessage{
			ID:        messageID,
			Timestamp: msg.Timestamp.Format("2006-01-02T15:04:05Z"),
			Content:   msg.Content,
			Sender:    sender,
		})
	}

	// Calculate pagination info
	hasNextPage := page < result.TotalPages
	hasPreviousPage := page > 1

	response := ConversationResponse{
		Messages:        apiMessages,
		TotalMessages:   result.TotalMessages,
		Page:            page,
		TotalPages:      result.TotalPages,
		HasNextPage:     hasNextPage,
		HasPreviousPage: hasPreviousPage,
	}

	return c.JSON(response)
}


// appMessageCountHandler trata POST /crm/messages-count
func (s *Server) appMessageCountHandler(c fiber.Ctx) error {
    log.Info().Msg("Received app message count request")

    var req AppMessageCountRequest

    log.Info().
        Str("application_id", req.ApplicationID).
        Str("month", req.Month).
        Int("count", req.Count).
        Msg("Stored monthly message count")

    return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success"})
}