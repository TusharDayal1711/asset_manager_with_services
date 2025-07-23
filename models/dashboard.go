package models

import (
	"github.com/google/uuid"
	"time"
)

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
