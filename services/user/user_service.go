package userservice

import (
	"asset/middlewares"
	"asset/providers"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	firebaseauth "firebase.google.com/go/v4/auth"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type UserService interface {
	ChangeUserRole(ctx context.Context, req UpdateUserRoleReq, adminID uuid.UUID) error
	DeleteUser(ctx context.Context, userID uuid.UUID, managerRole string) error
	GetEmployeesWithFilters(ctx context.Context, filter EmployeeFilter) ([]EmployeeResponseModel, error)
	GetEmployeeTimeline(ctx context.Context, userID uuid.UUID) ([]UserTimelineRes, error)
	PublicRegister(ctx context.Context, req PublicUserReq) (uuid.UUID, string, error)
	RegisterEmployeeByManager(ctx context.Context, req ManagerRegisterReq, managerID uuid.UUID) (uuid.UUID, error)
	UpdateEmployee(ctx context.Context, req UpdateEmployeeReq, managerID uuid.UUID) error
	GetDashboard(ctx context.Context, userID uuid.UUID) (UserDashboardRes, error)
	UserLogin(ctx context.Context, req PublicUserReq) (uuid.UUID, string, string, error)
	GoogleAuth(ctx context.Context, idToken string) (uuid.UUID, string, string, error)
	CreateFirstAdmin() bool
	FirebaseUserRegistration(ctx context.Context, idToken string) (*FirebaseRegistrationResponse, error)
}

type userServiceStruct struct {
	repo           UserRepository
	db             *sqlx.DB
	logger         providers.ZapLoggerProvider
	firebase       providers.FirebaseProvider
	AuthMiddleware providers.AuthMiddlewareService
}

func NewUserService(repo UserRepository, db *sqlx.DB, logger providers.ZapLoggerProvider, firebase providers.FirebaseProvider, AuthMiddleware providers.AuthMiddlewareService) UserService {
	return &userServiceStruct{repo: repo, db: db, logger: logger, firebase: firebase, AuthMiddleware: AuthMiddleware}
}

func (s *userServiceStruct) ChangeUserRole(ctx context.Context, req UpdateUserRoleReq, adminID uuid.UUID) error {
	s.logger.GetLogger().Info("change user role", zap.String("targetUserID", req.UserID), zap.String("newRole", req.Role), zap.String("adminID", adminID.String()))
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		s.logger.GetLogger().Error("failed to begin transaction for ChangeUserRole", zap.Error(err))
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			s.logger.GetLogger().Error("panic recovered during ChangeUserRole transaction", zap.Any("recover_info", r))
			tx.Rollback()
		} else if err != nil {
			s.logger.GetLogger().Error("rolling back transaction for ChangeUserRole", zap.Error(err))
			tx.Rollback()
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				s.logger.GetLogger().Error("failed to commit transaction for ChangeUserRole", zap.Error(commitErr))
			} else {
				s.logger.GetLogger().Info("transaction committed successfully for ChangeUserRole")
			}
		}
	}()

	userUUID, err := uuid.Parse(req.UserID)
	if err != nil {
		s.logger.GetLogger().Error("failed to parse userID in ChangeUserRole", zap.String("userID", req.UserID), zap.Error(err))
		return err
	}

	err = s.repo.UpdateUserRole(ctx, tx, userUUID, req.Role, adminID)
	if err != nil {
		if strings.Contains(err.Error(), "already has the role") {
			s.logger.GetLogger().Warn("user already has the requested role", zap.String("userID", req.UserID), zap.String("role", req.Role))
			return errors.New("user already has this role")
		}
		s.logger.GetLogger().Error("Failed to update user role in repository", zap.String("userID", req.UserID), zap.Error(err))
		return err
	}
	s.logger.GetLogger().Info("User role updated successfully", zap.String("userID", req.UserID), zap.String("newRole", req.Role))
	return nil
}

