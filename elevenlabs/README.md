# ElevenLabs Speech-to-Text Integration

This package provides integration with ElevenLabs' speech-to-text API for transcribing audio messages in the chatbot.

## Features

- **Audio Transcription**: Convert audio messages to text using ElevenLabs' advanced speech-to-text models
- **Multiple Audio Formats**: Support for various audio and video formats (MP3, WAV, MP4, etc.)
- **Language Detection**: Automatic language detection or manual language specification
- **WhatsApp Integration**: Seamless integration with Vonage WhatsApp messages
- **Error Handling**: Comprehensive error handling and logging
- **Flexible Input**: Support for both file uploads and cloud storage URLs

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

### Basic Client Usage

```go
package main

import (
    "chatbot/elevenlabs"
    "net/http"
    "strings"
)

func main() {
    // Initialize the client
    client := elevenlabs.NewClient("your-api-key", &http.Client{})
    
    // Transcribe from a file
    audioFile := strings.NewReader("audio data here")
    response, err := client.TranscribeAudioFile(audioFile, "audio.mp3", "en")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Transcribed text:", response.Text)
}
```

### Audio Handler Usage

```go
package main

import (
    "chatbot/elevenlabs"
    "net/http"
)

func main() {
    // Initialize client and handler
    client := elevenlabs.NewClient("your-api-key", &http.Client{})
    handler := elevenlabs.NewAudioHandler(&client, &http.Client{})
    
    // Process audio message
    audioMsg := elevenlabs.AudioMessage{
        URL:         "https://example.com/audio.mp3",
        ContentType: "audio/mpeg",
        Language:    "en", // Optional, auto-detect if empty
    }
    
    response, err := handler.ProcessAudioMessage(audioMsg)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Transcribed text:", response.Text)
}
```

## WhatsApp Integration

The integration automatically handles WhatsApp audio messages received through Vonage. When a user sends an audio message:

1. The webhook receives the message with an `audio` field containing the URL
2. The system automatically downloads and transcribes the audio
3. The transcribed text is combined with any existing text message
4. The combined message is processed by the chatbot

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

#### `NewClient(apiKey string, httpClient *http.Client) Client`

Creates a new ElevenLabs client.

#### `SpeechToText(req SpeechToTextRequest) (*SpeechToTextResponse, error)`

Main method for speech-to-text conversion with full control over parameters.

#### `TranscribeAudioFile(file io.Reader, fileName string, languageCode string) (*SpeechToTextResponse, error)`

Convenience method for transcribing audio files.

#### `TranscribeAudioURL(cloudStorageURL string, languageCode string) (*SpeechToTextResponse, error)`

Convenience method for transcribing audio from URLs.

### AudioHandler

#### `NewAudioHandler(client *Client, httpClient *http.Client) *AudioHandler`

Creates a new audio handler.

#### `ProcessAudioMessage(audioMsg AudioMessage) (*SpeechToTextResponse, error)`

Downloads and transcribes an audio message.

#### `ProcessAudioURL(url string, language string) (*SpeechToTextResponse, error)`

Transcribes audio directly from a URL.

### Types

#### `SpeechToTextRequest`

```go
type SpeechToTextRequest struct {
    ModelID               string             // Model to use (default: "scribe_v1")
    File                  io.Reader          // Audio file data
    FileName              string             // File name
    LanguageCode          string             // Language code (optional)
    TagAudioEvents        *bool              // Tag audio events like (laughter)
    NumSpeakers           *int               // Maximum number of speakers
    TimestampsGranularity string             // "word" or "character"
    Diarize               *bool              // Speaker identification
    FileFormat            string             // "pcm_s16le_16" or "other"
    CloudStorageURL       string             // Alternative to File
    Webhook               *bool              // Async processing
    EnableLogging         *bool              // Enable/disable logging
}
```

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
- **Video**: MP4, WebM, QuickTime (MOV)
- **Raw**: PCM 16-bit 16kHz (for lower latency)

## Error Handling

The package provides comprehensive error handling:

```go
response, err := client.TranscribeAudioFile(file, "audio.mp3", "en")
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
- **Timeout**: 30 seconds
- **Tag Audio Events**: `true`
- **Timestamps Granularity**: `word`

## Examples

### Transcribe WhatsApp Audio

```go
// This happens automatically when audio messages are received
// The transcribed text is added to the message processing pipeline
```

### Manual Transcription

```go
// From file
file, _ := os.Open("audio.mp3")
response, err := client.TranscribeAudioFile(file, "audio.mp3", "en")

// From URL
response, err := client.TranscribeAudioURL("https://example.com/audio.mp3", "en")

// Advanced usage
req := elevenlabs.SpeechToTextRequest{
    ModelID:               "scribe_v1",
    File:                  file,
    FileName:              "audio.mp3",
    LanguageCode:          "en",
    TagAudioEvents:        &[]bool{true}[0],
    TimestampsGranularity: "word",
    Diarize:               &[]bool{true}[0],
}
response, err := client.SpeechToText(req)
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

## License

This integration is part of the chatbot project and follows the same license terms. 