package assetservice

import (
	"asset/models"
	"context"
	"encoding/json"

	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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

type assetService struct {
	repo AssetRepository
	db   *sqlx.DB
}

func NewAssetService(repo AssetRepository, db *sqlx.DB) AssetService {
	return &assetService{repo: repo, db: db}
}

func (s *assetService) AddAssetWithConfig(ctx context.Context, req models.AddAssetWithConfigReq, addedBy uuid.UUID) (err error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
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

	assetID, err := s.repo.AddAsset(ctx, tx, req, addedBy)
	if err != nil {
		return fmt.Errorf("failed to add asset: %w", err)
	}

	switch req.Type {
	case "laptop":
		var cfg models.Laptop_config_req
		if err = json.Unmarshal(req.Config, &cfg); err != nil {
			return fmt.Errorf("invalid laptop config: %w", err)
		}
		err = s.repo.AddLaptopConfig(ctx, tx, cfg, assetID)
	case "mouse":
		var cfg models.Mouse_config_req
		if err = json.Unmarshal(req.Config, &cfg); err != nil {
			return fmt.Errorf("invalid mouse config: %w", err)
		}
		err = s.repo.AddMouseConfig(ctx, tx, cfg, assetID)
	case "monitor":
		var cfg models.Monitor_config_req
		if err = json.Unmarshal(req.Config, &cfg); err != nil {
			return fmt.Errorf("invalid monitor config: %w", err)
		}
		err = s.repo.AddMonitorConfig(ctx, tx, cfg, assetID)
	case "hard_disk":
		var cfg models.Hard_disk_config_req
		if err = json.Unmarshal(req.Config, &cfg); err != nil {
			return fmt.Errorf("invalid hard disk config: %w", err)
		}

		err = s.repo.AddHardDiskConfig(ctx, tx, cfg, assetID)
	case "pen_drive":
		var cfg models.Pen_drive_config_req
		if err = json.Unmarshal(req.Config, &cfg); err != nil {
			return fmt.Errorf("invalid pen drive config: %w", err)
		}

		err = s.repo.AddPenDriveConfig(ctx, tx, cfg, assetID)
	case "mobile":
		var cfg models.Mobile_config_req
		if err = json.Unmarshal(req.Config, &cfg); err != nil {
			return fmt.Errorf("invalid mobile config: %w", err)
		}

		err = s.repo.AddMobileConfig(ctx, tx, cfg, assetID)
	case "sim":
		var cfg models.Sim_config_req
		if err = json.Unmarshal(req.Config, &cfg); err != nil {
			return fmt.Errorf("invalid sim config: %w", err)
		}

		err = s.repo.AddSimConfig(ctx, tx, cfg, assetID)
	case "accessory":
		var cfg models.Accessories_config_req
		if err = json.Unmarshal(req.Config, &cfg); err != nil {
			return fmt.Errorf("invalid accessory config: %w", err)
		}

		err = s.repo.AddAccessoryConfig(ctx, tx, cfg, assetID)
	default:
		return errors.New("unsupported asset type")
	}

	if err != nil {
		return fmt.Errorf("failed to add asset configuration: %w", err)
	}
	return nil
}

func (s *assetService) AssignAsset(ctx context.Context, assetID, employeeID, managerID uuid.UUID) (err error) {

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
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

	err = s.repo.AssignAssetByID(ctx, tx, assetID, employeeID, managerID)
	if err != nil {
		return fmt.Errorf("failed to assign asset: %w", err)
	}
	return nil
}

func (s *assetService) DeleteAsset(ctx context.Context, assetID uuid.UUID) error {

	return s.repo.DeleteAssetByID(ctx, assetID)
}

func (s *assetService) GetAllAssets(ctx context.Context, filter models.AssetFilter) ([]models.AssetWithConfigRes, error) {

	return s.repo.SearchAssetsWithFilter(ctx, filter)
}

func (s *assetService) GetAssetTimeline(ctx context.Context, assetID uuid.UUID) ([]models.AssetTimelineEvent, error) {
	return s.repo.GetAssetTimeline(ctx, assetID)
}

func (s *assetService) ReceiveAssetFromService(ctx context.Context, assetID uuid.UUID) error {
	return s.repo.RecivedAssetFromService(ctx, assetID)
}

func (s *assetService) RetrieveAsset(ctx context.Context, req models.AssetReturnReq) (err error) {

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
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

	err = s.repo.RetrieveAsset(ctx, tx, uuid.MustParse(req.AssetID), uuid.MustParse(req.EmployeeID), req.ReturnReason)
	if err != nil {
		return fmt.Errorf("failed to retrieve asset: %w", err)
	}
	return nil
}

func (s *assetService) SendAssetToService(ctx context.Context, req models.AssetServiceReq, managerID uuid.UUID) error {

	return s.repo.SendAssetForService(ctx, req, managerID)
}

func (s *assetService) UpdateAsset(ctx context.Context, req models.UpdateAssetReq) error {
	return s.repo.UpdateAssetWithConfig(ctx, req)
}

func (s *assetService) UpdateAssetWithConfig(ctx context.Context, req models.UpdateAssetReq) error {
	return s.repo.UpdateAssetWithConfig(ctx, req)
}

func (s *assetService) GetAllAssetsWithFilters(ctx context.Context, filter models.AssetFilter) ([]models.AssetWithConfigRes, error) {
	return s.repo.SearchAssetsWithFilter(ctx, filter)
}
