package asset

import (
	"asset/models"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"log"
)

type AssetRepository interface {
	AddAsset(ctx context.Context, tx *sqlx.Tx, req models.AddAssetWithConfigReq, addedBy uuid.UUID) (uuid.UUID, error)

	AddLaptopConfig(ctx context.Context, tx *sqlx.Tx, cfg models.Laptop_config_req, assetID uuid.UUID) error
	AddMouseConfig(ctx context.Context, tx *sqlx.Tx, cfg models.Mouse_config_req, assetID uuid.UUID) error
	AddMonitorConfig(ctx context.Context, tx *sqlx.Tx, cfg models.Monitor_config_req, assetID uuid.UUID) error
	AddHardDiskConfig(ctx context.Context, tx *sqlx.Tx, cfg models.Hard_disk_config_req, assetID uuid.UUID) error
	AddPenDriveConfig(ctx context.Context, tx *sqlx.Tx, cfg models.Pen_drive_config_req, assetID uuid.UUID) error
	AddMobileConfig(ctx context.Context, tx *sqlx.Tx, cfg models.Mobile_config_req, assetID uuid.UUID) error
	AddSimConfig(ctx context.Context, tx *sqlx.Tx, cfg models.Sim_config_req, assetID uuid.UUID) error
	AddAccessoryConfig(ctx context.Context, tx *sqlx.Tx, cfg models.Accessories_config_req, assetID uuid.UUID) error
	AssignAssetByID(ctx context.Context, tx *sqlx.Tx, assetID, employeeID, managerID uuid.UUID) error
	DeleteAssetByID(ctx context.Context, assetID uuid.UUID) error
	SearchAssetsWithFilter(ctx context.Context, filter models.AssetFilter) ([]models.AssetWithConfigRes, error)
	GetAssetTimeline(ctx context.Context, assetID uuid.UUID) ([]models.AssetTimelineEvent, error)
	RecivedAssetFromService(ctx context.Context, assetID uuid.UUID) error
	RetrieveAsset(ctx context.Context, tx *sqlx.Tx, assetID, employeeID uuid.UUID, reason string) error
	SendAssetForService(ctx context.Context, req models.AssetServiceReq, managerID uuid.UUID) error
	UpdateAssetWithConfig(ctx context.Context, req models.UpdateAssetReq) error
}

type PostgresAssetRepository struct {
	DB *sqlx.DB
}

func NewAssetRepository(db *sqlx.DB) AssetRepository {
	return &PostgresAssetRepository{DB: db}
}

func (r *PostgresAssetRepository) AddAsset(ctx context.Context, tx *sqlx.Tx, assetReq models.AddAssetWithConfigReq, addedBy uuid.UUID) (uuid.UUID, error) {
	var assetID uuid.UUID
	err := tx.GetContext(ctx, &assetID, `
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

func (r *PostgresAssetRepository) AddLaptopConfig(ctx context.Context, tx *sqlx.Tx, config models.Laptop_config_req, assetID uuid.UUID) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO laptop_config (asset_id, processor, ram, os)
		VALUES ($1, $2, $3, $4)`,
		assetID, config.Processor, config.Ram, config.Os)
	if err != nil {
		return fmt.Errorf("failed to insert laptop config: %w", err)
	}
	return nil
}

func (r *PostgresAssetRepository) AddMouseConfig(ctx context.Context, tx *sqlx.Tx, config models.Mouse_config_req, assetID uuid.UUID) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO mouse_config (asset_id, dpi)
		VALUES ($1, $2)`,
		assetID, config.DPI)
	if err != nil {
		return fmt.Errorf("failed to insert mouse config: %w", err)
	}
	return nil
}

func (r *PostgresAssetRepository) AddMonitorConfig(ctx context.Context, tx *sqlx.Tx, config models.Monitor_config_req, assetID uuid.UUID) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO monitor_config (asset_id, display, resolution, port)
		VALUES ($1, $2, $3, $4)`,
		assetID, config.Display, config.Resolution, config.Port)
	if err != nil {
		return fmt.Errorf("failed to insert monitor config: %w", err)
	}
	return nil
}

