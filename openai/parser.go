package openai

import (
	"encoding/json"
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
	return p.parseNewMessages()
}

// parseNewMessages scans the buffer for complete message objects and parses them.
// It maintains the parsing position to avoid re-parsing already processed messages.
func (p *StreamingJSONParser) parseNewMessages() []Message {
	content := p.buffer.String()
	var parsedMessages []Message

	if !p.foundMessages && strings.Contains(content, `"messages":[`) {
		p.foundMessages = true
	}

	searchContent := content[p.lastParsedPos:]

	for {
		startIdx := strings.Index(searchContent, `{"content":`)
		if startIdx == -1 {
			startIdx = strings.Index(searchContent, `{"type":`)
			if startIdx == -1 {
				break
			}
		}

		fullStartIdx := p.lastParsedPos + startIdx

		endIdx := p.findMessageEnd(content, fullStartIdx)
		if endIdx == -1 {
			break
		}

		messageJSON := content[fullStartIdx : endIdx+1]

		var msg Message
		if err := json.Unmarshal([]byte(messageJSON), &msg); err == nil {
			p.MsgCount++
			parsedMessages = append(parsedMessages, msg)
		}

		p.lastParsedPos = endIdx + 1
		searchContent = content[p.lastParsedPos:]
	}

	return parsedMessages
}

// findMessageEnd locates the closing brace of a JSON object starting at startIdx.
// It properly handles nested objects and string escaping to ensure we find the
// correct closing brace for the message object.
func (p *StreamingJSONParser) findMessageEnd(content string, startIdx int) int {
	braceCount := 0
	inString := false
	escaped := false

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
			if char == '{' {
				braceCount++
			} else if char == '}' {
				braceCount--
				if braceCount == 0 {
					return i
				}
			}
		}
	}

	return -1
}
