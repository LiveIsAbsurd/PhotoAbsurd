package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"photogallery/internal/model"
)

// ErrAlbumNotFound возвращается, если альбом с запрошенным ID не существует
var ErrAlbumNotFound = errors.New("альбом не найден")

// AlbumRepository отвечает за сохранение и получение тематических альбомов/коллекций
type AlbumRepository struct {
	database *DB
}

// NewAlbumRepository — конструктор репозитория альбомов
func NewAlbumRepository(database *DB) *AlbumRepository {
	return &AlbumRepository{database: database}
}

// Create сохраняет новый альбом в базе данных
func (repository *AlbumRepository) Create(executionContext context.Context, album *model.Album) error {
	insertQuery := `
		INSERT INTO albums (id, user_id, title, description, cover_photo_id, is_public, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, executionError := repository.database.Pool.Exec(
		executionContext,
		insertQuery,
		album.ID,
		album.UserID,
		album.Title,
		album.Description,
		album.CoverPhotoID,
		album.IsPublic,
		album.CreatedAt,
		album.UpdatedAt,
	)
	return executionError
}

// GetByID находит альбом по его ID, подтягивает путь к обложке и подсчитывает фото внутри.
// Проверяет видимость альбома: публичный альбом, владелец или администратор.
func (repository *AlbumRepository) GetByID(
	executionContext context.Context,
	albumID uuid.UUID,
	currentUserID *uuid.UUID,
	currentUserRole string,
) (*model.Album, error) {
	selectQuery := `
		SELECT 
			albums.id, albums.user_id, albums.title, albums.description, albums.cover_photo_id, 
			albums.is_public, albums.created_at, albums.updated_at,
			photos.file_path_thumb AS cover_photo,
			(SELECT COUNT(*) FROM photos WHERE photos.album_id = albums.id) AS photos_count
		FROM albums
		LEFT JOIN photos ON albums.cover_photo_id = photos.id
		WHERE albums.id = $1
		  AND (albums.is_public = TRUE OR $3 = 'admin' OR ($2::uuid IS NOT NULL AND albums.user_id = $2))
	`
	var album model.Album
	queryError := repository.database.Pool.QueryRow(
		executionContext,
		selectQuery,
		albumID,
		currentUserID,
		currentUserRole,
	).Scan(
		&album.ID,
		&album.UserID,
		&album.Title,
		&album.Description,
		&album.CoverPhotoID,
		&album.IsPublic,
		&album.CreatedAt,
		&album.UpdatedAt,
		&album.CoverPhoto,
		&album.PhotosCount,
	)
	if queryError != nil {
		if errors.Is(queryError, pgx.ErrNoRows) {
			return nil, ErrAlbumNotFound
		}
		return nil, queryError
	}
	return &album, nil
}

// List возвращает список публичных альбомов, а также приватные альбомы текущего пользователя (или все для администратора)
func (repository *AlbumRepository) List(executionContext context.Context, currentUserID *uuid.UUID, currentUserRole string) ([]model.Album, error) {
	selectQuery := `
		SELECT 
			albums.id, albums.user_id, albums.title, albums.description, albums.cover_photo_id, 
			albums.is_public, albums.created_at, albums.updated_at,
			photos.file_path_thumb AS cover_photo,
			(SELECT COUNT(*) FROM photos WHERE photos.album_id = albums.id) AS photos_count
		FROM albums
		LEFT JOIN photos ON albums.cover_photo_id = photos.id
		WHERE albums.is_public = TRUE OR $2 = 'admin' OR ($1::uuid IS NOT NULL AND albums.user_id = $1)
		ORDER BY albums.created_at DESC
	`
	rows, queryError := repository.database.Pool.Query(executionContext, selectQuery, currentUserID, currentUserRole)
	if queryError != nil {
		return nil, queryError
	}
	defer rows.Close()

	albumsList := []model.Album{}
	for rows.Next() {
		var albumItem model.Album
		scanError := rows.Scan(
			&albumItem.ID,
			&albumItem.UserID,
			&albumItem.Title,
			&albumItem.Description,
			&albumItem.CoverPhotoID,
			&albumItem.IsPublic,
			&albumItem.CreatedAt,
			&albumItem.UpdatedAt,
			&albumItem.CoverPhoto,
			&albumItem.PhotosCount,
		)
		if scanError != nil {
			return nil, scanError
		}
		albumsList = append(albumsList, albumItem)
	}
	return albumsList, rows.Err()
}

// Delete удаляет альбом, если запрос поступил от владельца
func (repository *AlbumRepository) Delete(executionContext context.Context, albumID, currentUserID uuid.UUID) error {
	commandTag, executionError := repository.database.Pool.Exec(
		executionContext,
		`DELETE FROM albums WHERE id = $1 AND user_id = $2`,
		albumID,
		currentUserID,
	)
	if executionError != nil {
		return executionError
	}
	if commandTag.RowsAffected() == 0 {
		return ErrForbidden
	}
	return nil
}

// DeleteAsAdmin удаляет любой альбом администратором без проверки авторства
func (repository *AlbumRepository) DeleteAsAdmin(executionContext context.Context, albumID uuid.UUID) error {
	commandTag, executionError := repository.database.Pool.Exec(
		executionContext,
		`DELETE FROM albums WHERE id = $1`,
		albumID,
	)
	if executionError != nil {
		return executionError
	}
	if commandTag.RowsAffected() == 0 {
		return ErrAlbumNotFound
	}
	return nil
}

// Update обновляет название, описание и публичный статус альбома
func (repository *AlbumRepository) Update(executionContext context.Context, albumID uuid.UUID, title, description string, isPublic bool) error {
	_, executionError := repository.database.Pool.Exec(
		executionContext,
		`UPDATE albums SET title = $2, description = $3, is_public = $4, updated_at = NOW() WHERE id = $1`,
		albumID,
		title,
		description,
		isPublic,
	)
	return executionError
}
