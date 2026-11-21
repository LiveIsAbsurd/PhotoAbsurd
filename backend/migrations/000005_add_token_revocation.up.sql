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

-- Обезвреживаем демо-пароль в существующей базе данных, если он всё ещё установлен
UPDATE users 
SET password_hash = '!LOCKED_DEMO_ACCOUNT_PASSWORD_SET_VIA_ADMIN' 
WHERE username = 'demo_user' 
  AND password_hash = '$2a$12$Z0E0rM1y5H3W8j2L6n1X6.R3mF8wQ7vA9pD5sK2tL1mN4oP7qR9s.';
