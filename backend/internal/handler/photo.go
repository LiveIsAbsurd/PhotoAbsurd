package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"photogallery/internal/middleware"
	"photogallery/internal/model"
	"photogallery/internal/repository"
	"photogallery/internal/service"
)

// PhotoHandler обрабатывает все операции с медиафайлами и галереей
type PhotoHandler struct {
	photoRepository *repository.PhotoRepository
	albumRepository *repository.AlbumRepository
	likeRepository  *repository.LikeRepository
	imageService    *service.ImageService
	uploadDirectory string
}

// NewPhotoHandler — конструктор обработчика фотографий
func NewPhotoHandler(
	photoRepository *repository.PhotoRepository,
	albumRepository *repository.AlbumRepository,
	likeRepository *repository.LikeRepository,
	imageService *service.ImageService,
	uploadDirectory string,
) *PhotoHandler {
	return &PhotoHandler{
		photoRepository: photoRepository,
		albumRepository: albumRepository,
		likeRepository:  likeRepository,
		imageService:    imageService,
		uploadDirectory: uploadDirectory,
	}
}

// List обрабатывает GET /api/photos.
// Возвращает публичную ленту снимков с поддержкой пагинации, фильтра по альбому и сортировки.
func (handlerInstance *PhotoHandler) List(responseWriter http.ResponseWriter, httpRequest *http.Request) {
	pageNumber, _ := strconv.Atoi(httpRequest.URL.Query().Get("page"))
	limitPerPage, _ := strconv.Atoi(httpRequest.URL.Query().Get("limit"))
	sortType := httpRequest.URL.Query().Get("sort")

	var filterAlbumUUID *uuid.UUID
	albumQuery := httpRequest.URL.Query().Get("album_id")
	if albumQuery == "" {
		albumQuery = httpRequest.URL.Query().Get("album")
	}
	if albumQuery != "" {
		if parsedAlbumUUID, parseError := uuid.Parse(albumQuery); parseError == nil {
			filterAlbumUUID = &parsedAlbumUUID
		}
	}

	var filterUserUUID *uuid.UUID
	var filterUsername *string
	if userIDQueryString := httpRequest.URL.Query().Get("user_id"); userIDQueryString != "" {
		if parsedUserUUID, parseError := uuid.Parse(userIDQueryString); parseError == nil {
			filterUserUUID = &parsedUserUUID
		}
	}
	if usernameQueryString := httpRequest.URL.Query().Get("username"); usernameQueryString != "" {
		cleanUsername := strings.TrimPrefix(usernameQueryString, "@")
		filterUsername = &cleanUsername
	}
	if authorQueryString := httpRequest.URL.Query().Get("author"); authorQueryString != "" {
		if parsedUUID, parseError := uuid.Parse(authorQueryString); parseError == nil {
			filterUserUUID = &parsedUUID
		} else {
			cleanUsername := strings.TrimPrefix(authorQueryString, "@")
			filterUsername = &cleanUsername
		}
	}

	photosList, queryError := handlerInstance.photoRepository.List(httpRequest.Context(), repository.PhotoFilter{
		Page:            pageNumber,
		Limit:           limitPerPage,
		Sort:            sortType,
		UserID:          filterUserUUID,
		Username:        filterUsername,
		AlbumID:         filterAlbumUUID,
		CurrentUserID:   middleware.GetUserID(httpRequest.Context()),
		CurrentUserRole: middleware.GetRole(httpRequest.Context()),
	})
	if queryError != nil {
		log.Printf("[ERROR] PhotoHandler.List failed: %v", queryError)
		sendJSONResponse(responseWriter, http.StatusInternalServerError, map[string]string{"error": "внутренняя ошибка сервера"})
		return
	}

	if photosList == nil {
		photosList = []model.Photo{}
	}
	sendJSONResponse(responseWriter, http.StatusOK, photosList)
}

