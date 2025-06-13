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

// checkClientTool defines the tool for verifying if a client exists by phone number
var checkClientTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "check_cliente",
		Description: openai.String("Verifica se o cliente existe com base no número de telefone."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"phone_number": map[string]string{
					"type":        "string",
					"description": "Número de telefone do cliente no formato DDD+Número (ex: 11999998888)",
				},
			},
			"required": []string{"phone_number"},
		},
	},
}

// reagendarServicoTool define o tool para reagendar um serviço
var reagendarServicoTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "reagendar_servico",
		Description: openai.String("Altera a data e/ou hora de um agendamento existente."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"appointment_id": map[string]string{
					"type":        "string",
					"description": "ID do agendamento a ser reagendado",
				},
				"new_date": map[string]string{
					"type":        "string",
					"description": "Nova data no formato YYYY-MM-DD",
				},
				"new_time": map[string]string{
					"type":        "string",
					"description": "Novo horário no formato HH:MM",
				},
			},
			"required": []string{"appointment_id", "new_date", "new_time"},
		},
	},
}

var verificarHorariosDisponiveisTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "verificar_horarios_disponiveis",
		Description: openai.String("Retorna os horários disponíveis para um dado serviço em uma data específica."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"service_id": map[string]string{
					"type":        "string",
					"description": "ID do serviço para o qual checar horários",
				},
				"date": map[string]string{
					"type":        "string",
					"description": "Data no formato YYYY-MM-DD para verificar disponibilidade",
				},
			},
			"required": []string{"service_id", "date"},
		},
	},
}

var cadastrarClientesTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "cadastrar_clientes",
		Description: openai.String("Cadastra um novo cliente na base pelo nome e telefone."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]string{
					"type":        "string",
					"description": "Nome completo do cliente",
				},
				"phone_number": map[string]string{
					"type":        "string",
					"description": "Telefone no formato DDD+Número (ex: 11999998888)",
				},
				"email": map[string]string{
					"type":        "string",
					"description": "E-mail de contato (opcional)",
				},
			},
			"required": []string{"name", "phone_number"},
		},
	},
}

var agendamentosClienteTool = openai.ChatCompletionToolParam{
  Function: openai.FunctionDefinitionParam{
    Name:        "agendamentos_cliente",
    Description: openai.String("Retorna os agendamentos agendados para um cliente específico pelo ID."),
    Parameters: openai.FunctionParameters{
      "type": "object",
      "properties": map[string]any{
        "client_id": map[string]string{
          "type":        "string",
          "description": "ID do cliente para consulta de agendamentos",
        },
      },
      "required": []string{"client_id"},
    },
  },
}

// cancelarAgendamentoTool define o tool para cancelar um agendamento pelo ID
var cancelarAgendamentoTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "cancelar_agendamento",
		Description: openai.String("Cancela um agendamento existente pelo ID."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"appointment_id": map[string]string{
					"type":        "string",
					"description": "ID do agendamento a ser cancelado",
				},
			},
			"required": []string{"appointment_id"},
		},
	},
}

// CancelAppointmentRequest representa o payload de cancelamento
type CancelAppointmentRequest struct {
	AppointmentID string `json:"appointment_id"`
}

// RescheduleAppointmentRequest payload para reagendar
type RescheduleAppointmentRequest struct {
	AppointmentID string `json:"appointment_id"`
	NewDate       string `json:"new_date"`
	NewTime       string `json:"new_time"`
}

// RescheduleAppointmentResponse resposta do reagendamento
type RescheduleAppointmentResponse struct {
	AppointmentID string `json:"appointment_id"`
	Status        string `json:"status"`  // e.g. "rescheduled" ou "error"
	Message       string `json:"message"` // detalhes em caso de erro
}

// CancelAppointmentResponse representa a resposta do cancelamento
type CancelAppointmentResponse struct {
	AppointmentID string `json:"appointment_id"`
	Status        string `json:"status"`  // e.g. "cancelled" ou "error"
	Message       string `json:"message"` // detalhe em caso de erro
}

type ClientAppointmentsRequest struct {
  ClientID string `json:"client_id"`
}

