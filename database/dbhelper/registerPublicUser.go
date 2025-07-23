package dbhelper

import (
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func IsUserExists(tx *sqlx.Tx, email string) (bool, error) {
	var id uuid.UUID
	err := tx.QueryRow(`
		SELECT id FROM users 
		WHERE email = $1 AND archived_at IS NULL
	`, email).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check existing user: %w", err)
	}
	return true, nil
}

func InsertIntoUser(tx *sqlx.Tx, username, email string) (uuid.UUID, error) {
	var id uuid.UUID

	err := tx.Get(&id, `
		INSERT INTO users (username, email)
		VALUES ($1, $2)
		RETURNING id
	`, username, email)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert user: %w", err)
	}
	_, err = tx.Exec(`
		UPDATE users SET created_by = $1 WHERE id = $1
	`, id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to update created_by: %w", err)
	}
	return id, nil
}

func InsertIntoUserRole(tx *sqlx.Tx, userId uuid.UUID, role string, createdBy uuid.UUID) error {
	_, err := tx.Exec(`
		INSERT INTO user_roles (role, user_id, created_by)
		VALUES ($1, $2, $3)
	`, role, userId, createdBy)
	if err != nil {
		return fmt.Errorf("failed to insert user role: %w", err)
	}
	return nil
}

func InsertIntoUserType(tx *sqlx.Tx, userId uuid.UUID, employeeType string, createdBy uuid.UUID) error {
	_, err := tx.Exec(`
		INSERT INTO user_type (type, user_id, created_by)
		VALUES ($1, $2, $3)
	`, employeeType, userId, createdBy)
	if err != nil {
		return fmt.Errorf("failed to insert user type: %w", err)
	}
	return nil
}
