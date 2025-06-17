package openai

import (
	"chatbot/trinks"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/rs/zerolog/log"
)

// ============================================================================
// DEFINIÇÕES DAS TOOLS
// ============================================================================

var checkServicesTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "check_services",
		Description: openai.String("Lista todos os serviços disponíveis organizados por categoria."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"categoria_filtro": map[string]any{
					"type":        "string",
					"description": "Categoria para filtrar (opcional). Ex: 'Cabelo', 'Barba'",
				},
				"mostrar_resumo": map[string]any{
					"type":        "boolean",
					"description": "Se deve incluir resumo estatístico por categoria (padrão: true)",
					"default":     true,
				},
			},
			"required": []string{},
		},
	},
}

var registerClientTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "register_client",
		Description: openai.String("Cadastra um novo cliente no sistema da barbearia."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "Nome completo do cliente",
				},
				"email": map[string]any{
					"type":        "string",
					"description": "E-mail do cliente (deve ser único)",
				},
				"phone": map[string]any{
					"type":        "string",
					"description": "Telefone com DDD (ex: 11987654321)",
				},
			},
			"required": []string{"name", "email", "phone"},
		},
	},
}

// Adicionar essas tools junto com as existentes (checkServicesTool e registerClientTool)

var checkClientTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "check_cliente",
		Description: openai.String("Verifica se o cliente existe com base no número de telefone."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"phone_number": map[string]any{
					"type":        "string",
					"description": "Número de telefone do cliente no formato DDD+Número (ex: 11999998888)",
				},
			},
			"required": []string{"phone_number"},
		},
	},
}

var fazerAgendamentoTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "fazer_agendamento",
		Description: openai.String("Cria um novo agendamento para um cliente."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"client_id": map[string]any{
					"type":        "string",
					"description": "ID do cliente",
				},
				"service_id": map[string]any{
					"type":        "string",
					"description": "ID do serviço a ser agendado",
				},
				"date": map[string]any{
					"type":        "string",
					"description": "Data do agendamento no formato YYYY-MM-DD",
				},
				"time": map[string]any{
					"type":        "string",
					"description": "Horário do agendamento no formato HH:MM",
				},
			},
			"required": []string{"client_id", "service_id", "date", "time"},
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
				"service_id": map[string]any{
					"type":        "string",
					"description": "ID do serviço para o qual checar horários",
				},
				"date": map[string]any{
					"type":        "string",
					"description": "Data no formato YYYY-MM-DD para verificar disponibilidade",
				},
			},
			"required": []string{"service_id", "date"},
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
				"client_id": map[string]any{
					"type":        "string",
					"description": "ID do cliente para consulta de agendamentos",
				},
			},
			"required": []string{"client_id"},
		},
	},
}

var cancelarAgendamentoTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "cancelar_agendamento",
		Description: openai.String("Cancela um agendamento existente pelo ID."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"appointment_id": map[string]any{
					"type":        "string",
					"description": "ID do agendamento a ser cancelado",
				},
			},
			"required": []string{"appointment_id"},
		},
	},
}

var reagendarServicoTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "reagendar_servico",
		Description: openai.String("Altera a data e/ou hora de um agendamento existente."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"appointment_id": map[string]any{
					"type":        "string",
					"description": "ID do agendamento a ser reagendado",
				},
				"new_date": map[string]any{
					"type":        "string",
					"description": "Nova data no formato YYYY-MM-DD",
				},
				"new_time": map[string]any{
					"type":        "string",
					"description": "Novo horário no formato HH:MM",
				},
			},
			"required": []string{"appointment_id", "new_date", "new_time"},
		},
	},
}

// ============================================================================
// ESTRUTURAS DE DADOS
// ============================================================================

// Serviços e categorias
type ServiceSearchRequest struct {
	CategoriaFiltro string `json:"categoria_filtro,omitempty"`
	MostrarResumo   bool   `json:"mostrar_resumo"`
}

