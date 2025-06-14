package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {

	if video.VideoURL == nil {
		return video, nil
	}

	splitURL := strings.Split(*video.VideoURL, ",")
	if len(splitURL) < 2 {
		return video, nil
	}

	fmt.Println("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	fmt.Println(splitURL)

	preSignedURL, err := generatePresignedURL(cfg.s3Client, splitURL[0], splitURL[1], 5*time.Minute)

	if err != nil {
		return video, fmt.Errorf("can't generate presigned url %w", err)
	}

	video.VideoURL = &preSignedURL

	return video, nil
}

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {

	signedClient := s3.NewPresignClient(s3Client)
	preSignedHTTPRequest, err := signedClient.PresignGetObject(
		context.TODO(),
		&s3.GetObjectInput{
			Key:    aws.String(key),
			Bucket: aws.String(bucket)},
		s3.WithPresignExpires(expireTime))

	if err != nil {
		return "", fmt.Errorf("can't generate PreSignedHttpRequest %w", err)
	}

	return preSignedHTTPRequest.URL, nil
}