func (s *userServiceStruct) DeleteUser(ctx context.Context, userID uuid.UUID, managerRole string) error {
	s.logger.GetLogger().Info("inside delete user", zap.String("userID", userID.String()), zap.String("managerRole", managerRole))
	userRole, err := s.repo.GetUserRoleById(ctx, userID)
	if err != nil {
		s.logger.GetLogger().Error("failed to get user role by ID for deletion", zap.String("userID", userID.String()), zap.Error(err))
		return err
	}
	s.logger.GetLogger().Debug("retrieved user role for deletion target", zap.String("userID", userID.String()), zap.String("userRole", userRole))

	if managerRole != "admin" && (userRole == "admin" || userRole == "asset_manager" || userRole == "inventory_manager") {
		s.logger.GetLogger().Warn("unauthorized attempt to delete privileged user role", zap.String("managerRole", managerRole), zap.String("targetUserRole", userRole))
		return errors.New("only admin can delete admin or manager roles")
	}

	userEmail, err := s.repo.GetEmailByUserID(ctx, userID)
	if err != nil {
		s.logger.GetLogger().Error("failed to get user email for deletion", zap.String("userID", userID.String()), zap.Error(err))
		return errors.New("failed to get user email from user table")
	}

	firebaseUserRecords, err := s.firebase.GetUserByEmail(ctx, userEmail)
	if err != nil {
		s.logger.GetLogger().Error("failed to get user UID from firebase user table", zap.String("userID", userID.String()), zap.Error(err))
		return errors.New("failed to get user UID from firebase user table")
	}
	s.logger.GetLogger().Info("userRecords from firebase", zap.Any("firebaseUserRecords", firebaseUserRecords))
	err = s.firebase.DeleteAuthUser(ctx, firebaseUserRecords.UID)
	if err != nil {
		s.logger.GetLogger().Error("failed to delete auth user from firebase", zap.String("userID", userID.String()), zap.Error(err))
		return errors.New("failed to delete auth user from firebase")
	}
	err = s.repo.DeleteUserByID(ctx, userID)
	if err != nil {
		s.logger.GetLogger().Error("failed to delete user by ID", zap.String("userID", userID.String()), zap.Error(err))
		return err
	}
	s.logger.GetLogger().Info("user deleted successfully", zap.String("userID", userID.String()))
	return nil
}

func (s *userServiceStruct) GetEmployeesWithFilters(ctx context.Context, filter EmployeeFilter) ([]EmployeeResponseModel, error) {
	s.logger.GetLogger().Info("fetching employees with filters", zap.Any("filter", filter))
	employees, err := s.repo.GetFilteredEmployeesWithAssets(ctx, filter)
	if err != nil {
		s.logger.GetLogger().Error("failed to get filtered employees with assets", zap.Error(err))
		return nil, err
	}
	s.logger.GetLogger().Info("successfully fetched employees with filters", zap.Int("count", len(employees)))
	return employees, nil
}

func (s *userServiceStruct) GetEmployeeTimeline(ctx context.Context, userID uuid.UUID) ([]UserTimelineRes, error) {
	s.logger.GetLogger().Info("fetching employee timeline", zap.String("userID", userID.String()))
	timeline, err := s.repo.GetUserAssetTimeline(ctx, userID)
	if err != nil {
		s.logger.GetLogger().Error("failed to get user asset timeline", zap.String("userID", userID.String()), zap.Error(err))
		return nil, err
	}
	s.logger.GetLogger().Info("successfully fetched employee timeline", zap.String("userID", userID.String()), zap.Int("timelineEvents", len(timeline)))
	return timeline, nil
}