type ServiceSearchResponse struct {
	ServicosPorCategoria  map[string][]ServiceInfo   `json:"servicos_por_categoria"`
	ResumoCategoria       map[string]CategorySummary `json:"resumo_categoria,omitempty"`
	TotalServicos         int                        `json:"total_servicos"`
	CategoriasDisponiveis []string                   `json:"categorias_disponiveis"`
}

type ServiceInfo struct {
	ID        string  `json:"id"`
	Nome      string  `json:"nome"`
	Descricao string  `json:"descricao"`
	Duracao   int     `json:"duracao"`
	Preco     float64 `json:"preco"`
	Visivel   bool    `json:"visivel"`
}

type CategorySummary struct {
	Quantidade   int     `json:"quantidade"`
	PrecoMedio   float64 `json:"preco_medio"`
	DuracaoMedia int     `json:"duracao_media"`
}

// Cadastro de cliente
type ClientRegisterRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type ClientRegisterResponse struct {
	Success   bool   `json:"success"`
	ClientID  int    `json:"client_id,omitempty"`
	Message   string `json:"message"`
	ErrorCode string `json:"error_code,omitempty"`
}

// AppointmentRequest representa os dados necessários para criar um agendamento
type AppointmentRequest struct {
	ClientID  string `json:"client_id"`
	ServiceID string `json:"service_id"`
	Date      string `json:"date"` // formato: "2025-06-12"
	Time      string `json:"time"` // formato: "14:30"
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
	Time     string `json:"time"`     // ex: "09:00"
	Duration int    `json:"duration"` // duração em minutos
}

type AvailableSlotsResponse struct {
	ServiceID string          `json:"service_id"`
	Date      string          `json:"date"`
	Slots     []AvailableSlot `json:"slots"`
}

type ClientCheckRequest struct {
	PhoneNumber string `json:"phone_number"`
}

