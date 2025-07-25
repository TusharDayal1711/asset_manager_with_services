package userservice

import (
	"asset/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type UserRepository interface {
	DeleteUserByID(ctx context.Context, userID uuid.UUID) error
	GetUserByEmail(ctx context.Context, userEmail string) (uuid.UUID, error)
	GetUserDashboardById(ctx context.Context, userID uuid.UUID) (UserDashboardRes, error)
	GetUserRoleById(ctx context.Context, userId uuid.UUID) (string, error)
	GetUserAssetTimeline(ctx context.Context, userID uuid.UUID) ([]UserTimelineRes, error)
	IsUserExists(ctx context.Context, tx *sqlx.Tx, email string) (bool, error)
	CreateNewEmployee(ctx context.Context, tx *sqlx.Tx, req ManagerRegisterReq, managerUUID uuid.UUID) (uuid.UUID, error)
	GetFilteredEmployeesWithAssets(ctx context.Context, filter EmployeeFilter) ([]EmployeeResponseModel, error)
	UpdateEmployeeInfo(ctx context.Context, req UpdateEmployeeReq, adminUUID uuid.UUID) error
	GetCurrentUserRole(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID) (string, error)
	InsertIntoUser(ctx context.Context, tx *sqlx.Tx, username, email string) (uuid.UUID, error)
	InsertIntoUserType(ctx context.Context, tx *sqlx.Tx, userId uuid.UUID, employeeType string, createdBy uuid.UUID) error
	UpdateUserRole(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID, newRole string, updatedBy uuid.UUID) error
	InsertIntoUserRole(ctx context.Context, tx *sqlx.Tx, userId uuid.UUID, role string, createdBy uuid.UUID) error // Added this back as it was in the interface
	InsertUserRole(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID, role string, createdBy uuid.UUID) error     // This seems to be a duplicate of InsertIntoUserRole, consider consolidating
	ArchiveUserRoles(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID) error
}

type PostgresUserRepository struct {
	DB *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) UserRepository {
	return &PostgresUserRepository{DB: db}
}

func (r *PostgresUserRepository) DeleteUserByID(ctx context.Context, userID uuid.UUID) (err error) {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	var count int
	err = tx.GetContext(ctx, &count, `
		SELECT count(*) FROM asset_assign 
		WHERE employee_id = $1 AND returned_at IS NULL AND archived_at IS NULL LIMIT 1
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to check asset assignment: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete user, still have asset assigned")
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE users SET archived_at = now() WHERE id = $1 AND archived_at IS NULL
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE user_roles SET archived_at = now() WHERE user_id = $1 AND archived_at IS NULL
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user roles: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE user_type SET archived_at = now() WHERE user_id = $1 AND archived_at IS NULL
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user type: %w", err)
	}

	return nil
}

func (r *PostgresUserRepository) GetUserByEmail(ctx context.Context, userEmail string) (uuid.UUID, error) {
	var userId uuid.UUID

	err := r.DB.GetContext(ctx, &userId, `
		SELECT id FROM users
		WHERE email = $1 AND archived_at IS NULL
	`, userEmail)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, sql.ErrNoRows
		}
		return uuid.Nil, fmt.Errorf("failed to fetch user by email: %w", err)
	}
	return userId, nil
}

func (r *PostgresUserRepository) GetUserDashboardById(ctx context.Context, userID uuid.UUID) (user UserDashboardRes, err error) {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return user, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	err = tx.GetContext(ctx, &user, `
		SELECT u.id, u.username, u.email, u.contact_no, ut.type
		FROM users u
		LEFT JOIN user_type ut ON ut.user_id = u.id AND ut.archived_at IS NULL
		WHERE u.id = $1 AND u.archived_at IS NULL
	`, userID)
	if err != nil {
		return user, fmt.Errorf("failed to fetch user: %w", err)
	}

	err = tx.SelectContext(ctx, &user.Roles, `
		SELECT role FROM user_roles 
		WHERE user_id = $1 AND archived_at IS NULL
	`, userID)
	if err != nil {
		return user, fmt.Errorf("failed to fetch roles: %w", err)
	}

	err = tx.SelectContext(ctx, &user.AssignedAssets, `
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

func (r *PostgresUserRepository) GetUserRoleById(ctx context.Context, userId uuid.UUID) (string, error) {
	var userRole string
	err := r.DB.GetContext(ctx, &userRole, `
		SELECT role FROM user_roles 
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

func (r *PostgresUserRepository) GetUserAssetTimeline(ctx context.Context, userID uuid.UUID) ([]UserTimelineRes, error) {
	timeline := make([]UserTimelineRes, 0)

	err := r.DB.SelectContext(ctx, &timeline, `
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
		return nil, fmt.Errorf("failed to get user timeline: %w", err)
	}
	return timeline, nil
}

func (r *PostgresUserRepository) CreateNewEmployee(ctx context.Context, tx *sqlx.Tx, req ManagerRegisterReq, managerUUID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	err := tx.GetContext(ctx, &userID, `
		INSERT INTO users (username, email, contact_no)
		VALUES ($1, $2, $3)
		RETURNING id
	`, req.Username, req.Email, req.ContactNo)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert employee: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_type (user_id, type, created_by)
		VALUES ($1, $2, $3)
	`, userID, req.Type, managerUUID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert employee type: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_roles (user_id, role, created_by)
		VALUES ($1, 'employee', $2)
	`, userID, managerUUID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert employee role: %w", err)
	}

	return userID, nil
}

func (r *PostgresUserRepository) GetFilteredEmployeesWithAssets(ctx context.Context, filter EmployeeFilter) ([]EmployeeResponseModel, error) {
	args := []interface{}{
		!filter.IsSearchText,
		filter.SearchText,
		pq.Array(filter.Type),
		pq.Array(filter.Role),
		pq.Array(filter.AssetStatus),
		filter.Limit,
		filter.Offset,
	}

	query := `SELECT
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
AND (
	$1 OR (
		u.username ILIKE '%' || $2 || '%'
		OR u.email ILIKE '%' || $2 || '%'
		OR u.contact_no ILIKE '%' || $2 || '%'
	)
)
AND ($3::text[] IS NULL OR ut.type::text = ANY($3))
AND ($4::text[] IS NULL OR ur.role::text = ANY($4))
AND ($5::text[] IS NULL OR a.status::text = ANY($5) OR a.id IS NULL)
GROUP BY u.id, ut.type, u.created_at
ORDER BY u.created_at DESC
LIMIT $6 OFFSET $7;

    `

	rows := []EmployeeResponseModel{}
	err := r.DB.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *PostgresUserRepository) UpdateEmployeeInfo(ctx context.Context, req UpdateEmployeeReq, adminUUID uuid.UUID) error {
	query := `UPDATE users SET `
	args := []interface{}{}
	argPos := 1

	if req.Username != "" {
		query += fmt.Sprintf("username = $%d, ", argPos)
		args = append(args, req.Username)
		argPos++
	}
	if req.Email != "" {
		query += fmt.Sprintf("email = $%d, ", argPos)
		args = append(args, req.Email)
		argPos++
	}
	if req.ContactNo != "" {
		query += fmt.Sprintf("contact_no = $%d, ", argPos)
		args = append(args, req.ContactNo)
		argPos++
	}

	query += fmt.Sprintf("updated_by = $%d ", argPos)
	args = append(args, adminUUID)
	argPos++

	query = strings.TrimSuffix(query, ", ")
	query += fmt.Sprintf("WHERE id = $%d AND archived_at IS NULL", argPos)
	args = append(args, req.UserID)

	result, err := r.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no user found or nothing updated")
	}

	return nil
}

