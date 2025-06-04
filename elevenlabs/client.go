package elevenlabs

import (
	"net/http"

	"github.com/aws/aws-sdk-go/aws/session"
)

const (
	BaseURL      = "https://api.elevenlabs.io/v1"
	DefaultModel = "scribe_v1"
)

type Client struct {
	APIKey       string
	LanguageCode string
	HTTPClient   *http.Client
	S3Session    *session.Session
	S3Bucket     string
	S3Region     string
}

func NewClient(apiKey string, httpClient http.Client, s3Session *session.Session, s3Bucket string, s3Region string) Client {
	return Client{
		APIKey:       apiKey,
		LanguageCode: "pt",
		HTTPClient:   &httpClient,
		S3Session:    s3Session,
		S3Bucket:     s3Bucket,
		S3Region:     s3Region,
	}
}
