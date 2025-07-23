--user table
CREATE TABLE IF NOT EXISTS users(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT NOT NULL,
    email TEXT NOT NULL,
    contact_no TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    archived_at TIMESTAMP
);

--assets table
CREATE TABLE IF NOT EXISTS assets(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brand TEXT NOT NULL,
    model TEXT NOT NULL,
    serial_no TEXT NOT NULL,
    purchase_date timestamp,
    owned_by ownership NOT NULL DEFAULT 'remotestate',
    type asset_type,
    warranty_start TIMESTAMP,
    warranty_expire TIMESTAMP,
    added_by UUID REFERENCES users(id),
    added_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    status asset_status NOT NULL DEFAULT 'available',
    archived_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS asset_assign(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID NOT NULL REFERENCES assets(id),
    employee_id UUID NOT NULL REFERENCES users(id),
    assigned_at TIMESTAMP DEFAULT now(),
    returned_at TIMESTAMP,
    return_reason TEXT,
    archived_at TIMESTAMP
);


CREATE TABLE IF NOT EXISTS user_roles(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role employee_role,
    user_id UUID references users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    created_by uuid NOT NUll,
    last_updated_at TIMESTAMP WITH TIME ZONE,
    archived_at TIMESTAMP WITH TIME ZONE
);


CREATE TABLE IF NOT EXISTS user_type(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type employee_type,
    user_id UUID references users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    created_by uuid NOT NUll,
    last_updated_at TIMESTAMP WITH TIME ZONE,
    archived_at TIMESTAMP WITH TIME ZONE
);