package dbhelper

import (
	"asset/database"
	"asset/models"
	"fmt"
	"github.com/google/uuid"
)

func GetAssetTimeline(assetUUID uuid.UUID) ([]models.AssetTimelineEvent, error) {
	timeline := []models.AssetTimelineEvent{}

	query := `
		SELECT 
			'assigned' AS event_type,
			assigned_at AS start_time,
			returned_at AS end_time,
			'Assigned to employee' AS details,
			asset_id
		FROM asset_assign
		WHERE asset_id = $1 AND archived_at IS NULL

		UNION ALL

		SELECT 
			'went_for_service' AS event_type,
			service_start AS start_time,
			service_end AS end_time,
			reason AS details,
			asset_id
		FROM asset_service
		WHERE asset_id = $1 AND archived_at IS NULL

		ORDER BY start_time ASC
	`

	err := database.DB.Select(&timeline, query, assetUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch asset timeline: %w", err)
	}

	return timeline, nil
}
