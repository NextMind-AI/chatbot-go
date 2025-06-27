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

	fmt.Printf("DEBUG: parseNewMessages called, buffer length: %d, lastParsedPos: %d\n", len(content), p.lastParsedPos)

	// Look for "messages" array with flexible whitespace
	if !p.foundMessages {
		// Try different variations of the messages array pattern
		patterns := []string{
			`"messages":[`,
			`"messages": [`,  // with space before bracket
			`"messages" :[`,  // with space before colon
			`"messages" : [`, // with spaces around colon
		}

		for _, pattern := range patterns {
			if strings.Contains(content, pattern) {
				p.foundMessages = true
				fmt.Printf("DEBUG: Found 'messages' array in content with pattern: %q\n", pattern)
				break
			}
		}
	}

	searchContent := content[p.lastParsedPos:]
	fmt.Printf("DEBUG: Searching in content: %q\n", searchContent)

	for {
		// Look for opening braces that might start a message object
		braceIdx := strings.Index(searchContent, "{")
		if braceIdx == -1 {
			fmt.Printf("DEBUG: No more opening braces found in search content\n")
			break
		}

		// Skip this brace if we haven't found the messages array yet
		if !p.foundMessages {
			searchContent = searchContent[braceIdx+1:]
			p.lastParsedPos++
			continue
		}

		// Check if this is a message object by looking for "content" or "type" fields
		// directly after the opening brace (allowing for whitespace)
		checkEnd := braceIdx + 50
		if checkEnd > len(searchContent) {
			checkEnd = len(searchContent)
		}
		checkContent := searchContent[braceIdx:checkEnd]

		// Look for patterns that indicate this is a message object
		isMessage := false
		patterns := []string{
			`{"content"`,
			`{ "content"`,
			`{` + "\n" + `"content"`,
			`{` + "\t" + `"content"`,
			`{` + "\r\n" + `"content"`,
			`{"type"`,
			`{ "type"`,
			`{` + "\n" + `"type"`,
			`{` + "\t" + `"type"`,
			`{` + "\r\n" + `"type"`,
			`{` + "\n\t\t\t\t" + `"content"`,
			`{` + "\n\t\t\t" + `"content"`,
		}

		for _, pattern := range patterns {
			if strings.HasPrefix(checkContent, pattern) {
				isMessage = true
				break
			}
		}

		if !isMessage {
			// This brace doesn't start a message object, skip it
			searchContent = searchContent[braceIdx+1:]
			p.lastParsedPos++
			continue
		}

		startIdx := braceIdx
		foundPattern := "message object"

		fullStartIdx := p.lastParsedPos + startIdx
		fmt.Printf("DEBUG: Found potential message start at index %d using pattern: %q\n", fullStartIdx, foundPattern)

		endIdx := p.findMessageEnd(content, fullStartIdx)
		if endIdx == -1 {
			fmt.Printf("DEBUG: Could not find message end for message starting at %d\n", fullStartIdx)
			break
		}

		messageJSON := content[fullStartIdx : endIdx+1]
		fmt.Printf("DEBUG: Extracted message JSON: %q\n", messageJSON)

		var msg Message
		if err := json.Unmarshal([]byte(messageJSON), &msg); err == nil {
			p.MsgCount++
			parsedMessages = append(parsedMessages, msg)
			fmt.Printf("DEBUG: Successfully parsed message %d: content=%q, type=%q\n", p.MsgCount, msg.Content, msg.Type)
		} else {
			fmt.Printf("DEBUG: Failed to unmarshal message JSON: %v\n", err)
		}

		p.lastParsedPos = endIdx + 1
		searchContent = content[p.lastParsedPos:]
		fmt.Printf("DEBUG: Updated lastParsedPos to %d\n", p.lastParsedPos)
	}

	fmt.Printf("DEBUG: parseNewMessages returning %d messages\n", len(parsedMessages))
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
