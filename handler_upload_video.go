package main

import (
	"net/http"
	"mime"
	"log"
	"fmt"
	"crypto/rand"
	"encoding/hex"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"errors"
	"io"
	"os"

)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	const MaxReaderMemory = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, MaxReaderMemory)
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't find JWT", err)
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}
	videoDB, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed getting the video", err)
		return
	}
	if userID != videoDB.UserID {
		respondWithError(w, http.StatusUnauthorized, "Not the owner", errors.New("User doesn't have access"))
		return
	}

	file, header, err := r.FormFile("video")
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
		respondWithError(w, http.StatusBadRequest, "Unsuported video format", err)
		return
	}
	if ext != "mp4" {
		respondWithError(w, http.StatusBadRequest, "format is not mp4", err)
		return
	}
	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to create file", err)
		return
	}
	log.Println(tempFile.Name())


	defer os.Remove(tempFile.Name())
	defer tempFile.Close()


	_, err = io.Copy(tempFile, file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to copy file", err)
		return
	}
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to reset pointer", err)
		return

	}
	
	newPath, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to create path", err)
		return
	}
	fastFileStart, err := os.Open(newPath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to open new file", err)
		return
	}
	defer fastFileStart.Close()
	defer os.Remove(fastFileStart.Name())

	key := make([]byte, 32)
	rand.Read(key)
	encodedStr := hex.EncodeToString(key)

	scale, err := getVideoAspectRatio(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to get ratio", err)
		return
	}
	ratio := "other"
	if scale == "16/9"{
		ratio = "landscape"
	}
	if scale == "9/16" {
		ratio = "portrait"
	}

	keyFormated := fmt.Sprintf("%s/%s.%s", ratio, encodedStr, ext)
	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:			&cfg.s3Bucket,
		Key:			&keyFormated,
		Body:			fastFileStart,
		ContentType:	&mediaType,
	})
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to upload Video", err)
		return
	}
	videoURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, keyFormated)
	videoDB.VideoURL = &videoURL
	err = cfg.db.UpdateVideo(videoDB)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to update the video", err)
		return
	}
	respondWithJSON(w, http.StatusOK, videoDB)


}