package files

type FileEntity struct {
	ID                    uint32
	Name                  string
	Size                  int64
	IsDirectory           bool
	MimeType              string
	Path                  string
	RelativePath          string
	ThumbnailPath         string
	ThumbnailRelativePath string
	UpdatedAt             int64
	LastAccessed          int64
}

type File struct {
	Id          uint32 `json:"id,omitempty"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	IsDirectory bool   `json:"isDirectory"`
	MimeType    string `json:"mimeType,omitempty"`
	Path        string `json:"path"`
	Thumbnail   string `json:"thumbnail,omitempty"`
	UpdatedAt   int64  `json:"updatedAt"`
	Url         string `json:"url,omitempty"`
}

func (f FileEntity) ToDomain() File {
	return File{
		Id:          f.ID,
		Name:        f.Name,
		Size:        f.Size,
		IsDirectory: f.IsDirectory,
		MimeType:    f.MimeType,
		Path:        f.RelativePath,
		Thumbnail:   f.ThumbnailRelativePath,
		UpdatedAt:   f.UpdatedAt,
	}
}
