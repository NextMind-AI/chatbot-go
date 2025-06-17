package trinks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ============================================================================
// ESTRUTURAS PARA AGENDAMENTOS
// ============================================================================

type DisponibilidadeResponse struct {
    Date                     string              `json:"date"`
    ProfissionalID           string              `json:"profissional_id,omitempty"`
    HorarioEspecifico        string              `json:"horario_especifico,omitempty"`
    DisponibilidadeGeral     map[string][]string `json:"disponibilidade_geral,omitempty"`
    ProfissionaisDisponiveis []string            `json:"profissionais_disponiveis,omitempty"`
    HorarioDisponivel        bool                `json:"horario_disponivel,omitempty"`
    Message                  string              `json:"message"`
    TipoConsulta             string              `json:"tipo_consulta"`
}

// Lista estática de profissionais
var PROFISSIONAIS_CADASTRADOS = []ProfissionalCadastrado{
    {
        ID:      749630,
        Nome:    "Deurivan Lima Hortegal",
        CPF:     "06446181167",
        Apelido: "Deurivan Hortegal",
    },
    {
        ID:      749578,
        Nome:    "Samuel Mariano Silva",
        CPF:     "04760386181",
        Apelido: "Samuel Mariano",
    },
    {
        ID:      745446,
        Nome:    "Yuri Waner L Tolentino",
        CPF:     "05095178117",
        Apelido: "Yuri Waner",
    },
}

type ProfissionalCadastrado struct {
    ID      int    `json:"id"`
    Nome    string `json:"nome"`
    CPF     string `json:"cpf"`
    Apelido string `json:"apelido"`
}

// ============================================================================
// FUNÇÕES DE AGENDAMENTOS
// ============================================================================

// CriarAgendamento cria um novo agendamento
func CriarAgendamento(ctx context.Context, clientID, serviceID, date, time string) (*Agendamento, error) {
    config := LoadTrinksConfig()
    client := &http.Client{Timeout: 15 * time.Second}

    payload := map[string]interface{}{
        "clienteId": clientID,
        "servicoId": serviceID,
        "data":      date,
        "hora":      time,
    }

    jsonData, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("erro ao codificar dados do agendamento: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, "POST", config.BaseURL+"/agendamentos", strings.NewReader(string(jsonData)))
    if err != nil {
        return nil, fmt.Errorf("erro ao criar requisição de agendamento: %w", err)
    }

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
        Data Agendamento `json:"data"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return nil, fmt.Errorf("erro ao decodificar resposta de agendamento: %w", err)
    }

    return &response.Data, nil
}

// CancelarAgendamento cancela um agendamento existente
func CancelarAgendamento(ctx context.Context, appointmentID string) error {
    config := LoadTrinksConfig()
    httpClient := &http.Client{Timeout: 10 * time.Second}

    url := config.BaseURL + "/agendamentos/" + appointmentID
    req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
    if err != nil {
        return err
    }

    for key, value := range config.GetHeaders() {
        req.Header.Set(key, value)
    }

    resp, err := httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("não foi possível cancelar: %s", string(body))
    }

    return nil
}

// ReagendarServico reagenda um agendamento existente
func ReagendarServico(ctx context.Context, appointmentID, newDate, newTime string) (*Agendamento, error) {
    config := LoadTrinksConfig()
    client := &http.Client{Timeout: 15 * time.Second}

    payload := map[string]interface{}{
        "novaData":    newDate,
        "novoHorario": newTime,
    }

    jsonData, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("erro ao codificar dados de reagendamento: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, "PATCH", fmt.Sprintf("%s/agendamentos/%s", config.BaseURL, appointmentID), strings.NewReader(string(jsonData)))
    if err != nil {
        return nil, fmt.Errorf("erro ao criar requisição de reagendamento: %w", err)
    }

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
        Data Agendamento `json:"data"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return nil, fmt.Errorf("erro ao decodificar resposta de reagendamento: %w", err)
    }

    return &response.Data, nil
}

