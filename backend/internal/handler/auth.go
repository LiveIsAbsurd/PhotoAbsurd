package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"photogallery/internal/middleware"
	"photogallery/internal/model"
	"photogallery/internal/repository"
	"photogallery/internal/service"
)

// dummyPasswordHash используется для выравнивания времени отклика при Timing Attack (cost 12 bcrypt)
const dummyPasswordHash = "$2a$12$Z0E0rM1y5H3W8j2L6n1X6.R3mF8wQ7vA9pD5sK2tL1mN4oP7qR9s."

// AuthHandler обрабатывает HTTP-запросы авторизации и управления учётными записями
type AuthHandler struct {
	userRepository  *repository.UserRepository
	photoRepository *repository.PhotoRepository
	authService     *service.AuthService
	uploadDirectory string
	isCookieSecure  bool
}

// NewAuthHandler — конструктор обработчика авторизации
func NewAuthHandler(
	userRepository *repository.UserRepository,
	photoRepository *repository.PhotoRepository,
	authService *service.AuthService,
	uploadDirectory string,
	isCookieSecure bool,
) *AuthHandler {
	return &AuthHandler{
		userRepository:  userRepository,
		photoRepository: photoRepository,
		authService:     authService,
		uploadDirectory: uploadDirectory,
		isCookieSecure:  isCookieSecure,
	}
}

// Login обрабатывает POST /api/auth/login.
// Аутентифицирует пользователя строго по связке Логин + Пароль (без email).
func (handlerInstance *AuthHandler) Login(responseWriter http.ResponseWriter, httpRequest *http.Request) {
	var loginPayload model.LoginRequest

	// Декодируем входящий JSON из тела запроса
	decodeError := json.NewDecoder(httpRequest.Body).Decode(&loginPayload)
	if decodeError != nil {
		sendJSONResponse(responseWriter, http.StatusBadRequest, map[string]string{"error": "некорректный формат JSON"})
		return
	}

	loginPayload.Username = strings.TrimSpace(loginPayload.Username)

	if loginPayload.Username == "" || loginPayload.Password == "" {
		sendJSONResponse(responseWriter, http.StatusBadRequest, map[string]string{"error": "логин и пароль обязательны для заполнения"})
		return
	}

	// Ищем пользователя в базе данных по логину
	foundUser, findError := handlerInstance.userRepository.GetByUsername(httpRequest.Context(), loginPayload.Username)
	if findError != nil {
		// Защита от Timing Attack: если логин не существует, выполняем вычисление хеша bcrypt
		// на фиктивном значении с тем же cost=12, чтобы время отклика сервера было идентичным.
		_ = handlerInstance.authService.CheckPassword(loginPayload.Password, dummyPasswordHash)
		sendJSONResponse(responseWriter, http.StatusUnauthorized, map[string]string{"error": "неверный логин или пароль"})
		return
	}

	if !handlerInstance.authService.CheckPassword(loginPayload.Password, foundUser.PasswordHash) {
		sendJSONResponse(responseWriter, http.StatusUnauthorized, map[string]string{"error": "неверный логин или пароль"})
		return
	}

	// Генерируем новый JWT токен на 7 дней
	tokenString, tokenError := handlerInstance.authService.GenerateToken(foundUser.ID, foundUser.Username, foundUser.Role, foundUser.TokenVersion)
	if tokenError != nil {
		sendJSONResponse(responseWriter, http.StatusInternalServerError, map[string]string{"error": "ошибка генерации токена авторизации"})
		return
	}

	// Устанавливаем защищенную HttpOnly cookie с токеном
	handlerInstance.setAuthenticationCookie(responseWriter, tokenString)
	sendJSONResponse(responseWriter, http.StatusOK, foundUser)
}

// Logout обрабатывает POST /api/auth/logout.
// Инвалидирует сессию в браузере и отзывает токен в черном списке базы данных.
func (handlerInstance *AuthHandler) Logout(responseWriter http.ResponseWriter, httpRequest *http.Request) {
	// Если передана валидная кука, заносим ID токена в черный список отозванных
	if authCookie, err := httpRequest.Cookie(middleware.AuthCookieName); err == nil && authCookie.Value != "" {
		if claims, parseErr := handlerInstance.authService.ValidateToken(authCookie.Value); parseErr == nil && claims != nil {
			var exp time.Time
			if claims.ExpiresAt != nil {
				exp = claims.ExpiresAt.Time
			} else {
				exp = time.Now().Add(7 * 24 * time.Hour)
			}
			if claims.ID != "" {
				_ = handlerInstance.userRepository.RevokeToken(httpRequest.Context(), claims.ID, exp)
			}
		}
	}

	http.SetCookie(responseWriter, &http.Cookie{
		Name:     middleware.AuthCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   handlerInstance.isCookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0), // 1 января 1970 года — браузер немедленно удалит куку
		MaxAge:   -1,
	})
	sendJSONResponse(responseWriter, http.StatusOK, map[string]string{"status": "logged out"})
}

