package openai

import (
	"chatbot/trinks"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// ============================================================================
// CONFIGURAÇÃO E UTILITÁRIOS
// ============================================================================

func getAllTools() []openai.ChatCompletionToolParam {
	return []openai.ChatCompletionToolParam{
		checkServicesTool,
		registerClientTool,
	}
}

// ============================================================================
// PROCESSAMENTO DE TOOLS
// ============================================================================

func (c *Client) handleToolCalls(
	ctx context.Context,
	userID string,
	toolCalls []openai.ChatCompletionMessageToolCall,
) []openai.ChatCompletionMessageParamUnion {
	var responses []openai.ChatCompletionMessageParamUnion

	for _, toolCall := range toolCalls {
		var response openai.ChatCompletionMessageParamUnion
		var success bool

		switch toolCall.Function.Name {
		case "check_services":
			response, success = c.processCheckServicesTool(ctx, userID, toolCall)
		case "register_client":
			response, success = c.processRegisterClientTool(ctx, userID, toolCall)
		default:
			response = openai.ToolMessage(
				fmt.Sprintf("Ferramenta '%s' não reconhecida", toolCall.Function.Name),
				toolCall.ID,
			)
			success = false
		}

		if success {
			log.Info().
				Str("user_id", userID).
				Str("tool", toolCall.Function.Name).
				Msg("Tool executada com sucesso")
		}

		responses = append(responses, response)
	}

	return responses
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
				"telefone": numero,
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
