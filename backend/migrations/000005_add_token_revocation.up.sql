-- 000005_add_token_revocation.up.sql

-- Таблица для черного списка отозванных JWT-токенов (при выходе из системы)
CREATE TABLE IF NOT EXISTS revoked_tokens (
    id VARCHAR(64) PRIMARY KEY,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_revoked_tokens_expires ON revoked_tokens(expires_at);

-- Версия токена пользователя: при смене пароля инкрементируется,
-- что делает недействительными все ранее выпущенные JWT пользователя.
ALTER TABLE users ADD COLUMN IF NOT EXISTS token_version INT NOT NULL DEFAULT 1;

-- Удаляем тестового пользователя, если он остался в базе
DELETE FROM users WHERE username = 'demo_user';
