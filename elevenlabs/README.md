# ElevenLabs Speech-to-Text Integration

Simple and clean integration with ElevenLabs' speech-to-text API for transcribing audio files.

## Features

- **Single Method API**: One simple method to transcribe audio files
- **Configurable Language**: Set language code in client constructor 
- **Multiple Audio Formats**: Support for MP3, WAV, OGG, AAC, FLAC, M4A, WebM
- **Clean Error Handling**: Simple error responses

## Setup

### Environment Variables

Add your ElevenLabs API key to your `.env` file:

```env
ELEVENLABS_API_KEY=your_elevenlabs_api_key_here
```

### Get ElevenLabs API Key

1. Sign up at [ElevenLabs](https://elevenlabs.io/)
2. Navigate to your profile settings
3. Generate an API key

## Usage

### Basic Usage

```go
package main

import (
    "chatbot/elevenlabs"
    "net/http"
    "os"
)

func main() {
    // Create client with language code (empty string for auto-detection)
    client := elevenlabs.NewClient("your-api-key", "", http.Client{})
    
    // Open audio file
    file, err := os.Open("audio.mp3")
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()
    
    // Transcribe audio - returns just the text
    text, err := client.TranscribeAudioFile(file, "audio.mp3")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Transcribed:", text)
}
```

### With Specific Language

```go
// Create client with specific language
client := elevenlabs.NewClient("your-api-key", "en", http.Client{})

text, err := client.TranscribeAudioFile(file, "audio.mp3")
if err != nil {
    log.Fatal(err)
}
```

### From HTTP Download

```go
// Download audio first
resp, err := http.Get("https://example.com/audio.mp3")
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()

// Transcribe the downloaded audio
text, err := client.TranscribeAudioFile(resp.Body, "audio.mp3")
if err != nil {
    log.Fatal(err)
}
```

### From Byte Data

```go
audioData := []byte{/* audio file content */}
reader := bytes.NewReader(audioData)

text, err := client.TranscribeAudioFile(reader, "audio.wav")
if err != nil {
    log.Fatal(err)
}
```

## API Reference

### Client

#### `NewClient(apiKey string, languageCode string, httpClient http.Client) Client`

Creates a new ElevenLabs client.

**Parameters:**
- `apiKey`: Your ElevenLabs API key
- `languageCode`: Language code (e.g., "en", "es", "fr") or empty string for auto-detection
- `httpClient`: HTTP client for making requests

#### `TranscribeAudioFile(file io.Reader, fileName string) (string, error)`

**The only public method.** Transcribes audio and returns the text.

**Parameters:**
- `file`: An `io.Reader` containing the audio data
- `fileName`: The name of the file (used for format detection)

**Returns:**
- `string`: The transcribed text
- `error`: Any error that occurred

### Types

#### `SpeechToTextResponse` (internal)

```go
type SpeechToTextResponse struct {
    LanguageCode        string  `json:"language_code"`
    LanguageProbability float64 `json:"language_probability"`
    Text                string  `json:"text"`
}
```

#### `APIError`

```go
type APIError struct {
    StatusCode int    `json:"status_code"`
    Message    string `json:"message"`
    Detail     string `json:"detail,omitempty"`
}
```

## Supported Audio Formats

- MP3, WAV, OGG, AAC, FLAC, M4A, WebM

## Error Handling

```go
text, err := client.TranscribeAudioFile(file, "audio.mp3")
if err != nil {
    if apiErr, ok := err.(elevenlabs.APIError); ok {
        fmt.Printf("API Error %d: %s\n", apiErr.StatusCode, apiErr.Message)
    } else {
        fmt.Printf("Other error: %s\n", err.Error())
    }
    return
}
```

## Configuration

### Language Codes

Common language codes:
- `""` - Auto-detection
- `"en"` - English
- `"es"` - Spanish  
- `"fr"` - French
- `"de"` - German
- `"it"` - Italian
- `"pt"` - Portuguese

### Model

Uses ElevenLabs' default `scribe_v1` model.

## Examples

### Auto Language Detection

```go
client := elevenlabs.NewClient(apiKey, "", httpClient)
text, err := client.TranscribeAudioFile(file, "audio.mp3")
```

### English Only

```go
client := elevenlabs.NewClient(apiKey, "en", httpClient)
text, err := client.TranscribeAudioFile(file, "audio.mp3")
```

### Multiple Files

```go
client := elevenlabs.NewClient(apiKey, "", httpClient)

files := []string{"audio1.mp3", "audio2.wav", "audio3.m4a"}
for _, filename := range files {
    file, err := os.Open(filename)
    if err != nil {
        continue
    }
    
    text, err := client.TranscribeAudioFile(file, filename)
    if err != nil {
        log.Printf("Error transcribing %s: %v", filename, err)
        file.Close()
        continue
    }
    
    fmt.Printf("%s: %s\n", filename, text)
    file.Close()
}
```

## Integration Example

```go
// In your main.go
ElevenLabsClient = elevenlabs.NewClient(
    appConfig.ElevenLabsAPIKey,
    "", // auto-detection
    httpClient,
)

// Usage in message processing
func processAudioMessage(audioURL string) (string, error) {
    // Download audio
    resp, err := http.Get(audioURL)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    // Transcribe
    return ElevenLabsClient.TranscribeAudioFile(resp.Body, "audio.mp3")
}
```

## Troubleshooting

### Common Issues

1. **Missing API Key**: Ensure `ELEVENLABS_API_KEY` is set
2. **File Size**: Maximum 1GB file size
3. **Format**: Ensure audio format is supported
4. **Network**: Check connectivity

### Debug Logging

The package logs transcription results automatically:

```
INFO: Audio transcription completed transcribed_text="Hello world" detected_language="en" confidence=0.98
``` 
