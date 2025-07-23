package dbhelper

import (
	"asset/database"
	"asset/models"
	"fmt"
	"github.com/lib/pq"
)

func SearchAssetsWithFilter(filter models.AssetFilter) ([]models.AssetWithConfigRes, error) {
	tx, err := database.DB.Beginx()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil || err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	args := []interface{}{
		!filter.IsSearchText,
		filter.SearchText,
		pq.Array(filter.Status),
		pq.Array(filter.OwnedBy),
		pq.Array(filter.Type),
		filter.Limit,
		filter.Offset,
	}

	query := `
		SELECT id, brand, model, serial_no, type, owned_by, status, purchase_date, warranty_start, warranty_expire
		FROM assets
		WHERE archived_at IS NULL
		AND (
			$1 OR (
				brand ILIKE $2 OR 
				model ILIKE $2 OR 
				serial_no ILIKE $2
			)
		)
		AND status = ANY($3)
		AND owned_by = ANY($4)
		AND type = ANY($5)
		ORDER BY added_at DESC
		LIMIT $6 OFFSET $7
	`

	var assets []models.AssetWithConfigRes
	err = tx.Select(&assets, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch assets: %w", err)
	}

	for i, asset := range assets {
		var config interface{}
		switch asset.Type {
		case "laptop":
			var configTemp models.Laptop_config_res
			err = tx.Get(&configTemp, `SELECT processor, ram, os FROM laptop_config WHERE asset_id = $1`, asset.ID)
			config = configTemp
		case "mouse":
			var configTemp models.Mouse_config_res
			err = tx.Get(&configTemp, `SELECT dpi FROM mouse_config WHERE asset_id = $1`, asset.ID)
			config = configTemp
		case "monitor":
			var configTemp models.Monitor_config_res
			err = tx.Get(&configTemp, `SELECT display, resolution, port FROM monitor_config WHERE asset_id = $1`, asset.ID)
			config = configTemp
		case "mobile":
			var configTemp models.Mobile_config_res
			err = tx.Get(&configTemp, `SELECT processor, ram, os, imei_1, imei_2 FROM mobile_config WHERE asset_id = $1`, asset.ID)
			config = configTemp
		case "hard_disk":
			var configTemp models.Hard_disk_config_res
			err = tx.Get(&configTemp, `SELECT type, storage FROM hard_disk_config WHERE asset_id = $1`, asset.ID)
			config = configTemp
		case "pen_drive":
			var configTemp models.Pen_drive_config_res
			err = tx.Get(&configTemp, `SELECT version, storage FROM pendrive_config WHERE asset_id = $1`, asset.ID)
			config = configTemp
		case "sim":
			var configTemp models.Sim_config_res
			err = tx.Get(&configTemp, `SELECT number FROM sim_config WHERE asset_id = $1`, asset.ID)
			config = configTemp
		case "accessory":
			var configTemp models.Accessories_config_res
			err = tx.Get(&configTemp, `SELECT type, additional_info FROM accessories_config WHERE asset_id = $1`, asset.ID)
			config = configTemp
		}
		if err != nil && err.Error() != "sql: no rows in result set" {
			return nil, fmt.Errorf("failed to fetch config for asset %s: %w", asset.ID, err)
		}

		assets[i].Config = config
	}

	return assets, nil
}
