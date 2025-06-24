package trinks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Estrutura para a resposta da API de profissionais
type ProfessionalResponse struct {
    Data []struct {
        ID      int    `json:"id"`
        Nome    string `json:"nome"`
        CPF     string `json:"cpf"`
        Apelido string `json:"apelido"`
    } `json:"data"`
    Page         int `json:"page"`
    PageSize     int `json:"pageSize"`
    TotalPages   int `json:"totalPages"`
    TotalRecords int `json:"totalRecords"`
}

// BuscarTodosProfissionais busca todos os profissionais disponíveis e formata como string
func BuscarTodosProfissionais(ctx context.Context) (string, error) {
    config := LoadTrinksConfig()
    client := &http.Client{Timeout: 10 * time.Second}

    req, err := http.NewRequestWithContext(ctx, "GET", config.BaseURL+"/profissionais", nil)
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

    var apiResponse ProfessionalResponse
    if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
        return "", err
    }

    // Formatar como string
    var resultado strings.Builder
    resultado.WriteString("=====PROFISSIONAIS DISPONÍVEIS=====\n")
    
    for _, profissional := range apiResponse.Data {
        resultado.WriteString(fmt.Sprintf("%s (ID: %d)\n", 
            profissional.Nome, profissional.ID))
        if profissional.Apelido != "" && profissional.Apelido != profissional.Nome {
            resultado.WriteString(fmt.Sprintf("  Conhecido como: %s\n", profissional.Apelido))
        }
    }
    resultado.WriteString("\n")

    return resultado.String(), nil
}