type ClientCheckResponse struct {
	Exists     bool   `json:"exists"`
	ClientID   string `json:"client_id,omitempty"`
	ClientName string `json:"client_name,omitempty"`
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

type CancelAppointmentRequest struct {
	AppointmentID string `json:"appointment_id"`
}

type CancelAppointmentResponse struct {
	AppointmentID string `json:"appointment_id"`
	Status        string `json:"status"`  // e.g. "cancelled" ou "error"
	Message       string `json:"message"` // detalhe em caso de erro
}

type RescheduleAppointmentRequest struct {
	AppointmentID string `json:"appointment_id"`
	NewDate       string `json:"new_date"`
	NewTime       string `json:"new_time"`
}

type RescheduleAppointmentResponse struct {
	AppointmentID string `json:"appointment_id"`
	Status        string `json:"status"`  // e.g. "rescheduled" ou "error"
	Message       string `json:"message"` // detalhes em caso de erro
}

// ============================================================================
// CONFIGURAÇÃO E UTILITÁRIOS
// ============================================================================

// Atualizar a função getAllTools() para incluir todas as tools
func getAllTools() []openai.ChatCompletionToolParam {
	return []openai.ChatCompletionToolParam{
		checkServicesTool,
		registerClientTool,
		checkClientTool,
		fazerAgendamentoTool,
		verificarHorariosDisponiveisTool,
		agendamentosClienteTool,
		cancelarAgendamentoTool,
		reagendarServicoTool,
	}
}

// ============================================================================
// PROCESSAMENTO DE TOOLS
// ============================================================================

func (c *Client) handleToolCalls(
	ctx context.Context,
	userID string,
	toolCalls []openai.ChatCompletionMessageToolCall,
) ([]openai.ChatCompletionMessageParamUnion, error) {
	var responses []openai.ChatCompletionMessageParamUnion

	for _, toolCall := range toolCalls {
		var response openai.ChatCompletionMessageParamUnion
		var success bool

		switch toolCall.Function.Name {
		case "check_services":
			response, success = c.processCheckServicesTool(ctx, userID, toolCall)
		case "register_client":
			response, success = c.processRegisterClientTool(ctx, userID, toolCall)
		case "check_cliente":
			response, success = c.processCheckClientTool(ctx, userID, toolCall)
		case "fazer_agendamento":
			response, success = c.processFazerAgendamentoTool(ctx, userID, toolCall)
		case "verificar_horarios_disponiveis":
			response, success = c.processVerificarHorarioDisponivelTool(ctx, userID, toolCall)
		case "agendamentos_cliente":
			response, success = c.processAgendamentoClienteTool(ctx, userID, toolCall)
		case "cancelar_agendamento":
			response, success = c.processCancelarAgendamentoTool(ctx, userID, toolCall)
		case "reagendar_servico":
			response, success = c.processReagendarServicoTool(ctx, userID, toolCall)
		default:
			response = openai.ToolMessage(
				fmt.Sprintf("Ferramenta '%s' não reconhecida", toolCall.Function.Name),
				toolCall.ID,
			)
			success = false
			log.Error().
				Str("user_id", userID).
				Str("tool_name", toolCall.Function.Name).
				Msg("Tool não reconhecida")
		}

		if success {
			log.Info().
				Str("user_id", userID).
				Str("tool", toolCall.Function.Name).
				Msg("Tool executada com sucesso")
		} else {
			log.Warn().
				Str("user_id", userID).
				Str("tool", toolCall.Function.Name).
				Msg("Falha na execução da tool")
		}

		responses = append(responses, response)
	}

	if len(responses) == 0 {
		return nil, fmt.Errorf("nenhuma resposta de tool foi gerada")
	}

	return responses, nil
}

// ============================================================================
// FUNCIONALIDADE: CONSULTA DE SERVIÇOS
// ============================================================================

func (c *Client) processCheckServicesTool(
	ctx context.Context,
	userID string,
	toolCall openai.ChatCompletionMessageToolCall,
) (openai.ChatCompletionMessageParamUnion, bool) {
	var request ServiceSearchRequest
	err := json.Unmarshal([]byte(toolCall.Function.Arguments), &request)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Erro ao interpretar argumentos de check_services")
		return openai.ToolMessage("Erro ao interpretar parâmetros de consulta de serviços", toolCall.ID), false
	}

	// Define mostrar_resumo como true por padrão se não especificado
	if request.MostrarResumo == false && request.CategoriaFiltro == "" {
		request.MostrarResumo = true
	}

	log.Info().Str("user_id", userID).Str("categoria_filtro", request.CategoriaFiltro).Bool("mostrar_resumo", request.MostrarResumo).Msg("Processando consulta de serviços")

	response, err := c.fetchServicesFromAPI(ctx, request)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Erro ao buscar serviços")
		return openai.ToolMessage("Erro ao consultar serviços disponíveis", toolCall.ID), false
	}

	// Verificar se filtro retornou resultados
	if request.CategoriaFiltro != "" && len(response.ServicosPorCategoria) == 0 {
		mensagemErro := fmt.Sprintf("Nenhuma categoria encontrada com o filtro: '%s'. Categorias disponíveis: %s",
			request.CategoriaFiltro, strings.Join(response.CategoriasDisponiveis, ", "))
		return openai.ToolMessage(mensagemErro, toolCall.ID), false
	}

	respJSON, err := json.Marshal(response)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao serializar resposta de serviços")
		return openai.ToolMessage("Erro ao processar resposta de serviços", toolCall.ID), false
	}

	return openai.ToolMessage(string(respJSON), toolCall.ID), true
}