// GetByID обрабатывает GET /api/photos/{id}.
// Возвращает карточку фотографии с метаданными EXIF и асинхронно увеличивает просмотры.
func (handlerInstance *PhotoHandler) GetByID(responseWriter http.ResponseWriter, httpRequest *http.Request) {
	photoIDURLParameter := chi.URLParam(httpRequest, "id")
	photoUUID, parseError := uuid.Parse(photoIDURLParameter)
	if parseError != nil {
		sendJSONResponse(responseWriter, http.StatusBadRequest, map[string]string{"error": "некорректный UUID фотографии"})
		return
	}

	foundPhoto, findError := handlerInstance.photoRepository.GetByID(
		httpRequest.Context(),
		photoUUID,
		middleware.GetUserID(httpRequest.Context()),
		middleware.GetRole(httpRequest.Context()),
		nil,
	)
	if findError != nil {
		if errors.Is(findError, repository.ErrPhotoNotFound) {
			sendJSONResponse(responseWriter, http.StatusNotFound, map[string]string{"error": "фотография не найдена"})
			return
		}
		log.Printf("[ERROR] PhotoHandler.GetByID failed for %s: %v", photoUUID, findError)
		sendJSONResponse(responseWriter, http.StatusInternalServerError, map[string]string{"error": "внутренняя ошибка сервера"})
		return
	}

	sendJSONResponse(responseWriter, http.StatusOK, foundPhoto)
}

// Upload обрабатывает POST /api/photos.
// Принимает multipart/form-data с одним или несколькими файлами,
// прогоняет через сервис оптимизации изображений и сохраняет в БД.
func (handlerInstance *PhotoHandler) Upload(responseWriter http.ResponseWriter, httpRequest *http.Request) {
	currentUserID := middleware.GetUserID(httpRequest.Context())
	if currentUserID == nil {
		sendJSONResponse(responseWriter, http.StatusUnauthorized, map[string]string{"error": "требуется авторизация"})
		return
	}

	// Ограничиваем максимальный размер тела запроса 20 МБ для защиты от DoS
	httpRequest.Body = http.MaxBytesReader(responseWriter, httpRequest.Body, 20<<20)

	// Ограничиваем буфер multipart-формы 20 МБ в оперативной памяти
	if parseFormError := httpRequest.ParseMultipartForm(20 << 20); parseFormError != nil {
		sendJSONResponse(responseWriter, http.StatusBadRequest, map[string]string{"error": "ошибка разбора multipart-формы: превышен максимальный размер 20 МБ"})
		return
	}

	photoTitle := httpRequest.FormValue("title")
	photoDescription := httpRequest.FormValue("description")
	isPhotoPublic := httpRequest.FormValue("is_public") != "false"

	var targetAlbumUUID *uuid.UUID
	if albumIDString := httpRequest.FormValue("album_id"); albumIDString != "" {
		parsedUUID, parseAlbumErr := uuid.Parse(albumIDString)
		if parseAlbumErr != nil {
			sendJSONResponse(responseWriter, http.StatusBadRequest, map[string]string{"error": "некорректный UUID альбома"})
			return
		}
		foundAlbum, albumErr := handlerInstance.albumRepository.GetByID(httpRequest.Context(), parsedUUID, currentUserID, middleware.GetRole(httpRequest.Context()))
		if albumErr != nil {
			sendJSONResponse(responseWriter, http.StatusNotFound, map[string]string{"error": "альбом не найден"})
			return
		}
		if foundAlbum.UserID != *currentUserID {
			sendJSONResponse(responseWriter, http.StatusForbidden, map[string]string{"error": "нет доступа к чужому альбому"})
			return
		}
		targetAlbumUUID = &parsedUUID
		// Если снимок загружается в приватный альбом, снимок автоматически становится приватным
		if !foundAlbum.IsPublic {
			isPhotoPublic = false
		}
	}

	// Поддерживаем как одиночное поле "file", так и пакетное "files"
	uploadedFileHeaders := httpRequest.MultipartForm.File["file"]
	if len(uploadedFileHeaders) == 0 {
		uploadedFileHeaders = httpRequest.MultipartForm.File["files"]
	}
	if len(uploadedFileHeaders) == 0 {
		sendJSONResponse(responseWriter, http.StatusBadRequest, map[string]string{"error": "не передано ни одного файла"})
		return
	}

	createdPhotosList := []model.Photo{}

	for _, fileHeader := range uploadedFileHeaders {
		openedFile, openError := fileHeader.Open()
		if openError != nil {
			sendJSONResponse(responseWriter, http.StatusBadRequest, map[string]string{"error": "не удалось открыть загруженный файл"})
			return
		}

		// Вызываем конвейер обработки изображения (проверка magic bytes, ресайз, WebP)
		processedResult, processError := handlerInstance.imageService.ProcessAndSave(openedFile)
		_ = openedFile.Close()
		if processError != nil {
			sendJSONResponse(responseWriter, http.StatusUnprocessableEntity, map[string]string{"error": processError.Error()})
			return
		}

		resolvedTitle := photoTitle
		if resolvedTitle == "" {
			// Если название не введено, используем оригинальное имя файла без расширения
			resolvedTitle = strings.TrimSuffix(fileHeader.Filename, filepath.Ext(fileHeader.Filename))
		}

		newPhotoRecord := model.Photo{
			ID:               processedResult.ID,
			UserID:           *currentUserID,
			AlbumID:          targetAlbumUUID,
			Title:            resolvedTitle,
			Description:      photoDescription,
			FilePathOriginal: processedResult.FilePathOriginal,
			FilePathMedium:   processedResult.FilePathMedium,
			FilePathThumb:    processedResult.FilePathThumb,
			Width:            processedResult.Width,
			Height:           processedResult.Height,
			FileSize:         processedResult.FileSize,
			CameraMeta:       processedResult.CameraMeta,
			LikesCount:       0,
			ViewsCount:       0,
			IsPublic:         isPhotoPublic,
			CreatedAt:        time.Now(),
		}

		if saveError := handlerInstance.photoRepository.Create(httpRequest.Context(), &newPhotoRecord); saveError != nil {
			// При сбое записи в БД удаляем созданные файлы с диска, предотвращая утечку дискового пространства
			_ = os.Remove(filepath.Join(handlerInstance.uploadDirectory, filepath.Base(processedResult.FilePathOriginal)))
			_ = os.Remove(filepath.Join(handlerInstance.uploadDirectory, filepath.Base(processedResult.FilePathMedium)))
			_ = os.Remove(filepath.Join(handlerInstance.uploadDirectory, filepath.Base(processedResult.FilePathThumb)))
			log.Printf("[ERROR] PhotoHandler.Upload failed to save photo record: %v", saveError)
			sendJSONResponse(responseWriter, http.StatusInternalServerError, map[string]string{
				"error": "внутренняя ошибка сервера",
			})
			return
		}

		createdPhotosList = append(createdPhotosList, newPhotoRecord)
	}

	sendJSONResponse(responseWriter, http.StatusCreated, createdPhotosList)
}

