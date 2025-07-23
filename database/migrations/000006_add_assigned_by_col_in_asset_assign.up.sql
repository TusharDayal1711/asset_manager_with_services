ALTER TABLE asset_assign
ADD assigned_by UUID REFERENCES users(id)