package model

import (
	"time"

	"github.com/google/uuid"
)

// User представляет учётную запись пользователя в базе данных.
// Теги после типов полей (например `json:"username"`) объясняют стандартному пакету
// encoding/json, как сериализовать структуру в формат JSON для отправки в браузер.
// Тег `json:"-"` означает, что поле PasswordHash никогда не попадет в JSON-ответ клиенту!
type User struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	Email        *string   `json:"email,omitempty"` // Указатель *string позволяет хранить NULL из базы данных
	PasswordHash string    `json:"-"`               // Хеш пароля (скрыт от фронтенда ради безопасности)
	Role         string    `json:"role"`            // Роль: "admin" или "user"
	TokenVersion int       `json:"-"`               // Версия сессионного токена (для мгновенной инвалидации)
	AvatarURL    string    `json:"avatar_url"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CameraMeta хранит технические метаданные EXIF фотоснимка (камера, объектив, экспозиция).
// Очищается от приватных данных геолокации (GPS) при сохранении.
type CameraMeta struct {
	Make         string `json:"make,omitempty"`         // Производитель камеры (например, "Sony")
	Model        string `json:"model,omitempty"`        // Модель камеры (например, "ILCE-7M3")
	Lens         string `json:"lens,omitempty"`         // Модель объектива
	FocalLength  string `json:"focal_length,omitempty"` // Фокусное расстояние (например, "35 mm")
	FNumber      string `json:"f_number,omitempty"`     // Диафрагменное число (например, "f/2.0")
	ExposureTime string `json:"exposure_time,omitempty"`// Выдержка (например, "1/500 s")
	ISO          string `json:"iso,omitempty"`          // Светочувствительность (например, "ISO 200")
	DateTime     string `json:"date_time,omitempty"`    // Исходная дата съёмки
}

// Photo представляет опубликованную фотографию.
type Photo struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	AlbumID          *uuid.UUID `json:"album_id,omitempty"` // *uuid.UUID позволяет полю быть nil (без альбома)
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	FilePathOriginal string     `json:"file_path_original"` // Путь к оптимизированному оригиналу
	FilePathMedium   string     `json:"file_path_medium"`   // Путь к средней версии для быстрого предпросмотра
	FilePathThumb    string     `json:"file_path_thumb"`    // Путь к квадратной миниатюре
	Width            int        `json:"width"`
	Height           int        `json:"height"`
	FileSize         int64      `json:"file_size"`
	CameraMeta       CameraMeta `json:"camera_meta"`
	LikesCount       int        `json:"likes_count"`
	ViewsCount       int        `json:"views_count"`
	IsPublic         bool       `json:"is_public"`
	CreatedAt        time.Time  `json:"created_at"`

	// Дополнительные вычисляемые поля для удобства отображения на клиенте:
	AuthorUsername string `json:"author_username,omitempty"` // Логин автора снимка
	AuthorAvatar   string `json:"author_avatar,omitempty"`   // Аватар автора
	IsLiked        bool   `json:"is_liked"`                  // Поставил ли текущий пользователь/гость лайк
}

// Album представляет тематическую коллекцию или альбом фотографий.
type Album struct {
	ID           uuid.UUID  `json:"id"`
	UserID       uuid.UUID  `json:"user_id"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	CoverPhotoID *uuid.UUID `json:"cover_photo_id,omitempty"`
	CoverPhoto   *string    `json:"cover_photo,omitempty"` // URL миниатюры обложки альбома
	PhotosCount  int        `json:"photos_count"`          // Общее количество фотографий в альбоме
	IsPublic     bool       `json:"is_public"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Like представляет запись о лайке от зарегистрированного пользователя или гостя.
type Like struct {
	ID             uuid.UUID  `json:"id"`
	PhotoID        uuid.UUID  `json:"photo_id"`
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	GuestSessionID *string    `json:"guest_session_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// LoginRequest — DTO (Data Transfer Object) входящего запроса на авторизацию.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// CreateUserRequest — DTO запроса от администратора на создание нового пользователя.
type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"` // "admin" или "user"
}

// AdminChangePasswordRequest — DTO запроса от администратора на смену пароля пользователю.
type AdminChangePasswordRequest struct {
	Password string `json:"password"`
}

// UpdatePhotoRequest — DTO для изменения метаданных фото.
// Поля оформлены в виде указателей (*string, *bool), чтобы отличать
// намеренно переданное пустое значение от непереданного поля (nil).
type UpdatePhotoRequest struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	AlbumID     *uuid.UUID `json:"album_id"`
	IsPublic    *bool      `json:"is_public"`
}

// LikeToggleResponse — ответ сервера после переключения лайка.
type LikeToggleResponse struct {
	Liked      bool `json:"liked"`       // true — лайк поставлен, false — лайк снят
	LikesCount int  `json:"likes_count"` // Актуальное общее количество лайков
}
