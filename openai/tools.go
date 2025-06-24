package openai

import (
	"chatbot/trinks"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/openai/openai-go"
	"github.com/rs/zerolog/log"
)

// Cache global para tool calls
var (
	toolCallCache = make(map[string]toolCacheEntry)
	cacheMutex    sync.RWMutex
	cacheCleanup  = time.NewTicker(30 * time.Minute)
)

type toolCacheEntry struct {
	response  openai.ChatCompletionMessageParamUnion
	timestamp time.Time
	executed  bool
}

func init() {
	// Iniciar limpeza automática do cache
	go func() {
		for range cacheCleanup.C {
			cleanExpiredCacheEntries()
		}
	}()
}

func cleanExpiredCacheEntries() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	now := time.Now()
	for key, entry := range toolCallCache {
		// Remove entradas mais antigas que 1 hora
		if now.Sub(entry.timestamp) > time.Hour {
			delete(toolCallCache, key)
		}
	}

	log.Debug().Int("cache_size", len(toolCallCache)).Msg("Cache de tools limpo")
}

func generateCacheKey(userID string, toolCall openai.ChatCompletionMessageToolCall) string {
	// Criar chave única baseada no userID, nome da função e argumentos
	data := fmt.Sprintf("%s_%s_%s", userID, toolCall.Function.Name, toolCall.Function.Arguments)
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// handleToolCalls com cache implementado
func (c *Client) handleToolCalls(
	ctx context.Context,
	userID string,
	toolCalls []openai.ChatCompletionMessageToolCall,
) ([]openai.ChatCompletionMessageParamUnion, error) {
	var responses []openai.ChatCompletionMessageParamUnion

	log.Info().
		Str("user_id", userID).
		Int("tool_calls_count", len(toolCalls)).
		Msg("Processando tool calls com cache")

	for _, toolCall := range toolCalls {
		// Gerar chave única para o tool call
		cacheKey := generateCacheKey(userID, toolCall)

		// Verificar se já foi executado
		cacheMutex.RLock()
		cachedEntry, alreadyExecuted := toolCallCache[cacheKey]
		cacheMutex.RUnlock()

		if alreadyExecuted && cachedEntry.executed {
			log.Info().
				Str("user_id", userID).
				Str("tool", toolCall.Function.Name).
				Str("cache_key", cacheKey).
				Msg("Tool call já executada, usando cache")

			responses = append(responses, cachedEntry.response)
			continue
		}

		// Executar tool call
		var response openai.ChatCompletionMessageParamUnion
		var success bool

		log.Info().
			Str("user_id", userID).
			Str("tool", toolCall.Function.Name).
			Str("cache_key", cacheKey).
			Msg("Executando nova tool call")

		switch toolCall.Function.Name {
		case "register_client":
			response, success = c.processRegisterClientTool(ctx, userID, toolCall)
		case "fazer_agendamento":
			response, success = c.processFazerAgendamentoTool(ctx, userID, toolCall)
		case "verificar_horarios_disponiveis":
			response, success = c.processVerificarHorarioDisponivelTool(ctx, userID, toolCall)
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

		// Armazenar no cache
		cacheMutex.Lock()
		toolCallCache[cacheKey] = toolCacheEntry{
			response:  response,
			timestamp: time.Now(),
			executed:  success,
		}
		cacheMutex.Unlock()

		if success {
			log.Info().
				Str("user_id", userID).
				Str("tool", toolCall.Function.Name).
				Str("cache_key", cacheKey).
				Msg("Tool executada com sucesso e armazenada no cache")
		} else {
			log.Warn().
				Str("user_id", userID).
				Str("tool", toolCall.Function.Name).
				Str("cache_key", cacheKey).
				Msg("Falha na execução da tool")
		}

		responses = append(responses, response)
	}

	if len(responses) == 0 {
		log.Warn().
			Str("user_id", userID).
			Msg("Nenhuma resposta de tool foi gerada")
		return nil, fmt.Errorf("nenhuma resposta de tool foi gerada")
	}

	return responses, nil
}

// GetCacheStatistics retorna estatísticas do cache para monitoramento
func GetCacheStatistics() map[string]interface{} {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	totalEntries := len(toolCallCache)
	executedEntries := 0

	for _, entry := range toolCallCache {
		if entry.executed {
			executedEntries++
		}
	}

	hitRate := float64(0)
	if totalEntries > 0 {
		hitRate = float64(executedEntries) / float64(totalEntries) * 100
	}

	return map[string]interface{}{
		"total_entries":    totalEntries,
		"executed_entries": executedEntries,
		"cache_hit_rate":   hitRate,
	}
}

// ============================================================================
// HANDLERS DAS TOOLS
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

	// Reconstituir telefone completo para validação
	telefoneCompleto := request.DDD + request.Phone

	// Validar dados usando função do utils.go
	erros := trinks.ValidarDadosCliente(request.Name, telefoneCompleto, request.Email)
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

	// Verificar se cliente já existe
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
	cliente, err := trinks.CadastrarCliente(ctx, request.Name, request.Email, request.DDD, request.Phone)
	if err != nil {
		trinks.LogError(err, userID, "CadastrarCliente")

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

	log.Info().
		Str("user_id", userID).
		Str("email_cliente", request.EmailCliente).
		Strs("ids_servicos", request.IdsServicos).
		Str("profissional_id", request.ProfissionalID).
		Str("data_hora_inicio", request.DataHoraInicio).
		Msg("Processando agendamento sequencial")

	agendamentos, err := trinks.AgendarServicosSequenciais(
		ctx,
		request.EmailCliente,
		request.IdsServicos,
		request.ProfissionalID,
		request.DataHoraInicio,
	)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Erro ao criar agendamentos")

		response := AppointmentResponse{
			Success:   false,
			Message:   err.Error(),
			ErrorCode: "SCHEDULING_ERROR",
		}

		respJSON, _ := json.Marshal(response)
		return openai.ToolMessage(string(respJSON), toolCall.ID), false
	}

	// Converter resultados para formato de resposta
	var resultados []AgendamentoResultado
	for _, ag := range agendamentos {
		resultados = append(resultados, AgendamentoResultado{
			AppointmentID: strconv.Itoa(ag.ID),
			ServicoNome:   ag.Servico.Nome,
			Horario:       ag.DataHoraInicio,
			Status:        "agendado",
		})
	}

	response := AppointmentResponse{
		Success:            true,
		Agendamentos: resultados,
		Message:            fmt.Sprintf("✅ %d serviços agendados com sucesso!", len(agendamentos)),
	}

	respJSON, err := json.Marshal(response)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao serializar resposta de agendamento")
		return openai.ToolMessage("Agendamentos criados, mas erro ao processar resposta", toolCall.ID), true
	}

	return openai.ToolMessage(string(respJSON), toolCall.ID), true
}

