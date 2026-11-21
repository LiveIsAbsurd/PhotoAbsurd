package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"photogallery/internal/model"
)

// Стандартные ошибки при работе с пользователями
var (
	ErrUserNotFound      = errors.New("пользователь не найден")
	ErrUserAlreadyExists = errors.New("пользователь с таким логином уже существует")
)

// UserRepository отвечает за сохранение и получение пользователей из базы данных
type UserRepository struct {
	database *DB
}

// NewUserRepository — конструктор репозитория пользователей
func NewUserRepository(database *DB) *UserRepository {
	return &UserRepository{database: database}
}

// Create сохраняет нового пользователя в таблице users
func (repository *UserRepository) Create(executionContext context.Context, newUser *model.User) error {
	if newUser.Role == "" {
		newUser.Role = "user"
	}
	if newUser.TokenVersion == 0 {
		newUser.TokenVersion = 1
	}
	insertQuery := `
		INSERT INTO users (id, username, email, password_hash, role, token_version, avatar_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, executionError := repository.database.Pool.Exec(
		executionContext,
		insertQuery,
		newUser.ID,
		newUser.Username,
		newUser.Email,
		newUser.PasswordHash,
		newUser.Role,
		newUser.TokenVersion,
		newUser.AvatarURL,
		newUser.CreatedAt,
		newUser.UpdatedAt,
	)
	return executionError
}

// GetByID находит пользователя по его первичному ключу UUID
func (repository *UserRepository) GetByID(executionContext context.Context, userID uuid.UUID) (*model.User, error) {
	selectQuery := `
		SELECT id, username, email, password_hash, role, token_version, avatar_url, created_at, updated_at
		FROM users WHERE id = $1
	`
	var foundUser model.User
	queryError := repository.database.Pool.QueryRow(executionContext, selectQuery, userID).Scan(
		&foundUser.ID,
		&foundUser.Username,
		&foundUser.Email,
		&foundUser.PasswordHash,
		&foundUser.Role,
		&foundUser.TokenVersion,
		&foundUser.AvatarURL,
		&foundUser.CreatedAt,
		&foundUser.UpdatedAt,
	)
	if queryError != nil {
		if errors.Is(queryError, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, queryError
	}
	return &foundUser, nil
}

// GetByUsername находит пользователя по уникальному логину (используется при авторизации)
func (repository *UserRepository) GetByUsername(executionContext context.Context, username string) (*model.User, error) {
	selectQuery := `
		SELECT id, username, email, password_hash, role, token_version, avatar_url, created_at, updated_at
		FROM users WHERE username = $1
	`
	var foundUser model.User
	queryError := repository.database.Pool.QueryRow(executionContext, selectQuery, username).Scan(
		&foundUser.ID,
		&foundUser.Username,
		&foundUser.Email,
		&foundUser.PasswordHash,
		&foundUser.Role,
		&foundUser.TokenVersion,
		&foundUser.AvatarURL,
		&foundUser.CreatedAt,
		&foundUser.UpdatedAt,
	)
	if queryError != nil {
		if errors.Is(queryError, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, queryError
	}
	return &foundUser, nil
}

// List возвращает список всех пользователей для панели администратора
func (repository *UserRepository) List(executionContext context.Context) ([]model.User, error) {
	selectQuery := `
		SELECT id, username, email, password_hash, role, token_version, avatar_url, created_at, updated_at
		FROM users ORDER BY created_at DESC
	`
	rows, queryError := repository.database.Pool.Query(executionContext, selectQuery)
	if queryError != nil {
		return nil, queryError
	}
	defer rows.Close()

	usersList := []model.User{}
	for rows.Next() {
		var userItem model.User
		scanError := rows.Scan(
			&userItem.ID,
			&userItem.Username,
			&userItem.Email,
			&userItem.PasswordHash,
			&userItem.Role,
			&userItem.TokenVersion,
			&userItem.AvatarURL,
			&userItem.CreatedAt,
			&userItem.UpdatedAt,
		)
		if scanError != nil {
			return nil, scanError
		}
		usersList = append(usersList, userItem)
	}
	return usersList, rows.Err()
}

// UpdatePassword обновляет пароль пользователя и инкрементирует token_version,
// что мгновенно аннулирует все ранее выпущенные JWT-токены данного пользователя.
func (repository *UserRepository) UpdatePassword(executionContext context.Context, userID uuid.UUID, passwordHash string) error {
	updateQuery := `
		UPDATE users
		SET password_hash = $1, token_version = token_version + 1, updated_at = NOW()
		WHERE id = $2
	`
	commandTag, executionError := repository.database.Pool.Exec(executionContext, updateQuery, passwordHash, userID)
	if executionError != nil {
		return executionError
	}
	if commandTag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// RevokeToken добавляет ID токена (jti) в черный список отозванных токенов
func (repository *UserRepository) RevokeToken(executionContext context.Context, tokenID string, expiresAt time.Time) error {
	insertQuery := `
		INSERT INTO revoked_tokens (id, expires_at, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (id) DO NOTHING
	`
	_, executionError := repository.database.Pool.Exec(executionContext, insertQuery, tokenID, expiresAt)
	return executionError
}

// IsTokenRevoked проверяет, находится ли ID токена в черном списке
func (repository *UserRepository) IsTokenRevoked(executionContext context.Context, tokenID string) (bool, error) {
	checkQuery := `SELECT EXISTS(SELECT 1 FROM revoked_tokens WHERE id = $1)`
	var isRevoked bool
	queryError := repository.database.Pool.QueryRow(executionContext, checkQuery, tokenID).Scan(&isRevoked)
	return isRevoked, queryError
}

// Delete удаляет пользователя по ID
func (repository *UserRepository) Delete(executionContext context.Context, targetUserID uuid.UUID) error {
	_, executionError := repository.database.Pool.Exec(executionContext, `DELETE FROM users WHERE id = $1`, targetUserID)
	return executionError
}