func (r *PostgresAssetRepository) AddHardDiskConfig(ctx context.Context, tx *sqlx.Tx, config models.Hard_disk_config_req, assetID uuid.UUID) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO hard_disk_config (asset_id, type, storage)
		VALUES ($1, $2, $3)`,
		assetID, config.Type, config.Storage)
	if err != nil {
		return fmt.Errorf("failed to insert hard disk config: %w", err)
	}
	return nil
}

func (r *PostgresAssetRepository) AddPenDriveConfig(ctx context.Context, tx *sqlx.Tx, config models.Pen_drive_config_req, assetID uuid.UUID) error {

	_, err := tx.ExecContext(ctx, `
		INSERT INTO pendrive_config (asset_id, version, storage)
		VALUES ($1, $2, $3)`,
		assetID, config.Version, config.Storage)
	if err != nil {
		return fmt.Errorf("failed to insert pen drive config: %w", err)
	}
	return nil
}

func (r *PostgresAssetRepository) AddMobileConfig(ctx context.Context, tx *sqlx.Tx, config models.Mobile_config_req, assetID uuid.UUID) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO mobile_config (asset_id, processor, ram, os, imei_1, imei_2)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, assetID, config.Processor, config.Ram, config.Os, config.IMEI1, config.IMEI2)
	if err != nil {
		return fmt.Errorf("failed to insert mobile config: %w", err)
	}
	return nil
}

func (r *PostgresAssetRepository) AddSimConfig(ctx context.Context, tx *sqlx.Tx, config models.Sim_config_req, assetID uuid.UUID) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO sim_config (asset_id, number)
		VALUES ($1, $2)`,
		assetID, config.Number)
	if err != nil {
		return fmt.Errorf("failed to insert sim config: %w", err)
	}
	return nil
}

