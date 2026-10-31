package main

import (
	"os"
	"os/exec"
	"errors"
	"bytes"
	"encoding/json"
	"fmt"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func (cfg *apiConfig) MediaTypeToExt(mediaType string) (string, error) {
	var ext string
	var err error
	switch mediaType {
	case "image/jpeg":
		ext = "jpg"
	case "image/png":
		ext = "png"
	case "video/mp4":
		ext = "mp4"
	default:
		return "", errors.New("Invalid format")
	}
	return ext, err
}

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	var buffer bytes.Buffer 
	cmd.Stdout = &buffer
	if err := cmd.Run(); err != nil {
		return "", err
	}

	type Aspects struct{
		Streams []struct {
			CodecType	string	`json:"codec_type"`
			Width		int 	`json:"width"`
			Height		int		`json:"height"`
		} `json:"streams"`
	}

	var aspects Aspects
	var width int
	var height int
	if err := json.Unmarshal(buffer.Bytes(), &aspects); err != nil {
		return "", err
	}
	for _, stream := range(aspects.Streams) {
		if stream.CodecType == "video" {
			width = stream.Width
			height = stream.Height
			break
		}
	}
	if width/height == 16/9 {
		return "16/9", nil
	} else if width/height == 9/16 {
		return "9/16", nil 
	} else {
		return "other", nil
	}
}

func processVideoForFastStart(filePath string) (string, error) {
	newPath := fmt.Sprintf("%s.processing", filePath)
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", newPath)
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return newPath, nil

}