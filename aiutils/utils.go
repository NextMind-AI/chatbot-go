package aiutils

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/openai/openai-go"
)

type ToolCall struct {
	ID        string                 `json:"id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	Raw       map[string]interface{} `json:"-"`
}

// ToolCallUserError representa um erro de tool handler com uma mensagem pronta para o usuário.
type ToolCallUserError struct {
    UserMessage string // texto destinado ao usuário (já formatado)
    Err         error  // causa original
}

func (e *ToolCallUserError) Error() string { return e.Err.Error() }
func (e *ToolCallUserError) Unwrap() error { return e.Err }

func ExtractToolCallsFromCompletion(rawCompletion interface{}) ([]ToolCall, string, error) {
    var root map[string]interface{}

    // Tentar tratamento direto para os tipos suportados
    switch v := rawCompletion.(type) {
    case []byte:
        if err := json.Unmarshal(v, &root); err != nil {
            return nil, "", fmt.Errorf("json unmarshal raw bytes: %w", err)
        }
    case string:
        if err := json.Unmarshal([]byte(v), &root); err != nil {
            return nil, "", fmt.Errorf("json unmarshal raw string: %w", err)
        }
    case map[string]interface{}:
        root = v
    default:
        // DEFAULT: tentar serializar qualquer outro tipo (ex: struct do SDK) -> robusto
        b, err := json.Marshal(rawCompletion)
        if err != nil {
            return nil, "", fmt.Errorf("unsupported rawCompletion type and marshal failed: %w", err)
        }
        if err := json.Unmarshal(b, &root); err != nil {
            return nil, "", fmt.Errorf("json unmarshal after marshal: %w", err)
        }
    }

	choices, ok := root["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, "", errors.New("no choices found in completion")
	}

	// iterate choices to find tool_calls or function_call.arguments
	for i := range choices {
		choice, _ := choices[i].(map[string]interface{})
		if choice == nil {
			continue
		}

		// prefer function_call.arguments (function calling)
		if fc, found := choice["message"].(map[string]interface{})["function_call"]; found && fc != nil {
			if fcMap, ok := fc.(map[string]interface{}); ok {
				if argsRaw, exists := fcMap["arguments"]; exists {
					// arguments podem ser string JSON ou objeto
					if parsed, ok := tryParseAny(argsRaw); ok {
						if tcs := findToolCallsInInterface(parsed); len(tcs) > 0 {
							correctedJSON, _ := json.Marshal(parsed) // normalized
							return tcs, string(correctedJSON), nil
						}
						// talvez os argumentos sejam o objeto direto da tool
						// tentar transformar em ToolCall único
						if argMap, ok := parsed.(map[string]interface{}); ok {
							tc := ToolCall{
								Arguments: argMap,
							}
							return []ToolCall{tc}, "", nil
						}
					}
				}
			}
		}

		// fallback: procurar message.content
		msg := choice["message"].(map[string]interface{})
		var contentCandidates []interface{}
		if c, exists := msg["content"]; exists && c != nil && c != "" {
			contentCandidates = append(contentCandidates, c)
		}
		// se vazio, verificar audio.content
		if audio, exists := msg["audio"].(map[string]interface{}); exists && audio != nil {
			if ac, ok := audio["content"]; ok && ac != nil && ac != "" {
				contentCandidates = append(contentCandidates, ac)
			}
		}
		// também checar annotations ou other fields se quiser
		// iterate candidates
		for _, cand := range contentCandidates {
			parsed, ok := tryParseAny(cand)
			if !ok {
				continue
			}
			if tcs := findToolCallsInInterface(parsed); len(tcs) > 0 {
				// normalized JSON do bloco "tool_calls"
				correctedJSONBytes, _ := json.Marshal(parsed)
				return tcs, string(correctedJSONBytes), nil
			}
		}
	}

	// nada encontrado
	return nil, "", nil
}

func tryParseAny(v interface{}) (interface{}, bool) {
	switch x := v.(type) {
	case nil:
		return nil, false
	case map[string]interface{}, []interface{}:
		return x, true
	case string:
		s := strings.TrimSpace(x)
		// 1) try direct JSON
		var out interface{}
		if err := json.Unmarshal([]byte(s), &out); err == nil {
			return out, true
		}
		// 2) try strconv.Unquote (remove aspas externas)
		if unq, err := strconv.Unquote(s); err == nil {
			if err2 := json.Unmarshal([]byte(unq), &out); err2 == nil {
				return out, true
			}
		}
		// 3) replace escaped quotes and try
		try := strings.ReplaceAll(s, `\"`, `"`)
		if err := json.Unmarshal([]byte(try), &out); err == nil {
			return out, true
		}
		// 4) try base64 decode (algumas vezes audio/content pode conter base64)
		if decoded, err := base64.StdEncoding.DecodeString(s); err == nil {
			if err2 := json.Unmarshal(decoded, &out); err2 == nil {
				return out, true
			}
		}
		// fail
		return nil, false
	default:
		// outros tipos (numeric, bool)
		return x, true
	}
}

func findToolCallsInInterface(obj interface{}) []ToolCall {
	if obj == nil {
		return nil
	}
	// se obj é map, verifica chave "tool_calls"
	if m, ok := obj.(map[string]interface{}); ok {
		// direct key
		if tcRaw, exists := m["tool_calls"]; exists {
			if tcs := parseToolCallsArray(tcRaw); len(tcs) > 0 {
				return tcs
			}
		}
		// às vezes o objeto inteiro é o array: { "tool_calls": [...] } já tratado acima.
		// procurar recursivamente nas sub-chaves
		for _, v := range m {
			if tcs := findToolCallsInInterface(v); len(tcs) > 0 {
				return tcs
			}
		}
		return nil
	}
	// se obj é array, iterar e tentar achar em elementos
	if arr, ok := obj.([]interface{}); ok {
		var aggregate []ToolCall
		for _, item := range arr {
			if tcs := findToolCallsInInterface(item); len(tcs) > 0 {
				aggregate = append(aggregate, tcs...)
			}
		}
		return aggregate
	}
	return nil
}

func parseToolCallsArray(raw interface{}) []ToolCall {
	arr, ok := raw.([]interface{})
	if !ok {
		// talvez seja string contendo um JSON array
		if parsed, ok2 := tryParseAny(raw); ok2 {
			if arr2, ok3 := parsed.([]interface{}); ok3 {
				arr = arr2
			} else {
				return nil
			}
		} else {
			return nil
		}
	}

	var result []ToolCall
	for _, it := range arr {
		if m, ok := it.(map[string]interface{}); ok {
			tc := ToolCall{Raw: m, Arguments: map[string]interface{}{}}
			if id, ok := m["id"].(string); ok {
				tc.ID = id
			}
			if name, ok := m["name"].(string); ok {
				tc.Name = name
			}
			if args, ok := m["arguments"]; ok {
				// arguments podem ser objeto ou string JSON
				if parsed, ok := tryParseAny(args); ok {
					if argMap, ok := parsed.(map[string]interface{}); ok {
						tc.Arguments = argMap
					} else {
						// se não for map, armazenar sob chave raw_args
						tc.Arguments["raw_arg"] = parsed
					}
				} else {
					// fallback: armazena o valor cru
					tc.Arguments["raw_arg"] = args
				}
			}
			result = append(result, tc)
		}
	}
	return result
}

// removeToolCallsRecursive remove a chave "tool_calls" em qualquer nível do objeto
func removeToolCallsRecursive(v interface{}) {
	switch x := v.(type) {
	case map[string]interface{}:
		// remove direto se existir
		if _, ok := x["tool_calls"]; ok {
			delete(x, "tool_calls")
		}
		// recursão nas sub-entradas
		for _, vv := range x {
			removeToolCallsRecursive(vv)
		}
	case []interface{}:
		for _, e := range x {
			removeToolCallsRecursive(e)
		}
	}
}

// replaceRoleToolRecursive troca "role":"tool" -> "role":"assistant" em qualquer nível
func replaceRoleToolRecursive(v interface{}) {
	switch x := v.(type) {
	case map[string]interface{}:
		if role, ok := x["role"].(string); ok && role == "tool" {
			x["role"] = "assistant"
		}
		for _, vv := range x {
			replaceRoleToolRecursive(vv)
		}
	case []interface{}:
		for _, e := range x {
			replaceRoleToolRecursive(e)
		}
	}
}

func SanitizeMessagesForNoTools(msgs []openai.ChatCompletionMessageParamUnion) []openai.ChatCompletionMessageParamUnion {
	sanitized := make([]openai.ChatCompletionMessageParamUnion, 0, len(msgs))

	for _, m := range msgs {
		b, err := json.Marshal(m)
		if err != nil {
			// não conseguimos serializar -> fallback para original
			sanitized = append(sanitized, m)
			continue
		}

		var obj interface{}
		if err := json.Unmarshal(b, &obj); err != nil {
			sanitized = append(sanitized, m)
			continue
		}

		// remove qualquer tool_calls e conserta role:"tool"
		removeToolCallsRecursive(obj)
		replaceRoleToolRecursive(obj)

		nb, err := json.Marshal(obj)
		if err != nil {
			sanitized = append(sanitized, m)
			continue
		}

		var nm openai.ChatCompletionMessageParamUnion
		if err := json.Unmarshal(nb, &nm); err != nil {
			// se desserializar pra union falhar, mantém original
			sanitized = append(sanitized, m)
			continue
		}

		sanitized = append(sanitized, nm)
	}

	return sanitized
}