func (r *PostgresAssetRepository) AddAccessoryConfig(ctx context.Context, tx *sqlx.Tx, config models.Accessories_config_req, assetID uuid.UUID) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO accessories_config (asset_id, type, additional_info)
		VALUES ($1, $2, $3)`,
		assetID, config.Type, config.AdditionalInfo)
	if err != nil {
		return fmt.Errorf("failed to insert accessory config: %w", err)
	}
	return nil
}

func (r *PostgresAssetRepository) AssignAssetByID(ctx context.Context, tx *sqlx.Tx, assetID uuid.UUID, employeeID uuid.UUID, assignedBy uuid.UUID) error {
	var exists int
	err := tx.GetContext(ctx, &exists, `
		SELECT 1 FROM asset_assign 
		WHERE asset_id = $1 AND returned_at IS NULL AND archived_at IS NULL
		LIMIT 1
	`, assetID)

	if err != nil {
		if err == sql.ErrNoRows {
			// No existing assignment, proceed
		} else {
			return fmt.Errorf("failed to check existing assignment: %w", err)
		}
	} else {
		return fmt.Errorf("asset already assigned")
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO asset_assign (asset_id, employee_id, assigned_by)
		VALUES ($1, $2, $3)
	`, assetID, employeeID, assignedBy)
	if err != nil {
		return fmt.Errorf("failed to insert into asset_assign table: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE assets SET status = 'assigned' WHERE id = $1
	`, assetID)
	if err != nil {
		return fmt.Errorf("failed to update assignment: %w", err)
	}
	return nil
}

func (r *PostgresAssetRepository) DeleteAssetByID(ctx context.Context, assetID uuid.UUID) (err error) {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	var exists bool
	err = tx.GetContext(ctx, &exists, `
		SELECT EXISTS (
			SELECT 1 FROM asset_assign 
			WHERE asset_id = $1 AND archived_at IS NULL AND returned_at IS NULL
		)
	`, assetID)
	if err != nil {
		return fmt.Errorf("failed to check asset assignment: %w", err)
	}
	if exists {
		return fmt.Errorf("asset currently assigned to a user")
	}

	_, err = tx.ExecContext(ctx, `UPDATE assets SET archived_at = now() WHERE id = $1`, assetID)
	if err != nil {
		return fmt.Errorf("failed to archive asset: %w", err)
	}
	return nil
}

func (r *PostgresAssetRepository) GetAssetTimeline(ctx context.Context, assetUUID uuid.UUID) ([]models.AssetTimelineEvent, error) {
	timeline := []models.AssetTimelineEvent{}

	query := `
		SELECT 
			'assigned' AS event_type,
			assigned_at AS start_time,
			returned_at AS end_time,
			'Assigned to employee' AS details,
			asset_id
		FROM asset_assign
		WHERE asset_id = $1 AND archived_at IS NULL

		UNION ALL

		SELECT 
			'went_for_service' AS event_type,
			service_start AS start_time,
			service_end AS end_time,
			reason AS details,
			asset_id
		FROM asset_service
		WHERE asset_id = $1 AND archived_at IS NULL

		ORDER BY start_time ASC
	`

	err := r.DB.SelectContext(ctx, &timeline, query, assetUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch asset timeline: %w", err)
	}

	return timeline, nil
}

func (r *PostgresAssetRepository) RecivedAssetFromService(ctx context.Context, assetID uuid.UUID) (err error) {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	var count int
	err = tx.GetContext(ctx, &count, `
		SELECT COUNT(*) FROM asset_service
		WHERE asset_id = $1 AND archived_at IS NULL AND service_end IS NULL
	`, assetID)
	if err != nil {
		return fmt.Errorf("failed to check service record: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("asset is not currently under service")
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE assets
		SET status = 'available'
		WHERE id = $1
	`, assetID)
	if err != nil {
		return fmt.Errorf("failed to update asset status: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE asset_service
		SET service_end = now()
		WHERE asset_id = $1 AND archived_at IS NULL AND service_end IS NULL
	`, assetID)
	if err != nil {
		return fmt.Errorf("failed to update asset_service end_date: %w", err)
	}

	return nil
}

func (r *PostgresAssetRepository) RetrieveAsset(ctx context.Context, tx *sqlx.Tx, assetID uuid.UUID, employeeID uuid.UUID, reason string) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE asset_assign 
		SET returned_at = now(), return_reason = $1
		WHERE asset_id = $2 AND employee_id = $3 AND returned_at IS NULL AND archived_at IS NULL
	`, reason, assetID, employeeID)
	if err != nil {
		return fmt.Errorf("failed to update asset_assign: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to fetch rows affected: %w", err)
	}
	fmt.Println("Rows affected (asset_assign):", rowsAffected)

	if rowsAffected == 0 {
		return fmt.Errorf("no matching asset assignment found or already returned")
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE assets SET status = 'available' WHERE id = $1 AND archived_at IS NULL
	`, assetID)
	if err != nil {
		return fmt.Errorf("failed to update asset status: %w", err)
	}
	fmt.Println("Asset status updated to 'available'")
	return nil
}

func (r *PostgresAssetRepository) SearchAssetsWithFilter(ctx context.Context, filter models.AssetFilter) (assets []models.AssetWithConfigRes, err error) {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
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

	// Use SelectContext to fetch assets.
	err = tx.SelectContext(ctx, &assets, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch assets: %w", err)
	}

	for i, asset := range assets {
		var config interface{}
		switch asset.Type {
		case "laptop":
			var configTemp models.Laptop_config_res
			err = tx.GetContext(ctx, &configTemp, `SELECT processor, ram, os FROM laptop_config WHERE asset_id = $1`, asset.ID)
			config = configTemp
		case "mouse":
			var configTemp models.Mouse_config_res
			err = tx.GetContext(ctx, &configTemp, `SELECT dpi FROM mouse_config WHERE asset_id = $1`, asset.ID)
			config = configTemp
		case "monitor":
			var configTemp models.Monitor_config_res
			err = tx.GetContext(ctx, &configTemp, `SELECT display, resolution, port FROM monitor_config WHERE asset_id = $1`, asset.ID)
			config = configTemp
		case "mobile":
			var configTemp models.Mobile_config_res
			err = tx.GetContext(ctx, &configTemp, `SELECT processor, ram, os, imei_1, imei_2 FROM mobile_config WHERE asset_id = $1`, asset.ID)
			config = configTemp
		case "hard_disk":
			var configTemp models.Hard_disk_config_res
			err = tx.GetContext(ctx, &configTemp, `SELECT type, storage FROM hard_disk_config WHERE asset_id = $1`, asset.ID)
			config = configTemp
		case "pen_drive":
			var configTemp models.Pen_drive_config_res
			err = tx.GetContext(ctx, &configTemp, `SELECT version, storage FROM pendrive_config WHERE asset_id = $1`, asset.ID)
			config = configTemp
		case "sim":
			var configTemp models.Sim_config_res
			err = tx.GetContext(ctx, &configTemp, `SELECT number FROM sim_config WHERE asset_id = $1`, asset.ID)
			config = configTemp
		case "accessory":
			var configTemp models.Accessories_config_res
			err = tx.GetContext(ctx, &configTemp, `SELECT type, additional_info FROM accessories_config WHERE asset_id = $1`, asset.ID)
			config = configTemp
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) { // Check for sql.ErrNoRows specifically
			return nil, fmt.Errorf("failed to fetch config for asset %s: %w", asset.ID, err)
		}

		assets[i].Config = config
	}

	return assets, nil
}

func (r *PostgresAssetRepository) SendAssetForService(ctx context.Context, req models.AssetServiceReq, managerUUID uuid.UUID) (err error) {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	var inService bool
	err = tx.GetContext(ctx, &inService, `
		SELECT EXISTS (
			SELECT 1 FROM asset_service 
			WHERE asset_id = $1 AND service_end IS NULL AND archived_at IS NULL
		)
	`, req.AssetID)
	if err != nil {
		return fmt.Errorf("failed to check service status: %w", err)
	}
	if inService {
		return fmt.Errorf("asset is already under service")
	}

	var currentStatus string
	err = tx.GetContext(ctx, &currentStatus, `
		SELECT status FROM assets 
		WHERE id = $1 AND archived_at IS NULL
	`, req.AssetID)
	if err != nil {
		return fmt.Errorf("failed to get asset status: %w", err)
	}

	if currentStatus != "available" && currentStatus != "waiting_for_service" {
		return fmt.Errorf("only assets with status 'available' or 'waiting_for_service' can be sent for service")
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO asset_service (asset_id, reason, created_by)
		VALUES ($1, $2, $3)
	`, req.AssetID, req.Reason, managerUUID)
	if err != nil {
		return fmt.Errorf("failed to insert service record: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE assets SET status = 'sent_for_service'
		WHERE id = $1 AND archived_at IS NULL
	`, req.AssetID)
	if err != nil {
		return fmt.Errorf("failed to update asset status: %w", err)
	}

	return nil
}

func (r *PostgresAssetRepository) UpdateAssetWithConfig(ctx context.Context, req models.UpdateAssetReq) (err error) {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		log.Println("transaction failed", err)
		return err
	}
	// Defer a rollback or commit based on the outcome of the function.
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
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

		_, err := tx.ExecContext(ctx, query, args...)
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
			_, err = tx.ExecContext(ctx, `UPDATE laptop_config SET processor = $1, ram = $2, os = $3 WHERE asset_id = $4`,
				cfg.Processor, cfg.Ram, cfg.Os, req.ID)
		case "mouse":
			var cfg models.Mouse_config_req
			if err := json.Unmarshal(req.Config, &cfg); err != nil {
				return fmt.Errorf("invalid mouse config: %w", err)
			}
			_, err = tx.ExecContext(ctx, `UPDATE mouse_config SET dpi = $1 WHERE asset_id = $2`, cfg.DPI, req.ID)
		case "monitor":
			var cfg models.Monitor_config_req
			if err := json.Unmarshal(req.Config, &cfg); err != nil {
				return fmt.Errorf("invalid monitor config: %w", err)
			}
			_, err = tx.ExecContext(ctx, `UPDATE monitor_config SET display = $1, resolution = $2, port = $3 WHERE asset_id = $4`,
				cfg.Display, cfg.Resolution, cfg.Port, req.ID)
		case "mobile":
			var cfg models.Mobile_config_req
			if err := json.Unmarshal(req.Config, &cfg); err != nil {
				return fmt.Errorf("invalid mobile config: %w", err)
			}
			_, err = tx.ExecContext(ctx, `UPDATE mobile_config SET processor = $1, ram = $2, os = $3, imei_1 = $4, imei_2 = $5 WHERE asset_id = $6`,
				cfg.Processor, cfg.Ram, cfg.Os, cfg.IMEI1, cfg.IMEI2, req.ID)
		case "hard_disk":
			var cfg models.Hard_disk_config_req
			if err := json.Unmarshal(req.Config, &cfg); err != nil {
				return fmt.Errorf("invalid hard disk config: %w", err)
			}
			_, err = tx.ExecContext(ctx, `UPDATE hard_disk_config SET type = $1, storage = $2 WHERE asset_id = $3`,
				cfg.Type, cfg.Storage, req.ID)
		case "pen_drive":
			var cfg models.Pen_drive_config_req
			if err := json.Unmarshal(req.Config, &cfg); err != nil {
				return fmt.Errorf("invalid pen drive config: %w", err)
			}
			_, err = tx.ExecContext(ctx, `UPDATE pendrive_config SET version = $1, storage = $2 WHERE asset_id = $3`,
				cfg.Version, cfg.Storage, req.ID)
		case "sim":
			var cfg models.Sim_config_req
			if err := json.Unmarshal(req.Config, &cfg); err != nil {
				return fmt.Errorf("invalid sim config: %w", err)
			}
			_, err = tx.ExecContext(ctx, `UPDATE sim_config SET number = $1 WHERE asset_id = $2`,
				cfg.Number, req.ID)
		case "accessory":
			var cfg models.Accessories_config_req
			if err := json.Unmarshal(req.Config, &cfg); err != nil {
				return fmt.Errorf("invalid accessory config: %w", err)
			}
			_, err = tx.ExecContext(ctx, `UPDATE accessories_config SET type = $1, additional_info = $2 WHERE asset_id = $3`,
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
