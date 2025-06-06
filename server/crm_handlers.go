package server

import (
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
)

// listConversationsHandler returns a list of all active conversations
func (s *Server) listConversationsHandler(c fiber.Ctx) error {
	log.Info().Msg("Received request to list conversations")

	// Get Redis client from message processor
	redisClient := s.messageProcessor.GetRedisClient()

	// Get all active conversations
	userIDs, err := redisClient.GetAllActiveConversations()
	if err != nil {
		log.Error().Err(err).Msg("Error getting active conversations")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve conversations",
		})
	}

	var summaries []ConversationSummary

	// Get summary for each conversation
	for _, userID := range userIDs {
		chatHistory, err := redisClient.GetChatHistory(userID)
		if err != nil {
			log.Error().Err(err).Str("user_id", userID).Msg("Error getting chat history")
			continue
		}

		if len(chatHistory) == 0 {
			continue
		}

		// Get last message
		lastMessage := chatHistory[len(chatHistory)-1]

		summary := ConversationSummary{
			UserID:          userID,
			LastMessage:     lastMessage.Content,
			LastMessageTime: lastMessage.Timestamp,
			MessageCount:    len(chatHistory),
			LastRole:        lastMessage.Role,
		}

		summaries = append(summaries, summary)
	}

	// Sort by last message time (most recent first)
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].LastMessageTime.After(summaries[j].LastMessageTime)
	})

	log.Info().Int("conversations_count", len(summaries)).Msg("Successfully retrieved conversations")
	return c.JSON(summaries)
}

// getConversationHandler returns the conversation history for a specific user
func (s *Server) getConversationHandler(c fiber.Ctx) error {
	userID := c.Params("user_id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User ID is required",
		})
	}

	log.Info().Str("user_id", userID).Msg("Received request to get conversation")

	// Parse query parameters
	page := 1
	pageSize := 20

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

	// Parse date filters
	var startTime, endTime *time.Time

	if startDateParam := c.Query("start_date"); startDateParam != "" {
		if t, err := time.Parse(time.RFC3339, startDateParam); err == nil {
			startTime = &t
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid start_date format. Use ISO format (e.g., 2025-05-01T00:00:00Z)",
			})
		}
	}

	if endDateParam := c.Query("end_date"); endDateParam != "" {
		if t, err := time.Parse(time.RFC3339, endDateParam); err == nil {
			endTime = &t
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid end_date format. Use ISO format (e.g., 2025-05-13T23:59:59Z)",
			})
		}
	}

	// Get Redis client
	redisClient := s.messageProcessor.GetRedisClient()

	// Get paginated chat history
	messages, totalCount, err := redisClient.GetChatHistoryWithPagination(userID, page, pageSize, startTime, endTime)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Error getting chat history")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve conversation history",
		})
	}

	// Calculate pagination info
	totalPages := int(math.Ceil(float64(totalCount) / float64(pageSize)))
	hasNext := page < totalPages
	hasPrevious := page > 1

	response := ConversationResponse{
		UserID:      userID,
		Messages:    messages,
		TotalCount:  totalCount,
		Page:        page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		HasNext:     hasNext,
		HasPrevious: hasPrevious,
	}

	log.Info().
		Str("user_id", userID).
		Int("total_count", totalCount).
		Int("page", page).
		Int("page_size", pageSize).
		Msg("Successfully retrieved conversation history")

	return c.JSON(response)
}