func (s *userServiceStruct) PublicRegister(ctx context.Context, req PublicUserReq) (uuid.UUID, string, error) {
	s.logger.GetLogger().Info("starting public registration service", zap.String("email", req.Email))

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		s.logger.GetLogger().Error("failed to begin transaction for PublicRegister", zap.Error(err))
		return uuid.Nil, "", err
	}
	defer func() {
		if r := recover(); r != nil {
			s.logger.GetLogger().Error("panic recovered in PublicRegister", zap.Any("recover_info", r))
			tx.Rollback()
		} else if err != nil {
			s.logger.GetLogger().Error("rolling back transaction for PublicRegister due to error", zap.Error(err))
			tx.Rollback()
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				s.logger.GetLogger().Error("failed to commit transaction for PublicRegister", zap.Error(commitErr))
			} else {
				s.logger.GetLogger().Info("transaction committed successfully for PublicRegister")
			}
		}
	}()

	//validate email
	splitEmail := strings.Split(req.Email, "@")
	if len(splitEmail) != 2 || splitEmail[1] != "remotestate.com" {
		s.logger.GetLogger().Warn("Invalid email domain for public registration", zap.String("email", req.Email))
		return uuid.Nil, "", errors.New("only remotestate.com domain is valid")
	}

	//extract username from email
	usernameParts := strings.Split(splitEmail[0], ".")
	if len(usernameParts) != 2 || usernameParts[0] == "" || usernameParts[1] == "" {
		s.logger.GetLogger().Warn("Invalid email format for username extraction in PublicRegister", zap.String("email", req.Email))
		return uuid.Nil, "", errors.New("invalid email format for username")
	}
	username := usernameParts[0] + " " + usernameParts[1]
	s.logger.GetLogger().Debug("Parsed username from email", zap.String("username", username))

	//checking firebase db
	firebaseUID, err := s.firebase.GetAuthUserID(ctx, req.Email)
	if err != nil && !firebaseauth.IsUserNotFound(err) {
		s.logger.GetLogger().Error("Failed to check Firebase user", zap.String("email", req.Email), zap.Error(err))
		return uuid.Nil, "", fmt.Errorf("firebase lookup failed: %w", err)
	}
	if firebaseUID != "" {
		s.logger.GetLogger().Warn("User already exists in Firebase", zap.String("firebaseUID", firebaseUID))
		return uuid.Nil, "", errors.New("user already exists in Firebase")
	}

	//create user in Firebase
	firebaseUserRecord, err := s.firebase.CreateUser(ctx, req.Email)
	if err != nil {
		s.logger.GetLogger().Error("Failed to create user in Firebase", zap.Error(err))
		return uuid.Nil, "", fmt.Errorf("firebase creation failed: %w", err)
	}
	s.logger.GetLogger().Info("Firebase user created", zap.String("firebaseUID", firebaseUserRecord.UID))

	//check if user already exist in our db
	exists, err := s.repo.IsUserExists(ctx, tx, req.Email)
	if err != nil {
		s.logger.GetLogger().Error("Failed to check if user exists in PublicRegister", zap.Error(err))
		return uuid.Nil, "", err
	}
	if exists {
		s.logger.GetLogger().Warn("User already registered during public registration attempt in postgresSQL database", zap.String("email", req.Email))
		return uuid.Nil, "", errors.New("email already registered in postgresSQL database")
	}

	// Insert user into your DB
	userID, err := s.repo.InsertIntoUser(ctx, tx, username, req.Email, firebaseUserRecord.UID)
	if err != nil {
		s.logger.GetLogger().Error("Failed to insert into users table during PublicRegister", zap.Error(err))
		return uuid.Nil, "", err
	}
	s.logger.GetLogger().Info("New user inserted into users table", zap.String("userID", userID.String()))

	if err = s.repo.InsertIntoUserRole(ctx, tx, userID, "employee", userID); err != nil {
		s.logger.GetLogger().Error("Failed to insert user role during PublicRegister", zap.Error(err), zap.String("userID", userID.String()))
		return uuid.Nil, "", err
	}
	s.logger.GetLogger().Debug("Assigned user role 'employee'", zap.String("userID", userID.String()))

	if err = s.repo.InsertIntoUserType(ctx, tx, userID, "full_time", userID); err != nil {
		s.logger.GetLogger().Error("Failed to insert user type during PublicRegister", zap.Error(err), zap.String("userID", userID.String()))
		return uuid.Nil, "", err
	}
	s.logger.GetLogger().Debug("Assigned user type 'full_time'", zap.String("userID", userID.String()))

	s.logger.GetLogger().Info("Public registration completed successfully", zap.String("userID", userID.String()))
	return userID, firebaseUserRecord.UID, nil
}

