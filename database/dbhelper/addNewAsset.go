package dbhelper

import (
	"asset/models"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func AddAsset(tx *sqlx.Tx, assetReq models.AddAssetWithConfigReq, addedBy uuid.UUID) (uuid.UUID, error) {
	var assetID uuid.UUID
	err := tx.Get(&assetID, `
		INSERT INTO assets (
			brand, model, serial_no, purchase_date, 
			owned_by, type, warranty_start, warranty_expire, 
			added_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`,
		assetReq.Brand, assetReq.Model, assetReq.SerialNo, assetReq.PurchaseDate,
		assetReq.OwnedBy, assetReq.Type, assetReq.WarrantyStart, assetReq.WarrantyExpire,
		addedBy)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert asset: %w", err)
	}
	return assetID, nil
}

func AddLaptopConfig(tx *sqlx.Tx, config models.Laptop_config_req, assetID uuid.UUID) error {
	_, err := tx.Exec(`
		INSERT INTO laptop_config (asset_id, processor, ram, os)
		VALUES ($1, $2, $3, $4)`,
		assetID, config.Processor, config.Ram, config.Os)
	if err != nil {
		return fmt.Errorf("failed to insert laptop config: %w", err)
	}
	return nil
}

func AddMouseConfig(tx *sqlx.Tx, config models.Mouse_config_req, assetID uuid.UUID) error {
	_, err := tx.Exec(`
		INSERT INTO mouse_config (asset_id, dpi)
		VALUES ($1, $2)`,
		assetID, config.DPI)
	if err != nil {
		return fmt.Errorf("failed to insert mouse config: %w", err)
	}
	return nil
}

func AddMonitorConfig(tx *sqlx.Tx, config models.Monitor_config_req, assetID uuid.UUID) error {
	_, err := tx.Exec(`
		INSERT INTO monitor_config (asset_id, display, resolution, port)
		VALUES ($1, $2, $3, $4)`,
		assetID, config.Display, config.Resolution, config.Port)
	if err != nil {
		return fmt.Errorf("failed to insert monitor config: %w", err)
	}
	return nil
}

func AddHardDiskConfig(tx *sqlx.Tx, config models.Hard_disk_config_req, assetID uuid.UUID) error {
	_, err := tx.Exec(`
		INSERT INTO hard_disk_config (asset_id, type, storage)
		VALUES ($1, $2, $3)`,
		assetID, config.Type, config.Storage)
	if err != nil {
		return fmt.Errorf("failed to insert hard disk config: %w", err)
	}
	return nil
}

func AddPenDriveConfig(tx *sqlx.Tx, config models.Pen_drive_config_req, assetID uuid.UUID) error {
	_, err := tx.Exec(`
		INSERT INTO pendrive_config (asset_id, version, storage)
		VALUES ($1, $2, $3)`,
		assetID, config.Version, config.Storage)
	if err != nil {
		return fmt.Errorf("failed to insert pen drive config: %w", err)
	}
	return nil
}

func AddMobileConfig(tx *sqlx.Tx, config models.Mobile_config_req, assetID uuid.UUID) error {
	_, err := tx.Exec(`
		INSERT INTO mobile_config (asset_id, processor, ram, os, imei_1, imei_2)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, assetID, config.Processor, config.Ram, config.Os, config.IMEI1, config.IMEI2)
	if err != nil {
		return fmt.Errorf("failed to insert mobile config: %w", err)
	}
	return nil
}

func AddSimConfig(tx *sqlx.Tx, config models.Sim_config_req, assetID uuid.UUID) error {
	_, err := tx.Exec(`
		INSERT INTO sim_config (asset_id, number)
		VALUES ($1, $2)`,
		assetID, config.Number)
	if err != nil {
		return fmt.Errorf("failed to insert sim config: %w", err)
	}
	return nil
}

func AddAccessoryConfig(tx *sqlx.Tx, config models.Accessories_config_req, assetID uuid.UUID) error {
	_, err := tx.Exec(`
		INSERT INTO accessories_config (asset_id, type, additional_info)
		VALUES ($1, $2, $3)`,
		assetID, config.Type, config.AdditionalInfo)
	if err != nil {
		return fmt.Errorf("failed to insert accessory config: %w", err)
	}
	return nil
}
