package trinks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Estruturas necessárias
type ServicoParaAgendar struct {
	ID               string  `json:"id"`
	Nome             string  `json:"nome"`
	DuracaoEmMinutos int     `json:"duracaoEmMinutos"`
	Preco            float64 `json:"preco"`
}

type AgendamentoRealizado struct {
	ID          int    `json:"id,omitempty"`
	ServicoNome string `json:"servico_nome,omitempty"`
	Horario     string `json:"horario,omitempty"`
	Status      string `json:"status,omitempty"`
}

// AgendarServicos agenda uma lista de serviços de forma sequencial para um cliente e profissional
func AgendarServicos(ctx context.Context, clienteID, profissionalID, horarioInicio string, listaServicos []ServicoParaAgendar) ([]AgendamentoRealizado, error) {
	config := LoadTrinksConfig()
	httpClient := &http.Client{Timeout: 10 * time.Second}

	fmt.Printf("🗓️  Iniciando agendamento sequencial para o cliente ID: %s\n", clienteID)

	// Converte o horário de início para facilitar os cálculos
	horarioAtual, err := time.Parse("2006-01-02T15:04:05", horarioInicio)
	if err != nil {
		return nil, fmt.Errorf("formato de horário inválido: %v", err)
	}

	var agendamentosBemSucedidos []AgendamentoRealizado

	for _, servico := range listaServicos {
		// Monta o payload para a API
		payload := map[string]interface{}{
			"servicoId":        servico.ID,
			"clienteId":        clienteID,
			"profissionalId":   profissionalID,
			"dataHoraInicio":   horarioAtual.Format("2006-01-02T15:04:05"),
			"duracaoEmMinutos": servico.DuracaoEmMinutos,
			"valor":            servico.Preco,
		}

		fmt.Printf("  -> Tentando agendar '%s' para as %s...\n", servico.Nome, horarioAtual.Format("15:04"))

		jsonData, err := json.Marshal(payload)
		if err != nil {
			fmt.Printf("  ❌ Erro ao preparar dados para '%s': %v\n", servico.Nome, err)
			break
		}

		requisicao, err := http.NewRequestWithContext(ctx, "POST", config.BaseURL+"/agendamentos", strings.NewReader(string(jsonData)))
		if err != nil {
			fmt.Printf("  ❌ Erro ao criar requisição para '%s': %v\n", servico.Nome, err)
			break
		}

		// Adiciona os headers necessários
		for key, value := range config.GetHeaders() {
			requisicao.Header.Set(key, value)
		}
		requisicao.Header.Set("Content-Type", "application/json")

		resposta, err := httpClient.Do(requisicao)
		if err != nil {
			fmt.Printf("  ❌ Erro de conexão com a API: %v\n", err)
			fmt.Println("🛑 Processo de agendamento interrompido.")
			break
		}

		// A API retorna status 201 (Created) em caso de sucesso
		if resposta.StatusCode == 201 {
			fmt.Printf("  ✅ Sucesso! Agendamento para '%s' confirmado.\n", servico.Nome)

			var agendamentoInfo AgendamentoRealizado
			if err := json.NewDecoder(resposta.Body).Decode(&agendamentoInfo); err != nil {
				fmt.Printf("  ⚠️  Agendamento realizado, mas erro ao processar resposta: %v\n", err)
				// Cria um registro básico mesmo com erro na decodificação
				agendamentoInfo = AgendamentoRealizado{
					ServicoNome: servico.Nome,
					Horario:     horarioAtual.Format("2006-01-02T15:04:05"),
					Status:      "agendado",
				}
			}

			agendamentosBemSucedidos = append(agendamentosBemSucedidos, agendamentoInfo)

			// Atualiza o horário para o início do próximo serviço
			horarioAtual = horarioAtual.Add(time.Duration(servico.DuracaoEmMinutos) * time.Minute)
		} else {
			fmt.Printf("  ❌ Falha ao agendar '%s'.\n", servico.Nome)
			fmt.Printf("     Status: %d\n", resposta.StatusCode)

			// Lê a resposta de erro se possível
			if body, err := io.ReadAll(resposta.Body); err == nil {
				fmt.Printf("     Resposta: %s\n", string(body))
			}

			fmt.Println("🛑 Processo de agendamento interrompido devido a erro.")
			break
		}

		resposta.Body.Close()
	}

	fmt.Println("\nResumo do processo de agendamento concluído.")
	return agendamentosBemSucedidos, nil
}
