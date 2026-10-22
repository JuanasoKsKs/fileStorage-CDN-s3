package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"mime"
	"crypto/rand"
	"encoding/base64"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"

	"errors"
	"io"
	"os"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	// TODO: implement the upload here
	const MaxMemory = 10 << 20
	r.ParseMultipartForm(MaxMemory)
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()
	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type", err)
		return
	}

	ext, err := cfg.MediaTypeToExt(mediaType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unsuported image format", err)
		return
	}

	videoDB, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed getting the video", err)
		return
	}
	if userID != videoDB.UserID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", errors.New("User doesn't have access"))
		return
	}

	key := make([]byte, 32)
	rand.Read(key)
	encoded := base64.RawURLEncoding.EncodeToString(key)

	filePath := fmt.Sprintf("%v.%v", encoded,ext)
	fileCreated, err := os.Create(filepath.Join(cfg.assetsRoot, filePath))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to create file", err)
		return
	}
	defer fileCreated.Close()
	_, err = io.Copy(fileCreated, file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable copy to file", err)
		return
	}

	dataURL := fmt.Sprintf("http://localhost:%v/assets/%v", cfg.port, filePath)
	videoDB.ThumbnailURL = &dataURL

	err = cfg.db.UpdateVideo(videoDB)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to update the vido", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoDB)
}
