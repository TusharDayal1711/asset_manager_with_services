package dbhelper

import (
	"asset/database"
	"asset/models"
	"fmt"
	"github.com/google/uuid"
)

func GetUserDashboardById(userID uuid.UUID) (models.UserDashboardRes, error) {
	var user models.UserDashboardRes

	tx, err := database.DB.Beginx()
	if err != nil {
		return user, fmt.Errorf("failed to begin transcation: %v", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	//user's info
	err = tx.Get(&user, `
		SELECT u.id, u.username, u.email, u.contact_no, ut.type
		FROM users u
		LEFT JOIN user_type ut ON ut.user_id = u.id AND ut.archived_at IS NULL
		WHERE u.id = $1 AND u.archived_at IS NULL
	`, userID)
	if err != nil {
		return user, fmt.Errorf("failed to fetch user: %w", err)
	}

	//user role
	err = tx.Select(&user.Roles, `
		SELECT role FROM user_roles 
		WHERE user_id = $1 AND archived_at IS NULL
	`, userID)
	if err != nil {
		return user, fmt.Errorf("failed to fetch roles: %w", err)
	}

	//user assigned assets
	err = tx.Select(&user.AssignedAssets, `
		SELECT a.id, a.brand, a.model, a.serial_no, a.type, a.status, a.owned_by
		FROM assets a
		INNER JOIN asset_assign aa ON aa.asset_id = a.id
		WHERE aa.employee_id = $1 AND aa.returned_at IS NULL AND aa.archived_at IS NULL AND a.archived_at IS NULL
	`, userID)
	if err != nil {
		return user, fmt.Errorf("failed to fetch assigned assets: %w", err)
	}
	return user, nil
}
