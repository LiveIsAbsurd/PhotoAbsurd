package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"photogallery/internal/model"
)

// Ошибки репозитория фотографий для понятной обработки на уровне HTTP
var (
	ErrPhotoNotFound = errors.New("фотография не найдена")
	ErrForbidden     = errors.New("доступ запрещён: недостаточно прав")
)

// PhotoFilter содержит критерии фильтрации, пагинации и сортировки ленты фотографий
type PhotoFilter struct {
	Page            int        // Номер страницы (начиная с 1)
	Limit           int        // Количество элементов на странице
	Sort            string     // Тип сортировки: "newest" (свежие) или "popular" (по лайкам)
	AlbumID         *uuid.UUID // Фильтр по конкретному альбому (необязательно)
	UserID          *uuid.UUID // Фильтр по автору снимка (необязательно)
	Username        *string    // Фильтр по логину автора (необязательно)
	IncludePrivate  bool       // Показывать ли приватные снимки (только для личного профиля /me/photos)
	CurrentUserID   *uuid.UUID // ID текущего авторизованного пользователя (для вычисления is_liked)
	CurrentUserRole string     // Роль текущего пользователя: "admin" или "user"
	GuestSessionID  *string    // ID сессии гостя (для вычисления is_liked у неавторизованных)
}

// PhotoRepository отвечает за выполнение SQL-запросов к таблице фотографий
type PhotoRepository struct {
	database *DB
}

// NewPhotoRepository — конструктор репозитория фотографий
func NewPhotoRepository(database *DB) *PhotoRepository {
	return &PhotoRepository{database: database}
}

// Create сохраняет запись о новой фотографии в базе данных PostgreSQL
func (repository *PhotoRepository) Create(executionContext context.Context, photo *model.Photo) error {
	// Сериализуем структуру метаданных камеры в формат JSON для хранения в колонке типа JSONB
	cameraMetaJSONBytes, jsonError := json.Marshal(photo.CameraMeta)
	if jsonError != nil {
		cameraMetaJSONBytes = []byte("{}")
	}

	insertQuery := `
		INSERT INTO photos (
			id, user_id, album_id, title, description,
			file_path_original, file_path_medium, file_path_thumb,
			width, height, file_size, camera_meta,
			likes_count, views_count, is_public, created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11, $12,
			$13, $14, $15, $16
		)
	`
	_, executionError := repository.database.Pool.Exec(
		executionContext,
		insertQuery,
		photo.ID,
		photo.UserID,
		photo.AlbumID,
		photo.Title,
		photo.Description,
		photo.FilePathOriginal,
		photo.FilePathMedium,
		photo.FilePathThumb,
		photo.Width,
		photo.Height,
		photo.FileSize,
		cameraMetaJSONBytes,
		photo.LikesCount,
		photo.ViewsCount,
		photo.IsPublic,
		photo.CreatedAt,
	)
	return executionError
}

// GetByID находит фотографию по её уникальному идентификатору (UUID).
func (repository *PhotoRepository) GetByID(
	executionContext context.Context,
	photoID uuid.UUID,
	currentUserID *uuid.UUID,
	currentUserRole string,
	guestSessionID *string,
) (*model.Photo, error) {
	selectQuery := `
		SELECT 
			photos.id, photos.user_id, photos.album_id, photos.title, photos.description,
			photos.file_path_original, photos.file_path_medium, photos.file_path_thumb,
			photos.width, photos.height, photos.file_size, photos.camera_meta,
			0 AS likes_count, 0 AS views_count, photos.is_public, photos.created_at,
			users.username, users.avatar_url,
			FALSE AS is_liked
		FROM photos
		JOIN users ON photos.user_id = users.id
		WHERE photos.id = $1
		  AND (photos.is_public = TRUE OR $3 = 'admin' OR ($2::uuid IS NOT NULL AND photos.user_id = $2))
	`
	var photo model.Photo
	var cameraMetaJSONBytes []byte

	scanError := repository.database.Pool.QueryRow(
		executionContext,
		selectQuery,
		photoID,
		currentUserID,
		currentUserRole,
	).Scan(
		&photo.ID,
		&photo.UserID,
		&photo.AlbumID,
		&photo.Title,
		&photo.Description,
		&photo.FilePathOriginal,
		&photo.FilePathMedium,
		&photo.FilePathThumb,
		&photo.Width,
		&photo.Height,
		&photo.FileSize,
		&cameraMetaJSONBytes,
		&photo.LikesCount,
		&photo.ViewsCount,
		&photo.IsPublic,
		&photo.CreatedAt,
		&photo.AuthorUsername,
		&photo.AuthorAvatar,
		&photo.IsLiked,
	)

	if scanError != nil {
		if errors.Is(scanError, pgx.ErrNoRows) {
			return nil, ErrPhotoNotFound
		}
		return nil, scanError
	}

	// Десериализуем JSON-метаданные обратно в структуру CameraMeta
	_ = json.Unmarshal(cameraMetaJSONBytes, &photo.CameraMeta)
	return &photo, nil
}

