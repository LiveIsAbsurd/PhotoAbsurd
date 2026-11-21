package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Ошибки сервиса авторизации
var (
	ErrInvalidCredentials = errors.New("неверный логин или пароль")
	ErrInvalidToken       = errors.New("недействительный или просроченный токен авторизации")
)

// Claims определяет полезную нагрузку (payload), зашитую внутри JWT-токена.
// В Go мы встраиваем структуру jwt.RegisteredClaims, которая содержит стандартные поля:
// срок действия (exp), дату выпуска (iat), субъект (sub), уникальный идентификатор (jti / id).
type Claims struct {
	UserID       uuid.UUID `json:"user_id"`
	Username     string    `json:"username"`
	Role         string    `json:"role"`
	TokenVersion int       `json:"token_version"`
	jwt.RegisteredClaims
}

// AuthService инкапсулирует криптографические операции: хеширование паролей и выпуск JWT
type AuthService struct {
	jwtSecretKeyBytes []byte
}

// NewAuthService — конструктор сервиса аутентификации
func NewAuthService(jwtSecretString string) *AuthService {
	return &AuthService{jwtSecretKeyBytes: []byte(jwtSecretString)}
}

// HashPassword генерирует криптографически стойкий хеш пароля с использованием алгоритма bcrypt.
// Параметр cost = 12 означает 2^12 (4096) итераций функции растяжения ключа,
// что делает подбор пароля перебором (брутфорс) невозможным даже на современных GPU.
func (service *AuthService) HashPassword(plainPassword string) (string, error) {
	hashedPasswordBytes, hashingError := bcrypt.GenerateFromPassword([]byte(plainPassword), 12)
	return string(hashedPasswordBytes), hashingError
}

// CheckPassword безопасно сравнивает открытый пароль с сохранённым хешем bcrypt.
// Сравнение выполняется за постоянное время (защита от атак по времени / Timing Attacks).
func (service *AuthService) CheckPassword(plainPassword, passwordHashFromDatabase string) bool {
	comparisonError := bcrypt.CompareHashAndPassword([]byte(passwordHashFromDatabase), []byte(plainPassword))
	return comparisonError == nil
}

// GenerateToken создаёт подписанный JWT-токен на 7 дней для авторизованного пользователя
func (service *AuthService) GenerateToken(userID uuid.UUID, username, userRole string, tokenVersion int) (string, error) {
	tokenClaims := Claims{
		UserID:       userID,
		Username:     username,
		Role:         userRole,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(), // Уникальный идентификатор токена (jti)
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // Срок жизни 7 суток
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID.String(),
		},
	}

	// Создаём токен с алгоритмом подписи HMAC-SHA256
	unsignedToken := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims)

	// Подписываем токен нашим секретным ключом
	signedTokenString, signError := unsignedToken.SignedString(service.jwtSecretKeyBytes)
	return signedTokenString, signError
}

// ValidateToken проверяет криптографическую подпись и срок годности токена,
// а также возвращает расшифрованные Claims с информацией о пользователе
func (service *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	parsedToken, parseError := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Убеждаемся, что алгоритм подписи строго HMAC (защита от подмены на "none" или RSA)
		if _, isHMAC := token.Method.(*jwt.SigningMethodHMAC); !isHMAC {
			return nil, ErrInvalidToken
		}
		return service.jwtSecretKeyBytes, nil
	})

	if parseError != nil {
		return nil, parseError
	}

	// Извлекаем и валидируем Claims
	tokenClaims, isClaimsValid := parsedToken.Claims.(*Claims)
	if !isClaimsValid || !parsedToken.Valid {
		return nil, ErrInvalidToken
	}

	return tokenClaims, nil
}
