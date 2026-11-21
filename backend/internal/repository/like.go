package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// LikeRepository отвечает за операции с лайками в базе данных
type LikeRepository struct {
	database *DB
}

// NewLikeRepository — конструктор репозитория лайков
func NewLikeRepository(database *DB) *LikeRepository {
	return &LikeRepository{database: database}
}

// ToggleLike атомарно переключает статус отметки "Нравится" (ставит или снимает лайк).
// Функция использует транзакцию PostgreSQL с блокировкой строки (FOR UPDATE),
// чтобы полностью исключить состояние гонки (Race Condition) при одновременных кликах.
func (repository *LikeRepository) ToggleLike(
	executionContext context.Context,
	photoID uuid.UUID,
	authenticatedUserID *uuid.UUID,
	guestSessionID *string,
) (isNowLiked bool, updatedLikesCount int, operationError error) {
	// Для установки лайка обязательно наличие либо ID пользователя, либо ID сессии гостя
	if authenticatedUserID == nil && (guestSessionID == nil || *guestSessionID == "") {
		return false, 0, fmt.Errorf("требуется идентификатор пользователя или сессия гостя")
	}

	// 1. Открываем транзакцию базы данных
	databaseTransaction, beginError := repository.database.Pool.BeginTx(executionContext, pgx.TxOptions{})
	if beginError != nil {
		return false, 0, beginError
	}
	// defer databaseTransaction.Rollback гарантирует откат изменений, если произойдёт ошибка до Commit
	defer databaseTransaction.Rollback(executionContext)

	// Блокируем родительскую запись фото (FOR UPDATE), чтобы сериализовать параллельные
	// клики и предотвратить состояние гонки (Race Condition) и дедлоки.
	var lockedPhotoID uuid.UUID
	lockError := databaseTransaction.QueryRow(
		executionContext,
		`SELECT id FROM photos WHERE id = $1 FOR UPDATE`,
		photoID,
	).Scan(&lockedPhotoID)
	if lockError != nil {
		if errors.Is(lockError, pgx.ErrNoRows) {
			return false, 0, ErrPhotoNotFound
		}
		return false, 0, lockError
	}

	var existingLikeID uuid.UUID
	var checkLikeQuery string
	var queryParameter interface{}

	if authenticatedUserID != nil {
		checkLikeQuery = `SELECT id FROM likes WHERE photo_id = $1 AND user_id = $2 FOR UPDATE`
		queryParameter = *authenticatedUserID
	} else {
		checkLikeQuery = `SELECT id FROM likes WHERE photo_id = $1 AND guest_session_id = $2 FOR UPDATE`
		queryParameter = *guestSessionID
	}

	searchError := databaseTransaction.QueryRow(executionContext, checkLikeQuery, photoID, queryParameter).Scan(&existingLikeID)

	if searchError == nil {
		// 2А. Лайк уже существует -> снимаем его (Unlike)
		_, deleteError := databaseTransaction.Exec(executionContext, `DELETE FROM likes WHERE id = $1`, existingLikeID)
		if deleteError != nil {
			return false, 0, deleteError
		}
		// Уменьшаем счетчик у фото, не опуская его ниже 0
		_, decrementError := databaseTransaction.Exec(
			executionContext,
			`UPDATE photos SET likes_count = GREATEST(likes_count - 1, 0) WHERE id = $1`,
			photoID,
		)
		if decrementError != nil {
			return false, 0, decrementError
		}
		isNowLiked = false

	} else if searchError == pgx.ErrNoRows {
		// 2Б. Лайка ещё нет -> добавляем новую запись
		insertLikeQuery := `
			INSERT INTO likes (id, photo_id, user_id, guest_session_id, created_at)
			VALUES ($1, $2, $3, $4, NOW())
		`
		newLikeUUID := uuid.New()
		_, insertError := databaseTransaction.Exec(
			executionContext,
			insertLikeQuery,
			newLikeUUID,
			photoID,
			authenticatedUserID,
			guestSessionID,
		)
		if insertError != nil {
			return false, 0, insertError
		}
		// Увеличиваем счетчик лайков фотографии на 1
		_, incrementError := databaseTransaction.Exec(
			executionContext,
			`UPDATE photos SET likes_count = likes_count + 1 WHERE id = $1`,
			photoID,
		)
		if incrementError != nil {
			return false, 0, incrementError
		}
		isNowLiked = true

	} else {
		return false, 0, searchError
	}

	// 3. Считываем свежее актуальное значение счетчика лайков
	countQueryError := databaseTransaction.QueryRow(
		executionContext,
		`SELECT likes_count FROM photos WHERE id = $1`,
		photoID,
	).Scan(&updatedLikesCount)
	if countQueryError != nil {
		return false, 0, countQueryError
	}

	// 4. Фиксируем транзакцию (Commit)
	commitError := databaseTransaction.Commit(executionContext)
	if commitError != nil {
		return false, 0, commitError
	}

	return isNowLiked, updatedLikesCount, nil
}