type AppointmentItem struct {
  AppointmentID string `json:"appointment_id"`
  ServiceID     string `json:"service_id"`
  Date          string `json:"date"`  // YYYY-MM-DD
  Time          string `json:"time"`  // HH:MM
  Status        string `json:"status"`
}

type ClientAppointmentsResponse struct {
  ClientID     string            `json:"client_id"`
  Appointments []AppointmentItem `json:"appointments"`
}

type ClientCheckRequest struct {
	PhoneNumber string `json:"phone_number"`
}

type ClientCheckResponse struct {
	Exists     bool   `json:"exists"`
	ClientID   string `json:"client_id,omitempty"`
	ClientName string `json:"client_name,omitempty"`
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

// AppointmentRequest representa os dados necessários para criar um agendamento
type AppointmentRequest struct {
	ClientID   string `json:"client_id"`
	ServiceID  string `json:"service_id"`
	Date       string `json:"date"`        // formato: "2025-06-12"
	Time       string `json:"time"`        // formato: "14:30"
}

// AppointmentResponse representa a resposta do agendamento
type AppointmentResponse struct {
	AppointmentID string `json:"appointment_id"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}

type AvailableSlotsRequest struct {
	ServiceID string `json:"service_id"`
	Date      string `json:"date"` // formato: "2025-06-13"
}

type AvailableSlot struct {
	Time     string `json:"time"`      // ex: "09:00"
	Duration int    `json:"duration"`  // duração em minutos
}

type AvailableSlotsResponse struct {
	ServiceID string          `json:"service_id"`
	Date      string          `json:"date"`
	Slots     []AvailableSlot `json:"slots"`
}

// ClientRegisterRequest representa payload de cadastro de cliente
type ClientRegisterRequest struct {
	Name        string `json:"nome"`
	PhoneNumber string `json:"telefone"`
	Email       string `json:"email,omitempty"`
}

// ClientRegisterResponse representa resposta do cadastro
type ClientRegisterResponse struct {
	ClientID string `json:"id"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
}

// loadTrinksConfig loads Trinks API configuration
func loadTrinksConfig() (apiKey, estabelecimentoID, baseURL string) {
	cfg := config.Load()
	return cfg.TrinksAPIKey, cfg.TrinksEstabelecimentoID, cfg.TrinksBaseURL
}

// cancelAppointmentRequest dispara DELETE /agendamentos/{id} para cancelar
func (c *Client) cancelAppointmentRequest(
	ctx context.Context,
	appointmentID string,
) (*CancelAppointmentResponse, error) {
	apiKey, estabelecimentoID, baseURL := loadTrinksConfig()
	httpClient := &http.Client{Timeout: 10 * time.Second}

	url := baseURL + "/agendamentos/" + appointmentID
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("estabelecimentoId", estabelecimentoID)
	req.Header.Set("X-Api-Key", apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp CancelAppointmentResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	// Ajusta status caso HTTP >=400
	if resp.StatusCode >= 400 {
		apiResp.Status = "error"
		apiResp.Message = "Não foi possível cancelar: " + resp.Status
	} else {
		apiResp.Status = "cancelled"
	}

	apiResp.AppointmentID = appointmentID
	return &apiResp, nil
}

// processCancelarAgendamentoTool trata o chamado do AI para cancelar
func (c *Client) processCancelarAgendamentoTool(
	ctx context.Context,
	userID string,
	toolCall openai.ChatCompletionMessageToolCall,
) (openai.ChatCompletionMessageParamUnion, bool) {
	var req CancelAppointmentRequest
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &req); err != nil {
		log.Error().Err(err).
			Str("user_id", userID).
			Msg("Erro ao interpretar argumentos de cancelar_agendamento")
		return openai.ToolMessage("Parâmetros inválidos para cancelamento de agendamento", toolCall.ID), false
	}

	log.Info().
		Str("user_id", userID).
		Str("appointment_id", req.AppointmentID).
		Msg("Cancelando agendamento")

	resp, err := c.cancelAppointmentRequest(ctx, req.AppointmentID)
	if err != nil {
		log.Error().Err(err).
			Str("user_id", userID).
			Msg("Erro ao chamar API para cancelar agendamento")
		return openai.ToolMessage("Falha ao cancelar o agendamento", toolCall.ID), false
	}

	out, err := json.Marshal(resp)
	if err != nil {
		log.Error().Err(err).
			Msg("Erro ao serializar resposta de cancelamento")
		return openai.ToolMessage("Erro ao processar resposta de cancelamento", toolCall.ID), false
	}

	return openai.ToolMessage(string(out), toolCall.ID), true
}

// sendClientRegistrationRequest dispara POST /clientes para registrar
func (c *Client) sendClientRegistrationRequest(
	ctx context.Context,
	req ClientRegisterRequest,
) (*ClientRegisterResponse, error) {
	apiKey, estabelecimentoID, baseURL := loadTrinksConfig()
	httpClient := &http.Client{Timeout: 10 * time.Second}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		baseURL+"/clientes",
		strings.NewReader(string(payload)),
	)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("estabelecimentoId", estabelecimentoID)
	httpReq.Header.Set("X-Api-Key", apiKey)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp ClientRegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	// Caso haja erro HTTP, normalize o status
	if resp.StatusCode >= 400 {
		apiResp.Status = "error"
		apiResp.Message = "Falha ao cadastrar cliente: " + resp.Status
	}

	return &apiResp, nil
}

