package aws

import (
	"bytes"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/rs/zerolog/log"
)

type Client struct {
	session  *session.Session
	bucket   string
	region   string
	uploader *s3manager.Uploader
	s3Client *s3.S3
}

func NewClient(region, bucket string) *Client {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create AWS session")
	}

	log.Info().
		Str("bucket", bucket).
		Str("region", region).
		Msg("AWS session created successfully")

	return &Client{
		session:  sess,
		bucket:   bucket,
		region:   region,
		uploader: s3manager.NewUploader(sess),
		s3Client: s3.New(sess),
	}
}

func (c *Client) UploadAudio(audioData []byte, voiceID string) (string, error) {
	key := fmt.Sprintf("audio/%s_%d.mp3", voiceID, time.Now().Unix())

	log.Info().
		Str("bucket", c.bucket).
		Str("region", c.region).
		Str("key", key).
		Int("content_size", len(audioData)).
		Msg("Starting S3 upload")

	uploadInput := &s3manager.UploadInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(audioData),
		ContentType: aws.String("audio/mpeg"),
	}

	result, err := c.uploader.Upload(uploadInput)
	if err != nil {
		log.Error().
			Err(err).
			Str("bucket", c.bucket).
			Str("region", c.region).
			Str("key", key).
			Msg("S3 upload failed with detailed info")
		return "", fmt.Errorf("failed to upload audio to S3: %w", err)
	}

	_, aclErr := c.s3Client.PutObjectAcl(&s3.PutObjectAclInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
		ACL:    aws.String("public-read"),
	})
	if aclErr != nil {
		log.Warn().
			Err(aclErr).
			Str("bucket", c.bucket).
			Str("key", key).
			Msg("Failed to set public-read ACL on uploaded object, file may not be publicly accessible")
	}

	publicURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", c.bucket, c.region, key)

	log.Info().
		Str("s3_url", publicURL).
		Str("s3_location", result.Location).
		Str("bucket", c.bucket).
		Str("region", c.region).
		Str("key", key).
		Msg("Audio uploaded to S3 successfully")

	return publicURL, nil
}
