package dbhelper

import (
	"asset/database"
	"fmt"
	"github.com/google/uuid"
)

func RecivedAssetFromService(assetID uuid.UUID) (err error) {
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

	var count int
	err = tx.Get(&count, `
		SELECT COUNT(*) FROM asset_service
		WHERE asset_id = $1 AND archived_at IS NULL AND service_end IS NULL
	`, assetID)
	if err != nil {
		return fmt.Errorf("failed to check service record: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("asset is not currently under service")
	}

	_, err = tx.Exec(`
		UPDATE assets
		SET status = 'available'
		WHERE id = $1
	`, assetID)
	if err != nil {
		return fmt.Errorf("failed to update asset status: %w", err)
	}

	_, err = tx.Exec(`
		UPDATE asset_service
		SET service_end = now()
		WHERE asset_id = $1 AND archived_at IS NULL AND service_end IS NULL
	`, assetID)
	if err != nil {
		return fmt.Errorf("failed to update asset_service end_date: %w", err)
	}

	return nil
}
