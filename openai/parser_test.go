package openai

import (
	"reflect"
	"testing"
)

func TestStreamingJSONParser_BasicParsing(t *testing.T) {
	parser := NewStreamingJSONParser()

	jsonData := `{
		"messages": [
			{"content": "Hello world", "type": "text"},
			{"content": "How are you?", "type": "text"}
		]
	}`

	messages := parser.AddChunk(jsonData)

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}

	if messages[0].Content != "Hello world" {
		t.Errorf("Expected first message content 'Hello world', got '%s'", messages[0].Content)
	}

	if messages[1].Content != "How are you?" {
		t.Errorf("Expected second message content 'How are you?', got '%s'", messages[1].Content)
	}
}

func TestStreamingJSONParser_VariousFormats(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []Message
	}{
		{
			name:  "No spaces",
			input: `{"messages":[{"content":"Test1","type":"text"}]}`,
			expected: []Message{
				{Content: "Test1", Type: "text"},
			},
		},
		{
			name:  "Spaces around colons",
			input: `{"messages" : [{"content" : "Test2", "type" : "text"}]}`,
			expected: []Message{
				{Content: "Test2", Type: "text"},
			},
		},
		{
			name:  "Spaces after braces",
			input: `{ "messages": [{ "content": "Test3", "type": "text" }] }`,
			expected: []Message{
				{Content: "Test3", Type: "text"},
			},
		},
		{
			name:  "Mixed spacing",
			input: `{"messages" :[{ "content":"Test4","type" : "text" }]}`,
			expected: []Message{
				{Content: "Test4", Type: "text"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewStreamingJSONParser()
			messages := parser.AddChunk(tc.input)

			if len(messages) != len(tc.expected) {
				t.Errorf("Expected %d messages, got %d", len(tc.expected), len(messages))
			}

			for i, msg := range messages {
				if msg.Content != tc.expected[i].Content {
					t.Errorf("Expected content '%s', got '%s'", tc.expected[i].Content, msg.Content)
				}
				if msg.Type != tc.expected[i].Type {
					t.Errorf("Expected type '%s', got '%s'", tc.expected[i].Type, msg.Type)
				}
			}
		})
	}
}

func TestStreamingJSONParser_StreamingChunks(t *testing.T) {
	parser := NewStreamingJSONParser()

	chunks := []string{
		`{"mes`,
		`sages": [`,
		`{"content": "First`,
		` message", "type": "text"},`,
		`{"content": "Second message",`,
		` "type": "text"}`,
		`]}`,
	}

	var allMessages []Message
	for _, chunk := range chunks {
		messages := parser.AddChunk(chunk)
		allMessages = append(allMessages, messages...)
	}

	if len(allMessages) != 2 {
		t.Errorf("Expected 2 messages total, got %d", len(allMessages))
	}

	if allMessages[0].Content != "First message" {
		t.Errorf("Expected first message 'First message', got '%s'", allMessages[0].Content)
	}

	if allMessages[1].Content != "Second message" {
		t.Errorf("Expected second message 'Second message', got '%s'", allMessages[1].Content)
	}
}

func TestStreamingJSONParser_NestedObjects(t *testing.T) {
	parser := NewStreamingJSONParser()

	jsonData := `{
		"messages": [
			{
				"content": "Message with {nested} braces",
				"type": "text",
				"metadata": {"key": "value"}
			}
		]
	}`

	messages := parser.AddChunk(jsonData)

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	if messages[0].Content != "Message with {nested} braces" {
		t.Errorf("Expected content with nested braces, got '%s'", messages[0].Content)
	}
}

