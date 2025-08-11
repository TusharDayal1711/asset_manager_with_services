package userservice

import (
	"asset/providers"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-jose/go-jose/v4/json"
	"go.uber.org/zap"
	"strings"
	"time"

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
	InsertIntoUser(ctx context.Context, tx *sqlx.Tx, username, email string, firebasetoken string) (uuid.UUID, error)
	InsertIntoUserType(ctx context.Context, tx *sqlx.Tx, userId uuid.UUID, employeeType string, createdBy uuid.UUID) error
	UpdateUserRole(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID, newRole string, updatedBy uuid.UUID) error
	InsertIntoUserRole(ctx context.Context, tx *sqlx.Tx, userId uuid.UUID, role string, createdBy uuid.UUID) error
	InsertUserRole(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID, role string, createdBy uuid.UUID) error
	CreateFirebaseUser(ctx context.Context, name, email string) (uuid.UUID, error)
	GetFirebase() providers.FirebaseProvider
	GetEmailByUserID(ctx context.Context, userId uuid.UUID) (string, error)
}

type PostgresUserRepository struct {
	DB       *sqlx.DB
	Logger   providers.ZapLoggerProvider
	Firebase providers.FirebaseProvider
	Redis    providers.RedisProvider
}

func NewUserRepository(db *sqlx.DB, log providers.ZapLoggerProvider, firebase providers.FirebaseProvider, redis providers.RedisProvider) UserRepository {
	return &PostgresUserRepository{DB: db, Logger: log, Firebase: firebase, Redis: redis}
}

func (r *PostgresUserRepository) GetUserByEmail(ctx context.Context, userEmail string) (uuid.UUID, error) {
	r.Logger.GetLogger().Info("fetching user by email", zap.String("email", userEmail))
	var userId uuid.UUID

	err := r.DB.GetContext(ctx, &userId, `
		SELECT id FROM users
		WHERE email = $1 AND archived_at IS NULL
	`, userEmail)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Logger.GetLogger().Warn("no user found for email", zap.String("email", userEmail), zap.Error(err))
			return uuid.Nil, sql.ErrNoRows
		}
		r.Logger.GetLogger().Error("failed to fetch user by email", zap.String("email", userEmail), zap.Error(err))
		return uuid.Nil, fmt.Errorf("failed to fetch user by email: %w", err)
	}
	r.Logger.GetLogger().Info("user found by email", zap.String("email", userEmail), zap.String("user_id", userId.String()))
	return userId, nil
}

