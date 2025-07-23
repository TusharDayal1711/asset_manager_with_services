package models

import "time"

type Laptop_config_res struct {
	Processor string `json:"processor" db:"processor"`
	Ram       string `json:"ram" db:"ram"`
	Os        string `json:"os" db:"os"`
}

type Mouse_config_res struct {
	DPI string `json:"dpi" db:"dpi"`
}

type Monitor_config_res struct {
	Display    string `json:"display" db:"display"`
	Resolution string `json:"resolution" db:"resolution"`
	Port       string `json:"port" db:"port"`
}

type Hard_disk_config_res struct {
	Type    string `json:"type" db:"type"`
	Storage string `json:"storage" db:"storage"`
}

type Pen_drive_config_res struct {
	Version string `json:"version" db:"version"`
	Storage string `json:"storage" db:"storage"`
}

type Mobile_config_res struct {
	Processor string `json:"processor" db:"processor"`
	Ram       string `json:"ram" db:"ram"`
	Os        string `json:"os" db:"os"`
	IMEI1     string `json:"imei_1" db:"imei_1"`
	IMEI2     string `json:"imei_2" db:"imei_2"`
}

type Sim_config_res struct {
	Number int `json:"number" db:"number"`
}

type Accessories_config_res struct {
	Type           string `json:"type" db:"type"`
	AdditionalInfo string `json:"additional_info" db:"additional_info"`
}

type AssetWithConfigRes struct {
	ID            string      `json:"id" db:"id"`
	Brand         string      `json:"brand" db:"brand"`
	Model         string      `json:"model" db:"model"`
	SerialNo      string      `json:"serial_no" db:"serial_no"`
	Type          string      `json:"type" db:"type"`
	OwnedBy       string      `json:"owned_by" db:"owned_by"`
	Status        string      `json:"status" db:"status"`
	PurchaseDate  time.Time   `json:"purchase_date" db:"purchase_date"`
	WarrantyStart time.Time   `json:"warranty_start" db:"warranty_start"`
	WarrantyEnd   time.Time   `json:"warranty_expire" db:"warranty_expire"`
	Config        interface{} `json:"config"`
}