//func (s *userService) PublicRegister(ctx context.Context, req PublicUserReq) (uuid.UUID, error) {
//	s.logger.GetLogger().Info("starting public registration service", zap.String("email", req.Email))
//	tx, err := s.db.BeginTxx(ctx, nil)
//	if err != nil {
//		s.logger.GetLogger().Error("failed to begin transaction for PublicRegister", zap.Error(err))
//		return uuid.Nil, err
//	}
//	defer func() {
//		if r := recover(); r != nil {
//			s.logger.GetLogger().Error("panic recovered in PublicRegister", zap.Any("recover_info", r))
//			tx.Rollback()
//		} else if err != nil {
//			s.logger.GetLogger().Error("rolling back transaction for PublicRegister due to error", zap.Error(err))
//			tx.Rollback()
//		} else {
//			if commitErr := tx.Commit(); commitErr != nil {
//				s.logger.GetLogger().Error("failed to commit transaction for PublicRegister", zap.Error(commitErr))
//			} else {
//				s.logger.GetLogger().Info("transaction committed successfully for PublicRegister")
//			}
//		}
//	}()
//
//	splitEmail := strings.Split(req.Email, "@")
//	if len(splitEmail) != 2 || splitEmail[1] != "remotestate.com" {
//		s.logger.GetLogger().Warn("Invalid email domain for public registration", zap.String("email", req.Email))
//		return uuid.Nil, errors.New("only remotestate.com domain is valid")
//	}
//
//	usernameParts := strings.Split(splitEmail[0], ".")
//	if len(usernameParts) != 2 || usernameParts[0] == "" || usernameParts[1] == "" {
//		s.logger.GetLogger().Warn("Invalid email format for username extraction in PublicRegister", zap.String("email", req.Email))
//		return uuid.Nil, errors.New("invalid email format for username")
//	}
//	username := usernameParts[0] + " " + usernameParts[1]
//
//	s.logger.GetLogger().Debug("Parsed username from email", zap.String("username", username))
//
//	exists, err := s.repo.IsUserExists(ctx, tx, req.Email)
//	if err != nil {
//		s.logger.GetLogger().Error("Failed to check if user exists in PublicRegister", zap.Error(err))
//		return uuid.Nil, err
//	}
//	if exists {
//		s.logger.GetLogger().Warn("User already registered during public registration attempt", zap.String("email", req.Email))
//		return uuid.Nil, errors.New("email already registered...")
//	}
//
//	userID, err := s.repo.InsertIntoUser(ctx, tx, username, req.Email)
//	if err != nil {
//		s.logger.GetLogger().Error("Failed to insert into users table during PublicRegister", zap.Error(err))
//		return uuid.Nil, err
//	}
//	s.logger.GetLogger().Info("New user inserted into users table", zap.String("userID", userID.String()))
//
//	if err = s.repo.InsertIntoUserRole(ctx, tx, userID, "employee", userID); err != nil {
//		s.logger.GetLogger().Error("Failed to insert user role during PublicRegister", zap.Error(err), zap.String("userID", userID.String()))
//		return uuid.Nil, err
//	}
//	s.logger.GetLogger().Debug("Assigned user role 'employee'", zap.String("userID", userID.String()))
//
//	if err = s.repo.InsertIntoUserType(ctx, tx, userID, "full_time", userID); err != nil {
//		s.logger.GetLogger().Error("Failed to insert user type during PublicRegister", zap.Error(err), zap.String("userID", userID.String()))
//		return uuid.Nil, err
//	}
//	s.logger.GetLogger().Debug("Assigned user type 'full_time'", zap.String("userID", userID.String()))
//	s.logger.GetLogger().Info("Public registration completed successfully", zap.String("userID", userID.String()))
//	return userID, nil
//}

func (s *userServiceStruct) RegisterEmployeeByManager(ctx context.Context, req ManagerRegisterReq, managerID uuid.UUID) (uuid.UUID, error) {
	s.logger.GetLogger().Info("Starting employee registration by manager", zap.String("managerID", managerID.String()), zap.String("employeeEmail", req.Email))

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		s.logger.GetLogger().Error("Failed to begin transaction for RegisterEmployeeByManager", zap.Error(err))
		return uuid.Nil, err
	}

	defer func() {
		if r := recover(); r != nil {
			s.logger.GetLogger().Error("Panic recovered during RegisterEmployeeByManager transaction", zap.Any("recover_info", r))
			_ = tx.Rollback()
		} else if err != nil {
			s.logger.GetLogger().Error("Rolling back transaction for RegisterEmployeeByManager due to error", zap.Error(err))
			_ = tx.Rollback()
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				s.logger.GetLogger().Error("Failed to commit transaction for RegisterEmployeeByManager", zap.Error(commitErr))
			} else {
				s.logger.GetLogger().Info("Transaction committed successfully for RegisterEmployeeByManager")
			}
		}
	}()

	// Create Firebase user
	userRecord, err := s.firebase.CreateUser(ctx, req.Email)
	if err != nil {
		s.logger.GetLogger().Error("Failed to create Firebase user", zap.Error(err))
		return uuid.Nil, fmt.Errorf("firebase user creation failed: %w", err)
	}
	s.logger.GetLogger().Info("Firebase user created", zap.String("firebaseUID", userRecord.UID))

	userID, err := s.repo.CreateNewEmployee(ctx, tx, req, managerID)
	if err != nil {
		s.logger.GetLogger().Error("Failed to create new employee in repository", zap.Error(err), zap.String("managerID", managerID.String()))
		return uuid.Nil, err
	}
	s.logger.GetLogger().Info("Employee registered successfully by manager", zap.String("managerID", managerID.String()), zap.String("employeeID", userID.String()))

	return userID, nil
}

