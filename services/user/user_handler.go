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
}

func NewUserHandler(service UserService, auth providers.AuthMiddlewareService) *UserHandler {
	return &UserHandler{
		Service:        service,
		AuthMiddleware: auth,
	}
}

func (h *UserHandler) ChangeUserRole(w http.ResponseWriter, r *http.Request) {
	adminID, roles, err := h.AuthMiddleware.GetUserAndRolesFromContext(r)
	if err != nil {
		utils.RespondError(w, http.StatusUnauthorized, err, "unauthorized")
		return
	}
	if len(roles) == 0 || roles[0] != "admin" {
		utils.RespondError(w, http.StatusForbidden, fmt.Errorf("unauthorized"), "only admin can update roles")
		return
	}

	var req UpdateUserRoleReq
	if err := utils.ParseJSONBody(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "invalid request body")
		return
	}
	if err := validator.New().Struct(req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "invalid role input")
		return
	}

	adminUUID, _ := uuid.Parse(adminID)
	err = h.Service.ChangeUserRole(r.Context(), req, adminUUID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	jsoniter.NewEncoder(w).Encode(map[string]string{"message": "user role changed successfully"})
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	_, roles, err := h.AuthMiddleware.GetUserAndRolesFromContext(r)
	if err != nil {
		utils.RespondError(w, http.StatusUnauthorized, err, "unauthorized")
		return
	}
	if roles[0] != "admin" && roles[0] != "asset_manager" {
		utils.RespondError(w, http.StatusForbidden, nil, "only admin and asset manager can delete users")
		return
	}
	userID := r.URL.Query().Get("user_id")
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "invalid user id")
		return
	}
	err = h.Service.DeleteUser(r.Context(), userUUID, roles[0])
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	jsoniter.NewEncoder(w).Encode(map[string]string{"message": "user deleted successfully"})
}

func (h *UserHandler) GetEmployeesWithFilters(w http.ResponseWriter, r *http.Request) {
	_, roles, err := h.AuthMiddleware.GetUserAndRolesFromContext(r)
	if err != nil {
		utils.RespondError(w, http.StatusUnauthorized, err, "unauthorized")
		return
	}
	if roles[0] != "admin" && roles[0] != "employee_manager" {
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

	employees, err := h.Service.GetEmployeesWithFilters(r.Context(), filter)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "failed to fetch employee data")
		return
	}

	w.WriteHeader(http.StatusOK)
	jsoniter.NewEncoder(w).Encode(map[string]interface{}{"employees": employees})
}

func (h *UserHandler) GetEmployeeTimeline(w http.ResponseWriter, r *http.Request) {
	_, _, err := h.AuthMiddleware.GetUserAndRolesFromContext(r)
	if err != nil {
		utils.RespondError(w, http.StatusUnauthorized, err, "unauthorized")
		return
	}
	userID := r.URL.Query().Get("user_id")
	userUUID, err := uuid.Parse(userID)
	timeline, err := h.Service.GetEmployeeTimeline(r.Context(), userUUID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "failed to fetch timeline")
		return
	}
	w.WriteHeader(http.StatusOK)
	jsoniter.NewEncoder(w).Encode(map[string]interface{}{"user_id": userID, "timeline": timeline})
}

func (h *UserHandler) PublicRegister(w http.ResponseWriter, r *http.Request) {
	var req PublicUserReq
	if err := utils.ParseJSONBody(r, &req); err != nil {
		utils.Logger.Error("Failed to parse request body", zap.Error(err))
		utils.RespondError(w, http.StatusBadRequest, err, "invalid input")
		return
	}
	if err := validator.New().Struct(req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "invalid input")
		return
	}
	userID, err := h.Service.PublicRegister(r.Context(), req)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsoniter.NewEncoder(w).Encode(map[string]interface{}{"message": "account created successfully", "userId": userID})
}

func (h *UserHandler) RegisterEmployeeByManager(w http.ResponseWriter, r *http.Request) {
	managerID, roles, err := h.AuthMiddleware.GetUserAndRolesFromContext(r)
	if err != nil || (roles[0] != "admin" && roles[0] != "employee_manager") {
		utils.RespondError(w, http.StatusUnauthorized, err, "unauthorized")
		return
	}
	var req ManagerRegisterReq
	if err := utils.ParseJSONBody(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "invalid input body")
		return
	}
	if err := validator.New().Struct(req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "invalid input")
		return
	}
	managerUUID, _ := uuid.Parse(managerID)
	userID, err := h.Service.RegisterEmployeeByManager(r.Context(), req, managerUUID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, err.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsoniter.NewEncoder(w).Encode(map[string]interface{}{"user created": userID})
}

func (h *UserHandler) UpdateEmployee(w http.ResponseWriter, r *http.Request) {
	managerID, roles, err := h.AuthMiddleware.GetUserAndRolesFromContext(r)
	if err != nil || (roles[0] != "admin" && roles[0] != "employee_manager") {
		utils.RespondError(w, http.StatusUnauthorized, err, "unauthorized")
		return
	}
	managerUUID, _ := uuid.Parse(managerID)

	var req UpdateEmployeeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "invalid request body")
		return
	}
	if err := validator.New().Struct(req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "invalid input")
		return
	}
	if req.Username == "" && req.Email == "" && req.ContactNo == "" {
		utils.RespondError(w, http.StatusBadRequest, nil, "at least one field must be provided for update")
		return
	}
	if err := h.Service.UpdateEmployee(r.Context(), req, managerUUID); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "failed to update employee")
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "employee updated successfully"})
}

func (h *UserHandler) UserLogin(w http.ResponseWriter, r *http.Request) {
	var req PublicUserReq
	if err := utils.ParseJSONBody(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "invalid input")
		return
	}
	if err := validator.New().Struct(req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "invalid input")
		return
	}
	userID, accessToken, refreshToken, err := h.Service.UserLogin(r.Context(), req)
	if err != nil {
		utils.RespondError(w, http.StatusUnauthorized, err, err.Error())
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":       userID,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

func (h *UserHandler) GetUserDashboard(w http.ResponseWriter, r *http.Request) {
	userID, _, err := h.AuthMiddleware.GetUserAndRolesFromContext(r)
	if err != nil {
		utils.RespondError(w, http.StatusUnauthorized, err, "unauthorized")
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		utils.RespondError(w, http.StatusUnauthorized, err, "invalid user id")
		return
	}

	dashboard, err := h.Service.GetDashboard(r.Context(), userUUID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "failed to fetch dashboard data")
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dashboard)
}
