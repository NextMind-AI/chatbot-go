package trinks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// ============================================================================
// CONFIGURAÇÃO E ESTRUTURAS BASE
// ============================================================================

type TrinksConfig struct {
    APIKey          string
    EstabelecimentoID string
    BaseURL         string
}

type Cliente struct {
    ID           int                    `json:"id"`
    Nome         string                 `json:"nome"`
    Email        string                 `json:"email,omitempty"`
    CPF          string                 `json:"cpf,omitempty"`
    DataCadastro string                 `json:"dataCadastro"`
    Telefones    []TelefoneCliente      `json:"telefones"`
    Detalhes     map[string]interface{} `json:"clienteDetalhes,omitempty"`
}

type TelefoneCliente struct {
    DDD      string `json:"ddd"`
    Telefone string `json:"telefone"`
    Numero   string `json:"numero,omitempty"` // Para compatibilidade
    TipoID   int    `json:"tipoId,omitempty"`
}

type Servico struct {
    ID                 int     `json:"id"`
    Nome               string  `json:"nome"`
    Categoria          string  `json:"categoria"`
    Descricao          string  `json:"descricao"`
    DuracaoEmMinutos   int     `json:"duracaoEmMinutos"`
    Preco              float64 `json:"preco"`
    VisivelParaCliente bool    `json:"visivelParaCliente"`
}

type Agendamento struct {
    ID                 int              `json:"id"`
    DataHoraInicio     string           `json:"dataHoraInicio"`
    DuracaoEmMinutos   int              `json:"duracaoEmMinutos"`
    Valor              float64          `json:"valor"`
    Cliente            ClienteResumo    `json:"cliente"`
    Profissional       ProfissionalResumo `json:"profissional"`
    Servico            ServicoResumo    `json:"servico"`
    Status             StatusAgendamento `json:"status"`
}

type ClienteResumo struct {
    ID   int    `json:"id"`
    Nome string `json:"nome"`
}

type ProfissionalResumo struct {
    ID   int    `json:"id"`
    Nome string `json:"nome"`
}

type ServicoResumo struct {
    ID   int    `json:"id"`
    Nome string `json:"nome"`
}

type StatusAgendamento struct {
    ID   int    `json:"id"`
    Nome string `json:"nome"`
}

// ============================================================================
// FUNÇÕES UTILITÁRIAS PRINCIPAIS
// ============================================================================

func LoadTrinksConfig() TrinksConfig {
    return TrinksConfig{
        APIKey:            "aYUuejFVLk32PLEV14kAw9mX8U7BxBwtnWS43Tdb",
        EstabelecimentoID: "222326",
        BaseURL:           "https://api.trinks.com/v1",
    }
}

func (c TrinksConfig) GetHeaders() map[string]string {
    return map[string]string{
        "accept":             "application/json",
        "content-type":       "application/json",
        "estabelecimentoId":  c.EstabelecimentoID,
        "X-Api-Key":          c.APIKey,
    }
}

// BuscarClientePorEmail - Busca cliente por e-mail (USADO EM: Agendamento, Reagendamento, Cancelamento, Listar)
func BuscarClientePorEmail(ctx context.Context, email string) (*Cliente, error) {
    config := LoadTrinksConfig()
    client := &http.Client{Timeout: 10 * time.Second}
    
    url := fmt.Sprintf("%s/clientes?email=%s&incluirDetalhes=true", config.BaseURL, email)
    
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
    
    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("erro na API: %d", resp.StatusCode)
    }
    
    var response struct {
        Data []Cliente `json:"data"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return nil, err
    }
    
    if len(response.Data) == 0 {
        return nil, fmt.Errorf("cliente não encontrado")
    }
    
    return &response.Data[0], nil
}

// BuscarServicosPorIDs - Busca múltiplos serviços por IDs (USADO EM: Agendamento, Reagendamento)
func BuscarServicosPorIDs(ctx context.Context, ids []int) ([]Servico, error) {
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
    
    var response struct {
        Data []Servico `json:"data"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return nil, err
    }
    
    // Filtrar apenas os serviços solicitados
    var servicosEncontrados []Servico
    for _, servico := range response.Data {
        for _, id := range ids {
            if servico.ID == id {
                servicosEncontrados = append(servicosEncontrados, servico)
                break
            }
        }
    }
    
    return servicosEncontrados, nil
}