// List возвращает список фотографий согласно переданным фильтрам и параметрам пагинации
func (repository *PhotoRepository) List(executionContext context.Context, filter PhotoFilter) ([]model.Photo, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 24
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Page > 100 {
		filter.Page = 100
	}
	offsetRowsCount := (filter.Page - 1) * filter.Limit

	whereClauses := []string{"1=1"}
	queryArguments := []interface{}{}
	argumentIndex := 1

	if filter.UserID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("photos.user_id = $%d", argumentIndex))
		queryArguments = append(queryArguments, *filter.UserID)
		argumentIndex++

		// Администратор, автор фото или запрос с флагом IncludePrivate видит в том числе приватные снимки
		canSeePrivate := filter.IncludePrivate || filter.CurrentUserRole == "admin" || (filter.CurrentUserID != nil && *filter.CurrentUserID == *filter.UserID)
		if !canSeePrivate {
			whereClauses = append(whereClauses, "photos.is_public = TRUE")
			whereClauses = append(whereClauses, `
				(photos.album_id IS NULL OR photos.album_id IN (
					SELECT albums.id FROM albums 
					WHERE albums.is_public = TRUE
				))`)
		}
	} else if filter.Username != nil && *filter.Username != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("users.username = $%d", argumentIndex))
		queryArguments = append(queryArguments, *filter.Username)
		argumentIndex++

		canSeePrivate := filter.IncludePrivate || filter.CurrentUserRole == "admin"
		if !canSeePrivate {
			whereClauses = append(whereClauses, "photos.is_public = TRUE")
			whereClauses = append(whereClauses, `
				(photos.album_id IS NULL OR photos.album_id IN (
					SELECT albums.id FROM albums 
					WHERE albums.is_public = TRUE
				))`)
		}
	} else if filter.AlbumID != nil {
		// Просмотр конкретного альбома (коллекции)
		whereClauses = append(whereClauses, fmt.Sprintf("photos.album_id = $%d", argumentIndex))
		queryArguments = append(queryArguments, *filter.AlbumID)
		argumentIndex++

		// Доступ к фотографиям в альбоме: альбом должен быть публичным, либо пользователь — админ, либо владелец
		if filter.CurrentUserRole != "admin" {
			whereClauses = append(whereClauses, fmt.Sprintf(`
				photos.album_id IN (
					SELECT albums.id FROM albums 
					WHERE albums.id = $%d AND (albums.is_public = TRUE OR ($%d::uuid IS NOT NULL AND albums.user_id = $%d))
				)`, argumentIndex-1, argumentIndex, argumentIndex))
			queryArguments = append(queryArguments, filter.CurrentUserID)
			argumentIndex++
		}
	} else {
		// В общей ленте: администратор видит все снимки, а обычные посетители — только публичные фото
		if filter.CurrentUserRole != "admin" {
			whereClauses = append(whereClauses, "photos.is_public = TRUE")
			whereClauses = append(whereClauses, fmt.Sprintf(`
				(photos.album_id IS NULL OR photos.album_id IN (
					SELECT albums.id FROM albums 
					WHERE albums.is_public = TRUE OR ($%d::uuid IS NOT NULL AND albums.user_id = $%d)
				))`, argumentIndex, argumentIndex))
			queryArguments = append(queryArguments, filter.CurrentUserID)
			argumentIndex++
		}
	}

	orderClause := "photos.created_at DESC"

	limitArgumentIndex := argumentIndex
	queryArguments = append(queryArguments, filter.Limit)
	argumentIndex++

	offsetArgumentIndex := argumentIndex
	queryArguments = append(queryArguments, offsetRowsCount)
	argumentIndex++

	dynamicSelectQuery := fmt.Sprintf(`
		SELECT 
			photos.id, photos.user_id, photos.album_id, photos.title, photos.description,
			photos.file_path_original, photos.file_path_medium, photos.file_path_thumb,
			photos.width, photos.height, photos.file_size, photos.camera_meta,
			0 AS likes_count, 0 AS views_count, photos.is_public, photos.created_at,
			users.username, users.avatar_url,
			FALSE AS is_liked
		FROM photos
		JOIN users ON photos.user_id = users.id
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`,
		strings.Join(whereClauses, " AND "),
		orderClause,
		limitArgumentIndex,
		offsetArgumentIndex,
	)

	rows, queryError := repository.database.Pool.Query(executionContext, dynamicSelectQuery, queryArguments...)
	if queryError != nil {
		return nil, queryError
	}
	defer rows.Close()

	photosList := []model.Photo{}
	for rows.Next() {
		var photoItem model.Photo
		var cameraMetaJSONBytes []byte

		scanError := rows.Scan(
			&photoItem.ID,
			&photoItem.UserID,
			&photoItem.AlbumID,
			&photoItem.Title,
			&photoItem.Description,
			&photoItem.FilePathOriginal,
			&photoItem.FilePathMedium,
			&photoItem.FilePathThumb,
			&photoItem.Width,
			&photoItem.Height,
			&photoItem.FileSize,
			&cameraMetaJSONBytes,
			&photoItem.LikesCount,
			&photoItem.ViewsCount,
			&photoItem.IsPublic,
			&photoItem.CreatedAt,
			&photoItem.AuthorUsername,
			&photoItem.AuthorAvatar,
			&photoItem.IsLiked,
		)
		if scanError != nil {
			return nil, scanError
		}
		_ = json.Unmarshal(cameraMetaJSONBytes, &photoItem.CameraMeta)
		photosList = append(photosList, photoItem)
	}

	return photosList, rows.Err()
}