func (c *Client) fetchServicesFromAPI(ctx context.Context, request ServiceSearchRequest) (*ServiceSearchResponse, error) {
	config := trinks.LoadTrinksConfig() // Usando função do utils.go
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", config.BaseURL+"/servicos", nil)
	if err != nil {
		return nil, err
	}

	// Usando GetHeaders() do utils.go
	for key, value := range config.GetHeaders() {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResponse struct {
		Data []struct {
			ID                 interface{} `json:"id"`
			Nome               string      `json:"nome"`
			Categoria          string      `json:"categoria"`
			DuracaoEmMinutos   int         `json:"duracaoEmMinutos"`
			Preco              float64     `json:"preco"`
			Descricao          string      `json:"descricao"`
			VisivelParaCliente bool        `json:"visivelParaCliente"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, err
	}

	return c.processServiceDataByCategory(apiResponse.Data, request), nil
}

func (c *Client) processServiceDataByCategory(rawData []struct {
	ID                 interface{} `json:"id"`
	Nome               string      `json:"nome"`
	Categoria          string      `json:"categoria"`
	DuracaoEmMinutos   int         `json:"duracaoEmMinutos"`
	Preco              float64     `json:"preco"`
	Descricao          string      `json:"descricao"`
	VisivelParaCliente bool        `json:"visivelParaCliente"`
}, request ServiceSearchRequest) *ServiceSearchResponse {

	servicosPorCategoria := make(map[string][]ServiceInfo)
	categoriasDisponiveis := make(map[string]bool)
	totalServicos := 0

	for _, service := range rawData {
		categoria := service.Categoria
		categoriasDisponiveis[categoria] = true

		var idStr string
		switch id := service.ID.(type) {
		case string:
			idStr = id
		case float64:
			idStr = fmt.Sprintf("%.0f", id)
		case int:
			idStr = fmt.Sprintf("%d", id)
		default:
			idStr = fmt.Sprintf("%v", id)
		}

		serviceInfo := ServiceInfo{
			ID:        idStr,
			Nome:      service.Nome,
			Descricao: service.Descricao,
			Duracao:   service.DuracaoEmMinutos,
			Preco:     service.Preco,
			Visivel:   service.VisivelParaCliente,
		}

		servicosPorCategoria[categoria] = append(servicosPorCategoria[categoria], serviceInfo)
		totalServicos++
	}

	// Aplicar filtro de categoria se especificado
	if request.CategoriaFiltro != "" {
		categoriasFiltradas := make(map[string][]ServiceInfo)
		filtroLower := strings.ToLower(request.CategoriaFiltro)

		for categoria, servicos := range servicosPorCategoria {
			if strings.Contains(strings.ToLower(categoria), filtroLower) {
				categoriasFiltradas[categoria] = servicos
			}
		}

		servicosPorCategoria = categoriasFiltradas
		totalServicos = 0
		for _, servicos := range servicosPorCategoria {
			totalServicos += len(servicos)
		}
	}

	var listaCategoriasDisponiveis []string
	for categoria := range categoriasDisponiveis {
		listaCategoriasDisponiveis = append(listaCategoriasDisponiveis, categoria)
	}

	response := &ServiceSearchResponse{
		ServicosPorCategoria:  servicosPorCategoria,
		TotalServicos:         totalServicos,
		CategoriasDisponiveis: listaCategoriasDisponiveis,
	}

	if request.MostrarResumo {
		response.ResumoCategoria = c.criarResumoCategoria(servicosPorCategoria)
	}

	return response
}

func (c *Client) criarResumoCategoria(servicosPorCategoria map[string][]ServiceInfo) map[string]CategorySummary {
	resumo := make(map[string]CategorySummary)

	for categoria, servicos := range servicosPorCategoria {
		if len(servicos) == 0 {
			continue
		}

		var somaPreco float64
		var somaDuracao int

		for _, servico := range servicos {
			somaPreco += servico.Preco
			somaDuracao += servico.Duracao
		}

		resumo[categoria] = CategorySummary{
			Quantidade:   len(servicos),
			PrecoMedio:   somaPreco / float64(len(servicos)),
			DuracaoMedia: somaDuracao / len(servicos),
		}
	}

	return resumo
}

// ============================================================================
// FUNCIONALIDADE: CADASTRO DE CLIENTE
// ============================================================================

func (c *Client) processRegisterClientTool(
	ctx context.Context,
	userID string,
	toolCall openai.ChatCompletionMessageToolCall,
) (openai.ChatCompletionMessageParamUnion, bool) {
	var request ClientRegisterRequest

	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &request); err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Erro ao decodificar parâmetros de cadastro")
		return openai.ToolMessage("Erro ao processar dados do cliente", toolCall.ID), false
	}

	// Validar dados usando função do utils.go
	erros := trinks.ValidarDadosCliente(request.Name, request.Phone, request.Email)
	if len(erros) > 0 {
		mensagemErro := fmt.Sprintf("Dados inválidos: %s", strings.Join(erros, ", "))
		response := ClientRegisterResponse{
			Success:   false,
			Message:   mensagemErro,
			ErrorCode: "INVALID_DATA",
		}

		respJSON, _ := json.Marshal(response)
		return openai.ToolMessage(string(respJSON), toolCall.ID), false
	}

	// Verificar se cliente já existe usando função do utils.go
	_, err := trinks.BuscarClientePorEmail(ctx, request.Email)
	if err == nil {
		response := ClientRegisterResponse{
			Success:   false,
			Message:   "Cliente já cadastrado com este e-mail",
			ErrorCode: "CLIENT_EXISTS",
		}

		respJSON, _ := json.Marshal(response)
		return openai.ToolMessage(string(respJSON), toolCall.ID), false
	}

	// Cadastrar cliente
	cliente, err := c.cadastrarClienteAPI(ctx, request)
	if err != nil {
		trinks.LogError(err, userID, "cadastrarClienteAPI")

		response := ClientRegisterResponse{
			Success:   false,
			Message:   "Erro interno ao cadastrar cliente",
			ErrorCode: "API_ERROR",
		}

		respJSON, _ := json.Marshal(response)
		return openai.ToolMessage(string(respJSON), toolCall.ID), false
	}

	response := ClientRegisterResponse{
		Success:  true,
		ClientID: cliente.ID,
		Message:  fmt.Sprintf("Cliente %s cadastrado com sucesso!", cliente.Nome),
	}

	respJSON, err := json.Marshal(response)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao serializar resposta de cadastro")
		return openai.ToolMessage("Cliente cadastrado com sucesso, mas erro ao processar resposta", toolCall.ID), true
	}

	return openai.ToolMessage(string(respJSON), toolCall.ID), true
}

func (c *Client) cadastrarClienteAPI(ctx context.Context, request ClientRegisterRequest) (*trinks.Cliente, error) {
	config := trinks.LoadTrinksConfig() // Usando função do utils.go
	client := &http.Client{Timeout: 15 * time.Second}

	// Usando função LimparTelefone do utils.go
	ddd, numero := trinks.LimparTelefone(request.Phone)

	payload := map[string]interface{}{
		"nome":  strings.ToUpper(strings.TrimSpace(request.Name)),
		"email": strings.ToLower(strings.TrimSpace(request.Email)),
		"telefones": []map[string]interface{}{
			{
				"ddd":      ddd,
				"numero": numero,
				"tipoId":   3, // Tipo 3 = Celular
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("erro ao codificar dados: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", config.BaseURL+"/clientes", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %w", err)
	}

	// Usando GetHeaders() do utils.go
	for key, value := range config.GetHeaders() {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro da API (%d): %s", resp.StatusCode, string(body))
	}

	var response struct {
		Data trinks.Cliente `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &response.Data, nil
}

// ============================================================================
// FUNCIONALIDADE: CONSULTA DE CLIENTE
// ============================================================================

func (c *Client) processCheckClientTool(
	ctx context.Context,
	userID string,
	toolCall openai.ChatCompletionMessageToolCall,
) (openai.ChatCompletionMessageParamUnion, bool) {
	var request ClientCheckRequest
	err := json.Unmarshal([]byte(toolCall.Function.Arguments), &request)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Erro ao interpretar argumentos de check_cliente")
		return openai.ToolMessage("Erro ao interpretar parâmetros de consulta de cliente", toolCall.ID), false
	}

	log.Info().Str("user_id", userID).Str("phone_number", request.PhoneNumber).Msg("Processando consulta de cliente")

	response, err := c.fetchClientFromAPI(ctx, request)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Erro ao buscar cliente")
		return openai.ToolMessage("Erro ao consultar cliente", toolCall.ID), false
	}

	respJSON, err := json.Marshal(response)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao serializar resposta de consulta de cliente")
		return openai.ToolMessage("Erro ao processar resposta de cliente", toolCall.ID), false
	}

	return openai.ToolMessage(string(respJSON), toolCall.ID), true
}