// BuscarAgendamentosCliente - Busca agendamentos futuros do cliente (USADO EM: Reagendamento, Cancelamento, Listar)
func BuscarAgendamentosCliente(ctx context.Context, clienteID int) ([]Agendamento, error) {
    config := LoadTrinksConfig()
    client := &http.Client{Timeout: 10 * time.Second}
    
    req, err := http.NewRequestWithContext(ctx, "GET", config.BaseURL+"/agendamentos", nil)
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
    
    var response struct {
        Data []Agendamento `json:"data"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return nil, err
    }
    
    // Filtrar agendamentos do cliente e futuros
    var agendamentosCliente []Agendamento
    agora := time.Now()
    
    for _, ag := range response.Data {
        if ag.Cliente.ID == clienteID {
            dataAgendamento, err := time.Parse("2006-01-02T15:04:05", ag.DataHoraInicio)
            if err == nil && dataAgendamento.After(agora) {
                agendamentosCliente = append(agendamentosCliente, ag)
            } else if err == nil { // Incluir todos se não conseguir parsear
                agendamentosCliente = append(agendamentosCliente, ag)
            }
        }
    }
    
    return agendamentosCliente, nil
}

// IdentificarAgendamentosSequenciais - Agrupa agendamentos sequenciais (USADO EM: Reagendamento, Cancelamento, Listar)
func IdentificarAgendamentosSequenciais(agendamentos []Agendamento) [][]Agendamento {
    if len(agendamentos) == 0 {
        return [][]Agendamento{}
    }
    
    var blocos [][]Agendamento
    blocoAtual := []Agendamento{agendamentos[0]}
    
    for i := 1; i < len(agendamentos); i++ {
        anterior := blocoAtual[len(blocoAtual)-1]
        atual := agendamentos[i]
        
        inicioAnterior, err1 := time.Parse("2006-01-02T15:04:05", anterior.DataHoraInicio)
        inicioAtual, err2 := time.Parse("2006-01-02T15:04:05", atual.DataHoraInicio)
        
        if err1 != nil || err2 != nil {
            // Se não conseguir parsear, considera como não sequencial
            blocos = append(blocos, blocoAtual)
            blocoAtual = []Agendamento{atual}
            continue
        }
        
        fimAnterior := inicioAnterior.Add(time.Duration(anterior.DuracaoEmMinutos) * time.Minute)
        
        // Considera sequencial se o próximo começa até 15 minutos após o anterior terminar
        // e é o mesmo profissional
        if inicioAtual.Sub(fimAnterior) <= 15*time.Minute && 
           inicioAtual.Sub(fimAnterior) >= -5*time.Minute && // Permite 5min de sobreposição
           anterior.Profissional.ID == atual.Profissional.ID {
            blocoAtual = append(blocoAtual, atual)
        } else {
            blocos = append(blocos, blocoAtual)
            blocoAtual = []Agendamento{atual}
        }
    }
    
    blocos = append(blocos, blocoAtual)
    return blocos
}

// ValidarDadosCliente - Valida dados antes do cadastro (USADO EM: Cadastro)
func ValidarDadosCliente(nome, telefone, email string) []string {
    var erros []string
    
    if strings.TrimSpace(nome) == "" || len(strings.TrimSpace(nome)) < 2 {
        erros = append(erros, "Nome deve ter pelo menos 2 caracteres")
    }
    
    telefaneLimpo := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(telefone, " ", ""), "(", ""), ")", "")
    telefaneLimpo = strings.ReplaceAll(telefaneLimpo, "-", "")
    if len(telefaneLimpo) < 9 {
        erros = append(erros, "Telefone deve ter pelo menos 9 dígitos")
    }
    
    if email != "" && !strings.Contains(email, "@") {
        erros = append(erros, "Email inválido")
    }
    
    return erros
}

// LimparTelefone - Remove formatação do telefone e separa DDD (USADO EM: Cadastro, Busca)
func LimparTelefone(telefone string) (ddd, numero string) {
    telefaneLimpo := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(telefone, " ", ""), "(", ""), ")", "")
    telefaneLimpo = strings.ReplaceAll(telefaneLimpo, "-", "")
    
    // Remove caracteres não numéricos
    var digitos strings.Builder
    for _, r := range telefaneLimpo {
        if r >= '0' && r <= '9' {
            digitos.WriteRune(r)
        }
    }
    telefaneLimpo = digitos.String()
    
    if len(telefaneLimpo) >= 10 {
        ddd = telefaneLimpo[:2]
        numero = telefaneLimpo[2:]
    } else if len(telefaneLimpo) >= 8 {
        ddd = "63" // DDD padrão para Palmas
        numero = telefaneLimpo
    }
    
    return ddd, numero
}

// ============================================================================
// FUNÇÕES DE FORMATAÇÃO E HELPERS
// ============================================================================

// FormatarDataHora - Formata data/hora para exibição
func FormatarDataHora(dataHoraISO string) (data, hora string, err error) {
    t, err := time.Parse("2006-01-02T15:04:05", dataHoraISO)
    if err != nil {
        return "", "", err
    }
    
    data = t.Format("02/01/2006")
    hora = t.Format("15:04")
    return data, hora, nil
}

// CalcularDuracaoTotal - Calcula duração total de uma lista de agendamentos
func CalcularDuracaoTotal(agendamentos []Agendamento) int {
    total := 0
    for _, ag := range agendamentos {
        total += ag.DuracaoEmMinutos
    }
    return total
}

// CalcularValorTotal - Calcula valor total de uma lista de agendamentos
func CalcularValorTotal(agendamentos []Agendamento) float64 {
    total := 0.0
    for _, ag := range agendamentos {
        total += ag.Valor
    }
    return total
}

// LogError - Helper para logging de erros
func LogError(err error, userID, funcao string) {
    log.Error().
        Err(err).
        Str("user_id", userID).
        Str("funcao", funcao).
        Msg("Erro na função utilitária")
}