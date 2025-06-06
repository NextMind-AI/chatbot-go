package openai

import (
	"chatbot/config"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/rs/zerolog/log"
)

// sleepTool defines the sleep tool that allows the AI to pause conversation for a specified duration.
// This tool can be used when the AI needs to simulate waiting or processing time.
var sleepTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "sleep",
		Description: openai.String("Wait for a specified number of seconds before continuing the conversation"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"seconds": map[string]string{
					"type":        "integer",
					"description": "Number of seconds to wait",
				},
			},
			"required": []string{"seconds"},
		},
	},
}

// checkServicesTool defines the tool for checking available services at the salon
var checkServicesTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "check_services",
		Description: openai.String("Search and filter available salon services by category, name, or general inquiries. Use when customer asks about services, treatments, or wants to know what's available."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"search_term": map[string]string{
					"type":        "string",
					"description": "Specific service name or treatment to search for (e.g., 'corte', 'barba', 'sobrancelha')",
				},
				"category": map[string]string{
					"type":        "string",
					"description": "Service category to filter by (e.g., 'Cabelo', 'Barba', 'Sobrancelha')",
				},
				"query_type": map[string]string{
					"type":        "string",
					"description": "Type of query: 'specific' for exact service lookup, 'category' for category browsing, 'general' for overview of all services",
					"enum":        "[\"specific\", \"category\", \"general\"]",
				},
			},
			"required": []string{"query_type"},
		},
	},
}

// ServiceSearchRequest represents the structure for service search
type ServiceSearchRequest struct {
	SearchTerm string `json:"search_term,omitempty"`
	Category   string `json:"category,omitempty"`
	QueryType  string `json:"query_type"`
}

// ServiceSearchResponse represents the response from service search
type ServiceSearchResponse struct {
	Services         []ServiceInfo            `json:"services"`
	Categories       []string                 `json:"categories"`
	TotalServices    int                      `json:"total_services"`
	SearchPerformed  bool                     `json:"search_performed"`
	CategorySummary  map[string]CategoryInfo  `json:"category_summary,omitempty"`
}

// ServiceInfo represents individual service information
type ServiceInfo struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	Duration    int     `json:"duration"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	Visible     bool    `json:"visible"`
}

// CategoryInfo represents category summary information
type CategoryInfo struct {
	Count       int     `json:"count"`
	AvgPrice    float64 `json:"avg_price"`
	AvgDuration int     `json:"avg_duration"`
}

// loadTrinksConfig loads Trinks API configuration
func loadTrinksConfig() (apiKey, estabelecimentoID, baseURL string) {
	cfg := config.Load()
	return cfg.TrinksAPIKey, cfg.TrinksEstabelecimentoID, cfg.TrinksBaseURL
}

// processSleepTool processes a sleep tool call from the AI.
// It parses the arguments, executes the sleep operation, and returns the result.
// Returns a tool message and a success flag indicating whether the operation completed successfully.
func (c *Client) processSleepTool(
	ctx context.Context,
	userID string,
	toolCall openai.ChatCompletionMessageToolCall,
) (openai.ChatCompletionMessageParamUnion, bool) {
	var args map[string]any
	err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error parsing sleep function arguments")
		return openai.ToolMessage("", ""), false
	}

	seconds, ok := args["seconds"].(float64)
	if !ok {
		log.Error().
			Str("user_id", userID).
			Msg("Invalid seconds parameter for sleep function")
		return openai.ToolMessage("", ""), false
	}

	log.Info().
		Str("user_id", userID).
		Float64("seconds", seconds).
		Msg("Sleeping before continuing conversation")

	sleepDuration := time.Duration(seconds) * time.Second
	select {
	case <-time.After(sleepDuration):
	case <-ctx.Done():
		log.Info().
			Str("user_id", userID).
			Msg("Sleep cancelled due to context cancellation")
		return openai.ToolMessage("", ""), false
	}

	return openai.ToolMessage("Sleep completed", toolCall.ID), true
}

// processCheckServicesTool processes a service search tool call from the AI.
// It parses the arguments, fetches the service data from the API, and returns the result.
// Returns a tool message and a success flag indicating whether the operation completed successfully.
func (c *Client) processCheckServicesTool(
	ctx context.Context,
	userID string,
	toolCall openai.ChatCompletionMessageToolCall,
) (openai.ChatCompletionMessageParamUnion, bool) {
	var request ServiceSearchRequest
	err := json.Unmarshal([]byte(toolCall.Function.Arguments), &request)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error parsing check_services function arguments")
		return openai.ToolMessage("Error parsing service search request", toolCall.ID), false
	}

	log.Info().
		Str("user_id", userID).
		Str("query_type", request.QueryType).
		Str("search_term", request.SearchTerm).
		Str("category", request.Category).
		Msg("Processing service search request")

	// Call Trinks API to get services
	response, err := c.fetchServicesFromAPI(ctx, request)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error fetching services from API")
		return openai.ToolMessage("Error fetching services information", toolCall.ID), false
	}

	// Convert response to JSON for the AI
	responseJSON, err := json.Marshal(response)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error marshaling service response")
		return openai.ToolMessage("Error processing service information", toolCall.ID), false
	}

	return openai.ToolMessage(string(responseJSON), toolCall.ID), true
}

// fetchServicesFromAPI calls the Trinks API to get service information
func (c *Client) fetchServicesFromAPI(ctx context.Context, request ServiceSearchRequest) (*ServiceSearchResponse, error) {
	// Load config directly in this function
	apiKey, estabelecimentoID, baseURL := loadTrinksConfig()

	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/servicos", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("estabelecimentoId", estabelecimentoID)
	req.Header.Set("X-Api-Key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResponse struct {
		Data []struct {
			ID                   string  `json:"id"`
			Nome                 string  `json:"nome"`
			Categoria            string  `json:"categoria"`
			DuracaoEmMinutos     int     `json:"duracaoEmMinutos"`
			Preco                float64 `json:"preco"`
			Descricao            string  `json:"descricao"`
			VisivelParaCliente   bool    `json:"visivelParaCliente"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, err
	}

	// Process and filter the response based on request
	return c.processServiceData(apiResponse.Data, request), nil
}

