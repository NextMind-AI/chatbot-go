package trinks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// CadastrarCliente cadastra novo cliente no sistema
func CadastrarClienteAPI(ctx context.Context, nome, telefone string, email, cpf, endereco *string) (*ClienteCriado, error) {
	config := LoadTrinksConfig()
	httpClient := &http.Client{Timeout: 10 * time.Second}

	// Separar DDD e telefone
	telefoneLimpo := regexp.MustCompile(`\D`).ReplaceAllString(telefone, "")

	var ddd, numero string
	if len(telefoneLimpo) >= 10 {
		ddd = telefoneLimpo[:2]
	} else {
		ddd = "63"
	}

	if len(telefoneLimpo) >= 9 {
		numero = telefoneLimpo[len(telefoneLimpo)-9:]
	} else {
		numero = telefoneLimpo
	}

	// Montar payload
	payload := map[string]any{
		"nome": strings.ToUpper(strings.TrimSpace(nome)),
		"telefones": []map[string]string{
			{
				"ddd":      ddd,
				"numero": numero,
				"tipoId":   "3", // Tipo 3 = Celular
			},
		},
	}

	// Adicionar campos opcionais se fornecidos
	if email != nil && *email != "" {
		payload["email"] = *email
	}

	if cpf != nil && *cpf != "" {
		cpfLimpo := regexp.MustCompile(`\D`).ReplaceAllString(*cpf, "")
		payload["cpf"] = cpfLimpo
	}

	if endereco != nil && *endereco != "" {
		payload["endereco"] = *endereco
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("erro ao preparar dados: %v", err)
	}

	requisicao, err := http.NewRequestWithContext(ctx, "POST", config.BaseURL+"/clientes", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %v", err)
	}

	// Adicionar headers
	for key, value := range config.GetHeaders() {
		requisicao.Header.Set(key, value)
	}
	requisicao.Header.Set("Content-Type", "application/json")

	resposta, err := httpClient.Do(requisicao)
	if err != nil {
		return nil, fmt.Errorf("erro de conexão: %v", err)
	}
	defer resposta.Body.Close()

	if resposta.StatusCode == 201 {
		var clienteCriado ClienteCriado
		if err := json.NewDecoder(resposta.Body).Decode(&clienteCriado); err != nil {
			return nil, fmt.Errorf("erro ao processar resposta: %v", err)
		}

		fmt.Printf("✅ Cliente cadastrado: %s (ID: %d)\n", clienteCriado.Nome, clienteCriado.ID)
		return &clienteCriado, nil
	} else {
		var errorResponse map[string]interface{}
		json.NewDecoder(resposta.Body).Decode(&errorResponse)

		errorMsg := fmt.Sprintf("Status: %d", resposta.StatusCode)
		if errorResponse != nil {
			if msg, ok := errorResponse["message"]; ok {
				errorMsg += fmt.Sprintf(" - %v", msg)
			} else if errors, ok := errorResponse["errors"]; ok {
				errorMsg += fmt.Sprintf(" - %v", errors)
			}
		}

		fmt.Printf("❌ Erro ao cadastrar: %s\n", errorMsg)
		return nil, fmt.Errorf("erro ao cadastrar cliente: %s", errorMsg)
	}
}

// Estrutura para resposta da API
type ClienteCriado struct {
	ID    int    `json:"id"`
	Nome  string `json:"nome"`
	Email string `json:"email,omitempty"`
	CPF   string `json:"cpf,omitempty"`
}