func (r *PostgresUserRepository) DeleteUserByID(ctx context.Context, userID uuid.UUID) (err error) {
	r.Logger.GetLogger().Info("starting transaction to delete user by id", zap.String("user_id", userID.String()))
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		r.Logger.GetLogger().Error("failed to start transaction for deleteuserbyid", zap.Error(err))
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			r.Logger.GetLogger().Error("panic recovered during deleteuserbyid transaction", zap.Any("recover_info", p))
			panic(p)
		} else if err != nil {
			tx.Rollback()
			r.Logger.GetLogger().Error("rolling back transaction for deleteuserbyid due to error", zap.Error(err))
		} else {
			err = tx.Commit()
			if err != nil {
				r.Logger.GetLogger().Error("failed to commit transaction for deleteuserbyid", zap.Error(err))
			} else {
				r.Logger.GetLogger().Info("transaction committed successfully for deleteuserbyid")
			}
		}
	}()

	var count int
	r.Logger.GetLogger().Debug("checking for assigned assets before deleting user", zap.String("user_id", userID.String()))
	err = tx.GetContext(ctx, &count, `
		SELECT count(*) FROM asset_assign 
		WHERE employee_id = $1 AND returned_at IS NULL AND archived_at IS NULL LIMIT 1
	`, userID)
	if err != nil {
		r.Logger.GetLogger().Error("failed to check asset assignment for user deletion", zap.String("user_id", userID.String()), zap.Error(err))
		return fmt.Errorf("failed to check asset assignment: %w", err)
	}
	if count > 0 {
		r.Logger.GetLogger().Warn("cannot delete user, still has assets assigned", zap.String("user_id", userID.String()))
		return fmt.Errorf("cannot delete user, still have asset assigned")
	}

	r.Logger.GetLogger().Debug("archiving user record", zap.String("user_id", userID.String()))
	_, err = tx.ExecContext(ctx, `
		UPDATE users SET archived_at = now() WHERE id = $1 AND archived_at IS NULL
	`, userID)
	if err != nil {
		r.Logger.GetLogger().Error("failed to archive user record", zap.String("user_id", userID.String()), zap.Error(err))
		return fmt.Errorf("failed to delete user: %w", err)
	}

	r.Logger.GetLogger().Debug("archiving user roles", zap.String("user_id", userID.String()))
	_, err = tx.ExecContext(ctx, `
		UPDATE user_roles SET archived_at = now() WHERE user_id = $1 AND archived_at IS NULL
	`, userID)
	if err != nil {
		r.Logger.GetLogger().Error("failed to archive user roles", zap.String("user_id", userID.String()), zap.Error(err))
		return fmt.Errorf("failed to delete user roles: %w", err)
	}

	r.Logger.GetLogger().Debug("archiving user type", zap.String("user_id", userID.String()))
	_, err = tx.ExecContext(ctx, `
		UPDATE user_type SET archived_at = now() WHERE user_id = $1 AND archived_at IS NULL
	`, userID)
	if err != nil {
		r.Logger.GetLogger().Error("failed to archive user type", zap.String("user_id", userID.String()), zap.Error(err))
		return fmt.Errorf("failed to delete user type: %w", err)
	}
	r.Logger.GetLogger().Info("user and associated records archived successfully", zap.String("user_id", userID.String()))
	return nil
}

// //GetUSer
//
//	func (r *userRepository) GetUserDashboardById(ctx context.Context, userID uuid.UUID) (*models.UserDashboard, error) {
//		cacheKey := fmt.Sprintf("user:dashboard:%s", userID.String())
//
//		// Try Redis cache first
//		cached, err := r.Redis.Get(ctx, cacheKey).Result()
//		if err == nil && cached != "" {
//			var dashboard models.UserDashboard
//			if err := json.Unmarshal([]byte(cached), &dashboard); err == nil {
//				return &dashboard, nil
//			}
//			// If unmarshal fails, continue to fetch from DB
//		}
//
//		// Fetch from DB
//		var dashboard models.UserDashboard
//
//		// 1. Get user basic info
//		query := `SELECT id, email, name, phone, address FROM users WHERE id = $1`
//		if err := r.DB.GetContext(ctx, &dashboard.User, query, userID); err != nil {
//			return nil, err
//		}
//
//		// 2. Get user roles
//		roleQuery := `SELECT r.name FROM roles r JOIN user_roles ur ON r.id = ur.role_id WHERE ur.user_id = $1`
//		if err := r.DB.SelectContext(ctx, &dashboard.Roles, roleQuery, userID); err != nil {
//			return nil, err
//		}
//
//		// 3. Get assigned assets
//		assetQuery := `SELECT a.id, a.name, a.status FROM assets a WHERE a.assigned_to = $1`
//		if err := r.DB.SelectContext(ctx, &dashboard.AssignedAssets, assetQuery, userID); err != nil {
//			return nil, err
//		}
//
//		// Save to Redis with 10 min TTL
//		jsonData, err := json.Marshal(dashboard)
//		if err == nil {
//			_ = r.Redis.Set(ctx, cacheKey, jsonData, 10*time.Minute).Err()
//		}
//
//		return &dashboard, nil
//	}
func (r *PostgresUserRepository) GetUserDashboardById(ctx context.Context, userID uuid.UUID) (user UserDashboardRes, err error) {

	start := time.Now()
	defer func() {
		elapsed := time.Since(start).Microseconds()
		r.Logger.GetLogger().Info("total execution time", zap.Int64("duration", elapsed))
	}()

	RedisCacheKey := fmt.Sprintf("user:dashboard:%s", userID.String())
	//get data if present
	cachedData, err := r.Redis.Get(ctx, RedisCacheKey)
	if err == nil && cachedData != "" {
		r.Logger.GetLogger().Info("user dashboard found in Redis cache", zap.String("user_id", userID.String()))
		err = json.Unmarshal([]byte(cachedData), &user)
		if err == nil {
			return user, nil
		}
		r.Logger.GetLogger().Warn("failed to unmarshal cached dashboard, fetching from DB", zap.Error(err))
	}

	r.Logger.GetLogger().Info("starting transaction to get user dashboard by id", zap.String("user_id", userID.String()))
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		r.Logger.GetLogger().Error("failed to begin transaction", zap.Error(err))
		return user, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			r.Logger.GetLogger().Error("panic recovered", zap.Any("recover_info", p))
			panic(p)
		} else if err != nil {
			tx.Rollback()
			r.Logger.GetLogger().Error("rolling back transaction", zap.Error(err))
		} else {
			err = tx.Commit()
			if err != nil {
				r.Logger.GetLogger().Error("failed to commit transaction", zap.Error(err))
			} else {
				r.Logger.GetLogger().Info("transaction committed successfully")
			}
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

	jsonData, err := json.Marshal(user)
	if err == nil {
		_ = r.Redis.Set(ctx, RedisCacheKey, jsonData, 5*time.Minute)
		r.Logger.GetLogger().Info("user dashboard cached in Redis", zap.String("user_id", userID.String()))
		fmt.Println(time.Now().Format(time.RFC3339))
	}

	return user, nil
}

