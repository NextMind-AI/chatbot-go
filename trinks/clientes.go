package trinks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv" // ADICIONAR esta linha
	"strings"
	"time"
)

// ============================================================================
// ESTRUTURAS PARA CLIENTES
// ============================================================================

type ClientCheckResponse struct {
	Exists     bool   `json:"exists"`
	ClientID   string `json:"client_id,omitempty"`
	ClientName string `json:"client_name,omitempty"`
}

// ============================================================================
// FUNÇÕES DE CLIENTES
// ============================================================================

// BuscarClientePorTelefone busca cliente por número de telefone
func BuscarClientePorTelefone(ctx context.Context, phoneNumber string) (*ClientCheckResponse, error) {
	config := LoadTrinksConfig()
	client := &http.Client{Timeout: 10 * time.Second}

	ddd, numero := LimparTelefone(phoneNumber)

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/clientes?ddd=%s&telefone=%s", config.BaseURL, ddd, numero), nil)
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

	if resp.StatusCode == http.StatusNotFound {
		return &ClientCheckResponse{Exists: false}, nil
	}

	var apiResponse struct {
		Data []struct {
			ID   any    `json:"id"`
			Nome string `json:"nome"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, err
	}

	if len(apiResponse.Data) == 0 {
		return &ClientCheckResponse{Exists: false}, nil
	}

	clientData := apiResponse.Data[0]
	var idStr string
	switch id := clientData.ID.(type) {
	case string:
		idStr = id
	case float64:
		idStr = fmt.Sprintf("%.0f", id)
	case int:
		idStr = fmt.Sprintf("%d", id)
	default:
		idStr = fmt.Sprintf("%v", id)
	}

	return &ClientCheckResponse{
		Exists:     true,
		ClientID:   idStr,
		ClientName: clientData.Nome,
	}, nil
}

// CadastrarCliente cadastra um novo cliente na API
func CadastrarCliente(ctx context.Context, name, email, ddd, phone string) (*Cliente, error) {
	config := LoadTrinksConfig()
	client := &http.Client{Timeout: 15 * time.Second}

	payload := map[string]any{
		"nome":  strings.ToUpper(strings.TrimSpace(name)),
		"email": strings.ToLower(strings.TrimSpace(email)),
		"telefones": []map[string]any{
			{
				"ddd":      ddd,
				"telefone": phone,
				"tipoId":   3, // Tipo 3 = Celular
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("erro ao codificar dados: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", config.BaseURL+"/clientes", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %w", err)
	}

	for key, value := range config.GetHeaders() {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro da API (%d): %s", resp.StatusCode, string(body))
	}

	var response struct {
		Data Cliente `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &response.Data, nil
}

// BuscarClientePorEmailResponse busca cliente por e-mail e retorna resposta formatada
func BuscarClientePorEmailResponse(ctx context.Context, email string) (*ClientCheckResponse, error) {
	cliente, err := BuscarClientePorEmail(ctx, email)
	if err != nil {
		return &ClientCheckResponse{
			Exists: false,
		}, nil
	}

	return &ClientCheckResponse{
		Exists:     true,
		ClientID:   strconv.Itoa(cliente.ID),
		ClientName: cliente.Nome,
	}, nil
}
