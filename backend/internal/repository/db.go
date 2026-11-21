package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB инкапсулирует пул постоянных соединений к базе данных PostgreSQL.
// Использование pgxpool вместо одиночного соединения позволяет обрабатывать
// тысячи одновременных запросов от разных пользователей без накладных расходов
// на постоянное открытие новых TCP-сокетов.
type DB struct {
	Pool *pgxpool.Pool
}

// NewDB создаёт и валидирует пул подключений к PostgreSQL.
// Функция выполняет несколько попыток подключения (retry logic) на случай,
// если сервер базы данных в Docker ещё не завершил инициализацию.
func NewDB(parentContext context.Context, databaseConnectionURL string) (*DB, error) {
	poolConfiguration, parseError := pgxpool.ParseConfig(databaseConnectionURL)
	if parseError != nil {
		return nil, fmt.Errorf("не удалось разобрать строку подключения: %w", parseError)
	}

	// Настройка параметров пула соединений:
	poolConfiguration.MaxConns = 25                      // Максимум 25 одновременных подключений к БД
	poolConfiguration.MinConns = 5                       // Минимум 5 всегда прогретых подключений в резерве
	poolConfiguration.MaxConnLifetime = 30 * time.Minute // Пересоздавать соединение раз в 30 минут
	poolConfiguration.MaxConnIdleTime = 5 * time.Minute  // Закрывать неиспользуемые соединения старше 5 минут

	var connectionPool *pgxpool.Pool
	var connectionError error

	// Цикл из 10 попыток подключения с паузой 2 секунды
	for attemptNumber := 1; attemptNumber <= 10; attemptNumber++ {
		connectionPool, connectionError = pgxpool.NewWithConfig(parentContext, poolConfiguration)
		if connectionError == nil {
			// Проверяем реальную связь с БД с помощью Ping
			pingError := connectionPool.Ping(parentContext)
			if pingError == nil {
				databaseInstance := &DB{Pool: connectionPool}
				// Выполняем автоматическое обновление структуры таблиц
				_ = databaseInstance.AutoMigrate(parentContext)
				return databaseInstance, nil
			}
			connectionPool.Close()
			connectionError = pingError
		}
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("не удалось подключиться к PostgreSQL после 10 попыток: %w", connectionError)
}

// AutoMigrate выполняет необходимые изменения структуры БД при запуске приложения:
// делает поле email необязательным, добавляет колонку роли, назначает права администратора
// и создаёт недостающие индексы производительности.
func (database *DB) AutoMigrate(executionContext context.Context) error {
	migrationQueries := `
		ALTER TABLE users ALTER COLUMN email DROP NOT NULL;
		ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'user';
		ALTER TABLE users ADD COLUMN IF NOT EXISTS token_version INT NOT NULL DEFAULT 1;
		UPDATE users SET role = 'admin' WHERE username = 'test_curator';
		CREATE INDEX IF NOT EXISTS idx_likes_user_created ON likes(user_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_photos_public_created ON photos(created_at DESC) WHERE is_public = TRUE;
		CREATE TABLE IF NOT EXISTS revoked_tokens (
			id VARCHAR(64) PRIMARY KEY,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_revoked_tokens_expires ON revoked_tokens(expires_at);
		UPDATE users 
		SET password_hash = '!LOCKED_DEMO_ACCOUNT_PASSWORD_SET_VIA_ADMIN' 
		WHERE username = 'demo_user' 
		  AND password_hash = '$2a$12$Z0E0rM1y5H3W8j2L6n1X6.R3mF8wQ7vA9pD5sK2tL1mN4oP7qR9s.';
	`
	_, migrationError := database.Pool.Exec(executionContext, migrationQueries)
	return migrationError
}

// Close закрывает все открытые соединения пула с базой данных.
func (database *DB) Close() {
	if database.Pool != nil {
		database.Pool.Close()
	}
}