//// /GetUserDashboard
//func (r *PostgresUserRepository) GetUserDashboardById(ctx context.Context, userID uuid.UUID) (user UserDashboardRes, err error) {
//	r.Logger.GetLogger().Info("starting transaction to get user dashboard by id", zap.String("user_id", userID.String()))
//	tx, err := r.DB.BeginTxx(ctx, nil)
//	if err != nil {
//		r.Logger.GetLogger().Error("failed to begin transaction for getuserdashboardbyid", zap.Error(err))
//		return user, fmt.Errorf("failed to begin transaction: %w", err)
//	}
//	defer func() {
//		if p := recover(); p != nil {
//			tx.Rollback()
//			r.Logger.GetLogger().Error("panic recovered during getuserdashboardbyid transaction", zap.Any("recover_info", p))
//			panic(p)
//		} else if err != nil {
//			tx.Rollback()
//			r.Logger.GetLogger().Error("rolling back transaction for getuserdashboardbyid due to error", zap.Error(err))
//		} else {
//			err = tx.Commit()
//			if err != nil {
//				r.Logger.GetLogger().Error("failed to commit transaction for getuserdashboardbyid", zap.Error(err))
//			} else {
//				r.Logger.GetLogger().Info("transaction committed successfully for getuserdashboardbyid")
//			}
//		}
//	}()
//
//	r.Logger.GetLogger().Debug("fetching user details for dashboard", zap.String("user_id", userID.String()))
//	err = tx.GetContext(ctx, &user, `
//		SELECT u.id, u.username, u.email, u.contact_no, ut.type
//		FROM users u
//		LEFT JOIN user_type ut ON ut.user_id = u.id AND ut.archived_at IS NULL
//		WHERE u.id = $1 AND u.archived_at IS NULL
//	`, userID)
//	if err != nil {
//		r.Logger.GetLogger().Error("failed to fetch user details for dashboard", zap.String("user_id", userID.String()), zap.Error(err))
//		return user, fmt.Errorf("failed to fetch user: %w", err)
//	}
//	r.Logger.GetLogger().Debug("user details fetched for dashboard", zap.String("user_id", userID.String()))
//
//	r.Logger.GetLogger().Debug("fetching user roles for dashboard", zap.String("user_id", userID.String()))
//	err = tx.SelectContext(ctx, &user.Roles, `
//		SELECT role FROM user_roles
//		WHERE user_id = $1 AND archived_at IS NULL
//	`, userID)
//	if err != nil {
//		r.Logger.GetLogger().Error("failed to fetch roles for dashboard", zap.String("user_id", userID.String()), zap.Error(err))
//		return user, fmt.Errorf("failed to fetch roles: %w", err)
//	}
//	r.Logger.GetLogger().Debug("user roles fetched for dashboard", zap.String("user_id", userID.String()))
//
//	r.Logger.GetLogger().Debug("fetching assigned assets for dashboard", zap.String("user_id", userID.String()))
//	err = tx.SelectContext(ctx, &user.AssignedAssets, `
//		SELECT a.id, a.brand, a.model, a.serial_no, a.type, a.status, a.owned_by
//		FROM assets a
//		INNER JOIN asset_assign aa ON aa.asset_id = a.id
//		WHERE aa.employee_id = $1 AND aa.returned_at IS NULL AND aa.archived_at IS NULL AND a.archived_at IS NULL
//	`, userID)
//	if err != nil {
//		r.Logger.GetLogger().Error("failed to fetch assigned assets for dashboard", zap.String("user_id", userID.String()), zap.Error(err))
//		return user, fmt.Errorf("failed to fetch assigned assets: %w", err)
//	}
//	r.Logger.GetLogger().Info("successfully fetched user dashboard by id", zap.String("user_id", userID.String()))
//	return user, nil
//}