// Update обрабатывает PATCH /api/photos/{id}.
// Обновляет название, описание или привязку к альбому (только для автора).
func (handlerInstance *PhotoHandler) Update(responseWriter http.ResponseWriter, httpRequest *http.Request) {
	currentUserID := middleware.GetUserID(httpRequest.Context())
	if currentUserID == nil {
		sendJSONResponse(responseWriter, http.StatusUnauthorized, map[string]string{"error": "требуется авторизация"})
		return
	}

	photoUUID, parseError := uuid.Parse(chi.URLParam(httpRequest, "id"))
	if parseError != nil {
		sendJSONResponse(responseWriter, http.StatusBadRequest, map[string]string{"error": "некорректный UUID фотографии"})
		return
	}

	var updatePayload model.UpdatePhotoRequest
	if decodeError := json.NewDecoder(httpRequest.Body).Decode(&updatePayload); decodeError != nil {
		sendJSONResponse(responseWriter, http.StatusBadRequest, map[string]string{"error": "некорректные данные запроса"})
		return
	}

	if updateError := handlerInstance.photoRepository.Update(httpRequest.Context(), photoUUID, *currentUserID, updatePayload); updateError != nil {
		if errors.Is(updateError, repository.ErrForbidden) {
			sendJSONResponse(responseWriter, http.StatusForbidden, map[string]string{"error": "доступ запрещён: фото не найдено или принадлежит другому пользователю"})
			return
		}
		log.Printf("[ERROR] PhotoHandler.Update failed for %s: %v", photoUUID, updateError)
		sendJSONResponse(responseWriter, http.StatusInternalServerError, map[string]string{"error": "внутренняя ошибка сервера"})
		return
	}

	updatedPhoto, _ := handlerInstance.photoRepository.GetByID(httpRequest.Context(), photoUUID, currentUserID, middleware.GetRole(httpRequest.Context()), nil)
	sendJSONResponse(responseWriter, http.StatusOK, updatedPhoto)
}

