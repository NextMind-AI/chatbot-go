package trinks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ============================================================================
// ESTRUTURAS PARA SERVIÇOS
// ============================================================================

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

// ============================================================================
// FUNÇÕES DE SERVIÇOS
// ============================================================================

// BuscarServicos busca todos os serviços disponíveis organizados por categoria
func BuscarServicos(ctx context.Context, categoriaFiltro string, mostrarResumo bool) (*ServiceSearchResponse, error) {
	config := LoadTrinksConfig()
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", config.BaseURL+"/servicos", nil)
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
		return nil, err
	}

	return processServiceDataByCategory(apiResponse.Data, categoriaFiltro, mostrarResumo), nil
}

// processServiceDataByCategory processa os dados brutos da API e organiza por categoria
func processServiceDataByCategory(rawData []struct {
	ID                 any     `json:"id"`
	Nome               string  `json:"nome"`
	Categoria          string  `json:"categoria"`
	DuracaoEmMinutos   int     `json:"duracaoEmMinutos"`
	Preco              float64 `json:"preco"`
	Descricao          string  `json:"descricao"`
	VisivelParaCliente bool    `json:"visivelParaCliente"`
}, categoriaFiltro string, mostrarResumo bool) *ServiceSearchResponse {

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
	if categoriaFiltro != "" {
		categoriasFiltradas := make(map[string][]ServiceInfo)
		filtroLower := strings.ToLower(categoriaFiltro)

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

	if mostrarResumo {
		response.ResumoCategoria = criarResumoCategoria(servicosPorCategoria)
	}

	return response
}

// criarResumoCategoria cria um resumo estatístico das categorias de serviços
func criarResumoCategoria(servicosPorCategoria map[string][]ServiceInfo) map[string]CategorySummary {
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
