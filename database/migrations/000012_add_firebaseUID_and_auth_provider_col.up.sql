ALTER TABLE users
ADD COLUMN firebase_uid TEXT UNIQUE,
ADD COLUMN auth_provider TEXT NOT NULL DEFAULT 'custom';
