package dbhelper

import (
	"asset/database"
	"asset/models"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

func UpdateAssetWithConfig(req models.UpdateAssetReq) error {
	tx, err := database.DB.Beginx()
	if err != nil {
		log.Println("transaction failed", err)
		return err
	}
	defer func() {
		if p := recover(); p != nil || err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	updateFields := []string{}
	args := []interface{}{}
	argPos := 1

	if req.Brand != "" {
		updateFields = append(updateFields, fmt.Sprintf("brand = $%d", argPos))
		args = append(args, req.Brand)
		argPos++
	}
	if req.Model != "" {
		updateFields = append(updateFields, fmt.Sprintf("model = $%d", argPos))
		args = append(args, req.Model)
		argPos++
	}
	if req.SerialNo != "" {
		updateFields = append(updateFields, fmt.Sprintf("serial_no = $%d", argPos))
		args = append(args, req.SerialNo)
		argPos++
	}
	if req.PurchaseDate != nil {
		updateFields = append(updateFields, fmt.Sprintf("purchase_date = $%d", argPos))
		args = append(args, *req.PurchaseDate)
		argPos++
	}
	if req.OwnedBy != "" {
		updateFields = append(updateFields, fmt.Sprintf("owned_by = $%d", argPos))
		args = append(args, req.OwnedBy)
		argPos++
	}
	if req.WarrantyStart != nil {
		updateFields = append(updateFields, fmt.Sprintf("warranty_start = $%d", argPos))
		args = append(args, *req.WarrantyStart)
		argPos++
	}
	if req.WarrantyExpire != nil {
		updateFields = append(updateFields, fmt.Sprintf("warranty_expire = $%d", argPos))
		args = append(args, *req.WarrantyExpire)
		argPos++
	}

	if len(updateFields) > 0 {
		query := fmt.Sprintf("UPDATE assets SET %s WHERE id = $%d AND archived_at IS NULL", strings.Join(updateFields, ", "), argPos)
		args = append(args, req.ID)

		_, err := tx.Exec(query, args...)
		if err != nil {
			return fmt.Errorf("failed to update asset: %w", err)
		}
	}

	if req.Config != nil && req.Type != "" {
		switch req.Type {
		case "laptop":
			var cfg models.Laptop_config_req
			if err := json.Unmarshal(req.Config, &cfg); err != nil {
				return fmt.Errorf("invalid laptop config: %w", err)
			}
			_, err = tx.Exec(`UPDATE laptop_config SET processor = $1, ram = $2, os = $3 WHERE asset_id = $4`,
				cfg.Processor, cfg.Ram, cfg.Os, req.ID)
		case "mouse":
			var cfg models.Mouse_config_req
			if err := json.Unmarshal(req.Config, &cfg); err != nil {
				return fmt.Errorf("invalid mouse config: %w", err)
			}
			_, err = tx.Exec(`UPDATE mouse_config SET dpi = $1 WHERE asset_id = $2`, cfg.DPI, req.ID)
		case "monitor":
			var cfg models.Monitor_config_req
			if err := json.Unmarshal(req.Config, &cfg); err != nil {
				return fmt.Errorf("invalid monitor config: %w", err)
			}
			_, err = tx.Exec(`UPDATE monitor_config SET display = $1, resolution = $2, port = $3 WHERE asset_id = $4`,
				cfg.Display, cfg.Resolution, cfg.Port, req.ID)
		case "mobile":
			var cfg models.Mobile_config_req
			if err := json.Unmarshal(req.Config, &cfg); err != nil {
				return fmt.Errorf("invalid mobile config: %w", err)
			}
			_, err = tx.Exec(`UPDATE mobile_config SET processor = $1, ram = $2, os = $3, imei_1 = $4, imei_2 = $5 WHERE asset_id = $6`,
				cfg.Processor, cfg.Ram, cfg.Os, cfg.IMEI1, cfg.IMEI2, req.ID)
		case "hard_disk":
			var cfg models.Hard_disk_config_req
			if err := json.Unmarshal(req.Config, &cfg); err != nil {
				return fmt.Errorf("invalid hard disk config: %w", err)
			}
			_, err = tx.Exec(`UPDATE hard_disk_config SET type = $1, storage = $2 WHERE asset_id = $3`,
				cfg.Type, cfg.Storage, req.ID)
		case "pen_drive":
			var cfg models.Pen_drive_config_req
			if err := json.Unmarshal(req.Config, &cfg); err != nil {
				return fmt.Errorf("invalid pen drive config: %w", err)
			}
			_, err = tx.Exec(`UPDATE pendrive_config SET version = $1, storage = $2 WHERE asset_id = $3`,
				cfg.Version, cfg.Storage, req.ID)
		case "sim":
			var cfg models.Sim_config_req
			if err := json.Unmarshal(req.Config, &cfg); err != nil {
				return fmt.Errorf("invalid sim config: %w", err)
			}
			_, err = tx.Exec(`UPDATE sim_config SET number = $1 WHERE asset_id = $2`,
				cfg.Number, req.ID)
		case "accessory":
			var cfg models.Accessories_config_req
			if err := json.Unmarshal(req.Config, &cfg); err != nil {
				return fmt.Errorf("invalid accessory config: %w", err)
			}
			_, err = tx.Exec(`UPDATE accessories_config SET type = $1, additional_info = $2 WHERE asset_id = $3`,
				cfg.Type, cfg.AdditionalInfo, req.ID)
		default:
			return fmt.Errorf("unsupported asset type for config update")
		}

		if err != nil {
			return fmt.Errorf("failed to update config: %w", err)
		}
	}

	return nil
}
