-- 000004_add_indexes.up.sql

-- Index to optimize querying liked photos by user ordered by creation time
CREATE INDEX IF NOT EXISTS idx_likes_user_created ON likes(user_id, created_at DESC);

-- Partial index for public photos feed ordered by creation time
CREATE INDEX IF NOT EXISTS idx_photos_public_created ON photos(created_at DESC) WHERE is_public = TRUE;
