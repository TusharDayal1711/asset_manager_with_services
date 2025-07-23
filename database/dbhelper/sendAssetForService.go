package dbhelper

import (
	"asset/database"
	"asset/models"
	"fmt"
	"github.com/google/uuid"
)

func SendAssetForService(req models.AssetServiceReq, managerUUID uuid.UUID) error {
	tx, err := database.DB.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	var inService bool
	err = tx.Get(&inService, `
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
	err = tx.Get(&currentStatus, `
		SELECT status FROM assets 
		WHERE id = $1 AND archived_at IS NULL
	`, req.AssetID)
	if err != nil {
		return fmt.Errorf("failed to get asset status: %w", err)
	}

	if currentStatus != "available" && currentStatus != "waiting_for_service" {
		return fmt.Errorf("only assets with status 'available' or 'waiting_for_service' can be sent for service")
	}

	_, err = tx.Exec(`
		INSERT INTO asset_service (asset_id, reason, created_by)
		VALUES ($1, $2, $3)
	`, req.AssetID, req.Reason, managerUUID)
	if err != nil {
		return fmt.Errorf("failed to insert service record: %w", err)
	}

	_, err = tx.Exec(`
		UPDATE assets SET status = 'sent_for_service'
		WHERE id = $1 AND archived_at IS NULL
	`, req.AssetID)
	if err != nil {
		return fmt.Errorf("failed to update asset status: %w", err)
	}

	return tx.Commit()
}
