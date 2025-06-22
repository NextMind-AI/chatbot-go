package trinks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// Estruturas necessárias
type AgendamentoExistente struct {
	ID               int     `json:"id"`
	DataHoraInicio   string  `json:"dataHoraInicio"`
	DuracaoEmMinutos int     `json:"duracaoEmMinutos"`
	Valor            float64 `json:"valor"`
	Servico          struct {
		ID   int    `json:"id"`
		Nome string `json:"nome"`
	} `json:"servico"`
	Cliente struct {
		ID int `json:"id"`
	} `json:"cliente"`
	Profissional struct {
		ID int `json:"id"`
	} `json:"profissional"`
}

// ReagendarListaDeServicos reagenda uma lista de serviços para um novo horário de início, mantendo a sequência
func ReagendarListaDeServicos(ctx context.Context, listaDeAgendamentos []AgendamentoExistente, novoHorarioInicial string) bool {
	if len(listaDeAgendamentos) == 0 {
		fmt.Println("A lista de agendamentos está vazia. Nada a fazer.")
		return false
	}

	fmt.Printf("🔄 Iniciando o reagendamento de %d serviço(s)...\n", len(listaDeAgendamentos))

	// 1. Ordena a lista de agendamentos pela data/hora de início para garantir a sequência correta
	agendamentosOrdenados := make([]AgendamentoExistente, len(listaDeAgendamentos))
	copy(agendamentosOrdenados, listaDeAgendamentos)

	sort.Slice(agendamentosOrdenados, func(i, j int) bool {
		return agendamentosOrdenados[i].DataHoraInicio < agendamentosOrdenados[j].DataHoraInicio
	})

	// 2. Itera sobre a lista ordenada, atualizando cada agendamento
	horarioAtual, err := time.Parse("2006-01-02T15:04:05", novoHorarioInicial)
	if err != nil {
		fmt.Printf("❌ Erro ao processar horário inicial: %v\n", err)
		return false
	}

	config := LoadTrinksConfig()
	httpClient := &http.Client{Timeout: 10 * time.Second}
	sucessoTotal := true

	for _, agendamento := range agendamentosOrdenados {
		idAgendamento := agendamento.ID
		nomeServico := agendamento.Servico.Nome

		urlPut := fmt.Sprintf("%s/agendamentos/%d", config.BaseURL, idAgendamento)

		// Monta o payload completo necessário para a requisição PUT
		payload := map[string]interface{}{
			"servicoId":        agendamento.Servico.ID,
			"clienteId":        agendamento.Cliente.ID,
			"profissionalId":   agendamento.Profissional.ID,
			"dataHoraInicio":   horarioAtual.Format("2006-01-02T15:04:05"), // Define o novo horário
			"duracaoEmMinutos": agendamento.DuracaoEmMinutos,
			"valor":            agendamento.Valor,
		}

		fmt.Printf("  -> Reagendando '%s' (ID: %d) para %s...\n",
			nomeServico,
			idAgendamento,
			horarioAtual.Format("02/01/2006 às 15:04"))

		jsonData, err := json.Marshal(payload)
		if err != nil {
			fmt.Printf("  ❌ Erro ao preparar dados para agendamento ID %d: %v\n", idAgendamento, err)
			fmt.Println("🛑 Processo de reagendamento interrompido.")
			sucessoTotal = false
			break
		}

		requisicao, err := http.NewRequestWithContext(ctx, "PUT", urlPut, strings.NewReader(string(jsonData)))
		if err != nil {
			fmt.Printf("  ❌ Erro ao criar requisição para agendamento ID %d: %v\n", idAgendamento, err)
			fmt.Println("🛑 Processo de reagendamento interrompido.")
			sucessoTotal = false
			break
		}

		// Adicionar headers
		for key, value := range config.GetHeaders() {
			requisicao.Header.Set(key, value)
		}
		requisicao.Header.Set("Content-Type", "application/json")

		resposta, err := httpClient.Do(requisicao)
		if err != nil {
			fmt.Printf("  ❌ Erro na requisição HTTP para agendamento ID %d: %v\n", idAgendamento, err)
			fmt.Println("🛑 Processo de reagendamento interrompido.")
			sucessoTotal = false
			break
		}
		defer resposta.Body.Close()

		if resposta.StatusCode != http.StatusOK {
			fmt.Printf("  ❌ Erro ao reagendar agendamento ID %d: Recebido status %s\n", idAgendamento, resposta.Status)
			sucessoTotal = false
			// Não interrompe o processo, continua para o próximo agendamento
			continue
		}

		fmt.Printf("  ✔️ Agendamento ID %d reagendado com sucesso!\n", idAgendamento)

		// Avança o horário para o próximo agendamento, respeitando a duração do agendamento atual
		horarioAtual = horarioAtual.Add(time.Duration(agendamento.DuracaoEmMinutos) * time.Minute)
	}

	if sucessoTotal {
		fmt.Println("✅ Todos os serviços foram reagendados com sucesso!")
	} else {
		fmt.Println("⚠️ O processo de reagendamento foi concluído, mas alguns serviços apresentaram erro.")
	}

	return sucessoTotal
}
