package dbhelper

import (
	"asset/database"
	"fmt"
	"github.com/google/uuid"
)

func DeleteUserByID(userID uuid.UUID) error {
	tx, err := database.DB.Beginx()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
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
		SELECT count(*) FROM asset_assign 
		         WHERE employee_id = $1 
		           AND returned_at IS NULL AND archived_at IS NULL LIMIT 1

	`, userID)
	if err != nil {
		return fmt.Errorf("failed to check asset assignment: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete user, still have asset assigned")
	}
	_, err = tx.Exec(`
		UPDATE users SET archived_at = now() WHERE id = $1 AND archived_at IS NULL
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	_, err = tx.Exec(`
		UPDATE user_roles SET archived_at = now() WHERE user_id = $1 AND archived_at IS NULL
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user roles: %w", err)
	}

	_, err = tx.Exec(`
		UPDATE user_type SET archived_at = now() WHERE user_id = $1 AND archived_at IS NULL
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user type: %w", err)
	}

	return nil
}
