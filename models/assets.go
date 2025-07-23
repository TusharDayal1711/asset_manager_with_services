package models

import (
	"encoding/json"
	"github.com/google/uuid"
	"time"
)

type AssetReq struct {
	Brand          string    `json:"brand" validate:"required"`
	Model          string    `json:"model" validate:"required"`
	SerialNo       string    `json:"serial_no" validate:"required"`
	PurchaseDate   time.Time `json:"purchase_date" validate:"required"`
	OwnedBy        string    `json:"owned_by" validate:"required"`
	Type           string    `json:"type" validate:"required"`
	WarrantyStart  time.Time `json:"warranty" validate:"required"`
	WarrantyExpire time.Time `json:"warranty_expire" validate:"required,gtfield=WarrantyStart"`
}

// Assets request model
type Laptop_config_req struct {
	Processor string `json:"processor"`
	Ram       string `json:"ram"`
	Os        string `json:"os"`
}
type Mouse_config_req struct {
	DPI string `json:"dpi"`
}

type Monitor_config_req struct {
	Display    string `json:"display"`
	Resolution string `json:"resolution"`
	Port       string `json:"port"`
}

type Hard_disk_config_req struct {
	Type    string `json:"type"`
	Storage string `json:"storage"`
}

type Pen_drive_config_req struct {
	Version string `json:"version"`
	Storage string `json:"storage"`
}

type Mobile_config_req struct {
	Processor string `json:"processor"`
	Ram       string `json:"ram"`
	Os        string `json:"os"`
	IMEI1     string `json:"imei"`
	IMEI2     string `json:"ime2"`
}

type Sim_config_req struct {
	Number int `json:"number"`
}

type Accessories_config_req struct {
	Type           string `json:"type"`
	AdditionalInfo string `json:"additional_info"`
}

type AddAssetWithConfigReq struct {
	AssetReq
	Config json.RawMessage `json:"config" `
}

type AssetAssignReq struct {
	UserID  string `json:"user_id"`
	AssetID string `json:"asset_id"`
}

type AssetRes struct {
	ID       string `json:"id" db:"id"`
	Brand    string `json:"brand" db:"brand"`
	Model    string `json:"model" db:"model"`
	SerialNo string `json:"serial_no" db:"serial_no"`
	Type     string `json:"type" db:"type"`
	OwnedBy  string `json:"owned_by" db:"owned_by"`
	Status   string `json:"status" db:"status"`
}

type AssetReturnReq struct {
	AssetID      string `json:"asset_id" validate:"required,uuid"`
	EmployeeID   string `json:"employee_id" validate:"required,uuid"`
	ReturnReason string `json:"return_reason"`
}

type AssetServiceReq struct {
	AssetID uuid.UUID `json:"asset_id" validate:"required"`
	Reason  string    `json:"reason" validate:"required"`
}

type UpdateAssetReq struct {
	ID             uuid.UUID       `json:"id" validate:"required"`
	Brand          string          `json:"brand,omitempty"`
	Model          string          `json:"model,omitempty"`
	SerialNo       string          `json:"serial_no,omitempty"`
	PurchaseDate   *time.Time      `json:"purchase_date,omitempty"`
	OwnedBy        string          `json:"owned_by,omitempty"`
	WarrantyStart  *time.Time      `json:"warranty_start,omitempty"`
	WarrantyExpire *time.Time      `json:"warranty_expire,omitempty"`
	Type           string          `json:"type,omitempty"` // For validation only
	Config         json.RawMessage `json:"config,omitempty"`
}
