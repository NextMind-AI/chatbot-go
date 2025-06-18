package trinks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// VerificarClientePorTelefone busca um cliente pelo telefone e retorna suas informações com agendamentos
func VerificarClientePorTelefone(ctx context.Context, telefone string) (string, error) {
    config := LoadTrinksConfig()
    client := &http.Client{Timeout: 10 * time.Second}

    // Buscar cliente por telefone
    url := fmt.Sprintf("%s/clientes/buscar-por-telefone?telefone=%s", config.BaseURL, telefone)
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return "", err
    }

    for key, value := range config.GetHeaders() {
        req.Header.Set(key, value)
    }

    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    var clientResponse struct {
        Success bool `json:"success"`
        Data    struct {
            ID       int    `json:"id"`
            Nome     string `json:"nome"`
            Email    string `json:"email"`
            Telefone string `json:"telefone"`
        } `json:"data"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&clientResponse); err != nil {
        return "", err
    }

    // Se cliente não encontrado
    if !clientResponse.Success || clientResponse.Data.ID == 0 {
        return "Não registrado", nil
    }

    var response strings.Builder
    response.WriteString("CLIENTE ENCONTRADO\n\n")
    response.WriteString(fmt.Sprintf("Nome: %s\n", clientResponse.Data.Nome))
    response.WriteString(fmt.Sprintf("ID: %d\n", clientResponse.Data.ID))
    response.WriteString(fmt.Sprintf("Email: %s\n\n", clientResponse.Data.Email))

    // Buscar agendamentos do cliente
    agendamentos, err := buscarAgendamentosDoCliente(ctx, clientResponse.Data.ID)
    if err == nil && len(agendamentos) > 0 {
        response.WriteString("AGENDAMENTOS:\n")
        for _, ag := range agendamentos {
            data, hora := formatarDataHoraExibicao(ag.DataHoraInicio)
            response.WriteString(fmt.Sprintf("• %s às %s - %s\n", data, hora, ag.Servico.Nome))
            response.WriteString(fmt.Sprintf("  Profissional: %s | Status: %s\n", ag.Profissional.Nome, ag.Status.Nome))
        }
    } else {
        response.WriteString("Nenhum agendamento futuro encontrado\n")
    }

    return response.String(), nil
}

// Estruturas para agendamentos
type AgendamentoInfo struct {
    ID              int    `json:"id"`
    DataHoraInicio  string `json:"dataHoraInicio"`
    Servico         struct {
        Nome string `json:"nome"`
    } `json:"servico"`
    Profissional struct {
        Nome string `json:"nome"`
    } `json:"profissional"`
    Status struct {
        Nome string `json:"nome"`
    } `json:"status"`
}

// buscarAgendamentosDoCliente busca os agendamentos de um cliente específico
func buscarAgendamentosDoCliente(ctx context.Context, clienteID int) ([]AgendamentoInfo, error) {
    config := LoadTrinksConfig()
    client := &http.Client{Timeout: 10 * time.Second}

    url := fmt.Sprintf("%s/agendamentos/cliente/%d", config.BaseURL, clienteID)
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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
        Data []AgendamentoInfo `json:"data"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
        return nil, err
    }

    return apiResponse.Data, nil
}

// formatarDataHoraExibicao formata uma string de data/hora para exibição
func formatarDataHoraExibicao(dataHora string) (string, string) {
    // Assumindo formato "2006-01-02T15:04:05"
    t, err := time.Parse("2006-01-02T15:04:05", dataHora)
    if err != nil {
        // Tentar outros formatos comuns
        if t, err = time.Parse("2006-01-02 15:04:05", dataHora); err != nil {
            return dataHora, ""
        }
    }

    data := t.Format("02/01/2006")
    hora := t.Format("15:04")
    
    return data, hora
}