func (c *Client) processVerificarHorarioDisponivelTool(
	ctx context.Context,
	userID string,
	toolCall openai.ChatCompletionMessageToolCall,
) (openai.ChatCompletionMessageParamUnion, bool) {
	var request trinks.VerificarHorariosRequest // USAR trinks.VerificarHorariosRequest
	err := json.Unmarshal([]byte(toolCall.Function.Arguments), &request)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Erro ao interpretar argumentos de verificar_horarios_disponiveis")
		return openai.ToolMessage("Erro ao interpretar parâmetros de verificação de horários", toolCall.ID), false
	}

	log.Info().Str("user_id", userID).Str("date", request.Date).Str("profissional_id", request.ProfissionalID).Str("horario_especifico", request.HorarioEspecifico).Msg("Verificando horários disponíveis")

	response, err := trinks.VerificarDisponibilidade(ctx, request.Date, request.ProfissionalID, request.HorarioEspecifico)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Erro ao verificar disponibilidade")
		return openai.ToolMessage("Erro ao verificar horários disponíveis", toolCall.ID), false
	}

	respJSON, err := json.Marshal(response)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao serializar resposta de horários disponíveis")
		return openai.ToolMessage("Erro ao processar resposta de horários disponíveis", toolCall.ID), false
	}

	return openai.ToolMessage(string(respJSON), toolCall.ID), true
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

	err := trinks.CancelarAgendamento(ctx, req.AppointmentID)
	if err != nil {
		log.Error().Err(err).
			Str("user_id", userID).
			Msg("Erro ao chamar API para cancelar agendamento")
		return openai.ToolMessage("Falha ao cancelar o agendamento", toolCall.ID), false
	}

	resp := &CancelAppointmentResponse{
		AppointmentID: req.AppointmentID,
		Status:        "cancelled",
		Message:       "Agendamento cancelado com sucesso",
	}

	out, err := json.Marshal(resp)
	if err != nil {
		log.Error().Err(err).
			Msg("Erro ao serializar resposta de cancelamento")
		return openai.ToolMessage("Erro ao processar resposta de cancelamento", toolCall.ID), false
	}

	return openai.ToolMessage(string(out), toolCall.ID), true
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

	agendamento, err := trinks.ReagendarServico(ctx, request.AppointmentID, request.NewDate, request.NewTime)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Erro ao reagendar serviço")
		return openai.ToolMessage("Erro ao reagendar serviço", toolCall.ID), false
	}

	response := RescheduleAppointmentResponse{
		AppointmentID: strconv.Itoa(agendamento.ID),
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

// ============================================================================
// ESTRUTURAS DE DADOS
// ============================================================================

// Cadastro de cliente
type ClientRegisterRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	DDD   string `json:"ddd"`
	Phone string `json:"phone"`
}

type ClientRegisterResponse struct {
	Success   bool   `json:"success"`
	ClientID  int    `json:"client_id,omitempty"`
	Message   string `json:"message"`
	ErrorCode string `json:"error_code,omitempty"`
}

// Agendamento
type AppointmentRequest struct {
	EmailCliente    string   `json:"email_cliente"`
	IdsServicos     []string `json:"ids_servicos"`
	ProfissionalID  string   `json:"profissional_id"`
	DataHoraInicio  string   `json:"data_hora_inicio"`
}

type AppointmentResponse struct {
	Success          bool                   `json:"success"`
	Agendamentos     []AgendamentoResultado `json:"agendamentos,omitempty"`
	Message          string                 `json:"message"`
	ErrorCode        string                 `json:"error_code,omitempty"`
}

type AgendamentoResultado struct {
	AppointmentID string `json:"appointment_id"`
	ServicoNome   string `json:"servico_nome"`
	Horario       string `json:"horario"`
	Status        string `json:"status"`
}

// Cancelamento
type CancelAppointmentRequest struct {
	AppointmentID string `json:"appointment_id"`
}

type CancelAppointmentResponse struct {
	AppointmentID string `json:"appointment_id"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}

// Reagendamento
type RescheduleAppointmentRequest struct {
	AppointmentID string `json:"appointment_id"`
	NewDate       string `json:"new_date"`
	NewTime       string `json:"new_time"`
}

type RescheduleAppointmentResponse struct {
	AppointmentID string `json:"appointment_id"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}