// processCadastralClienteTool recebe a chamada do AI e dispara o cadastro
func (c *Client) processCadastralClienteTool(
	ctx context.Context,
	userID string,
	toolCall openai.ChatCompletionMessageToolCall,
) (openai.ChatCompletionMessageParamUnion, bool) {
	var req ClientRegisterRequest
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &req); err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Erro ao interpretar argumentos de cadastrar_clientes")
		return openai.ToolMessage("Erro ao interpretar parâmetros de cadastro de cliente", toolCall.ID), false
	}

	log.Info().
		Str("user_id", userID).
		Str("name", req.Name).
		Str("phone", req.PhoneNumber).
		Msg("Cadastrando novo cliente")

	resp, err := c.sendClientRegistrationRequest(ctx, req)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Erro ao cadastrar cliente na API")
		return openai.ToolMessage("Erro ao cadastrar cliente", toolCall.ID), false
	}

	out, err := json.Marshal(resp)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Erro ao serializar resposta de cadastro")
		return openai.ToolMessage("Erro ao processar resposta do cadastro", toolCall.ID), false
	}

	return openai.ToolMessage(string(out), toolCall.ID), true
}

// sendAppointmentRequest envia os dados para a API da Trinks para criar o agendamento
func (c *Client) sendAppointmentRequest(ctx context.Context, req AppointmentRequest) (*AppointmentResponse, error) {
	apiKey, estabelecimentoID, baseURL := loadTrinksConfig()
	client := &http.Client{Timeout: 10 * time.Second}

	// Corpo da requisição (ajuste conforme o formato exigido pela Trinks)
	payload := map[string]any{
		"cliente_id":    req.ClientID,
		"servico_id":    req.ServiceID,
		"data":          req.Date,
		"hora":          req.Time,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/agendamentos", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("estabelecimentoId", estabelecimentoID)
	httpReq.Header.Set("X-Api-Key", apiKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp AppointmentResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	// Tratamento caso ocorra erro com status HTTP
	if resp.StatusCode >= 400 {
		apiResp.Status = "error"
		apiResp.Message = "Erro ao criar agendamento: " + resp.Status
	}

	return &apiResp, nil
}

// processFazerAgendamentoTool agenda um serviço para um cliente
func (c *Client) processFazerAgendamentoTool(
	ctx context.Context,
	userID string,
	toolCall openai.ChatCompletionMessageToolCall,
) (openai.ChatCompletionMessageParamUnion, bool) {
	var request AppointmentRequest
	err := json.Unmarshal([]byte(toolCall.Function.Arguments), &request)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Erro ao interpretar os dados do agendamento")
		return openai.ToolMessage("Erro ao interpretar os dados do agendamento", toolCall.ID), false
	}

	log.Info().
		Str("user_id", userID).
		Str("client_id", request.ClientID).
		Str("service_id", request.ServiceID).
		Str("date", request.Date).
		Str("time", request.Time).
		Msg("Processando agendamento")

	response, err := c.sendAppointmentRequest(ctx, request)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Erro ao tentar agendar")
		return openai.ToolMessage("Erro ao tentar agendar o serviço", toolCall.ID), false
	}

	respJSON, err := json.Marshal(response)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Erro ao serializar resposta de agendamento")
		return openai.ToolMessage("Erro ao processar retorno do agendamento", toolCall.ID), false
	}

	return openai.ToolMessage(string(respJSON), toolCall.ID), true
}