// processServiceData processes the raw API data and applies filtering
func (c *Client) processServiceData(rawData []struct {
	ID                   string  `json:"id"`
	Nome                 string  `json:"nome"`
	Categoria            string  `json:"categoria"`
	DuracaoEmMinutos     int     `json:"duracaoEmMinutos"`
	Preco                float64 `json:"preco"`
	Descricao            string  `json:"descricao"`
	VisivelParaCliente   bool    `json:"visivelParaCliente"`
}, request ServiceSearchRequest) *ServiceSearchResponse {

	var filteredServices []ServiceInfo
	categories := make(map[string]bool)
	categorySummary := make(map[string]CategoryInfo)

	// Convert raw data to ServiceInfo and apply filtering
	for _, service := range rawData {
		serviceInfo := ServiceInfo{
			ID:          service.ID,
			Name:        service.Nome,
			Category:    service.Categoria,
			Duration:    service.DuracaoEmMinutos,
			Price:       service.Preco,
			Description: service.Descricao,
			Visible:     service.VisivelParaCliente,
		}

		categories[service.Categoria] = true

		// Update category summary
		if summary, exists := categorySummary[service.Categoria]; exists {
			summary.Count++
			summary.AvgPrice = (summary.AvgPrice*(float64(summary.Count-1)) + service.Preco) / float64(summary.Count)
			summary.AvgDuration = (summary.AvgDuration*(summary.Count-1) + service.DuracaoEmMinutos) / summary.Count
			categorySummary[service.Categoria] = summary
		} else {
			categorySummary[service.Categoria] = CategoryInfo{
				Count:       1,
				AvgPrice:    service.Preco,
				AvgDuration: service.DuracaoEmMinutos,
			}
		}

		// Apply filtering based on request type
		include := false
		switch request.QueryType {
		case "general":
			include = true
		case "category":
			if request.Category != "" {
				include = strings.Contains(strings.ToLower(service.Categoria), strings.ToLower(request.Category))
			} else {
				include = true
			}
		case "specific":
			if request.SearchTerm != "" {
				searchTerm := strings.ToLower(request.SearchTerm)
				serviceName := strings.ToLower(service.Nome)
				include = strings.Contains(serviceName, searchTerm)
			} else {
				include = true
			}
		}

		if include {
			filteredServices = append(filteredServices, serviceInfo)
		}
	}

	// Convert categories map to slice
	var categoryList []string
	for category := range categories {
		categoryList = append(categoryList, category)
	}

	response := &ServiceSearchResponse{
		Services:        filteredServices,
		Categories:      categoryList,
		TotalServices:   len(filteredServices),
		SearchPerformed: true,
		CategorySummary: categorySummary,
	}

	// Limit results for general queries to avoid overwhelming response
	if request.QueryType == "general" && len(filteredServices) > 10 {
		response.Services = filteredServices[:10]
		response.TotalServices = len(rawData) // Keep total count of all services
	}

	return response
}

// handleToolCalls processes all tool calls from the AI's response.
// It iterates through the tool calls, executes them, and appends the results to the message history.
// Currently supports the sleep and check_services tools, but can be extended to handle other tools.
func (c *Client) handleToolCalls(
	ctx context.Context,
	userID string,
	messages []openai.ChatCompletionMessageParamUnion,
	toolCalls []openai.ChatCompletionMessageToolCall,
) ([]openai.ChatCompletionMessageParamUnion, error) {
	for _, toolCall := range toolCalls {
		switch toolCall.Function.Name {
		case "sleep":
			toolMessage, success := c.processSleepTool(ctx, userID, toolCall)
			if !success {
				continue
			}
			messages = append(messages, toolMessage)
		case "check_services":
			toolMessage, success := c.processCheckServicesTool(ctx, userID, toolCall)
			if !success {
				continue
			}
			messages = append(messages, toolMessage)
		}
	}
	return messages, nil
}
