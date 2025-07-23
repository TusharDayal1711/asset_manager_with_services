package dbhelper

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func RetrieveAsset(tx *sqlx.Tx, assetID uuid.UUID, employeeID uuid.UUID, reason string) error {
	res, err := tx.Exec(`
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

	//updating asset table
	_, err = tx.Exec(`
		UPDATE assets SET status = 'available' WHERE id = $1 AND archived_at IS NULL
	`, assetID)
	if err != nil {
		return fmt.Errorf("failed to update asset status: %w", err)
	}
	fmt.Println("Asset status updated to 'available'")
	return nil
}