func (c *Client) fetchClientFromAPI(ctx context.Context, request ClientCheckRequest) (*ClientCheckResponse, error) {
	config := trinks.LoadTrinksConfig() // Usando função do utils.go
	client := &http.Client{Timeout: 10 * time.Second}

	// Usando função LimparTelefone do utils.go
	ddd, numero := trinks.LimparTelefone(request.PhoneNumber)

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/clientes?ddd=%s&telefone=%s", config.BaseURL, ddd, numero), nil)
	if err != nil {
		return nil, err
	}

	// Usando GetHeaders() do utils.go
	for key, value := range config.GetHeaders() {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResponse struct {
		Data []struct {
			ID                 interface{} `json:"id"`
			Nome               string      `json:"nome"`
			Email              string      `json:"email"`
			Telefones          []struct {
				DDD      string `json:"ddd"`
				Telefone string `json:"telefone"`
			} `json:"telefones"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, err
	}

	if len(apiResponse.Data) == 0 {
		return &ClientCheckResponse{Exists: false}, nil
	}

	clientData := apiResponse.Data[0]

	var idStr string
	switch id := clientData.ID.(type) {
	case string:
		idStr = id
	case float64:
		idStr = fmt.Sprintf("%.0f", id)
	case int:
		idStr = fmt.Sprintf("%d", id)
	default:
		idStr = fmt.Sprintf("%v", id)
	}

	return &ClientCheckResponse{
		Exists:     true,
		ClientID:   idStr,
		ClientName: clientData.Nome,
	}, nil
}

// ============================================================================
// FUNCIONALIDADE: AGENDAMENTO
// ============================================================================

func (c *Client) processFazerAgendamentoTool(
	ctx context.Context,
	userID string,
	toolCall openai.ChatCompletionMessageToolCall,
) (openai.ChatCompletionMessageParamUnion, bool) {
	var request AppointmentRequest
	err := json.Unmarshal([]byte(toolCall.Function.Arguments), &request)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Erro ao interpretar argumentos de fazer_agendamento")
		return openai.ToolMessage("Erro ao interpretar parâmetros de agendamento", toolCall.ID), false
	}

	log.Info().Str("user_id", userID).Str("client_id", request.ClientID).Str("service_id", request.ServiceID).Msg("Processando agendamento")

	// Chamar função para criar agendamento
	agendamento, err := c.criarAgendamentoAPI(ctx, request)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Erro ao criar agendamento")
		return openai.ToolMessage("Erro ao criar agendamento", toolCall.ID), false
	}

	response := AppointmentResponse{
		AppointmentID: agendamento.ID,
		Status:        "agendado",
		Message:       fmt.Sprintf("Agendamento criado com sucesso! ID: %s", agendamento.ID),
	}

	respJSON, err := json.Marshal(response)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao serializar resposta de agendamento")
		return openai.ToolMessage("Agendamento criado, mas erro ao processar resposta", toolCall.ID), true
	}

	return openai.ToolMessage(string(respJSON), toolCall.ID), true
}

func (c *Client) criarAgendamentoAPI(ctx context.Context, request AppointmentRequest) (*trinks.Agendamento, error) {
	config := trinks.LoadTrinksConfig() // Usando função do utils.go
	client := &http.Client{Timeout: 15 * time.Second}

	// Montar payload para criação do agendamento
	payload := map[string]interface{}{
		"clienteId": request.ClientID,
		"servicoId": request.ServiceID,
		"data":      request.Date,
		"hora":      request.Time,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("erro ao codificar dados do agendamento: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", config.BaseURL+"/agendamentos", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição de agendamento: %w", err)
	}

	// Usando GetHeaders() do utils.go
	for key, value := range config.GetHeaders() {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição de agendamento: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro da API ao criar agendamento (%d): %s", resp.StatusCode, string(body))
	}

	var response struct {
		Data trinks.Agendamento `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta de agendamento: %w", err)
	}

	return &response.Data, nil
}

func (c *Client) processVerificarHorarioDisponivelTool(
	ctx context.Context,
	userID string,
	toolCall openai.ChatCompletionMessageToolCall,
) (openai.ChatCompletionMessageParamUnion, bool) {
	var request AvailableSlotsRequest
	err := json.Unmarshal([]byte(toolCall.Function.Arguments), &request)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Erro ao interpretar argumentos de verificar_horarios_disponiveis")
		return openai.ToolMessage("Erro ao interpretar parâmetros de verificação de horários", toolCall.ID), false
	}

	log.Info().Str("user_id", userID).Str("service_id", request.ServiceID).Str("date", request.Date).Msg("Verificando horários disponíveis")

	// Chamar função para verificar horários disponíveis
	disponibilidade, err := c.verificarDisponibilidadeAPI(ctx, request)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Erro ao verificar disponibilidade")
		return openai.ToolMessage("Erro ao verificar horários disponíveis", toolCall.ID), false
	}

	response := AvailableSlotsResponse{
		ServiceID: request.ServiceID,
		Date:      request.Date,
		Slots:     disponibilidade,
	}

	respJSON, err := json.Marshal(response)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao serializar resposta de horários disponíveis")
		return openai.ToolMessage("Erro ao processar resposta de horários disponíveis", toolCall.ID), false
	}

	return openai.ToolMessage(string(respJSON), toolCall.ID), true
}

