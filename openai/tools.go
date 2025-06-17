package openai

import (
	"chatbot/trinks"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/openai/openai-go"
	"github.com/rs/zerolog/log"
)

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
    DDD   string `json:"ddd"`
    Phone string `json:"phone"`
}

type ClientRegisterResponse struct {
    Success   bool   `json:"success"`
    ClientID  int    `json:"client_id,omitempty"`
    Message   string `json:"message"`
    ErrorCode string `json:"error_code,omitempty"`
}

// Verificação de cliente


type ClientCheckResponse struct {
    Exists     bool   `json:"exists"`
    ClientID   string `json:"client_id,omitempty"`
    ClientName string `json:"client_name,omitempty"`
}

// Agendamento
type AppointmentRequest struct {
    ClientID  string `json:"client_id"`
    ServiceID string `json:"service_id"`
    Date      string `json:"date"` // formato: "2025-06-12"
    Time      string `json:"time"` // formato: "14:30"
}

type AppointmentResponse struct {
    AppointmentID string `json:"appointment_id"`
    Status        string `json:"status"`
    Message       string `json:"message"`
}


// Agendamentos do cliente
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

// Cancelamento
type CancelAppointmentRequest struct {
    AppointmentID string `json:"appointment_id"`
}

type CancelAppointmentResponse struct {
    AppointmentID string `json:"appointment_id"`
    Status        string `json:"status"`  // e.g. "cancelled" ou "error"
    Message       string `json:"message"` // detalhe em caso de erro
}

// Reagendamento
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
// HANDLERS DAS TOOLS
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

    if request.MostrarResumo == false && request.CategoriaFiltro == "" {
        request.MostrarResumo = true
    }

    log.Info().Str("user_id", userID).Str("categoria_filtro", request.CategoriaFiltro).Bool("mostrar_resumo", request.MostrarResumo).Msg("Processando consulta de serviços")

    response, err := trinks.BuscarServicos(ctx, request.CategoriaFiltro, request.MostrarResumo)
    if err != nil {
        log.Error().Err(err).Str("user_id", userID).Msg("Erro ao buscar serviços")
        return openai.ToolMessage("Erro ao consultar serviços disponíveis", toolCall.ID), false
    }

    respJSON, err := json.Marshal(response)
    if err != nil {
        log.Error().Err(err).Msg("Erro ao serializar resposta de serviços")
        return openai.ToolMessage("Erro ao processar resposta de serviços", toolCall.ID), false
    }

    return openai.ToolMessage(string(respJSON), toolCall.ID), true
}

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

func (c *Client) processCheckClientTool(
    ctx context.Context,
    userID string,
    toolCall openai.ChatCompletionMessageToolCall,
) (openai.ChatCompletionMessageParamUnion, bool) {
    var request trinks.ClientCheckRequest // USAR trinks.ClientCheckRequest
    err := json.Unmarshal([]byte(toolCall.Function.Arguments), &request)
    if err != nil {
        log.Error().Err(err).Str("user_id", userID).Msg("Erro ao interpretar argumentos de check_cliente")
        return openai.ToolMessage("Erro ao interpretar parâmetros de consulta de cliente", toolCall.ID), false
    }

    log.Info().Str("user_id", userID).Str("email", request.Email).Msg("Processando consulta de cliente")

    response, err := trinks.BuscarClientePorEmailResponse(ctx, request.Email)
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

    agendamento, err := trinks.CriarAgendamento(ctx, request.ClientID, request.ServiceID, request.Date, request.Time)
    if err != nil {
        log.Error().Err(err).Str("user_id", userID).Msg("Erro ao criar agendamento")
        return openai.ToolMessage("Erro ao criar agendamento", toolCall.ID), false
    }

    response := AppointmentResponse{
        AppointmentID: strconv.Itoa(agendamento.ID),
        Status:        "agendado",
        Message:       fmt.Sprintf("Agendamento criado com sucesso! ID: %d", agendamento.ID),
    }

    respJSON, err := json.Marshal(response)
    if err != nil {
        log.Error().Err(err).Msg("Erro ao serializar resposta de agendamento")
        return openai.ToolMessage("Agendamento criado, mas erro ao processar resposta", toolCall.ID), true
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

// ...existing code...

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

    clienteIDInt, err := strconv.Atoi(req.ClientID)
    if err != nil {
        return openai.ToolMessage("ID do cliente inválido", toolCall.ID), false
    }

    agendamentos, err := trinks.BuscarAgendamentosCliente(ctx, clienteIDInt)
    if err != nil {
        log.Error().Err(err).Str("user_id", userID).
            Msg("Erro ao buscar agendamentos do cliente")
        return openai.ToolMessage("Erro ao consultar agendamentos do cliente", toolCall.ID), false
    }

    items := make([]AppointmentItem, len(agendamentos))
    for i, agendamento := range agendamentos {
        data, hora, _ := trinks.FormatarDataHora(agendamento.DataHoraInicio)

        items[i] = AppointmentItem{
            AppointmentID: strconv.Itoa(agendamento.ID),
            ServiceID:     strconv.Itoa(agendamento.Servico.ID),
            Date:          data,
            Time:          hora,
            Status:        agendamento.Status.Nome,
        }
    }

    resp := &ClientAppointmentsResponse{
        ClientID:     req.ClientID,
        Appointments: items,
    }

    out, err := json.Marshal(resp)
    if err != nil {
        log.Error().Err(err).
            Msg("Erro ao serializar resposta de agendamentos")
        return openai.ToolMessage("Erro ao processar resposta de agendamentos", toolCall.ID), false
    }
    return openai.ToolMessage(string(out), toolCall.ID), true
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