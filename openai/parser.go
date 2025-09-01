package openai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// StreamingJSONParser handles incremental parsing of JSON responses from OpenAI's streaming API.
// It's designed to extract complete Message objects as they become available in the stream,
// even when JSON data arrives in partial chunks.
type StreamingJSONParser struct {
	// buffer accumulates the incoming JSON chunks
	buffer strings.Builder
	// lastParsedPos tracks the position in buffer up to which we've already parsed
	lastParsedPos int
	// MsgCount keeps track of how many messages we've parsed so far (exported for external access)
	MsgCount int
	// foundMessages indicates whether we've encountered the "messages" array in the JSON
	foundMessages bool
}

// NewStreamingJSONParser creates a new instance of StreamingJSONParser.
func NewStreamingJSONParser() *StreamingJSONParser {
	return &StreamingJSONParser{}
}

// AddChunk appends a new chunk of JSON data to the parser's buffer and attempts
// to parse any complete Message objects. Returns a slice of newly parsed messages.
func (p *StreamingJSONParser) AddChunk(chunk string) []Message {
	p.buffer.WriteString(chunk)

	// Debug logging to understand what content we're receiving
	if len(chunk) > 0 {
		fmt.Printf("DEBUG: AddChunk received: %q\n", chunk)
		fmt.Printf("DEBUG: Buffer now contains: %q\n", p.buffer.String())
	}

	messages := p.parseNewMessages()
	if len(messages) > 0 {
		fmt.Printf("DEBUG: Parsed %d messages from chunk\n", len(messages))
	}

	return messages
}

// parseNewMessages scans the buffer for complete message objects and parses them.
// It maintains the parsing position to avoid re-parsing already processed messages.
func (p *StreamingJSONParser) parseNewMessages() []Message {
	content := p.buffer.String()
	var parsedMessages []Message

	fmt.Printf("DEBUG: parseNewMessages called, buffer length: %d, current MsgCount: %d\n", len(content), p.MsgCount)

	// Sempre tenta fazer parse do JSON completo se contém "messages"
	if strings.Contains(content, `"messages":[`) || strings.Contains(content, `"messages": [`) {
		fmt.Printf("DEBUG: Attempting to parse complete JSON\n")

		var messageList MessageList
		if err := json.Unmarshal([]byte(content), &messageList); err == nil {
			fmt.Printf("DEBUG: Successfully parsed complete JSON with %d total messages\n", len(messageList.Messages))

			// Retorna apenas as mensagens que ainda não foram processadas
			if len(messageList.Messages) > p.MsgCount {
				newMessages := messageList.Messages[p.MsgCount:]
				oldMsgCount := p.MsgCount
				p.MsgCount = len(messageList.Messages)

				fmt.Printf("DEBUG: Returning %d new messages (from %d to %d)\n", len(newMessages), oldMsgCount, p.MsgCount)

				for i, msg := range newMessages {
					fmt.Printf("DEBUG: New message %d: content=%q, type=%q\n", oldMsgCount+i+1, msg.Content, msg.Type)
				}

				return newMessages
			} else {
				fmt.Printf("DEBUG: No new messages to return (already processed %d messages)\n", p.MsgCount)
				return parsedMessages
			}
		} else {
			fmt.Printf("DEBUG: Failed to parse complete JSON: %v\n", err)
			fmt.Printf("DEBUG: JSON content: %q\n", content)
		}
	} else {
		fmt.Printf("DEBUG: No 'messages' array found in content yet\n")
	}

	fmt.Printf("DEBUG: parseNewMessages returning 0 messages (incomplete or invalid JSON)\n")
	return parsedMessages
}

// findMessageEnd locates the closing brace of a JSON object starting at startIdx.
// It properly handles nested objects and string escaping to ensure we find the
// correct closing brace for the message object.
func (p *StreamingJSONParser) findMessageEnd(content string, startIdx int) int {
	braceCount := 0
	inString := false
	escaped := false

	endPreview := startIdx + 50
	if endPreview > len(content) {
		endPreview = len(content)
	}
	fmt.Printf("DEBUG: findMessageEnd called for content starting at %d: %q\n", startIdx, content[startIdx:endPreview])

	for i := startIdx; i < len(content); i++ {
		char := content[i]

		if escaped {
			escaped = false
			continue
		}

		if char == '\\' {
			escaped = true
			continue
		}

		if char == '"' {
			inString = !inString
			continue
		}

		if !inString {
			switch char {
			case '{':
				braceCount++
			case '}':
				braceCount--
				if braceCount == 0 {
					fmt.Printf("DEBUG: findMessageEnd found closing brace at index %d\n", i)
					return i
				}
			}
		}
	}

	fmt.Printf("DEBUG: findMessageEnd could not find closing brace, returning -1\n")
	return -1
}