func (s *userServiceStruct) UpdateEmployee(ctx context.Context, req UpdateEmployeeReq, managerID uuid.UUID) error {
	s.logger.GetLogger().Info("Attempting to update employee information")
	err := s.repo.UpdateEmployeeInfo(ctx, req, managerID)
	if err != nil {
		s.logger.GetLogger().Error("failed to update employee information in repository")
		return err
	}
	s.logger.GetLogger().Info("employee information updated successfully")
	return nil
}

func (s *userServiceStruct) GetDashboard(ctx context.Context, userID uuid.UUID) (UserDashboardRes, error) {
	s.logger.GetLogger().Info("Fetching user dashboard data", zap.String("userID", userID.String()))
	dashboard, err := s.repo.GetUserDashboardById(ctx, userID)
	if err != nil {
		s.logger.GetLogger().Error("Failed to get user dashboard by ID", zap.String("userID", userID.String()), zap.Error(err))
		return UserDashboardRes{}, err
	}
	s.logger.GetLogger().Info("Successfully fetched user dashboard data", zap.String("userID", userID.String()))
	return dashboard, nil
}

func (s *userServiceStruct) UserLogin(ctx context.Context, req PublicUserReq) (uuid.UUID, string, string, error) {
	s.logger.GetLogger().Info("Attempting user login", zap.String("email", req.Email))
	userID, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.GetLogger().Warn("Login failed: User not found for email", zap.String("email", req.Email))
			return uuid.Nil, "", "", errors.New("invalid email")
		}
		s.logger.GetLogger().Error("Failed to get user by email during login", zap.String("email", req.Email), zap.Error(err))
		return uuid.Nil, "", "", err
	}
	s.logger.GetLogger().Debug("User found for login", zap.String("userID", userID.String()))

	userRole, err := s.repo.GetUserRoleById(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.GetLogger().Error("Login failed: User exists but no role found", zap.String("userID", userID.String()), zap.Error(err))
			return uuid.Nil, "", "", errors.New("user role not found")
		}
		s.logger.GetLogger().Error("Failed to get user role by ID during login", zap.String("userID", userID.String()), zap.Error(err))
		return uuid.Nil, "", "", err
	}
	s.logger.GetLogger().Debug("User role retrieved for login", zap.String("userID", userID.String()), zap.String("role", userRole))

	//accessToken, err := middlewares.GenerateJWT(userID.String(), []string{userRole})
	accessToken, err := s.AuthMiddleware.GenerateJWT(userID.String(), []string{userRole})
	if err != nil {
		s.logger.GetLogger().Error("Failed to generate access token during login", zap.String("userID", userID.String()), zap.Error(err))
		return uuid.Nil, "", "", err
	}
	refreshToken, err := s.AuthMiddleware.GenerateRefreshToken(userID.String())
	if err != nil {
		s.logger.GetLogger().Error("Failed to generate refresh token during login", zap.String("userID", userID.String()), zap.Error(err))
		return uuid.Nil, "", "", err
	}
	s.logger.GetLogger().Info("User login successful, tokens generated", zap.String("userID", userID.String()))
	return userID, accessToken, refreshToken, nil
}

