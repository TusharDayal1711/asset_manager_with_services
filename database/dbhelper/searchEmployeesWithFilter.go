package dbhelper

import (
	"asset/database"
	"asset/models"
	"github.com/lib/pq"
)

func GetFilteredEmployeesWithAssets(filter models.EmployeeFilter) ([]models.EmployeeResponseModel, error) {
	args := []interface{}{
		!filter.IsSearchText,
		filter.SearchText,
		pq.Array(filter.Type),
		pq.Array(filter.Role),
		pq.Array(filter.AssetStatus),
		filter.Limit,
		filter.Offset,
	}

	query := `
	SELECT
	u.id,
		u.username,
		u.email,
		u.contact_no,
		ut.type AS employee_type,
	COALESCE(array_agg(a.id) FILTER (WHERE a.id IS NOT NULL), '{}') AS assigned_assets
	FROM users u
	LEFT JOIN user_type ut ON u.id = ut.user_id AND ut.archived_at IS NULL
	LEFT JOIN user_roles ur ON u.id = ur.user_id AND ur.archived_at IS NULL
	LEFT JOIN asset_assign aa ON u.id = aa.employee_id AND aa.archived_at IS NULL
	LEFT JOIN assets a ON aa.asset_id = a.id AND a.archived_at IS NULL
	WHERE u.archived_at IS NULL
	AND ($1 OR (
		u.username ILIKE '%' || $2 || '%'
	OR u.email ILIKE '%' || $2 || '%'
	OR u.contact_no ILIKE '%' || $2 || '%'
	))
	AND ($3::text[] IS NULL OR ut.type::text = ANY($3))
	AND ($4::text[] IS NULL OR ur.role::text = ANY($4))
	AND ($5::text[] IS NULL OR a.status::text = ANY($5))
	GROUP BY u.id, ut.type, u.created_at
		ORDER BY u.created_at DESC
	LIMIT $6 OFFSET $7;
	
	`

	rows := []models.EmployeeResponseModel{}
	err := database.DB.Select(&rows, query, args...)
	if err != nil {
		return nil, err
	}
	return rows, nil
}