// VerificarDisponibilidade verifica horários disponíveis seguindo a lógica Python
func VerificarDisponibilidade(ctx context.Context, date, profissionalID, horarioEspecifico string) (*DisponibilidadeResponse, error) {
    // Buscar todos os agendamentos
    agendamentos, err := buscarTodosAgendamentos(ctx)
    if err != nil {
        return nil, err
    }

    // Definir profissionais a verificar
    var profissionaisAVerificar []ProfissionalCadastrado
    if profissionalID != "" {
        profissionalIDInt, err := strconv.Atoi(profissionalID)
        if err == nil {
            for _, prof := range PROFISSIONAIS_CADASTRADOS {
                if prof.ID == profissionalIDInt {
                    profissionaisAVerificar = append(profissionaisAVerificar, prof)
                    break
                }
            }
        }
        if len(profissionaisAVerificar) == 0 {
            return &DisponibilidadeResponse{
                Date:         date,
                Message:      fmt.Sprintf("Profissional com ID %s não encontrado", profissionalID),
                TipoConsulta: "erro",
            }, nil
        }
    } else {
        profissionaisAVerificar = PROFISSIONAIS_CADASTRADOS
    }

    // Calcular disponibilidade geral
    disponibilidadeGeral := calcularDisponibilidadeGeral(date, agendamentos, profissionaisAVerificar)

    // Processar resposta baseada no tipo de consulta
    response := &DisponibilidadeResponse{
        Date:                 date,
        ProfissionalID:       profissionalID,
        HorarioEspecifico:    horarioEspecifico,
        DisponibilidadeGeral: disponibilidadeGeral,
    }

    if horarioEspecifico != "" {
        // Verificação de horário específico
        var profissionaisDisponiveis []string
        for nome, horarios := range disponibilidadeGeral {
            for _, horario := range horarios {
                if horario == horarioEspecifico {
                    profissionaisDisponiveis = append(profissionaisDisponiveis, nome)
                    break
                }
            }
        }

        response.ProfissionaisDisponiveis = profissionaisDisponiveis
        response.HorarioDisponivel = len(profissionaisDisponiveis) > 0

        if profissionalID != "" {
            response.TipoConsulta = "profissional_horario_especifico"
            if len(profissionaisDisponiveis) > 0 {
                response.Message = fmt.Sprintf("✅ Sim, o profissional está disponível às %s no dia %s", horarioEspecifico, date)
            } else {
                response.Message = fmt.Sprintf("❌ Não, o profissional não está disponível às %s no dia %s", horarioEspecifico, date)
            }
        } else {
            response.TipoConsulta = "horario_especifico"
            if len(profissionaisDisponiveis) > 0 {
                response.Message = fmt.Sprintf("✅ Profissionais disponíveis às %s no dia %s: %s",
                    horarioEspecifico, date, strings.Join(profissionaisDisponiveis, ", "))
            } else {
                response.Message = fmt.Sprintf("❌ Nenhum profissional está disponível às %s no dia %s", horarioEspecifico, date)
            }
        }
    } else if profissionalID != "" {
        response.TipoConsulta = "profissional_especifico"
        totalHorarios := 0
        for _, horarios := range disponibilidadeGeral {
            totalHorarios += len(horarios)
        }
        response.Message = fmt.Sprintf("✅ Horários disponíveis para o profissional no dia %s. Total de slots: %d", date, totalHorarios)
    } else {
        response.TipoConsulta = "geral"
        totalHorarios := 0
        for _, horarios := range disponibilidadeGeral {
            totalHorarios += len(horarios)
        }
        response.Message = fmt.Sprintf("✅ Lista de horários disponíveis em %s. Total de slots: %d", date, totalHorarios)
    }

    return response, nil
}

// Funções auxiliares

func buscarTodosAgendamentos(ctx context.Context) ([]struct {
    ID                 int                `json:"id"`
    DataHoraInicio     string             `json:"dataHoraInicio"`
    DuracaoEmMinutos   int                `json:"duracaoEmMinutos"`
    Profissional       ProfissionalResumo `json:"profissional"`
}, error) {
    config := LoadTrinksConfig()
    client := &http.Client{Timeout: 10 * time.Second}

    req, err := http.NewRequestWithContext(ctx, "GET", config.BaseURL+"/agendamentos", nil)
    if err != nil {
        return nil, err
    }

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
            ID                 int                `json:"id"`
            DataHoraInicio     string             `json:"dataHoraInicio"`
            DuracaoEmMinutos   int                `json:"duracaoEmMinutos"`
            Profissional       ProfissionalResumo `json:"profissional"`
        } `json:"data"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
        return nil, err
    }

    return apiResponse.Data, nil
}

func calcularDisponibilidadeGeral(date string, agendamentos []struct {
    ID                 int                `json:"id"`
    DataHoraInicio     string             `json:"dataHoraInicio"`
    DuracaoEmMinutos   int                `json:"duracaoEmMinutos"`
    Profissional       ProfissionalResumo `json:"profissional"`
}, profissionaisAVerificar []ProfissionalCadastrado) map[string][]string {

    disponibilidadeGeral := make(map[string][]string)
    horarioAbertura, _ := time.Parse("2006-01-02 15:04:05", date+" 09:00:00")
    horarioFechamento, _ := time.Parse("2006-01-02 15:04:05", date+" 20:00:00")
    intervaloMinutos := 15

    for _, prof := range profissionaisAVerificar {
        // Criar todos os slots possíveis para este profissional
        slotsDisponiveis := make(map[string]bool)
        for atual := horarioAbertura; atual.Before(horarioFechamento); atual = atual.Add(time.Duration(intervaloMinutos) * time.Minute) {
            slotsDisponiveis[atual.Format("15:04")] = true
        }

        // Filtrar agendamentos do profissional no dia específico
        for _, agendamento := range agendamentos {
            if strings.HasPrefix(agendamento.DataHoraInicio, date) && agendamento.Profissional.ID == prof.ID {
                inicio, err := time.Parse("2006-01-02T15:04:05", agendamento.DataHoraInicio)
                if err != nil {
                    continue
                }

                fim := inicio.Add(time.Duration(agendamento.DuracaoEmMinutos) * time.Minute)

                // Remover slots ocupados
                for slotOcupado := inicio; slotOcupado.Before(fim); slotOcupado = slotOcupado.Add(time.Duration(intervaloMinutos) * time.Minute) {
                    delete(slotsDisponiveis, slotOcupado.Format("15:04"))
                }
            }
        }

        // Converter para slice ordenado
        var slots []string
        for slot := range slotsDisponiveis {
            slots = append(slots, slot)
        }

        // Ordenar slots
        for i := 0; i < len(slots)-1; i++ {
            for j := i + 1; j < len(slots); j++ {
                if slots[i] > slots[j] {
                    slots[i], slots[j] = slots[j], slots[i]
                }
            }
        }

        disponibilidadeGeral[prof.Nome] = slots
    }

    return disponibilidadeGeral
}