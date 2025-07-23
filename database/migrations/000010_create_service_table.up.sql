CREATE TABLE IF NOT EXISTS asset_service (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        asset_id UUID NOT NULL REFERENCES assets(id),
        service_start TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
        service_end TIMESTAMP WITH TIME ZONE,
        reason TEXT NOT NULL,
        created_by UUID NOT NULL REFERENCES users(id),
        created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
        archived_at TIMESTAMP WITH TIME ZONE
);


CREATE UNIQUE INDEX IF NOT EXISTS idx_asset_id_asset_service
    ON asset_service(asset_id)
    WHERE archived_at IS NULL AND service_end IS NULL;