// Delete обрабатывает DELETE /api/photos/{id}.
// Администратор может удалить любое фото, автор — только своё фото.
// Физические файлы на диске удаляются в фоне после удаления записи из БД.
func (handlerInstance *PhotoHandler) Delete(responseWriter http.ResponseWriter, httpRequest *http.Request) {
	currentUserID := middleware.GetUserID(httpRequest.Context())
	if currentUserID == nil {
		sendJSONResponse(responseWriter, http.StatusUnauthorized, map[string]string{"error": "требуется авторизация"})
		return
	}
	isCurrentUserAdmin := middleware.GetRole(httpRequest.Context()) == "admin"

	photoUUID, parseError := uuid.Parse(chi.URLParam(httpRequest, "id"))
	if parseError != nil {
		sendJSONResponse(responseWriter, http.StatusBadRequest, map[string]string{"error": "некорректный UUID фотографии"})
		return
	}

	deletedPhotoRecord, deleteError := handlerInstance.photoRepository.Delete(httpRequest.Context(), photoUUID, *currentUserID, isCurrentUserAdmin)
	if deleteError != nil {
		if errors.Is(deleteError, repository.ErrForbidden) {
			sendJSONResponse(responseWriter, http.StatusForbidden, map[string]string{"error": "доступ запрещён: вы не можете удалить этот снимок"})
			return
		}
		log.Printf("[ERROR] PhotoHandler.Delete failed for %s: %v", photoUUID, deleteError)
		sendJSONResponse(responseWriter, http.StatusInternalServerError, map[string]string{"error": "внутренняя ошибка сервера"})
		return
	}

	// Физическое удаление файлов с диска в фоновой горутине Go
	if deletedPhotoRecord != nil {
		go func(photoToDelete *model.Photo) {
			_ = os.Remove(filepath.Join(handlerInstance.uploadDirectory, filepath.Base(photoToDelete.FilePathOriginal)))
			_ = os.Remove(filepath.Join(handlerInstance.uploadDirectory, filepath.Base(photoToDelete.FilePathMedium)))
			_ = os.Remove(filepath.Join(handlerInstance.uploadDirectory, filepath.Base(photoToDelete.FilePathThumb)))
		}(deletedPhotoRecord)
	}

	sendJSONResponse(responseWriter, http.StatusOK, map[string]string{"status": "deleted"})
}

// ListUserPhotos обрабатывает GET /api/users/me/photos.
// Возвращает фотографии, опубликованные текущим пользователем.
func (handlerInstance *PhotoHandler) ListUserPhotos(responseWriter http.ResponseWriter, httpRequest *http.Request) {
	currentUserID := middleware.GetUserID(httpRequest.Context())
	if currentUserID == nil {
		sendJSONResponse(responseWriter, http.StatusUnauthorized, map[string]string{"error": "требуется авторизация"})
		return
	}

	pageNumber, _ := strconv.Atoi(httpRequest.URL.Query().Get("page"))
	limitPerPage, _ := strconv.Atoi(httpRequest.URL.Query().Get("limit"))

	userPhotosList, queryError := handlerInstance.photoRepository.List(httpRequest.Context(), repository.PhotoFilter{
		Page:            pageNumber,
		Limit:           limitPerPage,
		UserID:          currentUserID,
		IncludePrivate:  true,
		CurrentUserID:   currentUserID,
		CurrentUserRole: middleware.GetRole(httpRequest.Context()),
	})
	if queryError != nil {
		log.Printf("[ERROR] PhotoHandler.ListUserPhotos failed: %v", queryError)
		sendJSONResponse(responseWriter, http.StatusInternalServerError, map[string]string{"error": "внутренняя ошибка сервера"})
		return
	}

	if userPhotosList == nil {
		userPhotosList = []model.Photo{}
	}
	sendJSONResponse(responseWriter, http.StatusOK, userPhotosList)
}

// sendJSONResponse — вспомогательная функция: устанавливает заголовок Content-Type: application/json,
// HTTP-код ответа и сериализует переданную структуру в тело ответа.
func sendJSONResponse(responseWriter http.ResponseWriter, httpStatusCode int, payloadData interface{}) {
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(httpStatusCode)
	_ = json.NewEncoder(responseWriter).Encode(payloadData)
}
