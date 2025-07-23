package assethandler

import (
	"asset/middlewareprovider"
	"asset/models"
	"asset/utils"
	"context"
	"encoding/json"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"net/http"
	"strings"
)

type AssetService interface {
	AddAssetWithConfig(ctx context.Context, req models.AddAssetWithConfigReq, userID uuid.UUID) error
	AssignAsset(ctx context.Context, assetID, userID, managerUUID uuid.UUID) error
	DeleteAsset(ctx context.Context, assetID uuid.UUID) error
	GetAllAssetsWithFilters(ctx context.Context, filter models.AssetFilter) ([]models.AssetWithConfigRes, error)
	GetAssetTimeline(ctx context.Context, assetID uuid.UUID) ([]models.AssetTimelineEvent, error)
	ReceiveAssetFromService(ctx context.Context, assetID uuid.UUID) error
	RetrieveAsset(ctx context.Context, req models.AssetReturnReq) error
	SendAssetToService(ctx context.Context, req models.AssetServiceReq, managerID uuid.UUID) error
	UpdateAsset(ctx context.Context, req models.UpdateAssetReq) error
	UpdateAssetWithConfig(ctx context.Context, req models.UpdateAssetReq) error
}

type AssetHandler struct {
	Service        AssetService
	AuthMiddleware middlewareprovider.AuthMiddlewareService
}

func NewAssetHandler(service AssetService, auth middlewareprovider.AuthMiddlewareService) *AssetHandler {
	return &AssetHandler{
		Service:        service,
		AuthMiddleware: auth,
	}
}

func (h *AssetHandler) AddNewAssetWithConfig(w http.ResponseWriter, r *http.Request) {
	userIDStr, _, err := h.AuthMiddleware.GetUserAndRolesFromContext(r)
	if err != nil {
		utils.RespondError(w, http.StatusUnauthorized, err, "unauthorized user")
		return
	}

	var req models.AddAssetWithConfigReq
	if err := utils.ParseJSONBody(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "invalid asset input")
		return
	}

	if err := validator.New().Struct(req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "validation error")
		return
	}

	userID, _ := uuid.Parse(userIDStr)

	err = h.Service.AddAssetWithConfig(r.Context(), req, userID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "failed to add asset")
		return
	}

	utils.RespondJSON(w, http.StatusCreated, map[string]interface{}{
		"msg":   "Asset and configuration created successfully",
		"asset": req,
	})
}

func (h *AssetHandler) AssignAssetToUser(w http.ResponseWriter, r *http.Request) {
	managerID, roles, err := h.AuthMiddleware.GetUserAndRolesFromContext(r)
	if err != nil || (roles[0] != "admin" && roles[0] != "asset_manager") {
		utils.RespondError(w, http.StatusForbidden, err, "permission denied")
		return
	}

	var req models.AssetAssignReq
	if err := utils.ParseJSONBody(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "invalid request body")
		return
	}

	assetID, _ := uuid.Parse(req.AssetID)
	userID, _ := uuid.Parse(req.UserID)
	managerUUID, _ := uuid.Parse(managerID)

	err = h.Service.AssignAsset(r.Context(), assetID, userID, managerUUID)
	if err != nil {
		if strings.Contains(err.Error(), "already assigned") {
			utils.RespondError(w, http.StatusConflict, err, "asset already assigned")
			return
		}
		utils.RespondError(w, http.StatusInternalServerError, err, "failed to assign asset")
		return
	}

	utils.RespondJSON(w, http.StatusCreated, map[string]interface{}{
		"message":     "asset assigned successfully",
		"user_id":     userID,
		"asset_id":    assetID,
		"assigned_by": managerUUID,
	})
}

func (h *AssetHandler) DeleteAsset(w http.ResponseWriter, r *http.Request) {
	_, roles, err := h.AuthMiddleware.GetUserAndRolesFromContext(r)
	if err != nil || (roles[0] != "admin" && roles[0] != "asset_manager") {
		utils.RespondError(w, http.StatusForbidden, err, "permission denied")
		return
	}

	assetIDStr := r.URL.Query().Get("asset_id")
	assetID, err := uuid.Parse(assetIDStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "invalid asset id")
		return
	}

	err = h.Service.DeleteAsset(r.Context(), assetID)
	if err != nil {
		if err.Error() == "asset currently assigned to a user" {
			utils.RespondError(w, http.StatusConflict, err, "asset is currently assigned")
			return
		}
		utils.RespondError(w, http.StatusInternalServerError, err, "failed to delete asset")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]string{"message": "asset deleted successfully"})
}

