package openai

import (
	"encoding/json"
	"strings"

	"github.com/rs/zerolog/log"
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

// GetBuffer returns the current buffer content for debugging purposes
func (p *StreamingJSONParser) GetBuffer() string {
	return p.buffer.String()
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

	// TEMP: Add debug logging to understand parser behavior
	log.Info().
		Int("buffer_length", len(content)).
		Str("buffer_content", content).
		Int("last_parsed_pos", p.lastParsedPos).
		Bool("found_messages", p.foundMessages).
		Msg("PARSER DEBUG: Starting parseNewMessages")

	if !p.foundMessages && strings.Contains(content, `"messages":[`) {
		p.foundMessages = true
		log.Info().Msg("PARSER DEBUG: Found messages array in content")
	}

	searchContent := content[p.lastParsedPos:]

	// TEMP: Add debug logging for search content
	log.Info().
		Str("search_content", searchContent).
		Int("search_content_length", len(searchContent)).
		Msg("PARSER DEBUG: Search content for parsing")

	for {
		startIdx := strings.Index(searchContent, `{"content":`)
		if startIdx == -1 {
			startIdx = strings.Index(searchContent, `{"type":`)
			if startIdx == -1 {
				log.Info().
					Str("search_content", searchContent).
					Msg("PARSER DEBUG: No message start patterns found")
				break
			}
		}

		fullStartIdx := p.lastParsedPos + startIdx

		log.Info().
			Int("full_start_idx", fullStartIdx).
			Msg("PARSER DEBUG: Found potential message start")

		endIdx := p.findMessageEnd(content, fullStartIdx)
		if endIdx == -1 {
			log.Info().
				Int("full_start_idx", fullStartIdx).
				Msg("PARSER DEBUG: Could not find message end")
			break
		}

		messageJSON := content[fullStartIdx : endIdx+1]

		log.Info().
			Str("message_json", messageJSON).
			Msg("PARSER DEBUG: Attempting to parse message JSON")

		var msg Message
		if err := json.Unmarshal([]byte(messageJSON), &msg); err == nil {
			p.MsgCount++
			parsedMessages = append(parsedMessages, msg)
			log.Info().
				Str("parsed_content", msg.Content).
				Str("parsed_type", msg.Type).
				Int("msg_count", p.MsgCount).
				Msg("PARSER DEBUG: Successfully parsed message")
		} else {
			log.Info().
				Err(err).
				Str("message_json", messageJSON).
				Msg("PARSER DEBUG: Failed to unmarshal message JSON")
		}

		p.lastParsedPos = endIdx + 1
		searchContent = content[p.lastParsedPos:]
	}

	log.Info().
		Int("parsed_messages_count", len(parsedMessages)).
		Msg("PARSER DEBUG: Completed parseNewMessages")

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
