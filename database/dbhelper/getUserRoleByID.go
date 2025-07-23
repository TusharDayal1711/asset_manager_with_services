package dbhelper

import (
	"asset/database"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

func GetUserRoleById(userId uuid.UUID) (string, error) {
	var userRole string

	err := database.DB.Get(&userRole, `
		SELECT role 
		FROM user_roles 
		WHERE user_id = $1 AND archived_at IS NULL
	`, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("no role found for user ID: %s", userId)
		}
		return "", fmt.Errorf("failed to fetch user role: %w", err)
	}
	return userRole, nil
}
