--enums
CREATE TYPE employee_role AS ENUM (
    'admin',
    'asset_manager',
    'inventory_manager',
    'employee'
);

CREATE TYPE employee_type AS ENUM (
    'full_time',
    'intern',
    'freelancer'
);

CREATE TYPE asset_type AS ENUM (
    'laptop',
    'mouse',
    'monitor',
    'hard_disk',
    'pen_drive',
    'mobile',
    'sim',
    'accessory'
);

CREATE TYPE asset_status AS ENUM (
    'available',
    'assigned',
    'waiting for repair',
    'sent_for_service',
    'damaged'
);

CREATE TYPE ownership AS ENUM (
    'remotestate',
    'client'
);



