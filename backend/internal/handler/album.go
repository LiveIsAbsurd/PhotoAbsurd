package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"photogallery/internal/middleware"
	"photogallery/internal/model"
	"photogallery/internal/repository"
)

// AlbumHandler обрабатывает HTTP-запросы для работы с коллекциями и альбомами
type AlbumHandler struct {
	albumRepository *repository.AlbumRepository
	photoRepository *repository.PhotoRepository
}

// NewAlbumHandler — конструктор обработчика альбомов
func NewAlbumHandler(albumRepository *repository.AlbumRepository, photoRepository *repository.PhotoRepository) *AlbumHandler {
	return &AlbumHandler{
		albumRepository: albumRepository,
		photoRepository: photoRepository,
	}
}

// List обрабатывает GET /api/albums.
// Возвращает список всех доступных альбомов.
func (handlerInstance *AlbumHandler) List(responseWriter http.ResponseWriter, httpRequest *http.Request) {
	currentUserID := middleware.GetUserID(httpRequest.Context())
	currentUserRole := middleware.GetRole(httpRequest.Context())
	albumsList, queryError := handlerInstance.albumRepository.List(httpRequest.Context(), currentUserID, currentUserRole)
	if queryError != nil {
		log.Printf("[ERROR] AlbumHandler.List failed: %v", queryError)
		sendJSONResponse(responseWriter, http.StatusInternalServerError, map[string]string{"error": "внутренняя ошибка сервера"})
		return
	}
	if albumsList == nil {
		albumsList = []model.Album{}
	}
	sendJSONResponse(responseWriter, http.StatusOK, albumsList)
}

// GetByID обрабатывает GET /api/albums/{id}.
// Возвращает данные об альбоме и вложенные в него снимки.
func (handlerInstance *AlbumHandler) GetByID(responseWriter http.ResponseWriter, httpRequest *http.Request) {
	albumIDParameter := chi.URLParam(httpRequest, "id")
	albumUUID, parseError := uuid.Parse(albumIDParameter)
	if parseError != nil {
		sendJSONResponse(responseWriter, http.StatusBadRequest, map[string]string{"error": "некорректный UUID альбома"})
		return
	}

	currentUserID := middleware.GetUserID(httpRequest.Context())
	currentUserRole := middleware.GetRole(httpRequest.Context())
	foundAlbum, findError := handlerInstance.albumRepository.GetByID(httpRequest.Context(), albumUUID, currentUserID, currentUserRole)
	if findError != nil {
		if errors.Is(findError, repository.ErrAlbumNotFound) {
			sendJSONResponse(responseWriter, http.StatusNotFound, map[string]string{"error": "альбом не найден"})
			return
		}
		log.Printf("[ERROR] AlbumHandler.GetByID failed for %s: %v", albumUUID, findError)
		sendJSONResponse(responseWriter, http.StatusInternalServerError, map[string]string{"error": "внутренняя ошибка сервера"})
		return
	}

	// Получаем фотографии, принадлежащие этому альбому
	albumPhotosList, _ := handlerInstance.photoRepository.List(httpRequest.Context(), repository.PhotoFilter{
		AlbumID:         &foundAlbum.ID,
		CurrentUserID:   middleware.GetUserID(httpRequest.Context()),
		CurrentUserRole: middleware.GetRole(httpRequest.Context()),
		GuestSessionID:  nil,
		Limit:           100,
	})

	sendJSONResponse(responseWriter, http.StatusOK, map[string]interface{}{
		"album":  foundAlbum,
		"photos": albumPhotosList,
	})
}

// Create обрабатывает POST /api/albums.
// Создаёт новый альбом для авторизованного пользователя.
func (handlerInstance *AlbumHandler) Create(responseWriter http.ResponseWriter, httpRequest *http.Request) {
	currentUserID := middleware.GetUserID(httpRequest.Context())
	if currentUserID == nil {
		sendJSONResponse(responseWriter, http.StatusUnauthorized, map[string]string{"error": "требуется авторизация"})
		return
	}

	var createAlbumPayload struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		IsPublic    *bool  `json:"is_public"`
	}
	if decodeError := json.NewDecoder(httpRequest.Body).Decode(&createAlbumPayload); decodeError != nil || createAlbumPayload.Title == "" {
		sendJSONResponse(responseWriter, http.StatusBadRequest, map[string]string{"error": "название альбома обязательно"})
		return
	}

	isPublicFlag := true
	if createAlbumPayload.IsPublic != nil {
		isPublicFlag = *createAlbumPayload.IsPublic
	}

	newAlbumRecord := model.Album{
		ID:          uuid.New(),
		UserID:      *currentUserID,
		Title:       createAlbumPayload.Title,
		Description: createAlbumPayload.Description,
		IsPublic:    isPublicFlag,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if saveError := handlerInstance.albumRepository.Create(httpRequest.Context(), &newAlbumRecord); saveError != nil {
		log.Printf("[ERROR] AlbumHandler.Create failed: %v", saveError)
		sendJSONResponse(responseWriter, http.StatusInternalServerError, map[string]string{"error": "внутренняя ошибка сервера"})
		return
	}

	sendJSONResponse(responseWriter, http.StatusCreated, newAlbumRecord)
}

