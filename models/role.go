package models

type Role string

const (
	AdminRole          Role = "admin"
	EmployeeMangerRole Role = "employee_manager"
	AssetManagerRole   Role = "asset_manager"
)
