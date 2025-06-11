package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {

	//Setting max memory to 10mb
	const maxMemory = 10 << 20

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldnt parse multiform", err)
		return
	}
	// Get image data
	file, headers, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse from file", err)
		return
	}
	// Extract file type from headers
	fileType := headers.Header.Get("Content-Type")

	defer file.Close()

	thumbnailData, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't read data", err)
		return
	}

	oldVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video", err)
		return
	}

	if userID != oldVideo.UserID {
		respondWithError(w, http.StatusUnauthorized, "User has no permission to modify video", err)
		return
	}

	// Create new thumbnail struct
	newThumb := thumbnail{
		data:      thumbnailData,
		mediaType: fileType,
	}
	// Add new thumbnail to global map
	videoThumbnails[videoID] = newThumb

	// Update video metadata
	//Set new thumbnail url
	newThumbURL := fmt.Sprintf("http://localhost:%v/api/thumbnails/%s", cfg.port, videoID)

	oldVideo.ThumbnailURL = &newThumbURL

	err = cfg.db.UpdateVideo(oldVideo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	updatedVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, updatedVideo)
}
