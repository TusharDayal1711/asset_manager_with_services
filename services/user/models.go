package userservice

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
	"time"
)

type PublicUserReq struct {
	Email string `json:"email" validate:"required,email"`
}

type ManagerRegisterReq struct {
	Username  string `json:"username" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	ContactNo string `json:"contact_no" validate:"required"`
	Type      string `json:"type" validate:"required,oneof=full_time intern freelancer"`
}

type EmployeeResponseModel struct {
	ID             string         `json:"id" db:"id"`
	Username       string         `json:"username" db:"username"`
	Email          string         `json:"email" db:"email"`
	ContactNo      *string        `json:"contact_no" db:"contact_no"`
	EmployeeType   string         `json:"type" db:"employee_type"`
	AssignedAssets pq.StringArray `json:"assigned_assets" db:"assigned_assets"`
}

type UpdateUserRoleReq struct {
	UserID string `json:"user_id" validate:"required,uuid"`
	Role   string `json:"role" validate:"required,oneof=admin asset_manager employee_manager user"`
}

type UpdateEmployeeReq struct {
	UserID    uuid.UUID `json:"user_id" validate:"required"`
	Username  string    `json:"username,omitempty"`
	Email     string    `json:"email,omitempty"`
	ContactNo string    `json:"contact_no,omitempty"`
}

type UserTimelineRes struct {
	AssetID      string     `json:"asset_id" db:"asset_id"`
	Brand        string     `json:"brand" db:"brand"`
	Model        string     `json:"model" db:"model"`
	SerialNo     string     `json:"serial_no" db:"serial_no"`
	AssignedAt   time.Time  `json:"assigned_at" db:"assigned_at"`
	ReturnedAt   *time.Time `json:"returned_at,omitempty" db:"returned_at"`
	ReturnReason *string    `json:"return_reason,omitempty" db:"return_reason"`
}

// /search using filters
type EmployeeFilter struct {
	IsSearchText bool
	SearchText   string
	Type         []string
	Role         []string
	AssetStatus  []string
	Limit        int
	Offset       int
}

// user dashboard
type UserDashboardRes struct {
	ID             string         `json:"id" db:"id"`
	Username       string         `json:"username" db:"username"`
	Email          string         `json:"email" db:"email"`
	ContactNo      *string        `json:"contact_no,omitempty" db:"contact_no"`
	Type           *string        `json:"type,omitempty" db:"type"`
	Roles          []string       `json:"roles"`
	AssignedAssets []AssetDetails `json:"assigned_assets"`
}
type AssetDetails struct {
	ID         uuid.UUID `json:"id" db:"id"`
	Brand      string    `json:"brand" db:"brand"`
	Model      string    `json:"model" db:"model"`
	SerialNo   string    `json:"serial_no" db:"serial_no"`
	Type       string    `json:"type" db:"type"`
	Status     string    `json:"status" db:"status"`
	OwnedBy    string    `json:"owned_by" db:"owned_by"`
	AssignedAt time.Time `json:"assigned_at" db:"assigned_at"`
}