func (s *userServiceStruct) GoogleAuth(ctx context.Context, idToken string) (uuid.UUID, string, string, error) {
	s.logger.GetLogger().Info("Starting Google authentication process")
	token, err := s.repo.GetFirebase().VerifyIDToken(ctx, idToken)
	if err != nil {
		s.logger.GetLogger().Error("invalid ID token received during GoogleAuth", zap.Error(err))
		return uuid.Nil, "", "", fmt.Errorf("invalid id token: %w", err)
	}
	s.logger.GetLogger().Debug("firebase ID token verified successfully", zap.String("UID", token.UID))

	userRecord, err := s.repo.GetFirebase().GetUserByUID(ctx, token.UID)
	if err != nil {
		s.logger.GetLogger().Error("failed to get user record from Firebase", zap.String("UID", token.UID), zap.Error(err))
		return uuid.Nil, "", "", fmt.Errorf("failed to get user info from firebase: %w", err)
	}
	s.logger.GetLogger().Debug("User record retrieved from Firebase", zap.String("email", userRecord.Email), zap.String("displayName", userRecord.DisplayName))
	fmt.Println("user records data ::", userRecord.UID)
	email := userRecord.Email
	if email == "" {
		s.logger.GetLogger().Error("email not found in Firebase user record", zap.String("UID", token.UID))
		return uuid.Nil, "", "", fmt.Errorf("email not found in firebase database")
	}

	userID, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.GetLogger().Info("User not found in PostgreSQL, creating new user account", zap.String("email", email))
			name := userRecord.DisplayName
			userID, err = s.repo.CreateFirebaseUser(ctx, name, email)
			if err != nil {
				s.logger.GetLogger().Error("Failed to register new Firebase user in PostgreSQL", zap.String("email", email), zap.Error(err))
				return uuid.Nil, "", "", fmt.Errorf("failed to register user: %w", err)
			}
			s.logger.GetLogger().Info("new user created successfully via Google Auth", zap.String("userID", userID.String()))
		} else {
			s.logger.GetLogger().Error("failed to get user by email from PostgreSQL during GoogleAuth", zap.String("email", email), zap.Error(err))
			return uuid.Nil, "", "", err
		}
	} else {
		s.logger.GetLogger().Info("existing user found in PostgreSQL for Google Auth", zap.String("userID", userID.String()))
	}

	role, err := s.repo.GetUserRoleById(ctx, userID)
	if err != nil {
		s.logger.GetLogger().Error("failed to get user role from user_roles table during GoogleAuth", zap.String("userID", userID.String()), zap.Error(err))
		return uuid.Nil, "", "", fmt.Errorf("failed to get role: %w", err)
	}
	s.logger.GetLogger().Debug("user role retrieved for Google Auth", zap.String("userID", userID.String()), zap.String("role", role))

	accessToken, err := middlewares.GenerateJWT(userRecord.UID, []string{role})
	if err != nil {
		s.logger.GetLogger().Error("failed to generate access token for GoogleAuth", zap.String("userID", userID.String()), zap.Error(err))
		return uuid.Nil, "", "", err
	}
	s.logger.GetLogger().Debug("access token generated for GoogleAuth", zap.String("userID", userID.String()))

	refreshToken, err := middlewares.GenerateRefreshToken(userRecord.UID)
	if err != nil {
		s.logger.GetLogger().Error("failed to generate refresh token for GoogleAuth", zap.String("userID", userID.String()), zap.Error(err))
		return uuid.Nil, "", "", err
	}
	s.logger.GetLogger().Info("google based authentication completed successfully", zap.String("userID", userID.String()))
	return userID, accessToken, refreshToken, nil
}

