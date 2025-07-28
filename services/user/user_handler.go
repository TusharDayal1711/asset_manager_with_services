package userservice

import (
	"asset/providers"
	"asset/utils"
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
	"net/http"
	"strings"
)

type UserHandler struct {
	Service        UserService
	AuthMiddleware providers.AuthMiddlewareService
	Logger         providers.ZapLoggerProvider
}

func NewUserHandler(service UserService, auth providers.AuthMiddlewareService, log providers.ZapLoggerProvider) *UserHandler {
	return &UserHandler{
		Service:        service,
		AuthMiddleware: auth,
		Logger:         log,
	}
}

func (h *UserHandler) ChangeUserRole(w http.ResponseWriter, r *http.Request) {
	h.Logger.GetLogger().Info("ChangeUserRole request received")
	adminID, roles, err := h.AuthMiddleware.GetUserAndRolesFromContext(r)
	if err != nil {
		h.Logger.GetLogger().Error("Unauthorized access attempt in ChangeUserRole", zap.Error(err))
		utils.RespondError(w, http.StatusUnauthorized, err, "unauthorized")
		return
	}
	if len(roles) == 0 || roles[0] != "admin" {
		h.Logger.GetLogger().Warn("Forbidden access attempt in ChangeUserRole", zap.String("adminID", adminID), zap.Any("roles", roles))
		utils.RespondError(w, http.StatusForbidden, fmt.Errorf("unauthorized"), "only admin can update roles")
		return
	}

	var req UpdateUserRoleReq
	if err := utils.ParseJSONBody(r, &req); err != nil {
		h.Logger.GetLogger().Error("Invalid request body in ChangeUserRole", zap.Error(err))
		utils.RespondError(w, http.StatusBadRequest, err, "invalid request body")
		return
	}
	if err := validator.New().Struct(req); err != nil {
		h.Logger.GetLogger().Error("Invalid role input in ChangeUserRole", zap.Error(err))
		utils.RespondError(w, http.StatusBadRequest, err, "invalid role input")
		return
	}

	adminUUID, err := uuid.Parse(adminID)
	if err != nil {
		h.Logger.GetLogger().Error("Failed to parse adminID in ChangeUserRole", zap.String("adminID", adminID), zap.Error(err))
		utils.RespondError(w, http.StatusInternalServerError, err, "internal server error")
		return
	}

	h.Logger.GetLogger().Info("Attempting to change user role", zap.String("targetUserID", req.UserID), zap.String("newRole", req.Role), zap.String("adminID", adminID))
	err = h.Service.ChangeUserRole(r.Context(), req, adminUUID)
	if err != nil {
		h.Logger.GetLogger().Error("Failed to change user role", zap.Error(err))
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error())
		return
	}
	h.Logger.GetLogger().Info("User role changed successfully", zap.String("targetUserID", req.UserID))
	w.WriteHeader(http.StatusOK)
	jsoniter.NewEncoder(w).Encode(map[string]string{"message": "user role changed successfully"})
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	h.Logger.GetLogger().Info("DeleteUser request received")
	_, roles, err := h.AuthMiddleware.GetUserAndRolesFromContext(r)
	if err != nil {
		h.Logger.GetLogger().Error("Unauthorized access attempt in DeleteUser", zap.Error(err))
		utils.RespondError(w, http.StatusUnauthorized, err, "unauthorized")
		return
	}
	if len(roles) == 0 || (roles[0] != "admin" && roles[0] != "asset_manager") {
		h.Logger.GetLogger().Warn("Forbidden access attempt in DeleteUser", zap.Any("roles", roles))
		utils.RespondError(w, http.StatusForbidden, nil, "only admin and asset manager can delete users")
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		h.Logger.GetLogger().Error("Missing user_id in DeleteUser request")
		utils.RespondError(w, http.StatusBadRequest, fmt.Errorf("user_id is required"), "invalid user id")
		return
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		h.Logger.GetLogger().Error("Invalid user ID format in DeleteUser", zap.String("userID", userID), zap.Error(err))
		utils.RespondError(w, http.StatusBadRequest, err, "invalid user id")
		return
	}
	h.Logger.GetLogger().Info("Attempting to delete user", zap.String("userID", userID), zap.String("initiatingRole", roles[0]))
	err = h.Service.DeleteUser(r.Context(), userUUID, roles[0])
	if err != nil {
		h.Logger.GetLogger().Error("Failed to delete user", zap.String("userID", userID), zap.Error(err))
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error())
		return
	}
	h.Logger.GetLogger().Info("User deleted successfully", zap.String("userID", userID))
	w.WriteHeader(http.StatusOK)
	jsoniter.NewEncoder(w).Encode(map[string]string{"message": "user deleted successfully"})
}

