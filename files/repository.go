package files

import "database/sql"

type Repository interface {
	InsertFile(file FileEntity) error
	GetFile(id uint32) (FileEntity, error)
	UpdateFile(file FileEntity) error
}

type RepositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) GetFile(id uint32) (FileEntity, error) {
	row := r.db.QueryRow("SELECT * FROM files WHERE id = ?", id)

	var file FileEntity
	err := row.Scan(&file.ID, &file.Name, &file.Size, &file.IsDirectory, &file.MimeType, &file.Path, &file.RelativePath, &file.ThumbnailPath, &file.ThumbnailRelativePath, &file.UpdatedAt, &file.LastAccessed)
	if err != nil {
		return file, err
	}

	return file, nil
}

func (r *RepositoryImpl) UpdateFile(file FileEntity) error {
	_, err := r.db.Exec("UPDATE files SET name = ?, size = ?, is_directory = ?, mime_type = ?, path = ?, relative_path = ?, thumbnail_path = ?, thumbnail_relative_path = ?, updated_at = ?, last_accessed = ? WHERE id = ?", file.Name, file.Size, file.IsDirectory, file.MimeType, file.Path, file.RelativePath, file.ThumbnailPath, file.ThumbnailRelativePath, file.UpdatedAt, file.LastAccessed, file.ID)

	return err
}

func (r *RepositoryImpl) InsertFile(file FileEntity) error {
	_, err := r.db.Exec("INSERT INTO files (id, name, size, is_directory, mime_type, path, relative_path, thumbnail_path, thumbnail_relative_path, updated_at, last_accessed) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", file.ID, file.Name, file.Size, file.IsDirectory, file.MimeType, file.Path, file.RelativePath, file.ThumbnailPath, file.ThumbnailRelativePath, file.UpdatedAt, file.LastAccessed)

	return err
}