func (r *PostgresUserRepository) GetUserRoleById(ctx context.Context, userId uuid.UUID) (string, error) {
	r.Logger.GetLogger().Info("fetching user role by id", zap.String("user_id", userId.String()))

	redisKey := fmt.Sprintf("user:GetUserRoleById:%s", userId.String())

	//getting data from redis if present
	if cachedData, err := r.Redis.Get(ctx, redisKey); err == nil && cachedData != "" {
		r.Logger.GetLogger().Info("user role found in Redis cache", zap.String("user_id", userId.String()))
		return cachedData, nil
	}

	//if not run db query
	var userRole string
	err := r.DB.GetContext(ctx, &userRole, `
		SELECT role FROM user_roles 
		WHERE user_id = $1 AND archived_at IS NULL
	`, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Logger.GetLogger().Warn("no role found for user id", zap.String("user_id", userId.String()), zap.Error(err))
			return "", fmt.Errorf("no role found for user id: %s", userId)
		}
		r.Logger.GetLogger().Error("failed to fetch user role", zap.String("user_id", userId.String()), zap.Error(err))
		return "", fmt.Errorf("failed to fetch user role: %w", err)
	}

	cacheErr := r.Redis.Set(ctx, redisKey, userRole, 5*time.Minute)
	if cacheErr != nil {
		r.Logger.GetLogger().Warn("failed to cache user role in Redis", zap.Error(cacheErr))
	} else {
		r.Logger.GetLogger().Info("cached user role in Redis", zap.String("user_id", userId.String()))
	}

	return userRole, nil
}

func (r *PostgresUserRepository) GetUserAssetTimeline(ctx context.Context, userID uuid.UUID) ([]UserTimelineRes, error) {
	r.Logger.GetLogger().Info("fetching user asset timeline", zap.String("user_id", userID.String()))
	timeline := make([]UserTimelineRes, 0)

	//generate key
	redisKey := fmt.Sprintf("user:GetUserAssetTimeline:%s", userID.String())

	//get data from redis, if preset
	if cached, err := r.Redis.Get(ctx, redisKey); err == nil && cached != "" {
		r.Logger.GetLogger().Info("user asset timeline found in Redis cache", zap.String("user_id", userID.String()))
		if err := json.Unmarshal([]byte(cached), &timeline); err == nil {
			return timeline, nil
		}
		r.Logger.GetLogger().Warn("failed to unmarshal cached timeline, falling back to DB", zap.Error(err))
	}

	//if not present in redis, run query and then store data
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
		r.Logger.GetLogger().Error("failed to get user timeline", zap.String("user_id", userID.String()), zap.Error(err))
		return nil, fmt.Errorf("failed to get user timeline: %w", err)
	}

	//store data in cache
	cacheBytes, err := json.Marshal(timeline)
	if err == nil {
		cacheErr := r.Redis.Set(ctx, redisKey, string(cacheBytes), 5*time.Minute)
		if cacheErr != nil {
			r.Logger.GetLogger().Warn("failed to cache asset timeline in Redis", zap.Error(cacheErr))
		} else {
			r.Logger.GetLogger().Info("cached user asset timeline in Redis", zap.String("user_id", userID.String()))
		}
	}

	r.Logger.GetLogger().Info("successfully fetched user asset timeline", zap.String("user_id", userID.String()), zap.Int("timeline_entries", len(timeline)))
	return timeline, nil
}

