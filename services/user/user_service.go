package userservice

import (
	"asset/middlewares"
	"asset/utils"
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"strings"
)

type UserService interface {
	ChangeUserRole(ctx context.Context, req UpdateUserRoleReq, adminID uuid.UUID) error
	DeleteUser(ctx context.Context, userID uuid.UUID, managerRole string) error
	GetEmployeesWithFilters(ctx context.Context, filter EmployeeFilter) ([]EmployeeResponseModel, error)
	GetEmployeeTimeline(ctx context.Context, userID uuid.UUID) ([]UserTimelineRes, error)
	PublicRegister(ctx context.Context, req PublicUserReq) (uuid.UUID, error)
	RegisterEmployeeByManager(ctx context.Context, req ManagerRegisterReq, managerID uuid.UUID) (uuid.UUID, error)
	UpdateEmployee(ctx context.Context, req UpdateEmployeeReq, managerID uuid.UUID) error
	GetDashboard(ctx context.Context, userID uuid.UUID) (UserDashboardRes, error)
	UserLogin(ctx context.Context, req PublicUserReq) (uuid.UUID, string, string, error)
}

type userService struct {
	repo UserRepository
	db   *sqlx.DB
}

func NewUserService(repo UserRepository, db *sqlx.DB) UserService {
	return &userService{repo: repo, db: db}
}

func (s *userService) ChangeUserRole(ctx context.Context, req UpdateUserRoleReq, adminID uuid.UUID) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil || err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	userUUID, err := uuid.Parse(req.UserID)
	if err != nil {
		return err
	}

	err = s.repo.UpdateUserRole(ctx, tx, userUUID, req.Role, adminID)
	if err != nil && strings.Contains(err.Error(), "already has the role") {
		return errors.New("user already has this role")
	}
	return err
}

func (s *userService) DeleteUser(ctx context.Context, userID uuid.UUID, managerRole string) error {
	userRole, err := s.repo.GetUserRoleById(ctx, userID)
	if err != nil {
		return err
	}

	if managerRole != "admin" && (userRole == "admin" || userRole == "asset_manager" || userRole == "inventory_manager") {
		return errors.New("only admin can delete admin or manager roles")
	}

	return s.repo.DeleteUserByID(ctx, userID)
}

func (s *userService) GetEmployeesWithFilters(ctx context.Context, filter EmployeeFilter) ([]EmployeeResponseModel, error) {
	return s.repo.GetFilteredEmployeesWithAssets(ctx, filter)
}

func (s *userService) GetEmployeeTimeline(ctx context.Context, userID uuid.UUID) ([]UserTimelineRes, error) {
	return s.repo.GetUserAssetTimeline(ctx, userID)
}

func (s *userService) PublicRegister(ctx context.Context, req PublicUserReq) (uuid.UUID, error) {
	utils.Logger.Info("inside publicregistration service...", zap.String("email", req.Email))
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		utils.Logger.Error("Failed to begin transaction", zap.Error(err))
		return uuid.Nil, err
	}
	defer func() {
		if r := recover(); r != nil {
			utils.Logger.Error("Panic recovered in PublicRegister", zap.Any("recover_info", r))
			tx.Rollback()
		} else if err != nil {
			utils.Logger.Error("Rolling back transaction due to error", zap.Error(err))
			tx.Rollback()
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				utils.Logger.Error("Failed to commit transaction", zap.Error(commitErr))
			} else {
				utils.Logger.Info("Transaction committed successfully")
			}
		}
	}()

	splitEmail := strings.Split(req.Email, "@")
	if len(splitEmail) != 2 || splitEmail[1] != "remotestate.com" {
		utils.Logger.Error("Invalid email domain", zap.String("input email ::", req.Email), zap.String("Required ::", "firstname.secondname@remotestate.com"))
		return uuid.Nil, errors.New("only remotestate.com domain is valid")
	}

	usernameParts := strings.Split(splitEmail[0], ".")
	if len(usernameParts) != 2 {
		utils.Logger.Error("Invalid email format for username", zap.String("email", req.Email))
		return uuid.Nil, errors.New("invalid email format for username")
	}
	username := usernameParts[0] + " " + usernameParts[1]

	utils.Logger.Debug("Parsed username from email ", zap.String("username", username))

	exists, err := s.repo.IsUserExists(ctx, tx, req.Email)
	if err != nil {
		utils.Logger.Error("Failed to check if user exists", zap.Error(err))
		return uuid.Nil, err
	}
	if exists {
		utils.Logger.Warn("user already registered", zap.String("email", req.Email))
		return uuid.Nil, errors.New("email already registered...")
	}

	userID, err := s.repo.InsertIntoUser(ctx, tx, username, req.Email)
	if err != nil {
		utils.Logger.Error("failed to insert into users table...", zap.Error(err))
		return uuid.Nil, err
	}
	utils.Logger.Info("new user inserted into users table", zap.String("user_id", userID.String()))

	if err = s.repo.InsertIntoUserRole(ctx, tx, userID, "employee", userID); err != nil {
		utils.Logger.Error("failed to insert user role", zap.Error(err))
		return uuid.Nil, err
	}
	utils.Logger.Debug("assigned user role 'employee'", zap.String("user_id", userID.String()))

	if err = s.repo.InsertIntoUserType(ctx, tx, userID, "full_time", userID); err != nil {
		utils.Logger.Error("failed to insert user type", zap.Error(err))
		return uuid.Nil, err
	}
	utils.Logger.Debug("assigned user type 'full_time'", zap.String("user_id", userID.String()))
	return userID, nil
}

func (s *userService) RegisterEmployeeByManager(ctx context.Context, req ManagerRegisterReq, managerID uuid.UUID) (uuid.UUID, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return uuid.Nil, err
	}
	defer func() {
		if r := recover(); r != nil || err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	return s.repo.CreateNewEmployee(ctx, tx, req, managerID)
}

func (s *userService) UpdateEmployee(ctx context.Context, req UpdateEmployeeReq, managerID uuid.UUID) error {
	return s.repo.UpdateEmployeeInfo(ctx, req, managerID)
}

func (s *userService) GetDashboard(ctx context.Context, userID uuid.UUID) (UserDashboardRes, error) {
	return s.repo.GetUserDashboardById(ctx, userID)
}

func (s *userService) UserLogin(ctx context.Context, req PublicUserReq) (uuid.UUID, string, string, error) {
	userID, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, "", "", errors.New("invalid email")
		}
		return uuid.Nil, "", "", err
	}

	userRole, err := s.repo.GetUserRoleById(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, "", "", errors.New("invalid email")
		}
		return uuid.Nil, "", "", err
	}

	accessToken, err := middlewares.GenerateJWT(userID.String(), []string{userRole})
	if err != nil {
		return uuid.Nil, "", "", err
	}
	refreshToken, err := middlewares.GenerateRefreshToken(userID.String())
	if err != nil {
		return uuid.Nil, "", "", err
	}
	return userID, accessToken, refreshToken, nil
}