func (h *AssetHandler) GetAllAssetsWithFilters(w http.ResponseWriter, r *http.Request) {
	_, roles, err := h.AuthMiddleware.GetUserAndRolesFromContext(r)
	if err != nil || (roles[0] != "admin" && roles[0] != "asset_manager") {
		utils.RespondError(w, http.StatusForbidden, err, "permission denied")
		return
	}

	var filter models.AssetFilter
	filter.SearchText = r.URL.Query().Get("search")
	if filter.SearchText != "" {
		filter.IsSearchText = true
		filter.SearchText = "%" + filter.SearchText + "%"
	}
	if val := r.URL.Query().Get("status"); val != "" {
		filter.Status = strings.Split(val, ",")
	}
	if val := r.URL.Query().Get("owned_by"); val != "" {
		filter.OwnedBy = strings.Split(val, ",")
	}
	if val := r.URL.Query().Get("type"); val != "" {
		filter.Type = strings.Split(val, ",")
	}

	filter.Limit, filter.Offset = utils.GetPageLimitAndOffset(r)

	assets, err := h.Service.GetAllAssetsWithFilters(r.Context(), filter)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "failed to fetch records")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{"assets": assets})
}

func (h *AssetHandler) GetAssetTimeline(w http.ResponseWriter, r *http.Request) {
	_, _, err := h.AuthMiddleware.GetUserAndRolesFromContext(r)
	if err != nil {
		utils.RespondError(w, http.StatusUnauthorized, err, "unauthorized")
		return
	}

	assetIDStr := r.URL.Query().Get("asset_id")
	assetID, err := uuid.Parse(assetIDStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "invalid asset id")
		return
	}

	timeline, err := h.Service.GetAssetTimeline(r.Context(), assetID)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "failed to fetch asset timeline")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"asset_id": assetID,
		"timeline": timeline,
	})
}

func (h *AssetHandler) ReceivedFromService(w http.ResponseWriter, r *http.Request) {
	_, roles, err := h.AuthMiddleware.GetUserAndRolesFromContext(r)
	if err != nil || (roles[0] != "admin" && roles[0] != "asset_manager") {
		utils.RespondError(w, http.StatusForbidden, err, "permission denied")
		return
	}

	assetIDStr := r.URL.Query().Get("asset_id")
	assetID, err := uuid.Parse(assetIDStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "invalid asset ID")
		return
	}

	err = h.Service.ReceiveAssetFromService(r.Context(), assetID)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error())
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "Asset received",
		"asset_id": assetID,
	})
}

func (h *AssetHandler) RetrieveAsset(w http.ResponseWriter, r *http.Request) {
	_, roles, err := h.AuthMiddleware.GetUserAndRolesFromContext(r)
	if err != nil || (roles[0] != "admin" && roles[0] != "asset_manager") {
		utils.RespondError(w, http.StatusForbidden, err, "permission denied")
		return
	}

	var req models.AssetReturnReq
	if err := utils.ParseJSONBody(r, &req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "invalid request body")
		return
	}

	err = h.Service.RetrieveAsset(r.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "no matching asset assignment found") {
			utils.RespondError(w, http.StatusNotFound, err, "no such asset or already returned")
			return
		}
		utils.RespondError(w, http.StatusInternalServerError, err, "failed to retrieve asset")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]string{"message": "asset returned successfully"})
}

func (h *AssetHandler) SendAssetToService(w http.ResponseWriter, r *http.Request) {
	managerIDStr, roles, err := h.AuthMiddleware.GetUserAndRolesFromContext(r)
	if err != nil || (roles[0] != "admin" && roles[0] != "asset_manager") {
		utils.RespondError(w, http.StatusForbidden, err, "permission denied")
		return
	}

	var req models.AssetServiceReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "invalid request body")
		return
	}

	if err := validator.New().Struct(req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "validation failed")
		return
	}

	managerID, _ := uuid.Parse(managerIDStr)

	if err := h.Service.SendAssetToService(r.Context(), req, managerID); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, err.Error())
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]string{"message": "asset sent for servicing"})
}

func (h *AssetHandler) UpdateAssetWithConfig(w http.ResponseWriter, r *http.Request) {
	var req models.UpdateAssetReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "invalid request body")
		return
	}

	err := h.Service.UpdateAsset(r.Context(), req)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "failed to update asset")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]string{"message": "asset updated successfully"})
}

func (h *AssetHandler) UpdateAssetWithConfigHandler(w http.ResponseWriter, r *http.Request) {
	var req models.UpdateAssetReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err, "invalid request")
		return
	}

	err := h.Service.UpdateAssetWithConfig(r.Context(), req)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "failed to update asset")
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "asset updated successfully",
	})
}
