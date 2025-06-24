package openai

import "github.com/openai/openai-go"

// ============================================================================
// DEFINIÇÕES DAS TOOLS
// ============================================================================

var registerClientTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "register_client",
		Description: openai.String("Cadastra um novo cliente no sistema da barbearia."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "Nome completo do cliente",
				},
				"email": map[string]any{
					"type":        "string",
					"description": "E-mail do cliente (deve ser único)",
				},
				"ddd": map[string]any{
					"type":        "string",
					"description": "DDD do telefone (ex: 63)",
				},
				"phone": map[string]any{
					"type":        "string",
					"description": "Número do telefone sem DDD (ex: 991234567)",
				},
			},
			"required": []string{"name", "email", "ddd", "phone"},
		},
	},
}

var fazerAgendamentoTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "fazer_agendamento",
		Description: openai.String("Agenda uma sequência de serviços para um cliente em um determinado horário."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"email_cliente": map[string]any{
					"type":        "string",
					"description": "E-mail do cliente",
				},
				"ids_servicos": map[string]any{
					"type":        "array",
					"description": "Lista de IDs dos serviços a agendar",
					"items": map[string]any{
						"type": "string",
					},
				},
				"profissional_id": map[string]any{
					"type":        "string",
					"description": "ID do profissional",
				},
				"data_hora_inicio": map[string]any{
					"type":        "string",
					"description": "Data e hora de início no formato YYYY-MM-DDTHH:MM:SS",
				},
			},
			"required": []string{"email_cliente", "ids_servicos", "profissional_id", "data_hora_inicio"},
		},
	},
}

var verificarHorariosDisponiveisTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "verificar_horarios_disponiveis",
		Description: openai.String("Verifica e lista horários disponíveis em uma data específica, com opções de filtrar por profissional e/ou horário específico."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"date": map[string]any{
					"type":        "string",
					"description": "Data no formato YYYY-MM-DD para verificar disponibilidade",
				},
				"profissional_id": map[string]any{
					"type":        "string",
					"description": "ID do profissional específico para verificar (opcional)",
				},
				"horario_especifico": map[string]any{
					"type":        "string",
					"description": "Horário específico no formato HH:MM para verificar se está disponível (opcional)",
				},
			},
			"required": []string{"date"},
		},
	},
}

var cancelarAgendamentoTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "cancelar_agendamento",
		Description: openai.String("Cancela um agendamento existente pelo ID."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"appointment_id": map[string]any{
					"type":        "string",
					"description": "ID do agendamento a ser cancelado",
				},
			},
			"required": []string{"appointment_id"},
		},
	},
}

var reagendarServicoTool = openai.ChatCompletionToolParam{
	Function: openai.FunctionDefinitionParam{
		Name:        "reagendar_servico",
		Description: openai.String("Altera a data e/ou hora de um agendamento existente."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"appointment_id": map[string]any{
					"type":        "string",
					"description": "ID do agendamento a ser reagendado",
				},
				"new_date": map[string]any{
					"type":        "string",
					"description": "Nova data no formato YYYY-MM-DD",
				},
				"new_time": map[string]any{
					"type":        "string",
					"description": "Novo horário no formato HH:MM",
				},
			},
			"required": []string{"appointment_id", "new_date", "new_time"},
		},
	},
}

// getAllTools retorna todas as tools disponíveis
func getAllTools() []openai.ChatCompletionToolParam {
	return []openai.ChatCompletionToolParam{
		registerClientTool,
		fazerAgendamentoTool,
		verificarHorariosDisponiveisTool,
		cancelarAgendamentoTool,
		reagendarServicoTool,
	}
}