func (h *UserHandler) GetEmployeesWithFilters(w http.ResponseWriter, r *http.Request) {
	h.Logger.GetLogger().Info("GetEmployeesWithFilters request received")
	_, roles, err := h.AuthMiddleware.GetUserAndRolesFromContext(r)
	if err != nil {
		h.Logger.GetLogger().Error("Unauthorized access attempt in GetEmployeesWithFilters", zap.Error(err))
		utils.RespondError(w, http.StatusUnauthorized, err, "unauthorized")
		return
	}
	if len(roles) == 0 || (roles[0] != "admin" && roles[0] != "employee_manager") {
		h.Logger.GetLogger().Warn("Forbidden access attempt in GetEmployeesWithFilters", zap.Any("roles", roles))
		utils.RespondError(w, http.StatusForbidden, nil, "only admin and employee manager can access")
		return
	}

	filter := EmployeeFilter{
		SearchText:   r.URL.Query().Get("search"),
		IsSearchText: r.URL.Query().Get("search") != "",
		Type:         strings.Split(r.URL.Query().Get("type"), ","),
		Role:         strings.Split(r.URL.Query().Get("role"), ","),
		AssetStatus:  strings.Split(r.URL.Query().Get("asset_status"), ","),
	}
	filter.Limit, filter.Offset = utils.GetPageLimitAndOffset(r)

	h.Logger.GetLogger().Debug("Fetching employees with filters", zap.Any("filter", filter))
	employees, err := h.Service.GetEmployeesWithFilters(r.Context(), filter)
	if err != nil {
		h.Logger.GetLogger().Error("Failed to fetch employee data with filters", zap.Error(err))
		utils.RespondError(w, http.StatusInternalServerError, err, "failed to fetch employee data")
		return
	}

	h.Logger.GetLogger().Info("Successfully fetched employees with filters", zap.Int("count", len(employees)))
	w.WriteHeader(http.StatusOK)
	jsoniter.NewEncoder(w).Encode(map[string]interface{}{"employees": employees})
}

func (h *UserHandler) GetEmployeeTimeline(w http.ResponseWriter, r *http.Request) {
	h.Logger.GetLogger().Info("GetEmployeeTimeline request received")
	_, _, err := h.AuthMiddleware.GetUserAndRolesFromContext(r)
	if err != nil {
		h.Logger.GetLogger().Error("Unauthorized access attempt in GetEmployeeTimeline", zap.Error(err))
		utils.RespondError(w, http.StatusUnauthorized, err, "unauthorized")
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		h.Logger.GetLogger().Error("Missing user_id in GetEmployeeTimeline request")
		utils.RespondError(w, http.StatusBadRequest, fmt.Errorf("user_id is required"), "invalid user id")
		return
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		h.Logger.GetLogger().Error("Invalid user ID format in GetEmployeeTimeline", zap.String("userID", userID), zap.Error(err))
		utils.RespondError(w, http.StatusBadRequest, err, "invalid user id")
		return
	}
	h.Logger.GetLogger().Debug("Fetching timeline for user", zap.String("userID", userID))
	timeline, err := h.Service.GetEmployeeTimeline(r.Context(), userUUID)
	if err != nil {
		h.Logger.GetLogger().Error("Failed to fetch timeline for user", zap.String("userID", userID), zap.Error(err))
		utils.RespondError(w, http.StatusInternalServerError, err, "failed to fetch timeline")
		return
	}
	h.Logger.GetLogger().Info("Successfully fetched employee timeline", zap.String("userID", userID))
	w.WriteHeader(http.StatusOK)
	jsoniter.NewEncoder(w).Encode(map[string]interface{}{"user_id": userID, "timeline": timeline})
}

func (h *UserHandler) PublicRegister(w http.ResponseWriter, r *http.Request) {
	h.Logger.GetLogger().Info("PublicRegister request received")
	var req PublicUserReq
	if err := utils.ParseJSONBody(r, &req); err != nil {
		h.Logger.GetLogger().Error("Error parsing request body in PublicRegister", zap.Error(err))
		utils.RespondError(w, http.StatusBadRequest, err, "invalid input")
		return
	}
	h.Logger.GetLogger().Debug("PublicRegister request body parsed", zap.String("email", req.Email))
	if err := validator.New().Struct(req); err != nil {
		h.Logger.GetLogger().Error("Invalid input in PublicRegister", zap.Error(err))
		utils.RespondError(w, http.StatusBadRequest, err, "invalid input")
		return
	}
	h.Logger.GetLogger().Info("Attempting public registration", zap.String("email", req.Email))
	userID, err := h.Service.PublicRegister(r.Context(), req)
	if err != nil {
		h.Logger.GetLogger().Error("Public registration failed", zap.String("email", req.Email), zap.Error(err))
		utils.RespondError(w, http.StatusBadRequest, err, err.Error())
		return
	}
	h.Logger.GetLogger().Info("Public registration successful", zap.String("userID", userID.String()))
	w.WriteHeader(http.StatusCreated)
	jsoniter.NewEncoder(w).Encode(map[string]interface{}{"message": "account created successfully", "userId": userID})
}

