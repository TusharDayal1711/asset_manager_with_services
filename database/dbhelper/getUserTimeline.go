package dbhelper

import (
	"asset/database"
	"asset/models"
	"fmt"
	"github.com/google/uuid"
)

func GetUserAssetTimeline(userID uuid.UUID) ([]models.UserTimelineRes, error) {
	timeline := make([]models.UserTimelineRes, 0)
	err := database.DB.Select(&timeline, `
		SELECT 
			a.asset_id,
			at.brand,
			at.model,
			at.serial_no,
			a.assigned_at,
			a.returned_at,
			a.return_reason
		FROM asset_assign a
		JOIN assets at ON at.id = a.asset_id
		WHERE a.employee_id = $1 AND a.archived_at IS NULL
		ORDER BY a.assigned_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user timeline")
	}
	return timeline, nil
}