func (r *PostgresUserRepository) InsertUserRole(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID, role string, createdBy uuid.UUID) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO user_roles (role, user_id, created_by)
		VALUES ($1, $2, $3)
	`, role, userID, createdBy)
	if err != nil {
		return fmt.Errorf("failed to insert new role: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) UpdateUserRole(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID, newRole string, updatedBy uuid.UUID) error {
	currentRole, err := r.GetCurrentUserRole(ctx, tx, userID) // Pass ctx
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to fetch current role: %w", err)
	}
	if err == nil && currentRole == newRole {
		return fmt.Errorf("user already has the role: %s", newRole)
	}
	if err := r.ArchiveUserRoles(ctx, tx, userID); err != nil { // Pass ctx
		return err
	}
	if err := r.InsertUserRole(ctx, tx, userID, newRole, updatedBy); err != nil { // Pass ctx
		return err
	}
	return nil
}

func (r *PostgresUserRepository) GetCurrentUserRole(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID) (string, error) {
	var role string

	err := tx.GetContext(ctx, &role, `
		SELECT role FROM user_roles
		WHERE user_id = $1 AND archived_at IS NULL
	`, userID)
	if err != nil {
		return "", err
	}
	return role, nil
}

func (r *PostgresUserRepository) ArchiveUserRoles(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID) error {

	_, err := tx.ExecContext(ctx, `
		UPDATE user_roles
		SET archived_at = now(), last_updated_at = now()
		WHERE user_id = $1 AND archived_at IS NULL
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to archive existing roles: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) IsUserExists(ctx context.Context, tx *sqlx.Tx, email string) (bool, error) {
	utils.Logger.Info("inside IsUSerExits...", zap.String("incoming email:", email))
	var id uuid.UUID
	err := tx.QueryRowContext(ctx, `
		SELECT id FROM users 
		WHERE email = $1 AND archived_at IS NULL
	`, email).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		utils.Logger.Error("failed to check if user exists", zap.Error(err))
		return false, fmt.Errorf("failed to check existing user: %w", err)
	}
	utils.Logger.Info("user exists, returning true", zap.String("user", email))
	return true, nil
}

func (r *PostgresUserRepository) InsertIntoUser(ctx context.Context, tx *sqlx.Tx, username, email string) (uuid.UUID, error) {
	utils.Logger.Info("inside InsertIntoUser, with values ::", zap.String("email:", email), zap.String("username:", username))
	var id uuid.UUID
	err := tx.GetContext(ctx, &id, `
		INSERT INTO users (username, email)
		VALUES ($1, $2)
		RETURNING id
	`, username, email)
	if err != nil {
		utils.Logger.Error("failed to insert into users", zap.Error(err))
		return uuid.Nil, fmt.Errorf("failed to insert user: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE users SET created_by = $1 WHERE id = $1
	`, id)
	if err != nil {
		utils.Logger.Error("failed to insert into users", zap.Error(err))
		return uuid.Nil, fmt.Errorf("failed to update created_by: %w", err)
	}
	return id, nil
}

func (r *PostgresUserRepository) InsertIntoUserRole(ctx context.Context, tx *sqlx.Tx, userId uuid.UUID, role string, createdBy uuid.UUID) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO user_roles (role, user_id, created_by)
		VALUES ($1, $2, $3)
	`, role, userId, createdBy)
	if err != nil {
		return fmt.Errorf("failed to insert user role: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) InsertIntoUserType(ctx context.Context, tx *sqlx.Tx, userId uuid.UUID, employeeType string, createdBy uuid.UUID) error {

	_, err := tx.ExecContext(ctx, `
		INSERT INTO user_type (type, user_id, created_by)
		VALUES ($1, $2, $3)
	`, employeeType, userId, createdBy)
	if err != nil {
		return fmt.Errorf("failed to insert user type: %w", err)
	}
	return nil
}
