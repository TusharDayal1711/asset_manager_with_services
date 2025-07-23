package dbhelper

import (
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func AssignAssetByID(tx *sqlx.Tx, assetId uuid.UUID, userId uuid.UUID, assignedBy uuid.UUID) error {
	var exists int
	err := tx.Get(&exists, `
	SELECT 1 FROM asset_assign 
	WHERE asset_id = $1 AND returned_at IS NULL AND archived_at IS NULL
	LIMIT 1
`, assetId)

	if err != nil {
		if err == sql.ErrNoRows {
		} else {
			return fmt.Errorf("failed to check existing assignment: %w", err)
		}
	} else {
		return fmt.Errorf("asset already assigned")
	}

	//inserting into asset_assign table
	_, err = tx.Exec(`
		INSERT INTO asset_assign (asset_id, employee_id, assigned_by)
		VALUES ($1, $2, $3)
	`, assetId, userId, assignedBy)
	if err != nil {
		return fmt.Errorf("failed to insert into asset_assign table%w", err)
	}
	//updating asset status in asset table
	_, err = tx.Exec(`
		UPDATE assets SET status = 'assigned' WHERE id = $1
	`, assetId)
	if err != nil {
		return fmt.Errorf("failed to update assignment: %w", err)
	}
	return nil
}
