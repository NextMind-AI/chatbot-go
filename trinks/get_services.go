package trinks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// BuscarTodosServicos busca todos os serviços disponíveis e formata como string organizada por categoria
func BuscarTodosServicos(ctx context.Context) (string, error) {
	config := LoadTrinksConfig()
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", config.BaseURL+"/servicos", nil)
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

	var apiResponse struct {
		Data []struct {
			ID                 any     `json:"id"`
			Nome               string  `json:"nome"`
			Categoria          string  `json:"categoria"`
			DuracaoEmMinutos   int     `json:"duracaoEmMinutos"`
			Preco              float64 `json:"preco"`
			Descricao          string  `json:"descricao"`
			VisivelParaCliente bool    `json:"visivelParaCliente"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return "", err
	}

	// Organizar serviços por categoria
	servicosPorCategoria := make(map[string][]struct {
		ID        string
		Nome      string
		Descricao string
		Duracao   int
		Preco     float64
	})

	for _, service := range apiResponse.Data {
		if !service.VisivelParaCliente {
			continue
		}

		categoria := service.Categoria

		// Converter ID para string
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

		servicoInfo := struct {
			ID        string
			Nome      string
			Descricao string
			Duracao   int
			Preco     float64
		}{
			ID:        idStr,
			Nome:      service.Nome,
			Descricao: service.Descricao,
			Duracao:   service.DuracaoEmMinutos,
			Preco:     service.Preco,
		}

		servicosPorCategoria[categoria] = append(servicosPorCategoria[categoria], servicoInfo)
	}

	// Formatar como string
	var resultado strings.Builder

	for categoria, servicos := range servicosPorCategoria {
		if len(servicos) == 0 {
			continue
		}

		resultado.WriteString(fmt.Sprintf("=====%s=====\n", strings.ToUpper(categoria)))

		for _, servico := range servicos {
			resultado.WriteString(fmt.Sprintf("%s (ID: %s) - R$ %.2f - %d min\n",
				servico.Nome, servico.ID, servico.Preco, servico.Duracao))
			if servico.Descricao != "" {
				resultado.WriteString(fmt.Sprintf("  %s\n", servico.Descricao))
			}
		}
		resultado.WriteString("\n")
	}

	return resultado.String(), nil
}
