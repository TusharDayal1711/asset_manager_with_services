ALTER TABLE user_roles ALTER COLUMN role SET NOT NULL;
ALTER TABLE user_type ALTER COLUMN type SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_roles_unique_active
    ON user_roles(user_id, role)
    WHERE archived_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_type_unique_active
    ON user_type(user_id, type)
    WHERE archived_at IS NULL;