func (c *Client) verificarDisponibilidadeAPI(ctx context.Context, request AvailableSlotsRequest) ([]AvailableSlot, error) {
	config := trinks.LoadTrinksConfig() // Usando função do utils.go
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/agendamentos/disponibilidade?servicoId=%s&data=%s", config.BaseURL, request.ServiceID, request.Date), nil)
	if err != nil {
		return nil, err
	}

	// Usando GetHeaders() do utils.go
	for key, value := range config.GetHeaders() {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResponse struct {
		Data []struct {
			Hora     string `json:"hora"`
			Duracao  int    `json:"duracao"`
			Disponivel bool   `json:"disponivel"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, err
	}

	var slots []AvailableSlot
	for _, slot := range apiResponse.Data {
		if slot.Disponivel {
			slots = append(slots, AvailableSlot{
				Time:     slot.Hora,
				Duration: slot.Duracao,
			})
		}
	}

	return slots, nil
}

func (c *Client) fetchAppointmentsByClient(
	ctx context.Context,
	clientID string,
) (*ClientAppointmentsResponse, error) {
	// Usar as funções do utils.go
	clienteIDInt, err := strconv.Atoi(clientID)
	if err != nil {
		return nil, fmt.Errorf("ID do cliente inválido: %w", err)
	}

	agendamentos, err := trinks.BuscarAgendamentosCliente(ctx, clienteIDInt)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar agendamentos: %w", err)
	}

	items := make([]AppointmentItem, len(agendamentos))
	for i, agendamento := range agendamentos {
		// Converter int para string
		data, hora, _ := trinks.FormatarDataHora(agendamento.DataHoraInicio)

		items[i] = AppointmentItem{
			AppointmentID: strconv.Itoa(agendamento.ID), // Converter int para string
			ServiceID:     strconv.Itoa(agendamento.Servico.ID),
			Date:          data,
			Time:          hora,
			Status:        agendamento.Status.Nome,
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

func (c *Client) cancelAppointmentRequest(
	ctx context.Context,
	appointmentID string,
) (*CancelAppointmentResponse, error) {
	config := trinks.LoadTrinksConfig()
	httpClient := &http.Client{Timeout: 10 * time.Second}

	url := config.BaseURL + "/agendamentos/" + appointmentID
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return nil, err
	}

	// Usar GetHeaders() do utils.go
	for key, value := range config.GetHeaders() {
		req.Header.Set(key, value)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Criar resposta baseada no status HTTP
	response := &CancelAppointmentResponse{
		AppointmentID: appointmentID,
	}

	if resp.StatusCode >= 400 {
		response.Status = "error"
		body, _ := io.ReadAll(resp.Body)
		response.Message = fmt.Sprintf("Não foi possível cancelar: %s", string(body))
	} else {
		response.Status = "cancelled"
		response.Message = "Agendamento cancelado com sucesso"
	}

	return response, nil
}

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

func (c *Client) rescheduleAppointmentRequest(
	ctx context.Context,
	reqData RescheduleAppointmentRequest,
) (*RescheduleAppointmentResponse, error) {
	config := trinks.LoadTrinksConfig()
	httpClient := &http.Client{Timeout: 10 * time.Second}

	// Monta endpoint: /agendamentos/{id}
	url := config.BaseURL + "/agendamentos/" + reqData.AppointmentID

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

	// Usar GetHeaders() do utils.go
	for key, value := range config.GetHeaders() {
		httpReq.Header.Set(key, value)
	}

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Criar resposta baseada no status HTTP
	response := &RescheduleAppointmentResponse{
		AppointmentID: reqData.AppointmentID,
	}

	if resp.StatusCode >= 400 {
		response.Status = "error"
		body, _ := io.ReadAll(resp.Body)
		response.Message = fmt.Sprintf("Não foi possível reagendar: %s", string(body))
	} else {
		response.Status = "rescheduled"
		response.Message = "Agendamento reagendado com sucesso"
	}

	return response, nil
}

func (c *Client) processReagendarServicoTool(
	ctx context.Context,
	userID string,
	toolCall openai.ChatCompletionMessageToolCall,
) (openai.ChatCompletionMessageParamUnion, bool) {
	var request RescheduleAppointmentRequest
	err := json.Unmarshal([]byte(toolCall.Function.Arguments), &request)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Erro ao interpretar argumentos de reagendar_servico")
		return openai.ToolMessage("Erro ao interpretar parâmetros de reagendamento", toolCall.ID), false
	}

	log.Info().Str("user_id", userID).Str("appointment_id", request.AppointmentID).Str("new_date", request.NewDate).Str("new_time", request.NewTime).Msg("Reagendando serviço")

	// Chamar função para reagendar serviço
	resposta, err := c.reagendarServicoAPI(ctx, request)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Erro ao reagendar serviço")
		return openai.ToolMessage("Erro ao reagendar serviço", toolCall.ID), false
	}

	response := RescheduleAppointmentResponse{
		AppointmentID: resposta.AppointmentID,
		Status:        "rescheduled",
		Message:       "Serviço reagendado com sucesso",
	}

	respJSON, err := json.Marshal(response)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao serializar resposta de reagendamento")
		return openai.ToolMessage("Serviço reagendado, mas erro ao processar resposta", toolCall.ID), true
	}

	return openai.ToolMessage(string(respJSON), toolCall.ID), true
}

func (c *Client) reagendarServicoAPI(ctx context.Context, request RescheduleAppointmentRequest) (*trinks.Agendamento, error) {
	config := trinks.LoadTrinksConfig() // Usando função do utils.go
	client := &http.Client{Timeout: 15 * time.Second}

	// Montar payload para atualização do agendamento
	payload := map[string]interface{}{
		"novaData": request.NewDate,
		"novoHorario": request.NewTime,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("erro ao codificar dados de reagendamento: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", fmt.Sprintf("%s/agendamentos/%s", config.BaseURL, request.AppointmentID), strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição de reagendamento: %w", err)
	}

	// Usando GetHeaders() do utils.go
	for key, value := range config.GetHeaders() {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição de reagendamento: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro da API ao reagendar serviço (%d): %s", resp.StatusCode, string(body))
	}

	var response struct {
		Data trinks.Agendamento `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta de reagendamento: %w", err)
	}

	return &response.Data, nil
}
