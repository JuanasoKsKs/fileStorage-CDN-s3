package main

import (
	"os"
	"errors"
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