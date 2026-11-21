-- 000002_seed_demo_data.up.sql

-- Учётная запись demo_user инициализируется с заблокированным паролем ради безопасности.
-- Пароль может быть назначен администратором через панель управления или CLI.
INSERT INTO users (id, username, email, password_hash, avatar_url, created_at, updated_at)
VALUES (
    'a0000000-0000-0000-0000-000000000001',
    'demo_user',
    'demo@example.com',
    '!LOCKED_DEMO_ACCOUNT_PASSWORD_SET_VIA_ADMIN',
    '',
    NOW(),
    NOW()
) ON CONFLICT (email) DO NOTHING;

INSERT INTO albums (id, user_id, title, description, is_public, created_at, updated_at)
VALUES (
    'b0000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000001',
    'Путешествия и Природа',
    'Демонстрационный альбом с красивыми фотографиями',
    TRUE,
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;
