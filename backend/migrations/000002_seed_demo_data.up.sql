-- 000002_seed_demo_data.up.sql
-- Тестовые и демонстрационные пользователи отключены для продакшн-окружения.
DELETE FROM users WHERE username = 'demo_user';
