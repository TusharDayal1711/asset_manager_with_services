--laptop
CREATE TABLE IF NOT EXISTS laptop_config(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID UNIQUE REFERENCES assets(id),
    processor TEXT,
    ram TEXT,
    storage TEXT,
    os TEXT,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP,
    archived_at TiMESTAMP
);

--mouse
CREATE TABLE IF NOT EXISTS mouse_config(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID UNIQUE REFERENCES assets(id),
    DPI TEXT,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP,
    archived_at TiMESTAMP
);

--monitor
CREATE TABLE IF NOT EXISTS monitor_config(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID UNIQUE REFERENCES assets(id),
    display TEXT, -- lcd, led, oled
    resolution TEXT,
    port TEXT,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP,
    archived_at TiMESTAMP
);

--storage
CREATE TABLE IF NOT EXISTS hard_disk_config(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID UNIQUE REFERENCES assets(id),
    type TEXT DEFAULT 'HDD',
    storage TEXT,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP,
    archived_at TiMESTAMP
);

--Pendrive
CREATE TABLE IF NOT EXISTS pendrive_config(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID UNIQUE REFERENCES assets(id),
    version TEXT DEFAULT '2.0',
    storage TEXT,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP,
    archived_at TiMESTAMP
);

--mobile
CREATE TABLE IF NOT EXISTS mobile_config(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID UNIQUE REFERENCES assets(id),
    processor TEXT,
    ram TEXT,
    storage TEXT,
    os TEXT,
    imei_1 TEXT NOT NULL,
    imei_2 TEXT,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP,
    archived_at TiMESTAMP
);

--sim
CREATE TABLE IF NOT EXISTS sim_config(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID UNIQUE REFERENCES assets(id),
    number TEXT,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP,
    archived_at TiMESTAMP
);

--accessories
CREATE TABLE IF NOT EXISTS accessories_config(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID UNIQUE REFERENCES assets(id),
    type TEXT,
    additional_info TEXT,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP,
    archived_at TiMESTAMP
);