// Me обрабатывает GET /api/auth/me.
// Возвращает профиль текущего авторизованного пользователя.
func (handlerInstance *AuthHandler) Me(responseWriter http.ResponseWriter, httpRequest *http.Request) {
	currentUserID := middleware.GetUserID(httpRequest.Context())
	if currentUserID == nil {
		sendJSONResponse(responseWriter, http.StatusUnauthorized, map[string]string{"error": "пользователь не авторизован"})
		return
	}

	currentUser, findError := handlerInstance.userRepository.GetByID(httpRequest.Context(), *currentUserID)
	if findError != nil {
		if errors.Is(findError, repository.ErrUserNotFound) {
			sendJSONResponse(responseWriter, http.StatusNotFound, map[string]string{"error": "пользователь не найден"})
			return
		}
		log.Printf("[ERROR] Me: failed to get user by id %s: %v", *currentUserID, findError)
		sendJSONResponse(responseWriter, http.StatusInternalServerError, map[string]string{"error": "внутренняя ошибка сервера"})
		return
	}

	sendJSONResponse(responseWriter, http.StatusOK, currentUser)
}

// AdminCreateUser обрабатывает POST /api/admin/users.
// Позволяет администратору создавать новых авторов и администраторов.
func (handlerInstance *AuthHandler) AdminCreateUser(responseWriter http.ResponseWriter, httpRequest *http.Request) {
	var createUserPayload model.CreateUserRequest

	decodeError := json.NewDecoder(httpRequest.Body).Decode(&createUserPayload)
	if decodeError != nil {
		sendJSONResponse(responseWriter, http.StatusBadRequest, map[string]string{"error": "некорректные входные данные"})
		return
	}

	if createUserPayload.Username == "" || len(createUserPayload.Password) < 6 {
		sendJSONResponse(responseWriter, http.StatusBadRequest, map[string]string{
			"error": "логин и пароль (длиной не менее 6 символов) обязательны",
		})
		return
	}

	assignedRole := createUserPayload.Role
	if assignedRole != "admin" {
		assignedRole = "user"
	}

	hashedPassword, hashError := handlerInstance.authService.HashPassword(createUserPayload.Password)
	if hashError != nil {
		sendJSONResponse(responseWriter, http.StatusInternalServerError, map[string]string{"error": "ошибка хеширования пароля"})
		return
	}

	newCreatedUser := model.User{
		ID:           uuid.New(),
		Username:     createUserPayload.Username,
		PasswordHash: hashedPassword,
		Role:         assignedRole,
		AvatarURL:    "",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	saveError := handlerInstance.userRepository.Create(httpRequest.Context(), &newCreatedUser)
	if saveError != nil {
		sendJSONResponse(responseWriter, http.StatusConflict, map[string]string{"error": "пользователь с таким логином уже существует"})
		return
	}

	sendJSONResponse(responseWriter, http.StatusCreated, newCreatedUser)
}

// AdminListUsers обрабатывает GET /api/admin/users.
// Возвращает список всех зарегистрированных пользователей для панели управления.
func (handlerInstance *AuthHandler) AdminListUsers(responseWriter http.ResponseWriter, httpRequest *http.Request) {
	allUsersList, queryError := handlerInstance.userRepository.List(httpRequest.Context())
	if queryError != nil {
		log.Printf("[ERROR] AdminListUsers failed: %v", queryError)
		sendJSONResponse(responseWriter, http.StatusInternalServerError, map[string]string{"error": "внутренняя ошибка сервера"})
		return
	}
	if allUsersList == nil {
		allUsersList = []model.User{}
	}
	sendJSONResponse(responseWriter, http.StatusOK, allUsersList)
}

// AdminDeleteUser обрабатывает DELETE /api/admin/users/{id}.
// Позволяет администратору удалить учётную запись пользователя.
func (handlerInstance *AuthHandler) AdminDeleteUser(responseWriter http.ResponseWriter, httpRequest *http.Request) {
	userIDURLParameter := chi.URLParam(httpRequest, "id")
	targetUserUUID, parseError := uuid.Parse(userIDURLParameter)
	if parseError != nil {
		sendJSONResponse(responseWriter, http.StatusBadRequest, map[string]string{"error": "некорректный UUID пользователя"})
		return
	}

	// Запрещаем администратору случайно удалить собственную учётную запись
	currentAdminID := middleware.GetUserID(httpRequest.Context())
	if currentAdminID != nil && *currentAdminID == targetUserUUID {
		sendJSONResponse(responseWriter, http.StatusBadRequest, map[string]string{"error": "нельзя удалить собственный аккаунт"})
		return
	}

	// Перед каскадным удалением пользователя из БД предварительно удаляем с диска все его файлы WebP
	if handlerInstance.photoRepository != nil && handlerInstance.uploadDirectory != "" {
		if filePaths, err := handlerInstance.photoRepository.GetFilePathsByUserID(httpRequest.Context(), targetUserUUID); err == nil {
			for _, filePath := range filePaths {
				_ = os.Remove(filepath.Join(handlerInstance.uploadDirectory, filepath.Base(filePath)))
			}
		}
	}

	deleteError := handlerInstance.userRepository.Delete(httpRequest.Context(), targetUserUUID)
	if deleteError != nil {
		log.Printf("[ERROR] AdminDeleteUser: failed to delete user %s: %v", targetUserUUID, deleteError)
		sendJSONResponse(responseWriter, http.StatusInternalServerError, map[string]string{"error": "внутренняя ошибка сервера"})
		return
	}

	sendJSONResponse(responseWriter, http.StatusOK, map[string]string{"status": "deleted"})
}

// AdminChangeUserPassword обрабатывает PUT /api/admin/users/{id}/password.
// Позволяет администратору изменить пароль любого пользователя.
// При этом автоматически увеличивается token_version пользователя,
// что немедленно инвалидирует все активные JWT-токены этого пользователя на всех устройствах.
func (handlerInstance *AuthHandler) AdminChangeUserPassword(responseWriter http.ResponseWriter, httpRequest *http.Request) {
	userIDURLParameter := chi.URLParam(httpRequest, "id")
	targetUserUUID, parseError := uuid.Parse(userIDURLParameter)
	if parseError != nil {
		sendJSONResponse(responseWriter, http.StatusBadRequest, map[string]string{"error": "некорректный UUID пользователя"})
		return
	}

	var payload model.AdminChangePasswordRequest
	decodeError := json.NewDecoder(httpRequest.Body).Decode(&payload)
	if decodeError != nil {
		sendJSONResponse(responseWriter, http.StatusBadRequest, map[string]string{"error": "некорректный формат JSON"})
		return
	}

	if len(payload.Password) < 6 {
		sendJSONResponse(responseWriter, http.StatusBadRequest, map[string]string{
			"error": "пароль должен содержать не менее 6 символов",
		})
		return
	}

	hashedPassword, hashError := handlerInstance.authService.HashPassword(payload.Password)
	if hashError != nil {
		sendJSONResponse(responseWriter, http.StatusInternalServerError, map[string]string{"error": "ошибка хеширования пароля"})
		return
	}

	updateError := handlerInstance.userRepository.UpdatePassword(httpRequest.Context(), targetUserUUID, hashedPassword)
	if updateError != nil {
		if errors.Is(updateError, repository.ErrUserNotFound) {
			sendJSONResponse(responseWriter, http.StatusNotFound, map[string]string{"error": "пользователь не найден"})
			return
		}
		log.Printf("[ERROR] AdminChangeUserPassword: failed to update password for user %s: %v", targetUserUUID, updateError)
		sendJSONResponse(responseWriter, http.StatusInternalServerError, map[string]string{"error": "внутренняя ошибка сервера"})
		return
	}

	sendJSONResponse(responseWriter, http.StatusOK, map[string]string{"status": "password updated"})
}

// setAuthenticationCookie устанавливает защищённый cookie с JWT токеном
func (handlerInstance *AuthHandler) setAuthenticationCookie(responseWriter http.ResponseWriter, jwtTokenString string) {
	http.SetCookie(responseWriter, &http.Cookie{
		Name:     middleware.AuthCookieName,
		Value:    jwtTokenString,
		Path:     "/",
		HttpOnly: true, // Защита от кражи токена через скрипты XSS
		Secure:   handlerInstance.isCookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})
}
