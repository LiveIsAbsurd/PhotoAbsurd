package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"photogallery/internal/repository"
	"photogallery/internal/service"
)

// contextKey — пользовательский строковый тип для ключей контекста в Go.
// В Go не рекомендуется использовать базовый тип string в качестве ключей context.WithValue,
// чтобы избежать случайных коллизий ключей между разными пакетами.
type contextKey string

const (
	UserIDContextKey   contextKey = "user_id"
	UsernameContextKey contextKey = "username"
	RoleContextKey     contextKey = "role"

	AuthCookieName = "jwt_token"
)

// Middleware содержит зависимости для проверки авторизации
type Middleware struct {
	authService    *service.AuthService
	userRepository *repository.UserRepository
	cookieSecure   bool
}

// NewMiddleware — конструктор промежуточных обработчиков
func NewMiddleware(
	authService *service.AuthService,
	userRepository *repository.UserRepository,
	cookieSecure bool,
) *Middleware {
	return &Middleware{
		authService:    authService,
		userRepository: userRepository,
		cookieSecure:   cookieSecure,
	}
}

// SessionMiddleware — центральный middleware для управления сессиями:
// Куки автоматически НЕ создаются и НЕ отправляются гостям.
// Кука jwt_token устанавливается ТОЛЬКО при явном входе пользователя (POST /api/auth/login).
// Если кука jwt_token присутствует, проверяется подпись, отсутствие в черном списке
// и соответствие версии токена (token_version) для мгновенной инвалидации при смене пароля.
func (middlewareInstance *Middleware) SessionMiddleware(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, httpRequest *http.Request) {
		requestContext := httpRequest.Context()

		// Сессия авторизованного пользователя:
		authCookie, authCookieError := httpRequest.Cookie(AuthCookieName)
		if authCookieError == nil && authCookie.Value != "" {
			userClaims, validateError := middlewareInstance.authService.ValidateToken(authCookie.Value)
			if validateError == nil && userClaims != nil {
				// 1. Проверяем, не был ли данный токен отозван (черный список при logout)
				isRevoked := false
				if userClaims.ID != "" && middlewareInstance.userRepository != nil {
					revoked, checkErr := middlewareInstance.userRepository.IsTokenRevoked(requestContext, userClaims.ID)
					if checkErr == nil && revoked {
						isRevoked = true
					}
				}

				if !isRevoked {
					// 2. Проверяем пользователя и актуальность token_version (смена пароля)
					if middlewareInstance.userRepository != nil {
						currentUser, userErr := middlewareInstance.userRepository.GetByID(requestContext, userClaims.UserID)
						if userErr == nil && currentUser != nil && currentUser.TokenVersion == userClaims.TokenVersion {
							requestContext = context.WithValue(requestContext, UserIDContextKey, currentUser.ID)
							requestContext = context.WithValue(requestContext, UsernameContextKey, currentUser.Username)
							requestContext = context.WithValue(requestContext, RoleContextKey, currentUser.Role)
						}
					} else {
						requestContext = context.WithValue(requestContext, UserIDContextKey, userClaims.UserID)
						requestContext = context.WithValue(requestContext, UsernameContextKey, userClaims.Username)
						requestContext = context.WithValue(requestContext, RoleContextKey, userClaims.Role)
					}
				}
			}
		}

		// Передаем управление следующему обработчику в цепочке с контекстом
		nextHandler.ServeHTTP(responseWriter, httpRequest.WithContext(requestContext))
	})
}

// RequireAuth блокирует неавторизованные запросы с кодом ответа 401 Unauthorized
func (middlewareInstance *Middleware) RequireAuth(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, httpRequest *http.Request) {
		if GetUserID(httpRequest.Context()) == nil {
			http.Error(responseWriter, `{"error":"требуется авторизация"}`, http.StatusUnauthorized)
			return
		}
		nextHandler.ServeHTTP(responseWriter, httpRequest)
	})
}

// RequireAdmin блокирует запросы от обычных пользователей без роли admin с кодом 403 Forbidden
func (middlewareInstance *Middleware) RequireAdmin(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, httpRequest *http.Request) {
		currentUserRole := GetRole(httpRequest.Context())
		if currentUserRole != "admin" {
			http.Error(responseWriter, `{"error":"доступ запрещён: требуются права администратора"}`, http.StatusForbidden)
			return
		}
		nextHandler.ServeHTTP(responseWriter, httpRequest)
	})
}

// GetUserID безопасно извлекает ID пользователя из контекста запроса
func GetUserID(executionContext context.Context) *uuid.UUID {
	contextValue := executionContext.Value(UserIDContextKey)
	if userUUID, ok := contextValue.(uuid.UUID); ok {
		return &userUUID
	}
	return nil
}

// GetUsername извлекает логин пользователя из контекста запроса
func GetUsername(executionContext context.Context) string {
	contextValue := executionContext.Value(UsernameContextKey)
	if usernameString, ok := contextValue.(string); ok {
		return usernameString
	}
	return ""
}

// GetRole извлекает роль пользователя из контекста запроса ("admin", "user" или "")
func GetRole(executionContext context.Context) string {
	contextValue := executionContext.Value(RoleContextKey)
	if roleString, ok := contextValue.(string); ok {
		return roleString
	}
	return ""
}

// GetGuestSessionID возвращает nil, так как гостевые сессии были полностью упразднены
func GetGuestSessionID(executionContext context.Context) *string {
	return nil
}