func (h *UserHandler) RegisterEmployeeByManager(w http.ResponseWriter, r *http.Request) {
	h.Logger.GetLogger().Info("RegisterEmployeeByManager request received")
	managerID, roles, err := h.AuthMiddleware.GetUserAndRolesFromContext(r)
	if err != nil {
		h.Logger.GetLogger().Error("Unauthorized access attempt in RegisterEmployeeByManager", zap.Error(err))
		utils.RespondError(w, http.StatusUnauthorized, err, "unauthorized")
		return
	}
	if len(roles) == 0 || (roles[0] != "admin" && roles[0] != "employee_manager") {
		h.Logger.GetLogger().Warn("Forbidden access attempt in RegisterEmployeeByManager", zap.String("managerID", managerID), zap.Any("roles", roles))
		utils.RespondError(w, http.StatusUnauthorized, fmt.Errorf("unauthorized role"), "unauthorized")
		return
	}

	var req ManagerRegisterReq
	if err := utils.ParseJSONBody(r, &req); err != nil {
		h.Logger.GetLogger().Error("Invalid input body in RegisterEmployeeByManager", zap.Error(err))
		utils.RespondError(w, http.StatusBadRequest, err, "invalid input body")
		return
	}
	if err := validator.New().Struct(req); err != nil {
		h.Logger.GetLogger().Error("Invalid input in RegisterEmployeeByManager", zap.Error(err))
		utils.RespondError(w, http.StatusBadRequest, err, "invalid input")
		return
	}
	managerUUID, err := uuid.Parse(managerID)
	if err != nil {
		h.Logger.GetLogger().Error("Failed to parse managerID in RegisterEmployeeByManager", zap.String("managerID", managerID), zap.Error(err))
		utils.RespondError(w, http.StatusInternalServerError, err, "internal server error")
		return
	}

	h.Logger.GetLogger().Info("Attempting to register employee by manager", zap.String("managerID", managerID), zap.String("employeeEmail", req.Email))
	userID, err := h.Service.RegisterEmployeeByManager(r.Context(), req, managerUUID)
	if err != nil {
		h.Logger.GetLogger().Error("Failed to register employee by manager", zap.String("managerID", managerID), zap.Error(err))
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error())
		return
	}
	h.Logger.GetLogger().Info("Employee registered successfully by manager", zap.String("managerID", managerID), zap.String("userID", userID.String()))
	w.WriteHeader(http.StatusCreated)
	jsoniter.NewEncoder(w).Encode(map[string]interface{}{"user created": userID})
}

func (h *UserHandler) UpdateEmployee(w http.ResponseWriter, r *http.Request) {
	h.Logger.GetLogger().Info("UpdateEmployee request received")
	managerID, roles, err := h.AuthMiddleware.GetUserAndRolesFromContext(r)
	if err != nil {
		h.Logger.GetLogger().Error("Unauthorized access attempt in UpdateEmployee", zap.Error(err))
		utils.RespondError(w, http.StatusUnauthorized, err, "unauthorized")
		return
	}
	if len(roles) == 0 || (roles[0] != "admin" && roles[0] != "employee_manager") {
		h.Logger.GetLogger().Warn("Forbidden access attempt in UpdateEmployee", zap.String("managerID", managerID), zap.Any("roles", roles))
		utils.RespondError(w, http.StatusUnauthorized, fmt.Errorf("unauthorized role"), "unauthorized")
		return
	}
	managerUUID, err := uuid.Parse(managerID)
	if err != nil {
		h.Logger.GetLogger().Error("Failed to parse managerID in UpdateEmployee", zap.String("managerID", managerID), zap.Error(err))
		utils.RespondError(w, http.StatusInternalServerError, err, "internal server error")
		return
	}

	var req UpdateEmployeeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.GetLogger().Error("Invalid request body in UpdateEmployee", zap.Error(err))
		utils.RespondError(w, http.StatusBadRequest, err, "invalid request body")
		return
	}
	if err := validator.New().Struct(req); err != nil {
		h.Logger.GetLogger().Error("Invalid input in UpdateEmployee", zap.Error(err))
		utils.RespondError(w, http.StatusBadRequest, err, "invalid input")
		return
	}
	if req.Username == "" && req.Email == "" && req.ContactNo == "" {
		h.Logger.GetLogger().Warn("No update fields provided in UpdateEmployee request")
		utils.RespondError(w, http.StatusBadRequest, nil, "at least one field must be provided for update")
		return
	}

	h.Logger.GetLogger().Info("Attempting to update employee")
	if err := h.Service.UpdateEmployee(r.Context(), req, managerUUID); err != nil {
		h.Logger.GetLogger().Error("Failed to update employee")
		utils.RespondError(w, http.StatusInternalServerError, err, "failed to update employee")
		return
	}
	h.Logger.GetLogger().Info("Employee updated successfully")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "employee updated successfully"})
}

