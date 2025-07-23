CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email
    ON users(email)
    WHERE archived_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_contact_no
    ON users(contact_no)
    WHERE contact_no IS NOT NULL AND archived_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_assets_serial_no
    ON assets(serial_no)
    WHERE archived_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_asset_assignment
    ON asset_assign(asset_id)
    WHERE returned_at IS NULL AND archived_at IS NULL;




