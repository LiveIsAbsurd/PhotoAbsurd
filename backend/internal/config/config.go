package config

import (
	"os"
	"strconv"
	"strings"
)

// Config — структура, хранящая настройки всего приложения.
// В Go структуры (struct) объединяют связанные поля данных.
type Config struct {
	Port               string   // Порт, на котором слушает HTTP-сервер (по умолчанию "8080")
	DatabaseURL        string   // Строка подключения к PostgreSQL (DSN)
	UploadDir          string   // Путь к каталогу хранения загруженных фотографий на диске
	JWTSecret          string   // Секретный ключ для подписи и проверки JWT-токенов авторизации
	CookieSecure       bool     // Флаг Secure для cookies (true при использовании HTTPS, false для локального HTTP)
	MaxUploadSize      int64    // Максимальный размер загружаемого файла в байтах (20 МБ)
	CORSAllowedOrigins []string // Дополнительные разрешённые домены для CORS (например, https://photos.example.com)
}

// Load считывает переменные окружения операционной системы и возвращает заполненную конфигурацию.
// Звёздочка (*Config) означает, что функция возвращает указатель (ссылку на структуру в памяти),
// что позволяет избежать копирования больших объёмов данных.
func Load() *Config {
	serverPort := getEnvironmentVariable("PORT", "8080")
	databaseConnectionURL := getEnvironmentVariable(
		"DATABASE_URL",
		"postgres://photouser:photosecret@postgres:5432/photogallery?sslmode=disable",
	)
	fileUploadDirectory := getEnvironmentVariable("UPLOAD_DIR", "/uploads")
	jwtSecretSigningKey := getEnvironmentVariable("JWT_SECRET", "photogallery_jwt_secret_key_32charslong!")
	isCookieSecure := getEnvironmentVariable("COOKIE_SECURE", "false") == "true"

	// Преобразуем строковое значение лимита размера файла в целое число (int64)
	maxUploadSizeString := getEnvironmentVariable("MAX_UPLOAD_SIZE", "20971520") // 20 MB в байтах
	maxUploadSizeBytes, parseError := strconv.ParseInt(maxUploadSizeString, 10, 64)
	if parseError != nil {
		// Если значение некорректное — используем безопасный дефолт 20 МБ
		maxUploadSizeBytes = 20 * 1024 * 1024
	}

	// Разбор дополнительных доменов CORS через запятую
	corsOriginsString := os.Getenv("CORS_ALLOWED_ORIGINS")
	var corsAllowedOrigins []string
	if corsOriginsString != "" {
		for _, origin := range strings.Split(corsOriginsString, ",") {
			trimmedOrigin := strings.TrimSpace(origin)
			if trimmedOrigin != "" {
				corsAllowedOrigins = append(corsAllowedOrigins, trimmedOrigin)
			}
		}
	}

	return &Config{
		Port:               serverPort,
		DatabaseURL:        databaseConnectionURL,
		UploadDir:          fileUploadDirectory,
		JWTSecret:          jwtSecretSigningKey,
		CookieSecure:       isCookieSecure,
		MaxUploadSize:      maxUploadSizeBytes,
		CORSAllowedOrigins: corsAllowedOrigins,
	}
}

// getEnvironmentVariable — вспомогательная функция: получает значение переменной окружения
// по ключу key. Если переменная не задана в системе, возвращает значение defaultValue.
func getEnvironmentVariable(variableKey, defaultValue string) string {
	environmentValue := os.Getenv(variableKey)
	if environmentValue != "" {
		return environmentValue
	}
	return defaultValue
}