func (h *UserHandler) UserLogin(w http.ResponseWriter, r *http.Request) {
	h.Logger.GetLogger().Info("UserLogin request received")
	var req PublicUserReq
	if err := utils.ParseJSONBody(r, &req); err != nil {
		h.Logger.GetLogger().Error("Invalid input in UserLogin (parsing body)", zap.Error(err))
		utils.RespondError(w, http.StatusBadRequest, err, "invalid input")
		return
	}
	if err := validator.New().Struct(req); err != nil {
		h.Logger.GetLogger().Error("Invalid input in UserLogin (validation)", zap.Error(err))
		utils.RespondError(w, http.StatusBadRequest, err, "invalid input")
		return
	}
	h.Logger.GetLogger().Info("Attempting user login", zap.String("email", req.Email))
	userID, accessToken, refreshToken, err := h.Service.UserLogin(r.Context(), req)
	if err != nil {
		h.Logger.GetLogger().Error("User login failed", zap.String("email", req.Email), zap.Error(err))
		utils.RespondError(w, http.StatusUnauthorized, err, err.Error())
		return
	}
	h.Logger.GetLogger().Info("User login successful", zap.String("userID", userID.String()))
	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":       userID,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

func (h *UserHandler) GetUserDashboard(w http.ResponseWriter, r *http.Request) {
	h.Logger.GetLogger().Info("GetUserDashboard request received")
	userID, _, err := h.AuthMiddleware.GetUserAndRolesFromContext(r)
	if err != nil {
		h.Logger.GetLogger().Error("Unauthorized access attempt in GetUserDashboard", zap.Error(err))
		utils.RespondError(w, http.StatusUnauthorized, err, "unauthorized")
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		h.Logger.GetLogger().Error("Invalid user ID format in GetUserDashboard", zap.String("userID", userID), zap.Error(err))
		utils.RespondError(w, http.StatusUnauthorized, err, "invalid user id")
		return
	}

	h.Logger.GetLogger().Debug("Fetching dashboard for user", zap.String("userID", userID))
	dashboard, err := h.Service.GetDashboard(r.Context(), userUUID)
	if err != nil {
		h.Logger.GetLogger().Error("Failed to fetch dashboard data", zap.String("userID", userID), zap.Error(err))
		utils.RespondError(w, http.StatusInternalServerError, err, "failed to fetch dashboard data")
		return
	}

	h.Logger.GetLogger().Info("Successfully fetched user dashboard", zap.String("userID", userID))
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dashboard)
}

func (h *UserHandler) GoogleAuth(w http.ResponseWriter, r *http.Request) {
	h.Logger.GetLogger().Info("GoogleAuth request received")
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		h.Logger.GetLogger().Warn("Missing bearer token in GoogleAuth request")
		utils.RespondError(w, http.StatusBadRequest, fmt.Errorf("missing bearer token"), "unauthorized")
		return
	}
	idToken := strings.TrimPrefix(authHeader, "Bearer ")
	h.Logger.GetLogger().Debug("Google authentication called")
	userID, accessToken, refreshToken, err := h.Service.GoogleAuth(r.Context(), idToken)
	if err != nil {
		h.Logger.GetLogger().Error("Google authentication failed", zap.Error(err))
		utils.RespondError(w, http.StatusUnauthorized, err, "google auth failed")
		return
	}

	h.Logger.GetLogger().Info("Google authentication successful", zap.String("userID", userID.String()))
	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":       userID,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

func (h *UserHandler) CreateAdmin(w http.ResponseWriter, r *http.Request) {
	h.Logger.GetLogger().Info("CreateAdmin request received")
	is := h.Service.CreateFirstAdmin()
	if !is {
		jsoniter.NewEncoder(w).Encode(map[string]string{"message": "failed to create admin created"})
	} else {
		jsoniter.NewEncoder(w).Encode(map[string]string{"message": "admin created successfully"})
	}
}
