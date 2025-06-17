package openai

import "github.com/openai/openai-go"

// ============================================================================
// DEFINIÇÕES DAS TOOLS
// ============================================================================

var checkServicesTool = openai.ChatCompletionToolParam{
    Function: openai.FunctionDefinitionParam{
        Name:        "check_services",
        Description: openai.String("Lista todos os serviços disponíveis organizados por categoria."),
        Parameters: openai.FunctionParameters{
            "type": "object",
            "properties": map[string]any{
                "categoria_filtro": map[string]any{
                    "type":        "string",
                    "description": "Categoria para filtrar (opcional). Ex: 'Cabelo', 'Barba'",
                },
                "mostrar_resumo": map[string]any{
                    "type":        "boolean",
                    "description": "Se deve incluir resumo estatístico por categoria (padrão: true)",
                    "default":     true,
                },
            },
            "required": []string{},
        },
    },
}

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
                "phone": map[string]any{
                    "type":        "string",
                    "description": "Número do telefone sem DDD (ex: 991234567)",
                },
            },
            "required": []string{"name", "email", "phone"},
        },
    },
}

var checkClientTool = openai.ChatCompletionToolParam{
    Function: openai.FunctionDefinitionParam{
        Name:        "check_cliente",
        Description: openai.String("Verifica se o cliente existe com base no número de telefone."),
        Parameters: openai.FunctionParameters{
            "type": "object",
            "properties": map[string]any{
                "phone_number": map[string]any{
                    "type":        "string",
                    "description": "Número de telefone do cliente no formato DDD+Número (ex: 11999998888)",
                },
            },
            "required": []string{"phone_number"},
        },
    },
}

var fazerAgendamentoTool = openai.ChatCompletionToolParam{
    Function: openai.FunctionDefinitionParam{
        Name:        "fazer_agendamento",
        Description: openai.String("Cria um novo agendamento para um cliente."),
        Parameters: openai.FunctionParameters{
            "type": "object",
            "properties": map[string]any{
                "client_id": map[string]any{
                    "type":        "string",
                    "description": "ID do cliente",
                },
                "service_id": map[string]any{
                    "type":        "string",
                    "description": "ID do serviço a ser agendado",
                },
                "date": map[string]any{
                    "type":        "string",
                    "description": "Data do agendamento no formato YYYY-MM-DD",
                },
                "time": map[string]any{
                    "type":        "string",
                    "description": "Horário do agendamento no formato HH:MM",
                },
            },
            "required": []string{"client_id", "service_id", "date", "time"},
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

var agendamentosClienteTool = openai.ChatCompletionToolParam{
    Function: openai.FunctionDefinitionParam{
        Name:        "agendamentos_cliente",
        Description: openai.String("Retorna os agendamentos agendados para um cliente específico pelo ID."),
        Parameters: openai.FunctionParameters{
            "type": "object",
            "properties": map[string]any{
                "client_id": map[string]any{
                    "type":        "string",
                    "description": "ID do cliente para consulta de agendamentos",
                },
            },
            "required": []string{"client_id"},
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
        checkServicesTool,
        registerClientTool,
        checkClientTool,
        fazerAgendamentoTool,
        verificarHorariosDisponiveisTool,
        agendamentosClienteTool,
        cancelarAgendamentoTool,
        reagendarServicoTool,
    }
}