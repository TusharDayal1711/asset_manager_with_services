package dbhelper

import (
	"asset/models"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func CreateNewEmployee(tx *sqlx.Tx, req models.ManagerRegisterReq, managerUUID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := tx.Get(&userID, `
		INSERT INTO users (username, email, contact_no)
		VALUES ($1, $2, $3)
		RETURNING id
	`, req.Username, req.Email, req.ContactNo)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert employee: %w", err)
	}

	// Insert employee type in user_type table
	_, err = tx.Exec(`
		INSERT INTO user_type (user_id, type, created_by)
		VALUES ($1, $2, $3)
	`, userID, req.Type, managerUUID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert employee type: %w", err)
	}
	_, err = tx.Exec(`
		INSERT INTO user_roles (user_id, role, created_by)
		VALUES ($1, 'employee', $2)
	`, userID, managerUUID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert employee role: %w", err)
	}

	return userID, nil
}
