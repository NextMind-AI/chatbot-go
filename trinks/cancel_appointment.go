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

// CancelarListaDeAgendamentos cancela uma lista de agendamentos enviando uma requisição PATCH para cada um
func CancelarListaDeAgendamentos(ctx context.Context, listaDeAgendamentos []AgendamentoExistente, idQuemCancelou int, motivo string) bool {
	if len(listaDeAgendamentos) == 0 {
		fmt.Println("A lista de agendamentos está vazia. Nada para cancelar.")
		return false
	}

	// Se motivo não foi fornecido, usa o padrão
	if motivo == "" {
		motivo = "Cancelado pelo cliente"
	}

	fmt.Printf("🗑️  Iniciando o cancelamento de %d serviço(s)...\n", len(listaDeAgendamentos))

	config := LoadTrinksConfig()
	httpClient := &http.Client{Timeout: 10 * time.Second}
	sucessoTotal := true

	for _, agendamento := range listaDeAgendamentos {
		idAgendamento := agendamento.ID
		nomeServico := agendamento.Servico.Nome

		// URL específica para alterar o status do agendamento para 'cancelado'
		urlPatch := fmt.Sprintf("%s/agendamentos/%d/status/cancelado", config.BaseURL, idAgendamento)

		// Payload com as informações de quem cancelou e o motivo
		payload := map[string]interface{}{
			"quemCancelou": fmt.Sprintf("%d", idQuemCancelou), // A API pode esperar uma string
			"motivo":       motivo,
		}

		fmt.Printf("  -> Cancelando '%s' (ID: %d)...\n", nomeServico, idAgendamento)

		jsonData, err := json.Marshal(payload)
		if err != nil {
			fmt.Printf("  ❌ Erro ao preparar dados para agendamento ID %d: %v\n", idAgendamento, err)
			sucessoTotal = false
			continue // Continua para o próximo
		}

		requisicao, err := http.NewRequestWithContext(ctx, "PATCH", urlPatch, strings.NewReader(string(jsonData)))
		if err != nil {
			fmt.Printf("  ❌ Erro ao criar requisição para agendamento ID %d: %v\n", idAgendamento, err)
			sucessoTotal = false
			continue // Continua para o próximo
		}

		// Adicionar headers
		for key, value := range config.GetHeaders() {
			requisicao.Header.Set(key, value)
		}
		requisicao.Header.Set("Content-Type", "application/json")

		resposta, err := httpClient.Do(requisicao)
		if err != nil {
			fmt.Printf("  ❌ Erro de conexão com a API: %v\n", err)
			sucessoTotal = false
			break // Pode fazer sentido parar se a conexão falhar
		}

		// Um PATCH bem-sucedido geralmente retorna 200 OK ou 204 No Content
		if resposta.StatusCode == 200 || resposta.StatusCode == 204 {
			fmt.Println("  ✅ Sucesso! Agendamento cancelado.")
		} else {
			fmt.Printf("  ❌ Falha ao cancelar o agendamento ID %d.\n", idAgendamento)
			fmt.Printf("     Status: %d\n", resposta.StatusCode)

			// Lê a resposta de erro se possível
			if body, err := io.ReadAll(resposta.Body); err == nil {
				fmt.Printf("     Resposta: %s\n", string(body))
			}

			sucessoTotal = false
			// Continua para o próximo mesmo em caso de falha, para tentar cancelar todos.
		}

		resposta.Body.Close()
	}

	if sucessoTotal {
		fmt.Println("\n✅ Todos os agendamentos na lista foram processados para cancelamento com sucesso!")
	} else {
		fmt.Println("\n⚠️ Ocorreu um erro. Verifique o status dos agendamentos.")
	}

	return sucessoTotal
}
