package dbhelper

import (
	"asset/database"
	"fmt"
	"github.com/google/uuid"
)

func DeleteAssetByID(assetID uuid.UUID) error {
	tx, err := database.DB.Beginx()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	var exists bool
	err = tx.Get(&exists, `
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

	_, err = tx.Exec(`UPDATE assets SET archived_at = now() WHERE id = $1`, assetID)
	if err != nil {
		return fmt.Errorf("failed to archive asset: %w", err)
	}
	return nil
}
