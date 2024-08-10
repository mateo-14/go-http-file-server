package files

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/mateo-14/go-http-file-server/settings"
	"github.com/mateo-14/go-http-file-server/utils"
)

type Service interface {
	GetFilesInDirectory(path string) ([]File, error)
}

type ServiceImpl struct {
	repository Repository
	settings   settings.Settings
}

func NewService(repository Repository, settings settings.Settings) Service {
	return &ServiceImpl{repository: repository, settings: settings}
}

func (s *ServiceImpl) GetFilesInDirectory(relativePath string) ([]File, error) {
	maxGoroutinesEnv := os.Getenv("PROCESS_FILES_MAX_GOROUTINES")
	maxGoroutines := 4
	if max, err := strconv.Atoi(maxGoroutinesEnv); err == nil {
		if max > 0 {
			maxGoroutines = max
		}
	}

	absolutePath := strings.ReplaceAll(path.Join(s.settings.SharedDirectoryPath, relativePath), "\\", "/")
	entries, err := os.ReadDir(absolutePath)
	if err != nil {
		return nil, err
	}

	files := make([]File, len(entries))
	maxGoroutines = min(maxGoroutines, len(entries))
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxGoroutines)

	for i, entry := range entries {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, entry os.DirEntry) {
			defer wg.Done()
			defer func() { <-sem }()

			files[i] = s.processEntry(entry, absolutePath, relativePath)
		}(i, entry)
	}

	wg.Wait()

	// Sort directories first
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDirectory == files[j].IsDirectory {
			return strings.Compare(strings.ToLower(files[i].Name), strings.ToLower(files[j].Name)) < 0
		}
		return files[i].IsDirectory
	})

	return files, nil
}

func (s *ServiceImpl) processEntry(entry os.DirEntry, absolutePath, relativePath string) File {
	fileInfo, _ := entry.Info()

	fileModel := File{
		Name:        entry.Name(),
		Size:        fileInfo.Size(),
		IsDirectory: entry.IsDir(),
		Path:        strings.ReplaceAll(strings.Replace(path.Join(relativePath, entry.Name()), s.settings.SharedDirectoryPath, "", 1), "\\", "/"),
		UpdatedAt:   fileInfo.ModTime().Unix(),
	}

	isInDb := false
	fileModel.Id = utils.HashString(fileModel.Path)
	fileModel.Url = generateFileUrl(fileModel.Path, s.settings.Port)

	// Check if file is in database and if it is up to date
	fileEntity, err := s.repository.GetFile(fileModel.Id)

	if err == nil {
		if fileEntity.UpdatedAt == fileModel.UpdatedAt && fileModel.IsDirectory == fileEntity.IsDirectory && ((!fileModel.IsDirectory && fileEntity.Size == fileModel.Size) || fileModel.IsDirectory) {
			fileModel.Size = fileEntity.Size
			fileModel.Thumbnail = fileEntity.ThumbnailRelativePath
			fileModel.MimeType = fileEntity.MimeType
			fileModel.UpdatedAt = fileEntity.UpdatedAt

			// Generate thumbnail URL if it exists
			if fileModel.Thumbnail != "" {
				fileModel.Thumbnail = generateThumbnailUrl(fileModel.Thumbnail, s.settings.Port)
			}

			return fileModel
		} else {
			isInDb = true
			log.Printf("File %s is outdated. Updating\n", fileModel.Path)
		}
	}

	entryPath := strings.ReplaceAll(filepath.Join(absolutePath, entry.Name()), "\\", "/")

	// Check if file is a directory and calculate its size
	if entry.IsDir() {
		if dirSize, err := utils.DirSize(entryPath); err == nil {
			fileModel.Size = dirSize
		}
	} else {
		// Get MIME type and generate thumbnail for video files
		if mimeType, err := mimetype.DetectFile(entryPath); err == nil {
			if strings.Split(mimeType.String(), "/")[0] == "video" {
				relativePath := fmt.Sprintf("%s.webp", fileModel.Path[:strings.LastIndex(fileModel.Path, filepath.Ext(fileModel.Path))])
				outputPath := strings.ReplaceAll(filepath.Join(s.settings.ThumbnailCacheDir, relativePath), "\\", "/")
				if err := utils.GenerateVideoThumbnail(entryPath, outputPath); err == nil {
					fileModel.Thumbnail = relativePath
					fileEntity.ThumbnailPath = outputPath
				} else {
					log.Printf("Error generating thumbnail for %s. Error: %s", fileModel.Path, err.Error())
				}
			}

			fileModel.MimeType = mimeType.String()
		}
	}

	// Update or insert file in database
	fileEntity.ID = fileModel.Id
	fileEntity.Name = fileModel.Name
	fileEntity.Size = fileModel.Size
	fileEntity.IsDirectory = fileModel.IsDirectory
	fileEntity.MimeType = fileModel.MimeType
	fileEntity.Path = entryPath
	fileEntity.RelativePath = fileModel.Path
	fileEntity.ThumbnailRelativePath = fileModel.Thumbnail
	fileEntity.UpdatedAt = fileModel.UpdatedAt
	fileEntity.LastAccessed = time.Now().Unix()

	if isInDb {
		if err := s.repository.UpdateFile(fileEntity); err != nil {
			log.Printf("Error updating file %s in database. Error: %s", fileModel.Path, err.Error())
		}
	} else {
		// if err := s.repository.InsertFile(fileEntity); err != nil {
		// 	log.Printf("Error inserting file %s in database. Error: %s", fileModel.Path, err.Error())
		// }
	}

	if fileModel.Thumbnail != "" {
		fileModel.Thumbnail = generateThumbnailUrl(fileModel.Thumbnail, s.settings.Port)
	}

	return fileModel
}