// Update обрабатывает PATCH /api/albums/{id}.
// Обновляет настройки альбома (название, описание, публичный статус is_public).
func (handlerInstance *AlbumHandler) Update(responseWriter http.ResponseWriter, httpRequest *http.Request) {
	currentUserID := middleware.GetUserID(httpRequest.Context())
	if currentUserID == nil {
		sendJSONResponse(responseWriter, http.StatusUnauthorized, map[string]string{"error": "требуется авторизация"})
		return
	}

	albumUUID, parseError := uuid.Parse(chi.URLParam(httpRequest, "id"))
	if parseError != nil {
		sendJSONResponse(responseWriter, http.StatusBadRequest, map[string]string{"error": "некорректный UUID альбома"})
		return
	}

	currentUserRole := middleware.GetRole(httpRequest.Context())
	foundAlbum, findError := handlerInstance.albumRepository.GetByID(httpRequest.Context(), albumUUID, currentUserID, currentUserRole)
	if findError != nil {
		sendJSONResponse(responseWriter, http.StatusNotFound, map[string]string{"error": "альбом не найден"})
		return
	}

	isCurrentUserAdmin := middleware.GetRole(httpRequest.Context()) == "admin"
	if !isCurrentUserAdmin && foundAlbum.UserID != *currentUserID {
		sendJSONResponse(responseWriter, http.StatusForbidden, map[string]string{"error": "доступ запрещён: вы не являетесь автором альбома"})
		return
	}

	var updatePayload struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		IsPublic    *bool   `json:"is_public"`
	}
	if decodeError := json.NewDecoder(httpRequest.Body).Decode(&updatePayload); decodeError != nil {
		sendJSONResponse(responseWriter, http.StatusBadRequest, map[string]string{"error": "некорректный JSON"})
		return
	}

	newTitle := foundAlbum.Title
	if updatePayload.Title != nil && *updatePayload.Title != "" {
		newTitle = *updatePayload.Title
	}
	newDescription := foundAlbum.Description
	if updatePayload.Description != nil {
		newDescription = *updatePayload.Description
	}
	newIsPublic := foundAlbum.IsPublic
	if updatePayload.IsPublic != nil {
		newIsPublic = *updatePayload.IsPublic
	}

	if updateError := handlerInstance.albumRepository.Update(httpRequest.Context(), albumUUID, newTitle, newDescription, newIsPublic); updateError != nil {
		log.Printf("[ERROR] AlbumHandler.Update failed for %s: %v", albumUUID, updateError)
		sendJSONResponse(responseWriter, http.StatusInternalServerError, map[string]string{"error": "внутренняя ошибка сервера"})
		return
	}

	foundAlbum.Title = newTitle
	foundAlbum.Description = newDescription
	foundAlbum.IsPublic = newIsPublic
	sendJSONResponse(responseWriter, http.StatusOK, foundAlbum)
}

// Delete обрабатывает DELETE /api/albums/{id}.
// Удаляет альбом автора или администратором (фотографии при этом не удаляются, их album_id сбрасывается в NULL).
func (handlerInstance *AlbumHandler) Delete(responseWriter http.ResponseWriter, httpRequest *http.Request) {
	currentUserID := middleware.GetUserID(httpRequest.Context())
	if currentUserID == nil {
		sendJSONResponse(responseWriter, http.StatusUnauthorized, map[string]string{"error": "требуется авторизация"})
		return
	}

	albumUUID, parseError := uuid.Parse(chi.URLParam(httpRequest, "id"))
	if parseError != nil {
		sendJSONResponse(responseWriter, http.StatusBadRequest, map[string]string{"error": "некорректный UUID альбома"})
		return
	}

	isCurrentUserAdmin := middleware.GetRole(httpRequest.Context()) == "admin"
	var deleteError error
	if isCurrentUserAdmin {
		deleteError = handlerInstance.albumRepository.DeleteAsAdmin(httpRequest.Context(), albumUUID)
	} else {
		deleteError = handlerInstance.albumRepository.Delete(httpRequest.Context(), albumUUID, *currentUserID)
	}

	if deleteError != nil {
		if errors.Is(deleteError, repository.ErrForbidden) {
			sendJSONResponse(responseWriter, http.StatusForbidden, map[string]string{"error": "доступ запрещён: альбом не найден или принадлежит другому автору"})
			return
		}
		log.Printf("[ERROR] AlbumHandler.Delete failed for %s: %v", albumUUID, deleteError)
		sendJSONResponse(responseWriter, http.StatusInternalServerError, map[string]string{"error": "внутренняя ошибка сервера"})
		return
	}

	sendJSONResponse(responseWriter, http.StatusOK, map[string]string{"status": "deleted"})
}
