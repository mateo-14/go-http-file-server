package settings

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Settings struct {
	Port                string `json:"port"`
	SharedDirectoryPath string `json:"sharedDirectoryPath"`
	ThumbnailCacheDir   string `json:"thumbnailCacheDir"`
}

func GetSettings() Settings {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	settings := Settings{
		Port: "8080",
	}

	sharedDirectoryPath := os.Getenv("SHARED_PATH")
	if sharedDirectoryPath == "" {
		log.Fatal("SHARED_PATH is not set")
	}

	thumbnailCacheDir := os.Getenv("THUMBNAILS_CACHE_PATH")
	if thumbnailCacheDir == "" {
		log.Fatal("THUMBNAILS_CACHE_PATH is not set")
	}

	settings.SharedDirectoryPath = sharedDirectoryPath
	settings.ThumbnailCacheDir = thumbnailCacheDir

	return settings
}
