# ElevenLabs Speech-to-Text and Text-to-Speech Integration

Simple and clean integration with ElevenLabs' speech-to-text and text-to-speech APIs.

## Features

- **Speech-to-Text**: Transcribe audio files to text
- **Text-to-Speech**: Convert text to audio with voice selection and upload to S3
- **Single Method APIs**: Simple methods for both operations
- **Multiple Audio Formats**: Support for MP3, WAV, OGG, AAC, FLAC, M4A, WebM
- **S3 Integration**: Automatic upload of generated audio to S3 bucket
- **Clean Error Handling**: Simple error responses

## Setup

### Environment Variables

Add your ElevenLabs API key and AWS configuration to your `.env` file:

```env
ELEVENLABS_API_KEY=your_elevenlabs_api_key_here
AWS_S3_BUCKET=your-s3-bucket-name
AWS_REGION=us-east-2
AWS_ACCESS_KEY_ID=your_aws_access_key_id
AWS_SECRET_ACCESS_KEY=your_aws_secret_access_key
```

### Get ElevenLabs API Key

1. Sign up at [ElevenLabs](https://elevenlabs.io/)
2. Navigate to your profile settings
3. Generate an API key

### AWS Setup

1. **Create an S3 bucket with public read access**
   - Go to AWS S3 Console
   - Create a new bucket
   - Note the bucket name and region

2. **Create IAM User with S3 permissions**
   - Go to AWS IAM Console
   - Create a new user with programmatic access
   - Attach the `AmazonS3FullAccess` policy (or create a custom policy with s3:PutObject and s3:PutObjectAcl permissions)
   - Save the Access Key ID and Secret Access Key

3. **Configure bucket for public read access**
   - Either use bucket ACLs with `public-read` for uploaded objects
   - Or set up a bucket policy to allow public read access:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PublicReadGetObject",
      "Effect": "Allow",
      "Principal": "*",
      "Action": "s3:GetObject",
      "Resource": "arn:aws:s3:::your-bucket-name/*"
    }
  ]
}
```

## Usage

### Basic Usage

```go
package main

import (
    "chatbot/elevenlabs"
    "net/http"
    "os"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/credentials"
    "github.com/aws/aws-sdk-go/aws/session"
)

func main() {
    sess, _ := session.NewSession(&aws.Config{
        Region: aws.String("us-east-2"),
        Credentials: credentials.NewStaticCredentials(
            "your-access-key-id",
            "your-secret-access-key",
            "",
        ),
    })
    
    client := elevenlabs.NewClient("your-api-key", http.Client{}, sess, "your-bucket", "us-east-2")
    
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

### Text-to-Speech Usage

```go
package main

import (
    "chatbot/elevenlabs"
    "net/http"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/credentials"
    "github.com/aws/aws-sdk-go/aws/session"
)

func main() {
    sess, _ := session.NewSession(&aws.Config{
        Region: aws.String("us-east-2"),
        Credentials: credentials.NewStaticCredentials(
            "your-access-key-id",
            "your-secret-access-key",
            "",
        ),
    })
    
    client := elevenlabs.NewClient("your-api-key", http.Client{}, sess, "your-bucket", "us-east-2")
    
    voiceID := "JBFqnCBsd6RMkjVDRZzb"
    text := "The first move is what sets everything in motion."
    modelID := "eleven_multilingual_v2"
    
    audioURL, err := client.ConvertTextToSpeech(voiceID, text, modelID)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Audio available at:", audioURL)
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

#### `NewClient(apiKey string, httpClient http.Client, s3Session *session.Session, s3Bucket string, s3Region string) Client`

Creates a new ElevenLabs client with S3 integration.

**Parameters:**
- `apiKey`: Your ElevenLabs API key
- `httpClient`: HTTP client for making requests
- `s3Session`: AWS session for S3 operations
- `s3Bucket`: S3 bucket name for audio storage
- `s3Region`: AWS region for the S3 bucket

#### `TranscribeAudioFile(file io.Reader, fileName string) (string, error)`

**The only public method.** Transcribes audio and returns the text.

**Parameters:**
- `file`: An `io.Reader` containing the audio data
- `fileName`: The name of the file (used for format detection)

**Returns:**
- `string`: The transcribed text
- `error`: Any error that occurred

#### `ConvertTextToSpeech(voiceID string, text string, modelID string) (string, error)`

Converts text to speech, uploads to S3, and returns the public URL.

**Parameters:**
- `voiceID`: ID of the voice to be used
- `text`: The text to convert to speech
- `modelID`: Model ID (e.g., "eleven_multilingual_v2")

**Returns:**
- `string`: The public URL of the uploaded audio file
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

#### `TextToSpeechRequest` (internal)

```go
type TextToSpeechRequest struct {
    Text    string `json:"text"`
    ModelID string `json:"model_id,omitempty"`
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

### Model

Uses ElevenLabs' default `scribe_v1` model.

## Examples

### Multiple Files

```go
client := elevenlabs.NewClient(apiKey, httpClient)

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

## Text-to-Speech Examples

### Basic Usage

```go
sess, _ := session.NewSession(&aws.Config{
    Region: aws.String("us-east-2"),
    Credentials: credentials.NewStaticCredentials(
        "your-access-key-id",
        "your-secret-access-key",
        "",
    ),
})
client := elevenlabs.NewClient(apiKey, httpClient, sess, "my-bucket", "us-east-2")
audioURL, err := client.ConvertTextToSpeech("voice-id", "Hello world", "eleven_multilingual_v2")
```

### Share Audio URL

```go
audioURL, err := client.ConvertTextToSpeech(voiceID, text, modelID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Share this audio: %s\n", audioURL)
```

### Use with WhatsApp API

```go
func handleTextToSpeech(w http.ResponseWriter, r *http.Request) {
    audioURL, err := client.ConvertTextToSpeech(voiceID, text, modelID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Send audio URL via WhatsApp or other messaging service
    // The URL points to the public S3 file
}
```

## Integration Example

```go
// In your main.go
sess, err := session.NewSession(&aws.Config{
    Region: aws.String(appConfig.S3Region),
    Credentials: credentials.NewStaticCredentials(
        appConfig.AWSAccessKeyID,
        appConfig.AWSSecretAccessKey,
        "",
    ),
})
if err != nil {
    log.Fatal().Err(err).Msg("Failed to create AWS session")
}

ElevenLabsClient = elevenlabs.NewClient(
    appConfig.ElevenLabsAPIKey,
    httpClient,
    sess,
    appConfig.S3Bucket,
    appConfig.S3Region,
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
2. **Missing S3 Config**: Ensure `AWS_S3_BUCKET` and `AWS_REGION` are set
3. **Missing AWS Credentials**: Ensure `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` are set
4. **AWS Permissions**: Ensure your AWS user/role has S3 upload permissions
5. **S3 Bucket Policy**: Ensure your bucket allows public read access
6. **File Size**: Maximum 1GB file size
7. **Format**: Ensure audio format is supported
8. **Network**: Check connectivity

### AWS Credential Issues

If you get AWS credential errors:
- Verify your Access Key ID and Secret Access Key are correct
- Ensure the IAM user has the necessary S3 permissions
- Check that the bucket name and region are correct
- Test AWS credentials with AWS CLI: `aws s3 ls s3://your-bucket-name`

### Debug Logging

The package logs transcription and upload results automatically:

```
INFO: Audio transcription completed transcribed_text="Hello world" detected_language="en" confidence=0.98
INFO: Audio uploaded to S3 successfully voice_id="voice-123" s3_url="https://bucket.s3.region.amazonaws.com/audio/voice-123_1234567890.mp3"
``` 
