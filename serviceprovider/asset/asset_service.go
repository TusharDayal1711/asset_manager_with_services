package assetservice

import (
	"asset/models"
	"asset/repository/asset"
	"context" // Import context package
	"encoding/json"
	"errors"
	"fmt" // Import fmt for error formatting
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// AssetService defines the interface for asset-related business logic.
type AssetService interface {
	AddAssetWithConfig(ctx context.Context, req models.AddAssetWithConfigReq, addedBy uuid.UUID) error
	AssignAsset(ctx context.Context, assetID, employeeID, managerID uuid.UUID) error
	DeleteAsset(ctx context.Context, assetID uuid.UUID) error
	GetAllAssets(ctx context.Context, filter models.AssetFilter) ([]models.AssetWithConfigRes, error)
	GetAssetTimeline(ctx context.Context, assetID uuid.UUID) ([]models.AssetTimelineEvent, error)
	ReceiveAssetFromService(ctx context.Context, assetID uuid.UUID) error
	RetrieveAsset(ctx context.Context, req models.AssetReturnReq) error
	SendAssetToService(ctx context.Context, req models.AssetServiceReq, managerID uuid.UUID) error
	UpdateAsset(ctx context.Context, req models.UpdateAssetReq) error
	UpdateAssetWithConfig(ctx context.Context, req models.UpdateAssetReq) error
}

// assetService implements the AssetService interface.
type assetService struct {
	repo asset.AssetRepository
	db   *sqlx.DB
}

// NewAssetService creates a new AssetService.
func NewAssetService(repo asset.AssetRepository, db *sqlx.DB) AssetService {
	return &assetService{repo: repo, db: db}
}

// AddAssetWithConfig adds a new asset along with its configuration.
func (s *assetService) AddAssetWithConfig(ctx context.Context, req models.AddAssetWithConfigReq, addedBy uuid.UUID) (err error) {
	// Begin a new transaction with the provided context.
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Defer a rollback or commit based on the outcome of the function.
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // Re-throw panic after rollback
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// Pass context and transaction to the repository method.
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
		// Pass context and transaction.
		err = s.repo.AddLaptopConfig(ctx, tx, cfg, assetID)
	case "mouse":
		var cfg models.Mouse_config_req
		if err = json.Unmarshal(req.Config, &cfg); err != nil {
			return fmt.Errorf("invalid mouse config: %w", err)
		}
		// Pass context and transaction.
		err = s.repo.AddMouseConfig(ctx, tx, cfg, assetID)
	case "monitor":
		var cfg models.Monitor_config_req
		if err = json.Unmarshal(req.Config, &cfg); err != nil {
			return fmt.Errorf("invalid monitor config: %w", err)
		}
		// Pass context and transaction.
		err = s.repo.AddMonitorConfig(ctx, tx, cfg, assetID)
	case "hard_disk":
		var cfg models.Hard_disk_config_req
		if err = json.Unmarshal(req.Config, &cfg); err != nil {
			return fmt.Errorf("invalid hard disk config: %w", err)
		}
		// Pass context and transaction.
		err = s.repo.AddHardDiskConfig(ctx, tx, cfg, assetID)
	case "pen_drive":
		var cfg models.Pen_drive_config_req
		if err = json.Unmarshal(req.Config, &cfg); err != nil {
			return fmt.Errorf("invalid pen drive config: %w", err)
		}
		// Pass context and transaction.
		err = s.repo.AddPenDriveConfig(ctx, tx, cfg, assetID)
	case "mobile":
		var cfg models.Mobile_config_req
		if err = json.Unmarshal(req.Config, &cfg); err != nil {
			return fmt.Errorf("invalid mobile config: %w", err)
		}
		// Pass context and transaction.
		err = s.repo.AddMobileConfig(ctx, tx, cfg, assetID)
	case "sim":
		var cfg models.Sim_config_req
		if err = json.Unmarshal(req.Config, &cfg); err != nil {
			return fmt.Errorf("invalid sim config: %w", err)
		}
		// Pass context and transaction.
		err = s.repo.AddSimConfig(ctx, tx, cfg, assetID)
	case "accessory":
		var cfg models.Accessories_config_req
		if err = json.Unmarshal(req.Config, &cfg); err != nil {
			return fmt.Errorf("invalid accessory config: %w", err)
		}
		// Pass context and transaction.
		err = s.repo.AddAccessoryConfig(ctx, tx, cfg, assetID)
	default:
		return errors.New("unsupported asset type")
	}

	if err != nil {
		return fmt.Errorf("failed to add asset configuration: %w", err)
	}
	return nil // Transaction commit is handled by defer
}

// AssignAsset assigns an asset to an employee.
func (s *assetService) AssignAsset(ctx context.Context, assetID, employeeID, managerID uuid.UUID) (err error) {
	// Begin a new transaction with the provided context.
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Defer a rollback or commit based on the outcome of the function.
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

	// Pass context and transaction to the repository method.
	err = s.repo.AssignAssetByID(ctx, tx, assetID, employeeID, managerID)
	if err != nil {
		return fmt.Errorf("failed to assign asset: %w", err)
	}
	return nil // Transaction commit is handled by defer
}

// DeleteAsset deletes an asset.
func (s *assetService) DeleteAsset(ctx context.Context, assetID uuid.UUID) error {
	// Pass context to the repository method.
	return s.repo.DeleteAssetByID(ctx, assetID)
}

// GetAllAssets retrieves all assets with optional filters.
func (s *assetService) GetAllAssets(ctx context.Context, filter models.AssetFilter) ([]models.AssetWithConfigRes, error) {
	// Pass context to the repository method.
	return s.repo.SearchAssetsWithFilter(ctx, filter)
}

// GetAssetTimeline retrieves the timeline for a specific asset.
func (s *assetService) GetAssetTimeline(ctx context.Context, assetID uuid.UUID) ([]models.AssetTimelineEvent, error) {
	// Pass context to the repository method.
	return s.repo.GetAssetTimeline(ctx, assetID)
}

// ReceiveAssetFromService marks an asset as received from service.
func (s *assetService) ReceiveAssetFromService(ctx context.Context, assetID uuid.UUID) error {
	// Pass context to the repository method.
	return s.repo.RecivedAssetFromService(ctx, assetID)
}

// RetrieveAsset retrieves an asset from an employee.
func (s *assetService) RetrieveAsset(ctx context.Context, req models.AssetReturnReq) (err error) {
	// Begin a new transaction with the provided context.
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Defer a rollback or commit based on the outcome of the function.
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
	return nil // Transaction commit is handled by defer
}

func (s *assetService) SendAssetToService(ctx context.Context, req models.AssetServiceReq, managerID uuid.UUID) error {
	// Pass context to the repository method.
	return s.repo.SendAssetForService(ctx, req, managerID)
}

func (s *assetService) UpdateAsset(ctx context.Context, req models.UpdateAssetReq) error {
	return s.repo.UpdateAssetWithConfig(ctx, req)
}

func (s *assetService) UpdateAssetWithConfig(ctx context.Context, req models.UpdateAssetReq) error {
	return s.repo.UpdateAssetWithConfig(ctx, req)
}
