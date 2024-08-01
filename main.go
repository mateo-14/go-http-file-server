package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/mateo-14/go-http-file-server/files"
	"github.com/mateo-14/go-http-file-server/settings"
	"github.com/mateo-14/go-http-file-server/utils"
	_ "github.com/mattn/go-sqlite3"
)

// TODO - Refactor, separate in layers
// TODO - If file is in db and its has thumbnail check if thumbnail exists
// TODO - Image thumbnail generation
// TODO - Save last_accessed and use it to run job to delete old files and thumbnails

func main() {
	log.SetOutput(io.MultiWriter(os.Stdout, utils.GetLogFileWriter()))
	if !utils.CheckIfFFMPEGExists() {
		log.Println("FFMPEG is not installed")
	}

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	settings := settings.GetSettings()

	db, err := sql.Open("sqlite3", "files.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS files (id INTEGER PRIMARY KEY, name TEXT, size INTEGER, is_directory BOOLEAN, mime_type TEXT, path TEXT, relative_path TEXT, thumbnail_path TEXT, thumbnail_relative_path TEXT, updated_at INTEGER, last_accessed INTEGER)")

	if err != nil {
		log.Fatal(err)
	}

	httpServer := http.NewServeMux()
	filesRepository := files.NewRepository(db)
	filesService := files.NewService(filesRepository, settings)

	httpServer.HandleFunc("/explore", func(w http.ResponseWriter, r *http.Request) {
		pathq := r.URL.Query().Get("path")
		files, err := filesService.GetFilesInDirectory(pathq)

		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.Error(w, "Directory not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to read directory", http.StatusInternalServerError)
			}
			return
		}

		fileData, err := json.Marshal(files)
		if err != nil {
			http.Error(w, "Failed to marshal file data", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(fileData)

	})

	httpServer.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(settings.SharedDirectoryPath))))
	httpServer.Handle("/thumbnails/", http.StripPrefix("/thumbnails/", http.FileServer(http.Dir(settings.ThumbnailCacheDir))))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Server is running on port " + port)
	http.ListenAndServe(":"+port, httpServer)
}
