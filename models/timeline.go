package models

import (
	"github.com/google/uuid"
	"time"
)

type UserTimelineRes struct {
	AssetID      string     `json:"asset_id" db:"asset_id"`
	Brand        string     `json:"brand" db:"brand"`
	Model        string     `json:"model" db:"model"`
	SerialNo     string     `json:"serial_no" db:"serial_no"`
	AssignedAt   time.Time  `json:"assigned_at" db:"assigned_at"`
	ReturnedAt   *time.Time `json:"returned_at,omitempty" db:"returned_at"`
	ReturnReason *string    `json:"return_reason,omitempty" db:"return_reason"`
}

type AssetTimelineEvent struct {
	EventType string     `json:"event_type" db:"event_type"`
	StartTime time.Time  `json:"start_time" db:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty" db:"end_time"`
	Details   string     `json:"details,omitempty" db:"details"`
	AssetID   uuid.UUID  `json:"asset_id" db:"asset_id"`
}
