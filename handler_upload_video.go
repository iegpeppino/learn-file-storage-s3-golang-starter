package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {

	const maxUploadSize = int64(1 << 30) // Max 1Gb

	aspectRatioDict := map[string]string{
		"19:6": "landscape",
		"6:19": "portrait",
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	err := r.ParseMultipartForm(maxUploadSize)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "File too large", err)
		return
	}

	videoID, err := uuid.Parse(r.PathValue("videoID"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid video url", err)
		return
	}

	// Authenticate User
	tokenStr, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't get bearer token", err)
		return
	}

	// Validate user
	userID, err := auth.ValidateJWT(tokenStr, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized user", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't retrieve video data", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User is not video's owner", err)
		return
	}

	file, fileHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't form video file", err)
		return
	}

	// Close file at end of function
	defer file.Close()

	// Extract content type from file header
	contenType := fileHeader.Header.Get("Content-Type")

	// Parse media type
	mediaType, _, err := mime.ParseMediaType(contenType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid media type", err)
		return
	}

	// Validate the mediatype
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusUnsupportedMediaType, "Only video/mp4 is supported", nil)
		return
	}

	// Save uploaded file to temp file on sys
	originalTempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create temp file", err)
		return
	}

	defer originalTempFile.Close()

	defer os.Remove(originalTempFile.Name())

	//Copy contents from wire to the temp file
	_, err = io.Copy(originalTempFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't copy file", err)
		return
	}

	// Create and upload fast file to S3
	fastFilePath, err := processVideoForFastStart(originalTempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create fast file path", err)
		return
	}

	fastFilePtr, err := os.Open(fastFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get FastFile pointer", err)
		return
	}

	defer os.Remove(fastFilePath)

	defer fastFilePtr.Close()

	// With file.Open() this step is not needed  the file pointer already starts at beggining
	// Reset tempFile pointer to beggining (to read again from beggining)
	// _, err = fastFilePtr.Seek(0, io.SeekStart)
	// if err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "Couldn't reset pointer", err)
	// 	return
	// }

	aspectRatioNum, err := getVideoAspectRatio(fastFilePtr.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't read video metadata", err)
		return
	}

	// Creating random name for video file
	// Making random bytes
	randBytes := make([]byte, 32)
	// Generating random data
	_, err = rand.Read(randBytes)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate bytes", err)
		return
	}

	videoCode := hex.EncodeToString(randBytes) + ".mp4"

	var s3Prefix string
	if prefix, ok := aspectRatioDict[aspectRatioNum]; ok {
		s3Prefix = prefix
	} else {
		s3Prefix = "other"
	}

	key := fmt.Sprintf("%s/%s", s3Prefix, videoCode)

	params := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &key,
		Body:        fastFilePtr,
		ContentType: &mediaType,
	}

	_, err = cfg.s3Client.PutObject(r.Context(), &params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't save video", err)
		return
	}

	// Updating video URL to be pre-signed

	newVideoURL := fmt.Sprintf("%s,%s", cfg.s3Bucket, key)
	video.VideoURL = &newVideoURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	video, err = cfg.dbVideoToSignedVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldnt generate presigned videoURL", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
