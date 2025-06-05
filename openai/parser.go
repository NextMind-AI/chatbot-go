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

// NewStreamingJSONParser creates a new instance of StreamingJSONParser.
func NewStreamingJSONParser() *StreamingJSONParser {
	return &StreamingJSONParser{}
}

// AddChunk appends a new chunk of JSON data to the parser's buffer and attempts
// to parse any complete Message objects. Returns a slice of newly parsed messages.
func (p *StreamingJSONParser) AddChunk(chunk string) []Message {
	p.buffer.WriteString(chunk)
	messages := p.parseNewMessages()

	// Add debug logging to understand what's happening
	if len(messages) > 0 {
		log.Debug().
			Int("messages_found", len(messages)).
			Str("chunk", chunk).
			Str("buffer_content", p.buffer.String()).
			Msg("Parser found messages in chunk")
	}

	return messages
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

	// Log buffer state for debugging
	if len(searchContent) > 0 {
		// Check if we have potential message patterns
		hasContentPattern := strings.Contains(searchContent, `"content":`)
		hasTypePattern := strings.Contains(searchContent, `"type":`)
		hasMessagesPattern := strings.Contains(searchContent, `"messages":`)

		// Log detailed buffer state for debugging
		sample := searchContent
		if len(sample) > 300 {
			sample = sample[:300] + "..."
		}
		log.Debug().
			Int("buffer_length", len(searchContent)).
			Int("parsed_pos", p.lastParsedPos).
			Bool("has_content_pattern", hasContentPattern).
			Bool("has_type_pattern", hasTypePattern).
			Bool("has_messages_pattern", hasMessagesPattern).
			Bool("found_messages", p.foundMessages).
			Str("buffer_sample", sample).
			Msg("Parser buffer state")
	}

	for {
		// Look for message objects more flexibly, accounting for whitespace
		startIdx := p.findMessageStart(searchContent)
		if startIdx == -1 {
			break
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
		} else {
			log.Warn().
				Err(err).
				Str("json", messageJSON).
				Msg("Failed to parse message JSON")
		}

		p.lastParsedPos = endIdx + 1
		searchContent = content[p.lastParsedPos:]
	}

	return parsedMessages
}

// findMessageStart looks for the start of a message object, handling whitespace flexibly
func (p *StreamingJSONParser) findMessageStart(searchContent string) int {
	// First try exact patterns
	startIdx := strings.Index(searchContent, `{"content":`)
	if startIdx != -1 {
		log.Debug().
			Str("pattern", `{"content":`).
			Int("position", startIdx).
			Msg("Found message start with exact content pattern")
		return startIdx
	}

	startIdx = strings.Index(searchContent, `{"type":`)
	if startIdx != -1 {
		log.Debug().
			Str("pattern", `{"type":`).
			Int("position", startIdx).
			Msg("Found message start with exact type pattern")
		return startIdx
	}

	// Then try patterns with whitespace after opening brace
	startIdx = strings.Index(searchContent, `{ "content":`)
	if startIdx != -1 {
		log.Debug().
			Str("pattern", `{ "content":`).
			Int("position", startIdx).
			Msg("Found message start with spaced content pattern")
		return startIdx
	}

	startIdx = strings.Index(searchContent, `{ "type":`)
	if startIdx != -1 {
		log.Debug().
			Str("pattern", `{ "type":`).
			Int("position", startIdx).
			Msg("Found message start with spaced type pattern")
		return startIdx
	}

	// If no patterns found, log the search content for debugging
	if len(searchContent) > 0 {
		sample := searchContent
		if len(sample) > 100 {
			sample = sample[:100] + "..."
		}
		log.Debug().
			Str("search_content", sample).
			Int("content_length", len(searchContent)).
			Msg("No message patterns found in search content")
	}

	return -1
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
