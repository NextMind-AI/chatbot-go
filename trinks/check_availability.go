package trinks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// VerificarHorariosDisponiveis verifica a disponibilidade de horários para uma data específica
func VerificarHorariosDisponiveis(ctx context.Context, data, profissionalID, horarioEspecifico string) (string, error) {
	config := LoadTrinksConfig()
	httpClient := &http.Client{Timeout: 10 * time.Second}

	// Construir URL com parâmetros
	url := fmt.Sprintf("%s/disponibilidade?data=%s", config.BaseURL, data)
	if profissionalID != "" {
		url += fmt.Sprintf("&profissional_id=%s", profissionalID)
	}
	if horarioEspecifico != "" {
		url += fmt.Sprintf("&horario=%s", horarioEspecifico)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	for key, value := range config.GetHeaders() {
		req.Header.Set(key, value)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var disponibilidadeResponse struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			DisponibilidadeGeral map[string][]string `json:"disponibilidade_geral"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&disponibilidadeResponse); err != nil {
		return "", err
	}

	// Formatar resposta de disponibilidade
	var formatted strings.Builder
	formatted.WriteString(fmt.Sprintf("DISPONIBILIDADE PARA %s\n\n", data))

	if disponibilidadeResponse.Data.DisponibilidadeGeral != nil {
		for profissional, horarios := range disponibilidadeResponse.Data.DisponibilidadeGeral {
			formatted.WriteString(fmt.Sprintf("%s:\n", profissional))
			if len(horarios) > 0 {
				for i, horario := range horarios {
					formatted.WriteString(horario)
					if i < len(horarios)-1 {
						formatted.WriteString(" | ")
					}
					// Quebra linha a cada 6 horários para melhor formatação
					if (i+1)%6 == 0 && i < len(horarios)-1 {
						formatted.WriteString("\n  ")
					}
				}
				formatted.WriteString(fmt.Sprintf("\n  Total: %d horários\n\n", len(horarios)))
			} else {
				formatted.WriteString("  Nenhum horário disponível\n\n")
			}
		}
	}

	if disponibilidadeResponse.Message != "" {
		formatted.WriteString(fmt.Sprintf("%s", disponibilidadeResponse.Message))
	}

	return formatted.String(), nil
}

// Estrutura para requisições de verificação de horários
type RequestVerificarHorarios struct {
	Date              string `json:"date"`
	ProfissionalID    string `json:"profissional_id,omitempty"`
	HorarioEspecifico string `json:"horario_especifico,omitempty"`
}
