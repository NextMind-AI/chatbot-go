# ElevenLabs Speech-to-Text Integration

This package provides integration with ElevenLabs' speech-to-text API for transcribing audio messages in the chatbot.

## Features

- **Simple Audio Transcription**: Convert audio URLs to text with a single method call
- **Multiple Audio Formats**: Support for various audio formats (MP3, WAV, OGG, AAC, FLAC, M4A, WebM)
- **Language Detection**: Automatic language detection
- **WhatsApp Integration**: Seamless integration with Vonage WhatsApp messages
- **Error Handling**: Comprehensive error handling and logging

## Setup

### 1. Environment Variables

Add your ElevenLabs API key to your environment variables:

```bash
export ELEVENLABS_API_KEY="your_elevenlabs_api_key_here"
```

Or add it to your `.env` file:

```env
ELEVENLABS_API_KEY=your_elevenlabs_api_key_here
```

### 2. Get ElevenLabs API Key

1. Sign up at [ElevenLabs](https://elevenlabs.io/)
2. Navigate to your profile settings
3. Generate an API key
4. Copy the key to your environment variables

## Usage

### Simple Transcription

```go
package main

import (
    "chatbot/elevenlabs"
    "net/http"
)

func main() {
    client := elevenlabs.NewClient("your-api-key", http.Client{})
    
    // Transcribe audio from URL - this is all you need!
    transcribedText, err := client.TranscribeAudio("https://example.com/audio.mp3")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Transcribed text:", transcribedText)
}
```

### Advanced Usage

For more control over the transcription process:

```go
// Use the full response
response, err := client.ProcessAudioFromURL("https://example.com/audio.mp3", "en")
if err != nil {
    log.Fatal(err)
}

fmt.Println("Text:", response.Text)
fmt.Println("Language:", response.LanguageCode)
fmt.Println("Confidence:", response.LanguageProbability)

// From file
audioFile := strings.NewReader("audio data here")
response, err := client.TranscribeAudioFile(audioFile, "audio.mp3", "en")
if err != nil {
    log.Fatal(err)
}
```

## WhatsApp Integration

The integration automatically handles WhatsApp audio messages received through Vonage. When a user sends an audio message:

1. The webhook receives the message with an `audio` field containing the URL
2. The system calls `client.TranscribeAudio(audioURL)` 
3. The transcribed text is processed by the chatbot

### Message Processing

```go
func processAudioMessage(message InboundMessage) (string, error) {
    if message.Audio == nil || message.Audio.URL == "" {
        return "", nil
    }

    return ElevenLabsClient.TranscribeAudio(message.Audio.URL)
}
```

### Message Structure

WhatsApp audio messages have this structure:

```json
{
  "message_type": "audio",
  "audio": {
    "url": "https://example.com/audio.mp3"
  },
  "from": "1234567890",
  "message_uuid": "uuid-here"
}
```

## API Reference

### Client

#### `NewClient(apiKey string, httpClient http.Client) Client`

Creates a new ElevenLabs client.

#### `TranscribeAudio(url string) (string, error)`

**Main method for simple audio transcription.** Takes an audio URL and returns the transcribed text.

#### `ProcessAudioFromURL(url string, language string) (*SpeechToTextResponse, error)`

Advanced method that returns the full response with language detection and confidence scores.

#### `TranscribeAudioFile(file io.Reader, fileName string, languageCode string) (*SpeechToTextResponse, error)`

Transcribes audio files directly.

#### `SpeechToText(req SpeechToTextRequest) (*SpeechToTextResponse, error)`

Low-level method for full control over all parameters.

### Types

#### `SpeechToTextResponse`

```go
type SpeechToTextResponse struct {
    LanguageCode        string  // Detected language
    LanguageProbability float64 // Language confidence (0-1)
    Text                string  // Transcribed text
    Words               []Word  // Word-level details
}
```

#### `Word`

```go
type Word struct {
    Text      string  // Word text
    Type      string  // "word" or "spacing"
    LogProb   float64 // Log probability
    Start     float64 // Start time in seconds
    End       float64 // End time in seconds
    SpeakerID string  // Speaker identifier
}
```

## Supported Audio Formats

- **Audio**: MP3, WAV, OGG, AAC, FLAC, M4A, WebM
- **Raw**: PCM 16-bit 16kHz (for lower latency)

## Error Handling

The package provides comprehensive error handling:

```go
transcribedText, err := client.TranscribeAudio("https://example.com/audio.mp3")
if err != nil {
    if apiErr, ok := err.(elevenlabs.APIError); ok {
        fmt.Printf("API Error %d: %s\n", apiErr.StatusCode, apiErr.Message)
    } else {
        fmt.Printf("Other error: %s\n", err.Error())
    }
    return
}
```

## Logging

The package uses structured logging with zerolog. Log levels:

- **Debug**: Request/response details
- **Info**: Successful operations and results
- **Warn**: Non-critical issues (e.g., missing API key)
- **Error**: Failed operations

## Configuration

### Environment Variables

- `ELEVENLABS_API_KEY`: Your ElevenLabs API key (required)

### Default Settings

- **Model**: `scribe_v1`
- **Tag Audio Events**: `true`
- **Timestamps Granularity**: `word`
- **Language Detection**: Auto-detect

## Examples

### Basic Usage

```go
// Simple transcription - recommended approach
text, err := client.TranscribeAudio("https://example.com/audio.mp3")
if err != nil {
    log.Fatal(err)
}
fmt.Println("Transcribed:", text)
```

### With Language Detection Info

```go
response, err := client.ProcessAudioFromURL("https://example.com/audio.mp3", "")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Text: %s\n", response.Text)
fmt.Printf("Language: %s (%.2f confidence)\n", 
    response.LanguageCode, response.LanguageProbability)
```

## Troubleshooting

### Common Issues

1. **Missing API Key**: Ensure `ELEVENLABS_API_KEY` is set
2. **File Size**: Maximum file size is 1GB for uploads, 2GB for cloud URLs
3. **Network Issues**: Check internet connectivity and firewall settings
4. **Audio Format**: Ensure audio format is supported

### Debug Mode

Enable debug logging to see detailed request/response information:

```go
log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}).Level(zerolog.DebugLevel)
``` 