func (c *Client) fetchAppointmentsByClient(
	ctx context.Context,
	clientID string,
) (*ClientAppointmentsResponse, error) {
	apiKey, estabelecimentoID, baseURL := loadTrinksConfig()
	httpClient := &http.Client{Timeout: 10 * time.Second}
	// Monta URL: /agendamentos?cliente_id={clientID}
	url := baseURL + "/agendamentos?cliente_id=" + clientID
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("estabelecimentoId", estabelecimentoID)
	req.Header.Set("X-Api-Key", apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiData struct {
		Data []struct {
		ID        string `json:"id"`
		ServicoID string `json:"servico_id"`
		Data      string `json:"data"`
		Hora      string `json:"hora"`
		Status    string `json:"status"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiData); err != nil {
		return nil, err
	}

	items := make([]AppointmentItem, len(apiData.Data))
	for i, a := range apiData.Data {
		items[i] = AppointmentItem{
		AppointmentID: a.ID,
		ServiceID:     a.ServicoID,
		Date:          a.Data,
		Time:          a.Hora,
		Status:        a.Status,
		}
	}
	return &ClientAppointmentsResponse{
		ClientID:     clientID,
		Appointments: items,
	}, nil
}

func (c *Client) processAgendamentoClienteTool(
	ctx context.Context,
	userID string,
	toolCall openai.ChatCompletionMessageToolCall,
) (openai.ChatCompletionMessageParamUnion, bool) {
	var req ClientAppointmentsRequest
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &req); err != nil {
		log.Error().Err(err).Str("user_id", userID).
		Msg("Erro ao interpretar argumentos de agendamentos_cliente")
		return openai.ToolMessage("Erro nos parâmetros de consulta de agendamentos", toolCall.ID), false
	}

	log.Info().Str("user_id", userID).Str("client_id", req.ClientID).
		Msg("Buscando agendamentos do cliente")

	resp, err := c.fetchAppointmentsByClient(ctx, req.ClientID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).
		Msg("Erro ao buscar agendamentos do cliente")
		return openai.ToolMessage("Erro ao consultar agendamentos do cliente", toolCall.ID), false
	}

	out, err := json.Marshal(resp)
	if err != nil {
		log.Error().Err(err).
		Msg("Erro ao serializar resposta de agendamentos")
		return openai.ToolMessage("Erro ao processar resposta de agendamentos", toolCall.ID), false
	}
	return openai.ToolMessage(string(out), toolCall.ID), true
}

// fetchAvailableSlots consulta a API Trinks pelos horários livres
func (c *Client) fetchAvailableSlots(
	ctx context.Context,
	req AvailableSlotsRequest,
) (*AvailableSlotsResponse, error) {
	apiKey, estabelecimentoID, baseURL := loadTrinksConfig()
	httpClient := &http.Client{Timeout: 10 * time.Second}

	// Monta URL: /servicos/{service_id}/disponibilidade?data=YYYY-MM-DD
	url := baseURL + "/servicos/" + req.ServiceID + "/disponibilidade?data=" + req.Date
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("accept", "application/json")
	httpReq.Header.Set("estabelecimentoId", estabelecimentoID)
	httpReq.Header.Set("X-Api-Key", apiKey)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Estrutura de resposta esperada da API
	var apiData struct {
		Data []struct {
			Hora     string `json:"hora"`     // ex: "09:00"
			Duracao  int    `json:"duracao"`  // em minutos
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiData); err != nil {
		return nil, err
	}

	// Converte para AvailableSlotsResponse
	slots := make([]AvailableSlot, len(apiData.Data))
	for i, s := range apiData.Data {
		slots[i] = AvailableSlot{
			Time:     s.Hora,
			Duration: s.Duracao,
		}
	}

	return &AvailableSlotsResponse{
		ServiceID: req.ServiceID,
		Date:      req.Date,
		Slots:     slots,
	}, nil
}

// processVerificarHorarioDisponivelTool obtém os horários disponíveis de um serviço em determinada data
func (c *Client) processVerificarHorarioDisponivelTool(
	ctx context.Context,
	userID string,
	toolCall openai.ChatCompletionMessageToolCall,
) (openai.ChatCompletionMessageParamUnion, bool) {
	var req AvailableSlotsRequest
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &req); err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Erro ao interpretar argumentos de verificar_horarios_disponiveis")
		return openai.ToolMessage("Erro ao interpretar parâmetros de verificação de horários", toolCall.ID), false
	}

	log.Info().
		Str("user_id", userID).
		Str("service_id", req.ServiceID).
		Str("date", req.Date).
		Msg("Verificando horários disponíveis")

	resp, err := c.fetchAvailableSlots(ctx, req)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Erro ao buscar horários disponíveis")
		return openai.ToolMessage("Erro ao consultar horários disponíveis", toolCall.ID), false
	}

	out, err := json.Marshal(resp)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Erro ao serializar resposta de horários disponíveis")
		return openai.ToolMessage("Erro ao processar resposta de disponibilidade", toolCall.ID), false
	}

	return openai.ToolMessage(string(out), toolCall.ID), true
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

func (c *Client) fetchClientByPhone(ctx context.Context, phoneNumber string) (*ClientCheckResponse, error) {
	apiKey, estabelecimentoID, baseURL := loadTrinksConfig()
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/clientes?telefone="+phoneNumber, nil)
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

	if resp.StatusCode == http.StatusNotFound {
		return &ClientCheckResponse{Exists: false}, nil
	}

	var apiResponse struct {
		ID    string `json:"id"`
		Nome  string `json:"nome"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, err
	}

	return &ClientCheckResponse{
		Exists:     true,
		ClientID:   apiResponse.ID,
		ClientName: apiResponse.Nome,
	}, nil
}

func (c *Client) processCheckClientTool(
	ctx context.Context,
	userID string,
	toolCall openai.ChatCompletionMessageToolCall,
) (openai.ChatCompletionMessageParamUnion, bool) {
	var request ClientCheckRequest
	err := json.Unmarshal([]byte(toolCall.Function.Arguments), &request)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Error parsing check_cliente arguments")
		return openai.ToolMessage("Erro ao interpretar os dados do cliente", toolCall.ID), false
	}

	log.Info().
		Str("user_id", userID).
		Str("phone_number", request.PhoneNumber).
		Msg("Checking if client exists")

	response, err := c.fetchClientByPhone(ctx, request.PhoneNumber)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Erro ao buscar cliente")
		return openai.ToolMessage("Erro ao buscar informações do cliente", toolCall.ID), false
	}

	respJSON, err := json.Marshal(response)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Erro ao serializar resposta do cliente")
		return openai.ToolMessage("Erro ao processar dados do cliente", toolCall.ID), false
	}

	return openai.ToolMessage(string(respJSON), toolCall.ID), true
}

