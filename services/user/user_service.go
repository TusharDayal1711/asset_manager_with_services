package userservice

import (
	"asset/middlewares"
	"asset/models"
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"strings"
)

type UserService interface {
	ChangeUserRole(ctx context.Context, req models.UpdateUserRoleReq, adminID uuid.UUID) error
	DeleteUser(ctx context.Context, userID uuid.UUID, managerRole string) error
	GetEmployeesWithFilters(ctx context.Context, filter models.EmployeeFilter) ([]models.EmployeeResponseModel, error)
	GetEmployeeTimeline(ctx context.Context, userID uuid.UUID) ([]models.UserTimelineRes, error)
	PublicRegister(ctx context.Context, req models.PublicUserReq) (uuid.UUID, error)
	RegisterEmployeeByManager(ctx context.Context, req models.ManagerRegisterReq, managerID uuid.UUID) (uuid.UUID, error)
	UpdateEmployee(ctx context.Context, req models.UpdateEmployeeReq, managerID uuid.UUID) error
	GetDashboard(ctx context.Context, userID uuid.UUID) (models.UserDashboardRes, error)
	UserLogin(ctx context.Context, req models.PublicUserReq) (uuid.UUID, string, string, error)
}

type userService struct {
	repo UserRepository
	db   *sqlx.DB
}

func NewUserService(repo UserRepository, db *sqlx.DB) UserService {
	return &userService{repo: repo, db: db}
}

func (s *userService) ChangeUserRole(ctx context.Context, req models.UpdateUserRoleReq, adminID uuid.UUID) error {
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

func (s *userService) GetEmployeesWithFilters(ctx context.Context, filter models.EmployeeFilter) ([]models.EmployeeResponseModel, error) {
	return s.repo.GetFilteredEmployeesWithAssets(ctx, filter)
}

func (s *userService) GetEmployeeTimeline(ctx context.Context, userID uuid.UUID) ([]models.UserTimelineRes, error) {
	return s.repo.GetUserAssetTimeline(ctx, userID)
}

func (s *userService) PublicRegister(ctx context.Context, req models.PublicUserReq) (uuid.UUID, error) {
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

	splitEmail := strings.Split(req.Email, "@")
	if len(splitEmail) != 2 || splitEmail[1] != "remotestate.com" {
		return uuid.Nil, errors.New("only remotestate.com domain is valid")
	}
	usernameParts := strings.Split(splitEmail[0], ".")
	if len(usernameParts) != 2 {
		return uuid.Nil, errors.New("invalid email format for username")
	}
	username := usernameParts[0] + " " + usernameParts[1]

	exists, err := s.repo.IsUserExists(ctx, tx, req.Email)
	if err != nil {
		return uuid.Nil, err
	}
	if exists {
		return uuid.Nil, errors.New("email already registered")
	}

	userID, err := s.repo.InsertIntoUser(ctx, tx, username, req.Email)
	if err != nil {
		return uuid.Nil, err
	}

	if err = s.repo.InsertIntoUserRole(ctx, tx, userID, "employee", userID); err != nil {
		return uuid.Nil, err
	}
	if err = s.repo.InsertIntoUserType(ctx, tx, userID, "full_time", userID); err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

func (s *userService) RegisterEmployeeByManager(ctx context.Context, req models.ManagerRegisterReq, managerID uuid.UUID) (uuid.UUID, error) {
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

func (s *userService) UpdateEmployee(ctx context.Context, req models.UpdateEmployeeReq, managerID uuid.UUID) error {
	return s.repo.UpdateEmployeeInfo(ctx, req, managerID)
}

func (s *userService) GetDashboard(ctx context.Context, userID uuid.UUID) (models.UserDashboardRes, error) {
	return s.repo.GetUserDashboardById(ctx, userID)
}

func (s *userService) UserLogin(ctx context.Context, req models.PublicUserReq) (uuid.UUID, string, string, error) {
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