func (s *userServiceStruct) CreateFirstAdmin() bool {
	const adminEmail = "systemadmin@remotestate.com"
	const adminUsername = "System Admin"
	const Role = "admin"
	const Type = "full_time"

	var isExist uuid.UUID
	err := s.db.Get(&isExist, `
		SELECT id FROM users 
		WHERE email = $1 AND archived_at IS NULL
	`, adminEmail)
	if err == nil {
		log.Println("user id already exist", isExist)
		return false
	}

	tx, err := s.db.Beginx()
	if err != nil {
		log.Println("transaction failed", err)
		return false
	}

	defer func() {
		if p := recover(); p != nil || err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	var adminID uuid.UUID
	err = tx.Get(&adminID, `
		INSERT INTO users (username, email)
		VALUES ($1, $2)
		RETURNING id
	`, adminUsername, adminEmail)
	if err != nil {
		log.Println("failed to create new admin", err)
		return false
	}

	_, err = tx.Exec(`
		INSERT INTO user_roles (role, user_id, created_by)
		VALUES ($1, $2, $2)
	`, Role, adminID)
	if err != nil {
		log.Println("failed to assign role", err)
		return false
	}

	_, err = tx.Exec(`
		INSERT INTO user_type (type, user_id, created_by)
		VALUES ($1, $2, $2)
	`, Type, adminID)
	if err != nil {
		log.Println("failed to assign user type", err)
		return false
	}
	log.Println("admin created", adminID)
	return true
}

type FirebaseRegistrationResponse struct {
	UserID      uuid.UUID
	FirebaseUID string
}

func (s *userServiceStruct) FirebaseUserRegistration(ctx context.Context, idToken string) (*FirebaseRegistrationResponse, error) {
	s.logger.GetLogger().Info("Starting Firebase user registration")

	//verify ID Token
	token, err := s.firebase.VerifyIDToken(ctx, idToken)
	if err != nil {
		s.logger.GetLogger().Error("Firebase token verification failed", zap.Error(err))
		return nil, errors.New("invalid firebase token")
	}

	firebaseUID := token.UID
	claims := token.Claims
	email, _ := claims["email"].(string)
	displayName, _ := claims["name"].(string)

	if email == "" || displayName == "" {
		s.logger.GetLogger().Warn("Missing email or display name in token")
		return nil, errors.New("cannot register without email or display name")
	}

	//create new fireabase recod if not exist
	userRecord, err := s.firebase.GetUserByUID(ctx, firebaseUID)
	if err != nil {
		if firebaseauth.IsUserNotFound(err) {
			s.logger.GetLogger().Info("User not found in Firebase, creating new user")
			userRecord, err = s.firebase.CreateUser(ctx, email)
			if err != nil {
				s.logger.GetLogger().Error("Failed to create Firebase user", zap.Error(err))
				return nil, fmt.Errorf("firebase user creation failed: %w", err)
			}
			s.logger.GetLogger().Info("Firebase user created", zap.String("firebaseUID", userRecord.UID))
		} else {
			s.logger.GetLogger().Error("Failed to fetch Firebase user", zap.Error(err))
			return nil, fmt.Errorf("firebase lookup failed: %w", err)
		}
	} else {
		s.logger.GetLogger().Info("User already exists in Firebase", zap.String("firebaseUID", userRecord.UID))
	}

	//check if user exists in DB
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		s.logger.GetLogger().Error("Failed to begin DB transaction", zap.Error(err))
		return nil, err
	}
	defer func() {
		if r := recover(); r != nil {
			s.logger.GetLogger().Error("Panic recovered in FirebaseUserRegistration", zap.Any("recover_info", r))
			_ = tx.Rollback()
		} else if err != nil {
			s.logger.GetLogger().Error("Rolling back transaction", zap.Error(err))
			_ = tx.Rollback()
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				s.logger.GetLogger().Error("Failed to commit transaction", zap.Error(commitErr))
			} else {
				s.logger.GetLogger().Info("Transaction committed successfully")
			}
		}
	}()

	exists, err := s.repo.IsUserExists(ctx, tx, email)
	if err != nil {
		s.logger.GetLogger().Error("Failed to check if user exists in DB", zap.Error(err))
		return nil, err
	}
	if exists {
		s.logger.GetLogger().Warn("User already exists in DB", zap.String("email", email))
		return nil, errors.New("user already exists in postgresSQL database")
	}

	//dont need this in case of firebase auth login
	username := displayName
	if username == "" {
		splitEmail := strings.Split(email, "@")
		if len(splitEmail) == 2 {
			usernameParts := strings.Split(splitEmail[0], ".")
			if len(usernameParts) == 2 {
				username = usernameParts[0] + " " + usernameParts[1]
			} else {
				username = splitEmail[0]
			}
		}
	}
	s.logger.GetLogger().Debug("Parsed username", zap.String("username", username))

	//insert user into DB
	userID, err := s.repo.InsertIntoUser(ctx, tx, username, email, userRecord.UID)
	if err != nil {
		s.logger.GetLogger().Error("Failed to insert user into DB", zap.Error(err))
		return nil, err
	}

	if err = s.repo.InsertIntoUserRole(ctx, tx, userID, "employee", userID); err != nil {
		s.logger.GetLogger().Error("Failed to assign user role", zap.Error(err))
		return nil, err
	}

	if err = s.repo.InsertIntoUserType(ctx, tx, userID, "full_time", userID); err != nil {
		s.logger.GetLogger().Error("Failed to assign user type", zap.Error(err))
		return nil, err
	}

	s.logger.GetLogger().Info("Firebase user registration successful", zap.String("userID", userID.String()))

	return &FirebaseRegistrationResponse{
		UserID:      userID,
		FirebaseUID: firebaseUID,
	}, nil
}
