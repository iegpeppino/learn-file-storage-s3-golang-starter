package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	contentType := headers.Header.Get("Content-Type")
	mediatype, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Can't identify media type", err)
		return
	}

	if mediatype != "image/jpeg" && mediatype != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Content does not match type", nil)
		return
	}

	fileExt := strings.Split(mediatype, "/")[1]
	fmt.Println(fileExt)
	defer file.Close()

	thumbnailPath := filepath.Join(cfg.assetsRoot, videoID.String()+"."+fileExt)

	newThumbFile, err := os.Create(thumbnailPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create new file", err)
		return
	}

	defer newThumbFile.Close()

	_, err = io.Copy(newThumbFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't save thumbnail", err)
		return
	}

	// Encode Thumbnail data in base64 and save it to a string in thumbnailURL
	//encdThumb := base64.StdEncoding.EncodeToString(thumbnailData)

	//dataURL := fmt.Sprintf("data:%s;base64,%s", fileType, encdThumb)

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video", err)
		return
	}

	if userID != video.UserID {
		respondWithError(w, http.StatusUnauthorized, "User has no permission to modify video", err)
		return
	}

	// Update video metadata
	//Set new thumbnail url
	newThumbURL := fmt.Sprintf("http://localhost:%v/assets/%s.%s", cfg.port, videoID, fileExt)

	video.ThumbnailURL = &newThumbURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