func TestStreamingJSONParser_EscapedCharacters(t *testing.T) {
	parser := NewStreamingJSONParser()

	jsonData := `{
		"messages": [
			{"content": "Message with \"quotes\" and \\backslashes\\", "type": "text"},
			{"content": "Message with } brace in string", "type": "text"}
		]
	}`

	messages := parser.AddChunk(jsonData)

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}

	if messages[0].Content != `Message with "quotes" and \backslashes\` {
		t.Errorf("Expected escaped content, got '%s'", messages[0].Content)
	}

	if messages[1].Content != "Message with } brace in string" {
		t.Errorf("Expected content with brace in string, got '%s'", messages[1].Content)
	}
}

func TestStreamingJSONParser_MultipleChunksWithPartialMessages(t *testing.T) {
	parser := NewStreamingJSONParser()

	chunks := []string{
		`{"messages": [{"content": "Complete message", "type": "text"},`,
		`{"content": "Partial`,
		` message`,
		` that spans chunks", "type": "text"}]}`,
	}

	var messageCount []int
	for _, chunk := range chunks {
		messages := parser.AddChunk(chunk)
		messageCount = append(messageCount, len(messages))
	}

	expectedCounts := []int{1, 0, 0, 1}
	if !reflect.DeepEqual(messageCount, expectedCounts) {
		t.Errorf("Expected message counts %v, got %v", expectedCounts, messageCount)
	}

	if parser.MsgCount != 2 {
		t.Errorf("Expected total message count 2, got %d", parser.MsgCount)
	}
}

func TestStreamingJSONParser_EmptyMessages(t *testing.T) {
	parser := NewStreamingJSONParser()

	jsonData := `{"messages": []}`
	messages := parser.AddChunk(jsonData)

	if len(messages) != 0 {
		t.Errorf("Expected 0 messages for empty array, got %d", len(messages))
	}
}

func TestStreamingJSONParser_MessageWithOnlyType(t *testing.T) {
	parser := NewStreamingJSONParser()

	jsonData := `{
		"messages": [
			{"type": "system"},
			{"content": "User message", "type": "text"}
		]
	}`

	messages := parser.AddChunk(jsonData)

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}

	if messages[0].Type != "system" {
		t.Errorf("Expected first message type 'system', got '%s'", messages[0].Type)
	}

	if messages[0].Content != "" {
		t.Errorf("Expected empty content for first message, got '%s'", messages[0].Content)
	}
}

func TestStreamingJSONParser_RealWorldStreamingScenario(t *testing.T) {
	parser := NewStreamingJSONParser()

	chunks := []string{
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,`,
		`"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}],`,
		`"messages":[`,
		`{"content":"I'll help you with that.","type":"text"},`,
		`{"content":"Here's the solution:","type":"text"}`,
		`]}`,
	}

	var totalMessages int
	for _, chunk := range chunks {
		messages := parser.AddChunk(chunk)
		totalMessages += len(messages)
	}

	if totalMessages != 2 {
		t.Errorf("Expected 2 messages total, got %d", totalMessages)
	}

	if parser.MsgCount != 2 {
		t.Errorf("Expected MsgCount to be 2, got %d", parser.MsgCount)
	}
}

func TestStreamingJSONParser_IncompleteJSON(t *testing.T) {
	parser := NewStreamingJSONParser()

	chunks := []string{
		`{"messages": [{"content": "First message", "type": "text"},`,
		`{"content": "Incomplete message"`,
	}

	var totalMessages int
	for _, chunk := range chunks {
		messages := parser.AddChunk(chunk)
		totalMessages += len(messages)
	}

	if totalMessages != 1 {
		t.Errorf("Expected only 1 complete message, got %d", totalMessages)
	}
}

func TestStreamingJSONParser_ComplexNestedStructure(t *testing.T) {
	parser := NewStreamingJSONParser()

	jsonData := `{
		"messages": [
			{
				"content": "Complex message",
				"type": "text",
				"metadata": {
					"nested": {
						"deep": {
							"value": "test"
						}
					}
				}
			}
		]
	}`

	messages := parser.AddChunk(jsonData)

	if len(messages) != 1 {
		t.Errorf("Expected 1 message with complex structure, got %d", len(messages))
	}

	if messages[0].Content != "Complex message" {
		t.Errorf("Expected 'Complex message', got '%s'", messages[0].Content)
	}
}