// ListUserLikes возвращает фотографии, которые понравились указанному пользователю
func (repository *PhotoRepository) ListUserLikes(
	executionContext context.Context,
	targetUserID uuid.UUID,
	pageNumber, limitPerPage int,
) ([]model.Photo, error) {
	if limitPerPage <= 0 || limitPerPage > 100 {
		limitPerPage = 24
	}
	if pageNumber <= 0 {
		pageNumber = 1
	}
	if pageNumber > 100 {
		pageNumber = 100
	}
	offsetRowsCount := (pageNumber - 1) * limitPerPage

	selectQuery := `
		SELECT 
			photos.id, photos.user_id, photos.album_id, photos.title, photos.description,
			photos.file_path_original, photos.file_path_medium, photos.file_path_thumb,
			photos.width, photos.height, photos.file_size, photos.camera_meta,
			photos.likes_count, photos.views_count, photos.is_public, photos.created_at,
			users.username, users.avatar_url,
			TRUE AS is_liked
		FROM likes
		JOIN photos ON likes.photo_id = photos.id
		JOIN users ON photos.user_id = users.id
		WHERE likes.user_id = $1
		ORDER BY likes.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, queryError := repository.database.Pool.Query(executionContext, selectQuery, targetUserID, limitPerPage, offsetRowsCount)
	if queryError != nil {
		return nil, queryError
	}
	defer rows.Close()

	likedPhotosList := []model.Photo{}
	for rows.Next() {
		var photoItem model.Photo
		var cameraMetaJSONBytes []byte

		scanError := rows.Scan(
			&photoItem.ID,
			&photoItem.UserID,
			&photoItem.AlbumID,
			&photoItem.Title,
			&photoItem.Description,
			&photoItem.FilePathOriginal,
			&photoItem.FilePathMedium,
			&photoItem.FilePathThumb,
			&photoItem.Width,
			&photoItem.Height,
			&photoItem.FileSize,
			&cameraMetaJSONBytes,
			&photoItem.LikesCount,
			&photoItem.ViewsCount,
			&photoItem.IsPublic,
			&photoItem.CreatedAt,
			&photoItem.AuthorUsername,
			&photoItem.AuthorAvatar,
			&photoItem.IsLiked,
		)
		if scanError != nil {
			return nil, scanError
		}
		_ = json.Unmarshal(cameraMetaJSONBytes, &photoItem.CameraMeta)
		likedPhotosList = append(likedPhotosList, photoItem)
	}

	return likedPhotosList, rows.Err()
}

// Update обновляет метаданные фотографии (название, описание, альбом, видимость)
func (repository *PhotoRepository) Update(
	executionContext context.Context,
	photoID, currentUserID uuid.UUID,
	updateRequest model.UpdatePhotoRequest,
) error {
	updateClauses := []string{}
	queryArguments := []interface{}{photoID, currentUserID}
	argumentIndex := 3

	if updateRequest.Title != nil {
		updateClauses = append(updateClauses, fmt.Sprintf("title = $%d", argumentIndex))
		queryArguments = append(queryArguments, *updateRequest.Title)
		argumentIndex++
	}
	if updateRequest.Description != nil {
		updateClauses = append(updateClauses, fmt.Sprintf("description = $%d", argumentIndex))
		queryArguments = append(queryArguments, *updateRequest.Description)
		argumentIndex++
	}
	if updateRequest.AlbumID != nil {
		updateClauses = append(updateClauses, fmt.Sprintf("album_id = $%d", argumentIndex))
		queryArguments = append(queryArguments, *updateRequest.AlbumID)
		argumentIndex++
	}
	if updateRequest.IsPublic != nil {
		updateClauses = append(updateClauses, fmt.Sprintf("is_public = $%d", argumentIndex))
		queryArguments = append(queryArguments, *updateRequest.IsPublic)
		argumentIndex++
	}

	if len(updateClauses) == 0 {
		return nil
	}

	updateQuery := fmt.Sprintf(`
		UPDATE photos 
		SET %s 
		WHERE id = $1 AND user_id = $2
	`, strings.Join(updateClauses, ", "))

	commandTag, executionError := repository.database.Pool.Exec(executionContext, updateQuery, queryArguments...)
	if executionError != nil {
		return executionError
	}
	if commandTag.RowsAffected() == 0 {
		return ErrForbidden
	}
	return nil
}

// Delete удаляет фотографию: администратор может удалить любой снимок,
// а обычный пользователь — только загруженный им лично.
func (repository *PhotoRepository) Delete(
	executionContext context.Context,
	photoID, currentUserID uuid.UUID,
	isAdministrator bool,
) (*model.Photo, error) {
	// Сначала получаем данные о фото без ограничения видимости, чтобы знать пути к файлам для удаления
	var targetPhoto model.Photo
	query := `
		SELECT id, user_id, file_path_original, file_path_medium, file_path_thumb
		FROM photos
		WHERE id = $1
	`
	scanErr := repository.database.Pool.QueryRow(executionContext, query, photoID).Scan(
		&targetPhoto.ID,
		&targetPhoto.UserID,
		&targetPhoto.FilePathOriginal,
		&targetPhoto.FilePathMedium,
		&targetPhoto.FilePathThumb,
	)
	if scanErr != nil {
		if errors.Is(scanErr, pgx.ErrNoRows) {
			return nil, ErrPhotoNotFound
		}
		return nil, scanErr
	}

	// Проверка прав: если не админ и не владелец — отклоняем
	if !isAdministrator && targetPhoto.UserID != currentUserID {
		return nil, ErrForbidden
	}

	var executionError error
	if isAdministrator {
		// Администратор удаляет снимок любого пользователя
		_, executionError = repository.database.Pool.Exec(executionContext, `DELETE FROM photos WHERE id = $1`, photoID)
	} else {
		// Обычный пользователь удаляет только свой снимок
		_, executionError = repository.database.Pool.Exec(executionContext, `DELETE FROM photos WHERE id = $1 AND user_id = $2`, photoID, currentUserID)
	}

	if executionError != nil {
		return nil, executionError
	}
	return &targetPhoto, nil
}

// GetFilePathsByUserID возвращает пути ко всем файлам фотографий конкретного пользователя
func (repository *PhotoRepository) GetFilePathsByUserID(
	executionContext context.Context,
	userID uuid.UUID,
) ([]string, error) {
	selectQuery := `
		SELECT file_path_original, file_path_medium, file_path_thumb
		FROM photos
		WHERE user_id = $1
	`
	rows, queryError := repository.database.Pool.Query(executionContext, selectQuery, userID)
	if queryError != nil {
		return nil, queryError
	}
	defer rows.Close()

	var filePaths []string
	for rows.Next() {
		var orig, med, thumb string
		if scanErr := rows.Scan(&orig, &med, &thumb); scanErr != nil {
			return nil, scanErr
		}
		if orig != "" {
			filePaths = append(filePaths, orig)
		}
		if med != "" {
			filePaths = append(filePaths, med)
		}
		if thumb != "" {
			filePaths = append(filePaths, thumb)
		}
	}
	return filePaths, rows.Err()
}

// IncrementViews отключён (функционал подсчёта просмотров удалён)
func (repository *PhotoRepository) IncrementViews(executionContext context.Context, photoID uuid.UUID) {
}
