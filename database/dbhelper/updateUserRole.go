package dbhelper

import (
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func GetCurrentUserRole(tx *sqlx.Tx, userID uuid.UUID) (string, error) {
	var role string
	err := tx.Get(&role, `
		SELECT role FROM user_roles
		WHERE user_id = $1 AND archived_at IS NULL
	`, userID)
	if err != nil {
		return "", err
	}
	return role, nil
}

func ArchiveUserRoles(tx *sqlx.Tx, userID uuid.UUID) error {
	_, err := tx.Exec(`
		UPDATE user_roles
		SET archived_at = now(), last_updated_at = now()
		WHERE user_id = $1 AND archived_at IS NULL
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to archive existing roles: %w", err)
	}
	return nil
}

func InsertUserRole(tx *sqlx.Tx, userID uuid.UUID, role string, createdBy uuid.UUID) error {
	_, err := tx.Exec(`
		INSERT INTO user_roles (id, role, user_id, created_by)
		VALUES ($1, $2, $3)
	`, role, userID, createdBy)
	if err != nil {
		return fmt.Errorf("failed to insert new role: %w", err)
	}
	return nil
}

func UpdateUserRole(tx *sqlx.Tx, userID uuid.UUID, newRole string, updatedBy uuid.UUID) error {
	currentRole, err := GetCurrentUserRole(tx, userID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to fetch current role: %w", err)
	}
	if err == nil && currentRole == newRole {
		return fmt.Errorf("user already has the role: %s", newRole)
	}
	if err := ArchiveUserRoles(tx, userID); err != nil {
		return err
	}
	if err := InsertUserRole(tx, userID, newRole, updatedBy); err != nil {
		return err
	}
	return nil
}
