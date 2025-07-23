package dbhelper

import (
	"asset/database"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
)

func GetUserByEmail(userEmail string) (uuid.UUID, error) {
	var userId uuid.UUID
	err := database.DB.Get(&userId, `
		SELECT id FROM users
		WHERE email = $1 
		AND archived_at IS NULL
	`, userEmail)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, sql.ErrNoRows
		}
		return uuid.Nil, fmt.Errorf("failed to fetch user by email: %w", err)
	}
	return userId, nil
}