// rescheduleAppointmentRequest faz PATCH /agendamentos/{id} com nova data/hora
func (c *Client) rescheduleAppointmentRequest(
	ctx context.Context,
	reqData RescheduleAppointmentRequest,
) (*RescheduleAppointmentResponse, error) {
	apiKey, estabelecimentoID, baseURL := loadTrinksConfig()
	httpClient := &http.Client{Timeout: 10 * time.Second}

	// Monta endpoint: /agendamentos/{id}
	url := baseURL + "/agendamentos/" + reqData.AppointmentID

	// Payload com novas informações
	payload := map[string]any{
		"data": reqData.NewDate,
		"hora": reqData.NewTime,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "PATCH", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("estabelecimentoId", estabelecimentoID)
	httpReq.Header.Set("X-Api-Key", apiKey)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp RescheduleAppointmentResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		apiResp.Status = "error"
		apiResp.Message = "Não foi possível reagendar: " + resp.Status
	} else {
		apiResp.Status = "rescheduled"
	}
	apiResp.AppointmentID = reqData.AppointmentID
	return &apiResp, nil
}

// processReagendarServicoTool trata a chamada do AI para reagendar
func (c *Client) processReagendarServicoTool(
	ctx context.Context,
	userID string,
	toolCall openai.ChatCompletionMessageToolCall,
) (openai.ChatCompletionMessageParamUnion, bool) {
	var req RescheduleAppointmentRequest
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &req); err != nil {
		log.Error().Err(err).
			Str("user_id", userID).
			Msg("Erro ao interpretar argumentos de reagendar_servico")
		return openai.ToolMessage("Parâmetros inválidos para reagendamento", toolCall.ID), false
	}

	log.Info().
		Str("user_id", userID).
		Str("appointment_id", req.AppointmentID).
		Str("new_date", req.NewDate).
		Str("new_time", req.NewTime).
		Msg("Reagendando serviço")

	resp, err := c.rescheduleAppointmentRequest(ctx, req)
	if err != nil {
		log.Error().Err(err).
			Str("user_id", userID).
			Msg("Erro ao chamar API para reagendar")
		return openai.ToolMessage("Falha ao reagendar o serviço", toolCall.ID), false
	}

	out, err := json.Marshal(resp)
	if err != nil {
		log.Error().Err(err).
			Msg("Erro ao serializar resposta de reagendamento")
		return openai.ToolMessage("Erro ao processar resposta de reagendamento", toolCall.ID), false
	}

	return openai.ToolMessage(string(out), toolCall.ID), true
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
					toolMessage = openai.ToolMessage("Error processing service information", toolCall.ID)
					log.Error().
						Str("user_id", userID).
						Msg("Failed to process check_services tool call")
				}
				messages = append(messages, toolMessage)
			case "check_cliente":
				toolMessage, success := c.processCheckClientTool(ctx, userID, toolCall)
				if !success {
					toolMessage = openai.ToolMessage("Error processing service information", toolCall.ID)
					log.Error().
						Str("user_id", userID).
						Msg("Failed to process check_cliente tool call")
				}
				messages = append(messages, toolMessage)
			case "fazer_agendamento":
				toolMessage, success := c.processFazerAgendamentoTool(ctx, userID, toolCall)
				if !success {
					toolMessage = openai.ToolMessage("Error processing service information", toolCall.ID)
					log.Error().
						Str("user_id", userID).
						Msg("Failed to process fazer_agendamento tool call")
				}
				messages = append(messages, toolMessage)
			case "verificar_horarios_disponiveis": 
				toolMessage, success := c.processVerificarHorarioDisponivelTool(ctx, userID, toolCall)
				if !success {
					toolMessage = openai.ToolMessage("Error processing service information", toolCall.ID)
					log.Error().
						Str("user_id", userID).
						Msg("Failed to process verificar_horarios_disponiveis tool call")
				}
				messages = append(messages, toolMessage)
			case "cadastrar_clientes": 
				toolMessage, success := c.processCadastralClienteTool(ctx, userID, toolCall)
				if !success {
					toolMessage = openai.ToolMessage("Error processing service information", toolCall.ID)
					log.Error().
						Str("user_id", userID).
						Msg("Failed to process cadastrar_clientes tool call")
				}
				messages = append(messages, toolMessage)
			case "agendamentos_cliente": 
				toolMessage, success := c.processAgendamentoClienteTool(ctx, userID, toolCall)
				if !success {
					toolMessage = openai.ToolMessage("Error processing service information", toolCall.ID)
					log.Error().
						Str("user_id", userID).
						Msg("Failed to process agendamentos_cliente tool call")
				}
				messages = append(messages, toolMessage)
			case "cancelar_agendamento": 
				toolMessage, success := c.processCancelarAgendamentoTool(ctx, userID, toolCall)
				if !success {
					toolMessage = openai.ToolMessage("Error processing service information", toolCall.ID)
					log.Error().
						Str("user_id", userID).
						Msg("Failed to process cancelar_agendamento tool call")
				}
				messages = append(messages, toolMessage)
			case "reagendar_servico": 
				toolMessage, success := c.processReagendarServicoTool(ctx, userID, toolCall)
				if !success {
					toolMessage = openai.ToolMessage("Error processing service information", toolCall.ID)
					log.Error().
						Str("user_id", userID).
						Msg("Failed to process reagendar_servico tool call")
				}
				messages = append(messages, toolMessage)
		}
	}
	return messages, nil
}