func (r *PostgresUserRepository) CreateNewEmployee(ctx context.Context, tx *sqlx.Tx, req ManagerRegisterReq, managerUUID uuid.UUID) (uuid.UUID, error) {
	r.Logger.GetLogger().Info("creating new employee record", zap.String("email", req.Email), zap.String("manager_id", managerUUID.String()))
	var userID uuid.UUID
	err := tx.GetContext(ctx, &userID, `
		INSERT INTO users (username, email, contact_no, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, req.Username, req.Email, req.ContactNo, managerUUID)
	if err != nil {
		r.Logger.GetLogger().Error("failed to insert new employee into users table", zap.Error(err))
		return uuid.Nil, fmt.Errorf("failed to insert employee: %w", err)
	}
	r.Logger.GetLogger().Debug("new employee inserted into users table", zap.String("user_id", userID.String()))

	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_type (user_id, type, created_by)
		VALUES ($1, $2, $3)
	`, userID, req.Type, managerUUID)
	if err != nil {
		r.Logger.GetLogger().Error("failed to insert employee type", zap.String("user_id", userID.String()), zap.Error(err))
		return uuid.Nil, fmt.Errorf("failed to insert employee type: %w", err)
	}
	r.Logger.GetLogger().Debug("employee type inserted", zap.String("user_id", userID.String()), zap.String("type", req.Type))

	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_roles (user_id, role, created_by)
		VALUES ($1, 'employee', $2)
	`, userID, managerUUID)
	if err != nil {
		r.Logger.GetLogger().Error("failed to insert employee role", zap.String("user_id", userID.String()), zap.Error(err))
		return uuid.Nil, fmt.Errorf("failed to insert employee role: %w", err)
	}
	r.Logger.GetLogger().Info("new employee created successfully", zap.String("user_id", userID.String()))
	return userID, nil
}

func (r *PostgresUserRepository) GetFilteredEmployeesWithAssets(ctx context.Context, filter EmployeeFilter) ([]EmployeeResponseModel, error) {
	r.Logger.GetLogger().Info("fetching filtered employees with assets", zap.Any("filter", filter))
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
		r.Logger.GetLogger().Error("failed to select filtered employees with assets", zap.Error(err), zap.Any("filter", filter))
		return nil, err
	}
	r.Logger.GetLogger().Info("successfully fetched filtered employees with assets", zap.Int("count", len(rows)))
	return rows, nil
}

func (r *PostgresUserRepository) UpdateEmployeeInfo(ctx context.Context, req UpdateEmployeeReq, adminUUID uuid.UUID) error {
	r.Logger.GetLogger().Info("updating employee information", zap.String("admin_id", adminUUID.String()))
	query := `UPDATE users SET`
	args := []interface{}{}
	argPos := 1

	if req.Username != "" {
		query += fmt.Sprintf("username = $%d, ", argPos)
		args = append(args, req.Username)
		argPos++
		r.Logger.GetLogger().Debug("updating username", zap.String("username", req.Username))
	}
	if req.Email != "" {
		query += fmt.Sprintf("email = $%d, ", argPos)
		args = append(args, req.Email)
		argPos++
		r.Logger.GetLogger().Debug("updating email", zap.String("email", req.Email))
	}
	if req.ContactNo != "" {
		query += fmt.Sprintf("contact_no = $%d, ", argPos)
		args = append(args, req.ContactNo)
		argPos++
		r.Logger.GetLogger().Debug("updating contact_no", zap.String("contact_no", req.ContactNo))
	}

	query += fmt.Sprintf("updated_by = $%d ", argPos)
	args = append(args, adminUUID)
	argPos++

	query = strings.TrimSuffix(query, ", ")
	query += fmt.Sprintf("WHERE id = $%d AND archived_at IS NULL", argPos)
	args = append(args, req.UserID)

	result, err := r.DB.ExecContext(ctx, query, args...)
	if err != nil {
		r.Logger.GetLogger().Error("failed to update user in database")
		return fmt.Errorf("failed to update user: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		r.Logger.GetLogger().Warn("no user found or nothing updated for employee update")
		return fmt.Errorf("no user found or nothing updated")
	}
	r.Logger.GetLogger().Info("employee information updated successfully")
	return nil
}

func (r *PostgresUserRepository) InsertUserRole(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID, role string, createdBy uuid.UUID) error {
	r.Logger.GetLogger().Info("inserting new user role", zap.String("user_id", userID.String()), zap.String("role", role), zap.String("created_by", createdBy.String()))
	_, err := tx.ExecContext(ctx, `
		INSERT INTO user_roles (role, user_id, created_by)
		VALUES ($1, $2, $3)
	`, role, userID, createdBy)
	if err != nil {
		r.Logger.GetLogger().Error("failed to insert new role", zap.String("user_id", userID.String()), zap.String("role", role), zap.Error(err))
		return fmt.Errorf("failed to insert new role: %w", err)
	}
	r.Logger.GetLogger().Info("user role inserted successfully", zap.String("user_id", userID.String()), zap.String("role", role))
	return nil
}

func (r *PostgresUserRepository) UpdateUserRole(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID, newRole string, updatedBy uuid.UUID) error {
	r.Logger.GetLogger().Info("updating user role", zap.String("user_id", userID.String()), zap.String("new_role", newRole), zap.String("updated_by", updatedBy.String()))
	currentRole, err := r.GetCurrentUserRole(ctx, tx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		r.Logger.GetLogger().Error("failed to fetch current role for user role update", zap.String("user_id", userID.String()), zap.Error(err))
		return fmt.Errorf("failed to fetch current role: %w", err)
	}
	if err == nil && currentRole == newRole {
		r.Logger.GetLogger().Warn("user already has the requested role, no update needed", zap.String("user_id", userID.String()), zap.String("role", newRole))
		return fmt.Errorf("user already has the role: %s", newRole)
	}
	if err := r.ArchiveUserRoles(ctx, tx, userID); err != nil {
		r.Logger.GetLogger().Error("failed to archive old user roles before updating", zap.String("user_id", userID.String()), zap.Error(err))
		return err
	}
	if err := r.InsertUserRole(ctx, tx, userID, newRole, updatedBy); err != nil {
		r.Logger.GetLogger().Error("failed to insert new user role after archiving old roles", zap.String("user_id", userID.String()), zap.String("new_role", newRole), zap.Error(err))
		return err
	}
	r.Logger.GetLogger().Info("user role updated successfully", zap.String("user_id", userID.String()), zap.String("new_role", newRole))
	return nil
}

func (r *PostgresUserRepository) GetCurrentUserRole(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID) (string, error) {
	r.Logger.GetLogger().Debug("getting current user role", zap.String("user_id", userID.String()))
	var role string

	err := tx.GetContext(ctx, &role, `
		SELECT role FROM user_roles
		WHERE user_id = $1 AND archived_at IS NULL
	`, userID)
	if err != nil {
		r.Logger.GetLogger().Warn("no current role found for user or error fetching", zap.String("user_id", userID.String()), zap.Error(err))
		return "", err
	}
	r.Logger.GetLogger().Debug("current user role fetched", zap.String("user_id", userID.String()), zap.String("role", role))
	return role, nil
}

func (r *PostgresUserRepository) ArchiveUserRoles(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID) error {
	r.Logger.GetLogger().Debug("archiving user roles for user", zap.String("user_id", userID.String()))
	_, err := tx.ExecContext(ctx, `
		UPDATE user_roles
		SET archived_at = now(), last_updated_at = now()
		WHERE user_id = $1 AND archived_at IS NULL
	`, userID)
	if err != nil {
		r.Logger.GetLogger().Error("failed to archive existing roles", zap.String("user_id", userID.String()), zap.Error(err))
		return fmt.Errorf("failed to archive existing roles: %w", err)
	}
	r.Logger.GetLogger().Debug("user roles archived successfully", zap.String("user_id", userID.String()))
	return nil
}

func (r *PostgresUserRepository) IsUserExists(ctx context.Context, tx *sqlx.Tx, email string) (bool, error) {
	r.Logger.GetLogger().Info("checking if user exists by email", zap.String("email", email))

	redisKey := fmt.Sprintf("user:IsUserExists:%s", email)

	//get value from cache if present
	if cached, err := r.Redis.Get(ctx, redisKey); err == nil && cached != "" {
		r.Logger.GetLogger().Info("user existence found in Redis cache", zap.String("email", email))
		return cached == "true", nil
	}

	//run query and store value in redis
	var id uuid.UUID
	err := tx.QueryRowContext(ctx, `
		SELECT id FROM users 
		WHERE email = $1 AND archived_at IS NULL
	`, email).Scan(&id)

	if err == sql.ErrNoRows {
		r.Logger.GetLogger().Debug("user does not exist", zap.String("email", email))
		_ = r.Redis.Set(ctx, redisKey, "false", 10*time.Minute)
		return false, nil
	}
	if err != nil {
		r.Logger.GetLogger().Error("failed to check if user exists", zap.String("email", email), zap.Error(err))
		return false, fmt.Errorf("failed to check existing user: %w", err)
	}

	r.Logger.GetLogger().Info("user exists", zap.String("user_id", id.String()), zap.String("email", email))
	_ = r.Redis.Set(ctx, redisKey, "true", 10*time.Minute)
	return true, nil
}

func (r *PostgresUserRepository) InsertIntoUser(ctx context.Context, tx *sqlx.Tx, username, email string, firebasetoken string) (uuid.UUID, error) {
	r.Logger.GetLogger().Info("inserting new user into users table", zap.String("username", username), zap.String("email", email))
	var id uuid.UUID
	err := tx.GetContext(ctx, &id, `
		INSERT INTO users (username, email, firebase_uid)
		VALUES ($1, $2, $3)
		RETURNING id
	`, username, email, firebasetoken)
	if err != nil {
		r.Logger.GetLogger().Error("failed to insert into users table", zap.Error(err))
		return uuid.Nil, fmt.Errorf("failed to insert user: %w", err)
	}
	r.Logger.GetLogger().Debug("user inserted, updating created_by field", zap.String("user_id", id.String()))

	_, err = tx.ExecContext(ctx, `
		UPDATE users SET created_by = $1 WHERE id = $1
	`, id)
	if err != nil {
		r.Logger.GetLogger().Error("failed to update created_by for new user", zap.String("user_id", id.String()), zap.Error(err))
		return uuid.Nil, fmt.Errorf("failed to update created_by: %w", err)
	}
	r.Logger.GetLogger().Info("new user inserted and created_by updated", zap.String("user_id", id.String()))
	return id, nil
}

func (r *PostgresUserRepository) InsertIntoUserRole(ctx context.Context, tx *sqlx.Tx, userId uuid.UUID, role string, createdBy uuid.UUID) error {
	r.Logger.GetLogger().Info("inserting user role", zap.String("user_id", userId.String()), zap.String("role", role), zap.String("created_by", createdBy.String()))
	_, err := tx.ExecContext(ctx, `
		INSERT INTO user_roles (role, user_id, created_by)
		VALUES ($1, $2, $3)
	`, role, userId, createdBy)
	if err != nil {
		r.Logger.GetLogger().Error("failed to insert user role", zap.String("user_id", userId.String()), zap.String("role", role), zap.Error(err))
		return fmt.Errorf("failed to insert user role: %w", err)
	}
	r.Logger.GetLogger().Info("user role inserted successfully", zap.String("user_id", userId.String()), zap.String("role", role))
	return nil
}

func (r *PostgresUserRepository) InsertIntoUserType(ctx context.Context, tx *sqlx.Tx, userId uuid.UUID, employeeType string, createdBy uuid.UUID) error {
	r.Logger.GetLogger().Info("inserting user type", zap.String("user_id", userId.String()), zap.String("employee_type", employeeType), zap.String("created_by", createdBy.String()))
	_, err := tx.ExecContext(ctx, `
		INSERT INTO user_type (type, user_id, created_by)
		VALUES ($1, $2, $3)
	`, employeeType, userId, createdBy)
	if err != nil {
		r.Logger.GetLogger().Error("failed to insert user type", zap.String("user_id", userId.String()), zap.String("employee_type", employeeType), zap.Error(err))
		return fmt.Errorf("failed to insert user type: %w", err)
	}
	r.Logger.GetLogger().Info("user type inserted successfully", zap.String("user_id", userId.String()), zap.String("employee_type", employeeType))
	return nil
}

func (r *PostgresUserRepository) GetEmailByUserID(ctx context.Context, userId uuid.UUID) (string, error) {
	redisKey := fmt.Sprintf("user:GetEmailByUserID:%s", userId.String())
	//get data from redis, if present
	if cached, err := r.Redis.Get(ctx, redisKey); err == nil && cached != "" {
		r.Logger.GetLogger().Info("user email found in Redis cache", zap.String("user_id", userId.String()))
		return cached, nil
	}

	//if not present for user id , run query and then store in redis cace
	var userMail string
	err := r.DB.GetContext(ctx, &userMail, `SELECT email FROM users WHERE id = $1 AND archived_at IS NULL`, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Logger.GetLogger().Warn("no user found with given ID", zap.String("user_id", userId.String()))
			return "", nil
		}
		r.Logger.GetLogger().Error("failed to get user email", zap.String("user_id", userId.String()), zap.Error(err))
		return "", fmt.Errorf("failed to get user email: %w", err)
	}

	// Cache result in Redis
	if err := r.Redis.Set(ctx, redisKey, userMail, 5*time.Minute); err != nil {
		r.Logger.GetLogger().Warn("failed to cache user email in Redis", zap.Error(err))
	}

	return userMail, nil
}

func (r *PostgresUserRepository) CreateFirebaseUser(ctx context.Context, name, email string) (userID uuid.UUID, err error) {
	r.Logger.GetLogger().Info("creating firebase user in postgres repository", zap.String("name", name), zap.String("email", email))
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		r.Logger.GetLogger().Error("failed to begin transaction for createfirebaseuser", zap.Error(err))
		return uuid.Nil, err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			r.Logger.GetLogger().Error("panic recovered during createfirebaseuser transaction", zap.Any("recover_info", p))
			panic(p)
		} else if err != nil {
			_ = tx.Rollback()
			r.Logger.GetLogger().Error("rolling back transaction for createfirebaseuser due to error", zap.Error(err))
		} else {
			err = tx.Commit()
			if err != nil {
				r.Logger.GetLogger().Error("failed to commit transaction for createfirebaseuser", zap.Error(err))
			} else {
				r.Logger.GetLogger().Info("transaction committed successfully for createfirebaseuser")
			}
		}
	}()

	userID, err = r.InsertIntoUser(ctx, tx, name, email, "")
	if err != nil {
		r.Logger.GetLogger().Error("failed to insert user during firebase user creation", zap.Error(err))
		return uuid.Nil, err
	}
	err = r.InsertIntoUserRole(ctx, tx, userID, "employee", userID)
	if err != nil {
		r.Logger.GetLogger().Error("failed to insert user role during firebase user creation", zap.Error(err))
		return uuid.Nil, err
	}
	err = r.InsertIntoUserType(ctx, tx, userID, "full_time", userID)
	if err != nil {
		r.Logger.GetLogger().Error("failed to insert user type during firebase user creation", zap.Error(err))
		return uuid.Nil, err
	}
	r.Logger.GetLogger().Info("firebase user created successfully in postgres", zap.String("user_id", userID.String()))
	return userID, nil
}

func (r *PostgresUserRepository) GetFirebase() providers.FirebaseProvider {
	r.Logger.GetLogger().Debug("getting firebase provider instance")
	return r.Firebase
}
